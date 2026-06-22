# migrations

В директории находятся SQL-миграции схемы PostgreSQL.

## Файлы

Каждая версия состоит из пары:

```text
NNN_description.up.sql
NNN_description.down.sql
```

Текущие миграции:

- `001_create_metrics_table` - создаёт таблицу `metrics`
- `002_alter_metrics_table` - добавляет enum `metric_type` и `created_at`

`up` применяет изменение, `down` откатывает его.

## Запуск

Используется библиотека `golang-migrate`.

Если задан `DATABASE_DSN`, application root [`internal/server`](../internal/server/README.md) при старте:

1. открывает соединение с PostgreSQL
2. проверяет доступность БД
3. вызывает runner из `internal/server/migrations`
4. выполняет все доступные `up`-миграции
5. продолжает запуск приложения

`migrate.ErrNoChange` означает, что схема уже актуальна, и не считается ошибкой.

Путь к миграциям формируется относительно рабочей директории проекта, поэтому Сервер следует запускать из корня репозитория или обеспечить доступность каталога `migrations`.

## Локальный PostgreSQL

Создание и управление контейнером:

```bash
make postgres-up
make postgres-start
make postgres-stop
make postgres-rm
```

Подключение через `psql`:

```bash
make postgres-connect
```

Полезные команды:

```text
\dt
\d metrics
SELECT * FROM schema_migrations;
\q
```
