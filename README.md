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

## CI

Push to GitHub and Actions runs [`.github/workflows/ci.yml`](.github/workflows/ci.yml): compose file is valid, every Go/Nest/Next service still builds. No secrets needed. The workflow file is the interesting part on a resume — not a badge.
