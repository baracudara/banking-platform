.PHONY: up down proto lint breaking migrate-account run-account tidy build

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

migrate-account:
	migrate -path services/account/migrations \
		-database "postgres://banking:banking@localhost:5432/account?sslmode=disable" up

run-account:
	cd services/account && go run ./cmd/server

build:
	go build banking/protos/... banking/services/account/...

tidy:
	cd protos && go mod tidy
	cd services/account && go mod tidy
