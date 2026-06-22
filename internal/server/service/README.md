# internal/server/service

Пакет содержит transport-independent бизнес-логику работы с метриками и отделяет HTTP-слой от конкретной реализации хранилища.

## `MetricsService`

Сервис работает через интерфейс `MetricsStorage` и поддерживает:

- замену значения gauge
- накопление delta counter
- batch-обновление метрик
- чтение gauge и counter
- получение snapshot всех метрик

```text
HTTP handler
-> MetricsService
-> MetricsStorage
```

В качестве `MetricsStorage` используются `MemStorage` или `PostgresStorage` из [`internal/repository`](../../repository/README.md).

Транспортный слой зависит от интерфейса сервиса, а сервис не знает об HTTP-маршрутах, статусах и middleware. Поэтому одну реализацию `MetricsService` можно использовать из разных transport-адаптеров.

## Синхронное файловое сохранение

Метод `EnableSyncSave` подключает `Saver`. После этого каждое успешное изменение метрик вызывает сохранение snapshot.

Этот режим используется Сервером при:

```text
DATABASE_DSN пуст
FILE_STORAGE_PATH непустой
STORE_INTERVAL = 0
```

При положительном интервале периодическим и финальным сохранением управляет `internal/server.Server`.

## Обработка ошибок

Сервис:

- добавляет контекст к ошибкам repository и файлового сохранения
- сохраняет `repository.ErrMetricNotFound` доступной для проверки через `errors.Is`
- не зависит от HTTP-статусов и формата transport'а
