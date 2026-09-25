# Savor

Food delivery for Tbilisi, built as a small production-shaped platform: seven services, one database each, async checkout, and a three-role UI.

```
docker compose up --build
```

Then open [http://localhost](http://localhost).

| Account | Password | Role |
|---|---|---|
| `customer@savor.dev` | `savor1234` | Order food |
| `restaurant@savor.dev` | `savor1234` | Kitchen + menus (Nari, Soba Hour, Portico) |
| `kitchen@savor.dev` | `savor1234` | Kitchen (Ember, Dune, Green Room) |
| `courier@savor.dev` | `savor1234` | Deliveries |

Put `FAIL` in the delivery address to decline the mock charge.

## Demo path

1. Sign in as the customer. Add dishes from one kitchen. Checkout.
2. Watch the order page: `pending_payment` → `paid` (payment service, ~0.6s).
3. Sign in as the restaurant. Accept → cooking → ready.
4. Sign in as the courier, go online, pick up, deliver.
5. Mail lands in [Mailhog](http://localhost:8025). Traces in [Jaeger](http://localhost:16686). Metrics in [Grafana](http://localhost:3001) (`savor` / `savor`).

## Architecture

```
browser ──► Traefik :80 ──► Next.js
                         └► gateway (JWT, rate limit, WS)  /api  /ws
                                ├ identity     (Go)
                                ├ catalog      (NestJS + Redis)
                                ├ order        (Go, outbox)
                                ├ payment      (Go)
                                ├ courier      (NestJS)
                                └ notification (NestJS + SMTP)
```

Events travel on RabbitMQ topic `savor.events`. Order writes them through a transactional outbox so a crash after INSERT cannot drop `order.created`. Payment is idempotent on `orderId`. Courier assignment is a competing-consumer on `order.ready`.

Each service owns a Postgres database created by `infra/postgres/init.sql`. Internal ports are not published; only Traefik, the UIs, and observability ports are.

## Stack

- Go 1.23 — gateway, identity, order, payment
- NestJS 11 + Prisma — catalog, courier, notification
- Next.js 15 — customer, restaurant, courier in one app
- Postgres 16, Redis 7, RabbitMQ 4, MinIO, Mailhog
- Traefik 3, Prometheus, Grafana, Loki, Jaeger

## Why these service cuts

| Service | Exists because |
|---|---|
| identity | Auth is not a kitchen concern |
| catalog | Menus are read-heavy and cacheable |
| order | State machine + outbox belong in one place |
| payment | Charges must be idempotent and slow on purpose |
| courier | Shift + assignment is a different load pattern |
| notification | Side-effect fan-out should not sit in order |
| gateway | Browsers never talk to five origins |

## Repo

```
apps/web
services/{gateway,identity,order,payment,catalog,courier,notification}
contracts/{openapi,events}
infra/{postgres,prometheus,grafana,traefik,promtail}
```

```
make up        # compose up --build
make down-v    # wipe volumes and reseed
```

Copy `.env.example` to `.env` if you want to override secrets. Do not commit `.env`.

## Screenshots

### Customer Flow

Browse restaurants, add items to cart, and track your order in real-time.

![Customer browsing restaurants](docs/customer-browse.png)
![Customer cart](docs/customer-cart.png)
![Customer order tracking](docs/customer-order.png)

### Restaurant Management

Accept orders, manage your menu, and track preparation status.

![Restaurant order management](docs/restaurant-orders.png)
![Restaurant menu editor](docs/restaurant-menu.png)

### Courier Operations

Go online, accept deliveries, and complete drop-offs.

![Courier dashboard](docs/courier-dashboard.png)
![Courier active delivery](docs/courier-delivery.png)

> **Note:** Screenshots are captured from a local running instance. See [docs/SCREENSHOTS.md](docs/SCREENSHOTS.md) for the capture guide.

## Tests

The project includes automated tests for core business logic:

### Running Tests Locally

**NestJS services (catalog, courier, notification):**
```bash
cd services/catalog
npm install
npm test
```

**Go services (gateway, identity, order, payment):**
```bash
cd services/order
go test ./...

# Skip integration tests (require database):
go test -short ./...
```

### What's Tested

- **Catalog service**: Restaurant CRUD, dish management, ownership validation, cache invalidation
- **Courier service**: Delivery assignment, status transitions, pickup/delivery workflows
- **Order service**: Cart management, order creation, state machine transitions, outbox pattern
- **Payment service**: Idempotent charge processing, failure handling, order lookup

Integration tests run against a real Postgres database. Unit tests mock external dependencies.

## Deployment

### Single VPS Deployment

Deploy the entire stack to a single VPS with docker compose:

**Prerequisites:**
- Ubuntu 22.04+ VPS with 4GB+ RAM
- Docker and Docker Compose installed
- Domain pointed to your VPS (optional, for HTTPS)

**Steps:**

1. SSH into your VPS and clone the repo:
   ```bash
   git clone https://github.com/your-username/savor.git
   cd savor
   ```

2. Copy and edit environment variables:
   ```bash
   cp .env.example .env
   nano .env
   ```

   Update production secrets:
   - `JWT_SECRET`: Generate a strong 32+ character secret
   - `POSTGRES_PASSWORD`: Change from default
   - `RABBITMQ_PASSWORD`: Change from default
   - `MINIO_ROOT_PASSWORD`: Change from default
   - `NEXT_PUBLIC_API_URL`: Set to your domain (e.g., `https://savor.example.com`)

3. Start the application:
   ```bash
   docker compose up -d
   ```

4. Check all services are healthy:
   ```bash
   docker compose ps
   ```

5. Access the application:
   - **Frontend**: http://your-vps-ip (or your domain)
   - **Grafana**: http://your-vps-ip:3001 (user: savor, pass: savor)
   - **Jaeger**: http://your-vps-ip:16686
   - **Mailhog**: http://your-vps-ip:8025

**For HTTPS with Traefik:**

Update `infra/traefik/traefik.yml` to enable Let's Encrypt:
```yaml
certificatesResolvers:
  letsencrypt:
    acme:
      email: your-email@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
```

Add HTTPS entrypoint and mount volume in `docker-compose.yml`:
```yaml
services:
  traefik:
    command:
      - --configFile=/etc/traefik/traefik.yml
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./infra/traefik/traefik.yml:/etc/traefik/traefik.yml:ro
      - ./infra/traefik/dynamic.yml:/etc/traefik/dynamic.yml:ro
      - traefik-certs:/letsencrypt
```

**Monitoring:**

Prometheus, Grafana, Loki, and Jaeger are included. Access Grafana at port 3001 to view:
- Service metrics (request rates, latencies, errors)
- Logs from all services
- Distributed traces

**Backup:**

Postgres data is in the `postgres-data` volume. Backup regularly:
```bash
docker compose exec postgres pg_dump -U savor savor > backup.sql
```

## CI

GitHub Actions runs on every push and PR:
- **Compose validation**: Ensures docker-compose.yml is valid
- **Build checks**: All Go/NestJS/Next.js services build successfully
- **Linting**: `go vet` for Go services
- **Tests**: Unit tests for all services with Jest (NestJS) and Go test

See [`.github/workflows/ci.yml`](.github/workflows/ci.yml) for the full workflow.
