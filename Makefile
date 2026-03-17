.PHONY: help dev-up dev-down dev-build dev-restart dev-logs migrate-up migrate-down migrate-status build run clean mocks clean-mocks test

DC = docker-compose --env-file env/.env -f docker-compose.yml
DB_CONTAINER = postgres-starter
DB_USER = postgres
DB_NAME = go-gin-starter
API_CONTAINER=api

create-feature:
	@chmod +x create_feature.sh
	@./create_feature.sh $(feature)

swag-init:
	swag init -g main.go --output docs

up:
	$(DC) up -d

build:
	$(DC) build

up-build:
	$(DC) up --build

down:
	$(DC) down

restart:
	$(DC) restart

logs:
	$(DC) logs -f

logs-api:
	$(DC) logs -f api

ps:
	$(DC) ps

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "=X= Error: 'name' is required. Usage: make migrate-create name=migrastion_name"; \
		exit 1; \
	fi
	@go run cmd/migrate/main.go -action=create -name=$(name)

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