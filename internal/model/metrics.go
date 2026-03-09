package model

const (
	// Counter — тип метрики со счётчиком (монотонно растущее целое значение)
	Counter = "counter"

	// Gauge — тип метрики с произвольным вещественным значением
	Gauge = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Ограничиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// чтобы отличать значение "0", от не заданного значения
// и, соответственно, не кодировать в структуру.
type Metrics struct {
	// ID — имя метрики
	ID string `json:"id"`

	// MType — тип метрики: gauge или counter
	MType string `json:"type"`

	// Delta — значение метрики типа counter
	Delta *int64 `json:"delta,omitempty"`

	// Value — значение метрики типа gauge
	Value *float64 `json:"value,omitempty"`

	Hash string `json:"hash,omitempty"`
}
