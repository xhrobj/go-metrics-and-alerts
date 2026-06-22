# internal/agent

Пакет разделяет runtime Агента, сервис метрик и транспорт отправки.

```text
agent.Agent
-> agent/service.ReportingService
  -> agent/service.MetricsSender
    <- agent/transport/http.Sender
    <- agent/transport/grpc.Sender (И28)
```

```text
internal/agent/
├── agent.go
├── report.go
├── service/
│   ├── service.go
│   ├── poll.go
│   ├── report.go
│   └── sender.go
└── transport/
    └── http/
        └── sender.go
```

- `agent` управляет ticker'ами, очередью, worker'ами и graceful shutdown
- `agent/service` собирает метрики, хранит `PollCount`, формирует batch и восстанавливает счетчик при ошибках
- `agent/service.MetricsSender` задает транспортный порт отправки
- `agent/transport/http` реализует текущую HTTP-доставку
- `cmd/agent` создаёт конкретный транспорт, сервис и runtime Агента

## Runtime-метрики

Агент сохраняет в локальное хранилище gauge-метрики из `runtime.MemStats`.

### Общая память

- `Alloc`
- `TotalAlloc`
- `Sys`

### Heap

- `HeapAlloc`
- `HeapSys`
- `HeapIdle`
- `HeapInuse`
- `HeapReleased`
- `HeapObjects`

### Garbage Collector

- `NumGC`
- `NumForcedGC`
- `GCCPUFraction`
- `PauseTotalNs`
- `LastGC`
- `NextGC`
- `GCSys`

### Внутренние структуры аллокатора

- `MSpanInuse`
- `MSpanSys`
- `MCacheInuse`
- `MCacheSys`
- `BuckHashSys`
- `OtherSys`
- `Lookups`

### Стек

- `StackInuse`
- `StackSys`

### Счётчики аллокаций

- `Frees`
- `Mallocs`

## Дополнительные метрики

- `RandomValue` - gauge со случайным значением, обновляется вместе с runtime-метриками
- `PollCount` - counter с количеством runtime-опросов после последней успешной отправки отчёта

`PollCount` не хранится как обычная gauge-метрика. Значение накапливается отдельно и добавляется к каждому batch-отчёту как delta counter. При ошибке формирования или отправки отчёта значение возвращается в накопитель.

## Системные метрики

Через `gopsutil` собираются:

- `TotalMemory` - общий объём оперативной памяти
- `FreeMemory` - свободная оперативная память
- `CPUutilization1` ... `CPUutilizationN` - загрузка каждого логического CPU в процентах

Все системные метрики имеют тип gauge.

## Отправка

Отчёт формируется из снимка всех gauge-метрик и текущей delta `PollCount`. После чего `agent/service.ReportingService` передает готовый batch через `MetricsSender`.

```text
snapshot
-> []model.Metrics
-> MetricsSender
   -> agent/transport/http.Sender
      -> JSON
      -> gzip
      -> optional encryption
      -> optional HashSHA256
      -> POST /updates
```

Задачи отправки проходят через буферизированную очередь, которой управляет `Agent`. Количество worker'ов и размер очереди определяются `RATE_LIMIT`.



