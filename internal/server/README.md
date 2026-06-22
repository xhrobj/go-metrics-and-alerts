# internal/server

Пакет является корнем приложения Сервера: создаёт зависимости, управляет HTTP lifecycle, persistence, аудитом и соединением с PostgreSQL.

```text
cmd/server
-> server.New
-> server.Server
   -> server/config
   -> server/service
   -> server/transport/http
   -> repository
```

## Структура

```text
internal/server/
├── server.go       # создание, запуск и graceful shutdown Сервера
├── database.go     # подключение PostgreSQL, миграции и выбор repository
├── persistence.go  # восстановление и файловое сохранение метрик
├── audit_setup.go  # подключение observers аудита
├── security.go     # загрузка ключа и trusted subnet
├── config/         # загрузка и валидация конфигурации Сервера
├── audit/          # события аудита и observers
├── migrations/     # запуск SQL-миграций
├── service/        # бизнес-логика метрик
└── transport/
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

## Конфигурация

Загрузка и валидация параметров описаны в [`internal/server/config`](config/README.md).
