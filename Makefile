.PHONY: help dev-up dev-down dev-build dev-restart dev-logs migrate-up migrate-down migrate-status build run clean mocks clean-mocks test monitor-up monitor-down monitor-logs monitor-restart

DC = docker-compose --env-file env/.env -f docker-compose.yml
DC_MONITORING = docker-compose -f docker/monitoring/docker-compose.yml
DB_CONTAINER = postgres-starter
DB_USER = postgres
DB_NAME = go-gin-starter
API_CONTAINER=api

FEATURE_NAME=$(name)

create-repo:
	@chmod +x script/create_repository.sh
	@./script/create_repository.sh $(FEATURE_NAME)

create-service:
	@chmod +x script/create_service.sh
	@./script/create_service.sh $(FEATURE_NAME)

create-handler:
	@chmod +x script/create_handler.sh
	@./script/create_handler.sh $(FEATURE_NAME)

create-feature:
	@$(MAKE) create-repo name=$(FEATURE_NAME)
	@$(MAKE) create-service name=$(FEATURE_NAME)
	@$(MAKE) create-handler name=$(FEATURE_NAME)

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

monitor-up:
	$(DC_MONITORING) up -d

monitor-down:
	$(DC_MONITORING) down

monitor-logs:
	$(DC_MONITORING) logs -f

monitor-restart:
	$(DC_MONITORING) restart

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