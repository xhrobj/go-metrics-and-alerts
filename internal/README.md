# internal

В директории `internal/` находится код внутренних пакетов проекта. Эти пакеты нельзя импортировать из модулей, расположенных за пределами родительского модуля.

## Структура

```text
internal/
├── agent/          # runtime, сервис и transport Агента
├── buildinfo/      # вывод версии, даты и Git-коммита сборки
├── config/         # загрузка и валидация конфигурации
├── encryption/     # RSA-ключи и гибридное шифрование
├── hash/           # вычисление HashSHA256
├── logger/         # инициализация zap
├── model/          # общая модель метрики
├── protocol/       # общие транспортные константы
├── repository/     # хранилища метрик
└── server/         # приложение Сервера и его внутренние пакеты
    ├── audit/      # события аудита и observers
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
