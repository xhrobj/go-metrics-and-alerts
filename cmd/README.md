# cmd

В директории `cmd/` находятся точки входа CLI-приложений проекта:

- `agent` - собирает системные и runtime-метрики и отправляет их на Сервер
- `server` - принимает, хранит и отдаёт метрики
- `staticlint` - запускает собственный набор статических анализаторов проекта
- `reset` - генерирует методы `Reset()` для структур с комментарием `// generate:reset`

## Структура

```text
cmd/
├── agent/       # точка входа Агента
├── server/      # точка входа Сервера
├── staticlint/  # CLI-утилита статического анализа
└── reset/       # CLI-утилита генерации Reset-методов
```

## Взаимодействие Агента и Сервера

Агент периодически собирает системные и runtime-метрики, формирует batch-отчёты и отправляет их на Сервер через очередь задач и worker pool.

```text
agent -> queue/worker pool -> HTTP -> server -> service -> repository
```

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

Общая реализация загрузки и валидации параметров описана в [`internal/config`](../internal/config/README.md).

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

Защита запросов состоит из независимых механизмов:

- `KEY` включает подпись исходящих HTTP-сообщений и проверку `HashSHA256`, если заголовок присутствует во входящем сообщении
- `CRYPTO_KEY` включает гибридное шифрование RSA-OAEP + AES-GCM
- `TRUSTED_SUBNET` ограничивает приём метрик доверенной сетью
- Агент добавляет в запросы заголовок `X-Real-IP`

При включённых сжатии, шифровании и хешировании тело запроса обрабатывается в таком порядке:

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
-> проверка trusted subnet для update-маршрутов
-> handler
```

Агенту передаётся публичный RSA-ключ, Серверу - соответствующий приватный ключ.

Без `CRYPTO_KEY` сохраняется обычный сценарий обмена без шифрования. Подробности о ключах находятся в [`internal/encryption`](../internal/encryption/README.md).

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
make generate-reset
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

Полный локальный стенд с PostgreSQL, Сервером, Агентом, healthcheck'ами и шифрованием запускается командой:

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
```

Для другого env-файла используется `ENV_FILE`:

```bash
make run-server ENV_FILE=.env.local
```
