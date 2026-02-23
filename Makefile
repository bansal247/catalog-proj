.PHONY: all up down migrate proto test test-unit test-e2e run tidy

SPANNER_PROJECT  ?= test-project
SPANNER_INSTANCE ?= test-instance
SPANNER_DATABASE ?= catalog
SPANNER_EMULATOR_HOST ?= localhost:9010

GRPC_ADDR ?= :50051

## up: Start the Spanner emulator via docker-compose.
up:
	docker-compose up -d
	@echo "Waiting for emulator to be ready..."
	@sleep 3

## down: Stop and remove the Spanner emulator container.
down:
	docker-compose down

## migrate: Create the Spanner instance + database and apply schema DDL.
migrate:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	go run ./scripts/migrate/main.go \
		-project=$(SPANNER_PROJECT) \
		-instance=$(SPANNER_INSTANCE) \
		-database=$(SPANNER_DATABASE)

## proto: Regenerate gRPC code from .proto files (requires protoc + plugins).
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/product_service.proto

## test: Run all tests (unit + e2e against emulator).
test: test-e2e

## test-e2e: Run E2E tests against the Spanner emulator.
test-e2e:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	go test ./tests/... -v -count=1 -timeout=120s

## run: Start the gRPC server.
run:
	SPANNER_EMULATOR_HOST=$(SPANNER_EMULATOR_HOST) \
	SPANNER_PROJECT=$(SPANNER_PROJECT) \
	SPANNER_INSTANCE=$(SPANNER_INSTANCE) \
	SPANNER_DATABASE=$(SPANNER_DATABASE) \
	GRPC_ADDR=$(GRPC_ADDR) \
	go run ./cmd/server/main.go

## tidy: Tidy go modules.
tidy:
	go mod tidy
