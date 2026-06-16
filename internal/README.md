# internal

В директории `internal/` находится код внутренних пакетов проекта. Эти пакеты нельзя импортировать из модулей, расположенных за пределами родительского модуля.

## Структура

```text
internal/
├── agent/       # сбор, формирование и отправка метрик
├── audit/       # события аудита и observers
├── buildinfo/   # вывод версии, даты и Git-коммита сборки
├── config/      # загрузка и валидация конфигурации
├── encryption/  # RSA-ключи и гибридное шифрование
├── handler/     # HTTP-обработчики
├── hash/        # вычисление HashSHA256
├── logger/      # инициализация zap
├── middleware/  # HTTP middleware
├── migrations/  # запуск SQL-миграций
├── model/       # транспортная модель метрики
├── protocol/    # общие элементы транспортного протокола
├── repository/  # хранилища метрик
├── router/      # маршруты и middleware-цепочка
└── service/     # бизнес-логика метрик
```

Пакеты разделены по ответственности:

```text
HTTP -> router/middleware -> handler -> service -> repository
```

Агент использует собственную runtime-цепочку:

```text
poll loops -> local storage -> report queue -> HTTP
```

Подробности находятся в README отдельных пакетов.
