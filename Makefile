.PHONY: up down proto lint breaking migrate-account migrate-account-down run-account tidy build

up:
	docker compose up -d

down:
	docker compose down

# Generate Go stubs from the shared contracts in protos/.
proto:
	cd protos && buf generate

# Schema linter - catches naming/versioning violations before they ship.
lint:
	cd protos && buf lint

# Fails if the current contracts break wire compatibility with the last
# commit on main. This is the check that belongs in CI.
breaking:
	cd protos && buf breaking --against '../.git#branch=main,subdir=protos'

# Migrations run through the migrate/migrate image rather than a locally
# installed binary: nothing to install, and the exact same command works on
# macOS and on Windows. The container joins the compose network, so it
# addresses Postgres by service name, not through the published port.
MIGRATE = docker run --rm --network banking-platform_default \
	-v "$(CURDIR)/services/account/migrations:/migrations" migrate/migrate \
	-path=/migrations \
	-database "postgres://banking:banking@postgres-account:5432/account?sslmode=disable"

migrate-account:
	$(MIGRATE) up

migrate-account-down:
	$(MIGRATE) down 1

run-account:
	cd services/account && go run ./cmd/server

build:
	go build banking/protos/... banking/services/account/...

tidy:
	cd protos && go mod tidy
	cd services/account && go mod tidy
