# Миграции PostgreSQL

В директории `migrations/` находятся SQL-миграции схемы PostgreSQL.

## Структура

Каждая версия состоит из двух файлов:

```text
NNN_description.up.sql
NNN_description.down.sql
```

- `up.sql` применяет изменение схемы
- `down.sql` описывает обратное изменение для отката

Текущие миграции:

- `001_create_metrics_table` - создаёт таблицу `metrics`
- `002_alter_metrics_table` - добавляет тип `metric_type` и колонку `created_at`

После применения всех миграций таблица `metrics` содержит gauge- и counter-метрики. Метрика идентифицируется сочетанием имени и типа.

## Автоматическое применение

Для выполнения миграций используется библиотека `golang-migrate`.

Если задан `DATABASE_DSN`, Сервер при запуске:

1. открывает соединение с PostgreSQL
2. проверяет доступность БД
3. создаёт runner из пакета `internal/server/migrations`
4. применяет все доступные `up`-миграции
5. продолжает запуск приложения

Ошибка `migrate.ErrNoChange` означает, что схема уже актуальна, и не препятствует запуску Сервера.

`down`-миграции автоматически не применяются.

## Расположение файлов

Путь к миграциям формируется на основе текущей рабочей директории:

```text
<working directory>/migrations
```

Поэтому Сервер следует запускать из корня репозитория либо обеспечить наличие каталога `migrations/` в рабочей директории процесса.

## Локальный PostgreSQL

Создание контейнера:

```bash
make postgres-up
```

Управление контейнером:

```bash
make postgres-start
make postgres-stop
make postgres-rm
```

Подключение через `psql`:

```bash
make postgres-connect
```

Полезные команды `psql`:

```text
\dt
\d metrics
SELECT * FROM schema_migrations;
\q
```
