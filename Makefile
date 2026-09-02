COMPOSE := docker compose

.PHONY: up down logs ps restart seed fmt tidy

up:
	$(COMPOSE) up --build

up-d:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

down-v:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f --tail=200

ps:
	$(COMPOSE) ps

restart:
	$(COMPOSE) restart

# Demo accounts are seeded on first boot of identity/catalog.
# Recreate volumes if you need a clean slate: make down-v && make up
seed:
	@echo "Seed runs automatically when databases are empty."
	@echo "customer@savor.dev / savor1234"
	@echo "restaurant@savor.dev / savor1234"
	@echo "courier@savor.dev / savor1234"
