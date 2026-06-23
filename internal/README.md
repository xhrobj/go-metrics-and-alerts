# internal

В директории `internal/` находится код внутренних пакетов проекта. Эти пакеты нельзя импортировать из модулей, расположенных за пределами родительского модуля.

## Структура

```text
internal/
├── agent/          # runtime, конфигурация, сервис и транспорт Агента
│   ├── config/     # загрузка и валидация конфигурации Агента
│   ├── service/    # сбор метрик и подготовка отчётов
│   └── transport/  # реализации транспорта Агента
├── buildinfo/      # вывод версии, даты и Git-коммита сборки
├── encryption/     # RSA-ключи и гибридное шифрование
├── hash/           # вычисление HashSHA256
├── logger/         # инициализация zap
├── model/          # общая модель метрики
├── proto/          # сгенерированные protobuf-типы и интерфейсы gRPC
├── protocol/       # общие транспортные константы
├── repository/     # хранилища метрик
└── server/         # приложение Сервера и его внутренние пакеты
    ├── audit/      # события аудита и observers
    ├── config/     # загрузка и валидация конфигурации Сервера
    ├── migrations/ # запуск SQL-миграций
    ├── service/    # бизнес-логика метрик
    └── transport/
        └── http/
            ├── handler/    # HTTP-обработчики
            ├── middleware/ # HTTP middleware
            └── router/     # маршруты и middleware-цепочка
```

## Агент

Подробнее: [`internal/agent`](agent/README.md).

## Сервер

Подробнее: [`internal/server`](server/README.md).

## Protobuf

Пакет `internal/proto` содержит сгенерированные на основе [`api/metrics.proto`](../api/metrics.proto) Go-типы сообщений и интерфейсы gRPC-клиента и gRPC-Сервера.

## Транспортные константы

Подробнее: [`internal/protocol`](protocol/README.md).
