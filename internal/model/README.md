# internal/model

Пакет содержит транспортную модель метрики, используемую HTTP API, Агентом, handler'ами, service и repository.

## Типы метрик

- `gauge` - произвольное значение `float64`; новое значение заменяет предыдущее
- `counter` - delta `int64`; новое значение прибавляется к накопленному счётчику

## `Metrics`

```go
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
```

Поля:

- `ID` - имя метрики
- `MType` - `gauge` или `counter`
- `Delta` - значение counter
- `Value` - значение gauge
- `Hash` - поле транспортной модели, которое в текущей реализации не используется; подпись всего HTTP-тела передаётся через заголовок `HashSHA256`

`Delta` и `Value` являются указателями, чтобы различать:

- поле отсутствует
- поле присутствует и равно нулю

Это важно для корректной валидации JSON. Для gauge ожидается `Value`, для counter - `Delta`.
