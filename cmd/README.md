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

## Основные команды

Сборка приложений:

```bash
make build
```

Запуск тестов и проверок:

```bash
make ci
make staticlint
```

Генерация кода:

```bash
make generate-reset
```

Очистка сгенерированных файлов:

```bash
make clean-generated
```

Просмотр покрытия:

```bash
make show-coverage
```

Локальный PostgreSQL:

```bash
make postgres-up
make postgres-start
make postgres-stop
make postgres-rm
```

Запуск Сервера:

```bash
make run-server
make run-server-env
```

Запуск Агента:

```bash
make run-agent
make run-agent-env
```
