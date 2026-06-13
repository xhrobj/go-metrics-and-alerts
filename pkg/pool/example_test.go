package pool_test

import (
	"fmt"

	"github.com/xhrobj/go-metrics-and-alerts/pkg/pool"
)

type metricBatch struct {
	Metrics []string
}

func newMetricBatch() *metricBatch {
	return &metricBatch{
		Metrics: make([]string, 0, 4),
	}
}

func (b *metricBatch) Reset() {
	b.Metrics = b.Metrics[:0]
}

func ExamplePool() {
	batchPool, err := pool.New(newMetricBatch)
	if err != nil {
		fmt.Println(err)
		return
	}

	// берем временный объект из пула и заполняем его данными
	batch := batchPool.Get()
	batch.Metrics = append(batch.Metrics, "Alloc", "HeapAlloc")

	fmt.Println(len(batch.Metrics))

	// Put вызывает Reset перед возвратом объекта в пул
	batchPool.Put(batch)

	fmt.Println(len(batch.Metrics))

	// Output:
	// 2
	// 0
}
