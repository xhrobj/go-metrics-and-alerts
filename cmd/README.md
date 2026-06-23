# cmd

В директории `cmd/` находятся точки входа CLI-приложений проекта:

- `agent` - собирает системные и runtime-метрики и отправляет их на Сервер
- `server` - запускает Сервер, который принимает, хранит и отдаёт метрики
- `staticlint` - запускает собственный набор статических анализаторов проекта
- `reset` - генерирует методы `Reset()` для структур с комментарием `// generate:reset`

Точки входа остаются тонкими: они читают конфигурацию, создают зависимости и передают управление внутренним пакетам `internal/agent` и `internal/server`.

## Структура

```text
cmd/
├── agent/       # точка входа Агента
├── server/      # точка входа Сервера
├── staticlint/  # CLI-утилита статического анализа
└── reset/       # CLI-утилита генерации Reset-методов
```

## Взаимодействие Агента и Сервера

Агент периодически собирает системные и runtime-метрики, формирует batch-отчёты и отправляет их через выбранный транспорт.

```text
cmd/agent
-> agent.Agent
-> agent/service.ReportingService
-> agent/service.MetricsSender
   -> agent/transport/grpc.GRPCSender -> gRPC
   -> agent/transport/http.HTTPSender -> HTTP
```

Агент использует gRPC по умолчанию. HTTP остается доступным через `--transport=http`.

Сервер одновременно принимает оба транспорта и передает данные в общий сервис метрик и repository:

Пакет `internal/server` создаёт зависимости, управляет двумя listener'ами, persistence, аудитом и соединением с БД.

Подробности:

- [архитектура Агента](../internal/agent/README.md)
- [архитектура Сервера](../internal/server/README.md)

Сервер использует одно из runtime-хранилищ:

- `PostgresStorage` - если задан `DATABASE_DSN`
- `MemStorage` - если PostgreSQL не настроен

При работе с `MemStorage` состояние может дополнительно сохраняться и восстанавливаться через `FileStore`.

## Конфигурация

Агент и Сервер поддерживают четыре источника конфигурации:

- значения по умолчанию
- JSON-файл
- флаги командной строки
- переменные окружения

Приоритет источников:

```text
env > flag > JSON > default
```

Конкретные параметры описаны в README соответствующих приложений:

- [конфигурация Агента](agent/README.md)
- [конфигурация Сервера](server/README.md)

## Информация о сборке

При сборке через `Makefile` в бинарники Агента и Сервера подставляются:

- версия сборки
- дата сборки
- хеш Git-коммита

При старте приложения выводят значения в stdout, например:

```text
Build version: v0.9.0
Build date: 2026-06-16
Build commit: c0decafe42
```

Значения передаются через `ldflags`. Если build info не была передана, приложение выводит `N/A`.

## Защита обмена

Проверка trusted subnet работает для обоих транспортов. Агент передаёт локальный IPv4-адрес:

- HTTP - в заголовке `X-Real-IP`
- gRPC - в metadata `x-real-ip`

`KEY` и `CRYPTO_KEY` относятся только к HTTP-транспорту:

- `KEY` включает подпись исходящих HTTP-сообщений и проверку `HashSHA256`, если заголовок присутствует во входящем сообщении
- `CRYPTO_KEY` включает гибридное шифрование RSA-OAEP + AES-GCM

При включённых сжатии, шифровании и хешировании HTTP-тело обрабатывается в таком порядке:

```text
Агент:
JSON
-> gzip
-> AES-GCM
-> RSA-OAEP для AES-ключа
-> HashSHA256
-> HTTP
```

```text
Сервер:
HashSHA256
-> RSA-OAEP / AES-GCM
-> gzip
-> проверка trusted subnet
-> handler
```

Агенту передаётся публичный RSA-ключ, Серверу - соответствующий приватный ключ. Без `CRYPTO_KEY` сохраняется обычный HTTP-сценарий без шифрования. Подробности о ключах находятся в [`internal/encryption`](../internal/encryption/README.md).

## Основные команды

### Сборка

```bash
make build
make build-server
make build-agent
```

### Тесты и проверки

```bash
make ci
make test
make test-race
make show-coverage
make staticlint
```

### Генерация кода

```bash
make generate-mocks
make generate-reset
make generate-proto
make clean-generated
```

### Локальный PostgreSQL

```bash
make postgres-up
make postgres-start
make postgres-stop
make postgres-rm
make postgres-connect
```

### Локальный запуск приложений

```bash
make run-server
make run-server-env
make run-server-config
make run-server-crypto

make run-agent
make run-agent-env
make run-agent-config
make run-agent-crypto
```

### Docker Compose

Полный локальный стенд с PostgreSQL, Сервером, Агентом и healthcheck'ами запускается командой:

```bash
make compose-up
```

Просмотр логов и остановка:

```bash
make compose-logs
make compose-down
```

При необходимости локальная RSA-пара создаётся автоматически перед запуском Compose.

## Локальная конфигурация

`Makefile` при наличии автоматически читает корневой файл `.env`. Создайте его из примера:

```bash
cp .env.example .env
```

Сам `.env` игнорируется Git и не должен попадать в репозиторий. Агент и Сервер не читают env-файл напрямую: `Makefile` передаёт значения существующим флагам и переменным окружения.

Все команды работают и без `.env`, используя значения по умолчанию из `Makefile`.

Разово переопределить параметр можно в команде:

```bash
make run-server TRUSTED_SUBNET=10.0.0.0/8
make run-agent AGENT_TRANSPORT=http
```

Для другого env-файла используется `ENV_FILE`:

```bash
make run-server ENV_FILE=.env.local
```
