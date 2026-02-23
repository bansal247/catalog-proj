## Setup

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- (Optional) `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` for proto regeneration

### Start the Spanner Emulator

```bash
make up
```

### Apply Database Migrations

```bash
make migrate
```

### Run E2E Tests (requires running emulator)

```bash
make test-e2e
```

### Start the gRPC Server

```bash
make run
```

The server listens on `:50051` by default. Override with `GRPC_ADDR=:8080`.

### Regenerate Proto Stubs

```bash
make proto
```

## Assumptions
- Using reader in queries instead of using read_model in contracts