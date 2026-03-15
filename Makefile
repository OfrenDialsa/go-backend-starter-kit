.PHONY: help dev-up dev-down dev-build dev-restart dev-logs migrate-up migrate-down migrate-status build run clean mocks clean-mocks test

DC = docker-compose --env-file env/.env -f docker-compose.yml
DB_CONTAINER = postgres-starter
DB_USER = postgres
DB_NAME = go-gin-starter
API_CONTAINER=api

swag-init:
	swag init -g main.go --output docs

up:
	$(DC) up -d

down:
	$(DC) down

build:
	$(DC) build

f-build:
	docker compose -f docker-compose.yml up --build

restart:
	$(DC) restart

logs:
	$(DC) logs -f

logs-api:
	$(DC) logs -f api

ps:
	$(DC) ps

migrate-up:
	$(DC) exec $(API_CONTAINER) /app/migrate -action=up

migrate-down:
	$(DC) exec $(API_CONTAINER) /app/migrate -action=down

migrate-status:
	$(DC) exec $(API_CONTAINER) ls -l migrations

db-shell:
	docker exec -it $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME)

test:
	go test ./... -v

clean:
	go clean