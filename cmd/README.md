# cmd

В директории `cmd` находятся точки входа в исполняемые команды проекта. Здесь остаётся только код запуска: чтение конфигурации, сборка зависимостей и старт приложения или утилиты. Бизнес-логика живёт в `internal`.

## Команды проекта

- `agent` - собирает runtime-метрики и отправляет их на Сервер.
- `server` - принимает, хранит и отдаёт метрики.
- `staticlint` - запускает собственный набор статических анализаторов проекта.
- `reset` - генерирует методы `Reset()` для структур с комментарием `// generate:reset`.

## Структура

```text
cmd/
├── agent/       # точка входа Агента
├── server/      # точка входа Сервера
├── staticlint/  # CLI-утилита статического анализа
└── reset/       # CLI-утилита генерации Reset-методов
```

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
