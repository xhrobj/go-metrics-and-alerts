# cmd/server

В данной директории содержится код HTTP-сервера сбора метрик.

Сервер принимает метрики от агентов, хранит их в памяти и предоставляет HTTP-интерфейс для получения текущих значений.

## Конфигурация сервера

Сервер поддерживает настройку параметров через флаги командной строки и переменные окружения.

Приоритет источников конфигурации: `env > flag > default`

## Параметры запуска

- флаг `-a` или env `ADDRESS` —> адрес HTTP-сервера и порт (по умолчанию `localhost:8080`)
- флаг `-i` или env `STORE_INTERVAL` —> интервал сохранения метрик на диск в секундах (по умолчанию `300`)
- флаг `-f` или env `FILE_STORAGE_PATH` —> путь к файлу хранения метрик (по умолчанию `metrics-db.json`)
- флаг `-r` или env `RESTORE` —> загружать метрики из файла при старте (по умолчанию `false`)
- флаг `-d` или env `DATABASE_DSN` —> строка подключения к базе данных PostgreSQL (по умолчанию ``)

## Пример запуска

```bash
./server -a localhost:8080 -i 300 -f metrics-db.json -r=true -d=postgres://metrics:password@localhost:5432/metricsdb
```

или через переменные окружения:

```
ADDRESS=localhost:8080
STORE_INTERVAL=300
FILE_STORAGE_PATH=metrics-db.json
RESTORE=true
DATABASE_DSN=postgres://metrics:password@localhost:5432/metricsdb

./server
```
