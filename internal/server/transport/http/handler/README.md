# internal/server/transport/http/handler

Пакет содержит HTTP-обработчики Сервера. Handler преобразует HTTP-запросы в вызовы `internal/server/service`, формирует ответы и отправляет события аудита после успешного обновления метрик.

```text
HTTP request
-> router / middleware
-> handler
-> server/service.MetricsService
-> repository
```

## Эндпоинты

### JSON API

- `POST /update` - обновить одну метрику и вернуть её итоговое значение
- `POST /updates` - атомарно обновить batch метрик
- `POST /value` - получить значение одной метрики

### Path-based API

- `POST /update/{type}/{name}/{value}` - обновить одну метрику
- `GET /value/{type}/{name}` - получить значение в текстовом виде

Эти маршруты сохранены для совместимости с ранними инкрементами.

### Служебные и пользовательские маршруты

- `GET /ping` - проверить доступность PostgreSQL; без настроенной или доступной БД возвращает `500 Internal Server Error`
- `GET /` - получить HTML-страницу со всеми известными метриками

## Форматы

Модель метрики описана в [`internal/model`](../../../../model/README.md).

- JSON-запросы используют `Content-Type: application/json`
- legacy update-маршрут использует `text/plain`
- Сервер умеет принимать gzip-сжатые тела

## Доверенная подсеть

Проверка доверенной подсети выполняется middleware только для маршрутов записи:

- `POST /update/{type}/{name}/{value}`
- `POST /update`
- `POST /updates`

При включённом `TRUSTED_SUBNET` handler вызывается только после успешной проверки `X-Real-IP`.

## Ответственность handler'а

Handler отвечает за:

- декодирование и базовую валидацию входных данных
- выбор операции по типу метрики
- вызов бизнес-логики через интерфейс `Service`
- преобразование ошибок в HTTP-статусы
- формирование JSON, text и HTML-ответов
- уведомление аудитора после успешной записи

Маршрутизация, логирование, хеширование, расшифрование, gzip и trusted subnet находятся в соседних пакетах `router` и `middleware`.
