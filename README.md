# Banking Platform

A microservices banking backend in Go. The project is a place to practice
concurrency and the architecture of data-intensive systems the way they are
built in production: correctness of money handling, explicit API contracts,
automated delivery and reproducible infrastructure.

**Status:** the Account service runs and deploys automatically. The Ledger
service, the transactional outbox and Kafka are next — see [Roadmap](#roadmap).

## Stack

Go · gRPC · Protocol Buffers (buf) · PostgreSQL (pgx) · Docker / Docker Compose ·
GitHub Actions · Ansible · WireGuard · HAProxy

## Repository layout

```
protos/                   shared API contracts (module banking/protos)
  proto/account/v1/         Account service API
  proto/common/v1/          shared types, e.g. Money
  gen/go/                   generated Go code (committed)
services/account/         Account service (module banking/services/account)
  cmd/server/               entrypoint, dependency wiring, signal handling
  internal/domain/          domain model; knows nothing about the DB or transport
  internal/service/         business logic; depends on a Repository interface
  internal/repository/      PostgreSQL implementation (pgx pool)
  internal/transport/       gRPC handlers: adapter between proto and service
  internal/app/             gRPC server lifecycle, interceptors
  migrations/               SQL migrations
deploy/
  docker-compose.prod.yml   single-host production stack
  ansible/                  multi-node lab: WireGuard mesh, Postgres, HAProxy
.github/workflows/        CI and CD
```

## Design decisions

- **Monorepo with `go.work`, one Go module per service.** With a single
  developer, a repository per service would only add contract-syncing overhead.
- **Database per service.** Every service owns its PostgreSQL instance; there
  is no shared schema. Anything spanning services goes through APIs and events,
  which is why the outbox and saga patterns are on the roadmap.
- **Money is `int64` in minor units** (cents), never a float, and always
  travels together with its currency (`common.v1.Money`).
- **The ledger is the source of truth for balances.** `accounts.balance_cache`
  is a read cache protected by **optimistic concurrency control**: every update
  is `UPDATE ... WHERE id = $1 AND version = $2`, and a lost race surfaces as
  `ErrOptimisticLockFailed` instead of silently overwriting a newer value.
- **Layered architecture (ports and adapters).** `domain` imports nothing from
  the outer layers; `service` depends on a `Repository` interface rather than
  on pgx, so business logic can be tested without a database.
- **Central API contracts.** `.proto` files live in `protos/`, not inside
  services, so a consumer does not depend on the producer's module. Every RPC
  has its own response message, so adding a field to one RPC never breaks
  another. `buf lint` and `buf breaking` run in CI and block changes that would
  break the wire format (renamed fields, reused tag numbers).
- **Bounded graceful shutdown.** On SIGTERM the server calls `GracefulStop`
  raced against `shutdown_timeout`, then falls back to a hard `Stop`. A bare
  `GracefulStop` can wait forever, and the orchestrator would then kill the
  process in the middle of a request anyway.
- **Secure defaults.** gRPC reflection is enabled only outside production;
  interceptors log the fact of a call, not its payload (payloads contain account
  ids and amounts); the image is distroless and runs as a non-root user.
- **Configuration** via `cleanenv`: a YAML file plus environment variable
  overrides. `config/local.yaml` is committed because its credentials only work
  with the local `docker-compose.yml`; real secrets come from the environment.

## Running locally

Requirements: Go, Docker. [buf](https://buf.build) only if you change `.proto`
files — the generated code is committed.

```bash
make up                # PostgreSQL in Docker
make migrate-account   # apply migrations (runs the migrate/migrate image)
make run-account       # start the Account service on :50051
```

Reflection is enabled locally, so the API can be explored with
[grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext -d '{"user_id":"8f8b1c9e-5d3a-4f1e-9a57-0c2f6a1d4b11","currency":"USD"}' \
  localhost:50051 account.v1.AccountService/CreateAccount
```

Other targets: `make build`, `make proto`, `make lint`, `make breaking`, `make down`.

## CI/CD

- **CI** (`.github/workflows/ci.yml`, GitHub-hosted runners): build, `go vet`,
  `go test -race`, `buf lint`, `buf breaking` against the previous commit, and
  a Docker image build with layer caching.
- **CD** (`.github/workflows/deploy.yml`, self-hosted ARM64 runner): starts
  only after CI succeeds on a push to `main` (or manually for rollbacks). It
  builds the image natively on the host, applies migrations, rolls out with
  Docker Compose and prunes old images.

## Infrastructure lab (Ansible)

`deploy/ansible` provisions a three-node lab from scratch (one balancer, two
app nodes) in dependency order (`playbooks/site.yml`):

1. **base** — `ops` user with key-only SSH, hardened `sshd`, `ufw` with
   default-deny incoming.
2. **wireguard** — full-mesh overlay network; private keys are generated on
   each node and never leave it.
3. **docker**, **postgres** — PostgreSQL bound to the overlay address only,
   with migrations applied on provisioning.
4. **deploy-account** — the image is built once and streamed to each node
   (`docker save | docker load`), then rolled out with Compose.
5. **haproxy** — load balancer in front of the app nodes. In `http` mode it
   speaks HTTP/2 and balances individual gRPC requests; in `tcp` mode it
   balances connections, which is useful to show why L4 balancing is uneven
   for long-lived gRPC connections.

```bash
cd deploy/ansible
# put your server addresses into inventory.yml first
ansible-playbook playbooks/site.yml -e ansible_user=root
```

## Roadmap

1. Account service + PostgreSQL — **in progress** (unit and integration tests,
   input validation, error mapping to gRPC codes)
2. Ledger service: double-entry bookkeeping, idempotent transfers,
   service-to-service gRPC, transactional outbox
3. Messaging with Kafka: outbox relay, idempotent consumers, dead-letter topic
4. Sharding the Account service by `user_id`
5. Saga orchestrator for cross-shard transfers
6. Redis: cache, distributed rate limiter (token bucket), distributed lock
7. Observability: Prometheus, Grafana, OpenTelemetry
8. Load testing and resilience: circuit breaker, bulkhead, retries with
   backoff and jitter
