# banking-platform

Учебный проект: банковское приложение на микросервисах для отработки паттернов
конкурентности (Concurrency in Go) и архитектуры высоконагруженных систем
(Designing Data-Intensive Applications).

## Статус

Шаг 1 из плана: **Account service + PostgreSQL**, без gRPC-стабов (см. "Что дальше").

## Структура

```
services/account/
  cmd/server/          — entrypoint, сборка зависимостей
  internal/domain/     — доменная модель, ничего не знает о БД или транспорте
  internal/repository/ — реализация доступа к Postgres (pgx)
  internal/service/    — бизнес-логика, зависит от Repository как от интерфейса
  internal/transport/  — gRPC-хендлеры (адаптер между proto и service)
  proto/                — .proto контракт сервиса
  migrations/           — SQL-миграции
```

Слои изолированы намеренно: `domain` не импортирует ничего из `repository`
или `transport`, `service` зависит только от интерфейса `Repository` (не от
конкретной pgx-реализации) — это то, что делает юнит-тесты `service` без
поднятия реальной базы.

## Как запустить

```bash
make up              # поднять Postgres в docker
make migrate-account  # прогнать миграции (нужен golang-migrate)
make proto            # сгенерировать gRPC-стабы (нужен protoc)
make tidy             # go mod tidy — подтянет зависимости
make run-account      # запустить сервис
```

## Что дальше

1. `make proto` не запускался и не проверялся — сгенерируй стабы и посмотри,
   что говорит компилятор в `internal/transport/grpcserver`, там наверняка
   будут расхождения с реальным API protoc-gen-go, разберём вместе.
2. Auth service — второй сервис первого шага плана.
3. После Account + Auth — Transaction/Ledger с outbox pattern.
