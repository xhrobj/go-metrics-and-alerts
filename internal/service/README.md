# internal/service

Пакет содержит бизнес-логику работы с метриками и отделяет HTTP-слой от конкретной реализации хранилища.

## `MetricsService`

Сервис работает через интерфейс `MetricsStorage` и поддерживает:

- замену значения gauge
- накопление delta counter
- batch-обновление метрик
- чтение gauge и counter
- получение snapshot всех метрик

```text
handler -> MetricsService -> MetricsStorage
```

В качестве `MetricsStorage` используются `MemStorage` или `PostgresStorage`.

## Синхронное файловое сохранение

Метод `EnableSyncSave` подключает `Saver`. После этого каждое успешное изменение метрик вызывает сохранение snapshot.

Этот режим используется Сервером при:

```text
DATABASE_DSN пуст
FILE_STORAGE_PATH непустой
STORE_INTERVAL = 0
```

При положительном интервале периодическое сохранение управляется точкой входа Сервера, а не сервисом.

## Обработка ошибок

Service:

- добавляет контекст к ошибкам repository и файлового сохранения
- сохраняет `repository.ErrMetricNotFound` доступной для проверки через `errors.Is`
- не зависит от HTTP-статусов и формата транспорта
