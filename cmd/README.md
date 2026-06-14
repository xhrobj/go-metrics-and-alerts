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

Агент периодически опрашивает системные и runtime-метрики, складывает их в очередь и отправляет на Сервер по HTTP.

```text
agent -> queue/worker pool -> HTTP -> server -> storage
```

Сервер сохраняет метрики в одном из хранилищ:

- **in-memory** - хранение метрик в памяти
- **file storage** - периодическое сохранение метрик в файл
- **PostgreSQL** - хранение метрик в базе данных

## Конфигурация

Агент и Сервер поддерживают конфигурацию через:

- флаги командной строки
- переменные окружения
- значения по умолчанию

Приоритет источников:

```text
env > flag > default
```

Конкретные параметры описаны в README соответствующих приложений:

- [конфигурация Агента](agent/README.md)
- [конфигурация Сервера](server/README.md)

## Информация о сборке

При сборке через `Makefile` в бинарники Агента и Сервера подставляются:

- версия сборки
- дата сборки
- хэш Git-коммита

При старте приложения выводят эти значения в stdout:

```text
Build version: v0.8.0
Build date: 2026-06-14
Build commit: abc1234
```

Значения передаются через `ldflags`.

Если build info не была передана, приложение выводит `N/A`.

## Защита обмена

Защита запросов состоит из двух независимых механизмов:

- ключ `KEY` включает вычисление и проверку заголовка `HashSHA256`
- ключ `CRYPTO_KEY` включает гибридное шифрование RSA-OAEP + AES-GCM

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
-> handler
```

Агенту передаётся публичный RSA-ключ, Серверу - соответствующий приватный ключ.

Без `CRYPTO_KEY` сохраняется обычный сценарий обмена без шифрования.

Подробности о ключах и ручной генерации пары находятся в [`internal/encryption`](../internal/encryption/README.md).

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
make run-server-crypto

make run-agent
make run-agent-env
make run-agent-crypto
```

### Docker Compose

Полный локальный стенд с PostgreSQL, Сервером, Агентом, healthcheck'ами и шифрованием запускается командой:

```bash
make compose-up
```

Просмотр логов:

```bash
make compose-logs
```

Остановка контейнеров:

```bash
make compose-down
```

При необходимости локальная RSA-пара создаётся автоматически перед запуском Compose.
