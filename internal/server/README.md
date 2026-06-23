# internal/server

Пакет является корнем приложения Сервера: создаёт зависимости, управляет жизненным циклом HTTP- и gRPC-Серверов, persistence, аудитом и соединением с PostgreSQL.

## Структура

```text
internal/server/
├── server.go       # создание, запуск и graceful shutdown Сервера
├── grpc.go         # создание и graceful shutdown gRPC-сервера
├── database.go     # подключение PostgreSQL, миграции и выбор repository
├── persistence.go  # восстановление и файловое сохранение метрик
├── audit_setup.go  # создание общего диспетчера и подключение observers аудита
├── security.go     # загрузка ключа и trusted subnet
├── config/         # загрузка и валидация конфигурации Сервера
├── audit/          # события аудита и observers
├── migrations/     # запуск SQL-миграций
├── service/        # бизнес-логика метрик
└── transport/
    ├── grpc/       # gRPC-сервис, metadata, аудит, логирование и проверка trusted subnet
    └── http/
        ├── handler/    # HTTP-обработчики
        ├── middleware/ # HTTP middleware
        └── router/     # маршруты и middleware-цепочка
```

## HTTP-транспорт

HTTP-запрос проходит через router и middleware к handler'у, который вызывает общий сервис метрик:

```text
HTTP
-> transport/http/router
-> transport/http/middleware
-> transport/http/handler
-> service.MetricsService
-> repository
```

## gRPC-транспорт

gRPC-запрос проходит через UnaryInterceptor к реализации сервиса `Metrics`, которая преобразует protobuf-сообщения во внутреннюю модель и вызывает тот же сервис метрик:

```text
gRPC
-> transport/grpc/LoggingInterceptor
-> transport/grpc/TrustedSubnetInterceptor
-> transport/grpc/Server.UpdateMetrics
-> service.MetricsService
-> repository
```

HTTP- и gRPC-транспорты используют общий `MetricsService`, repository и диспетчер аудита. Сервер слушает отдельные адреса, заданные параметрами `ADDRESS` и `GRPC_ADDRESS`.

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

## Lifecycle

`Server.Run` запускает HTTP- и gRPC-Серверы параллельно. Отмена контекста или завершение одного транспорта инициирует остановку второго транспорта. HTTP использует `Shutdown`, gRPC использует `GracefulStop` с переходом к принудительному `Stop` после общего тайм-аута.
