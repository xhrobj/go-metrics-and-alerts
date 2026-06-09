package model

const (
	// Counter — тип метрики со счётчиком (монотонно растущее целое значение)
	Counter = "counter"

	// Gauge — тип метрики с произвольным вещественным значением
	Gauge = "gauge"
)

// Metrics описывает метрику, передаваемую через API.
//
// Модель намеренно сделана плоской, без вложенных структур.
//
// Поля Delta и Value объявлены указателями, чтобы можно было
// отличить нулевое значение (0) от отсутствующего значения.
type Metrics struct {
	// ID — имя метрики
	ID string `json:"id"`

	// MType — тип метрики: gauge или counter
	MType string `json:"type"`

	// Delta — значение метрики типа counter
	Delta *int64 `json:"delta,omitempty"`

	// Value — значение метрики типа gauge
	Value *float64 `json:"value,omitempty"`

	// Hash содержит подпись метрики, рассчитанную по ключу.
	Hash string `json:"hash,omitempty"`
}
