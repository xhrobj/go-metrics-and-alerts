# internal/agent

Список собираемых Агентом метрик.

## Runtime-метрики

Агент откидывает в хранилище следующий список метрик из `runtime.MemStats`:

### 1. Общая память (классика мониторинга)

- `Alloc` — сколько байт реально занято сейчас
- `TotalAlloc` — сколько байт всего когда-либо выделялось
- `Sys` — сколько памяти Go запросил у ОС

### 2. Heap (что происходит с кучей: занято, простаивает, отдано ОС)

- `HeapAlloc`
- `HeapSys`
- `HeapIdle`
- `HeapInuse`
- `HeapReleased`
- `HeapObjects`

### 3. GC (насколько GC мешает программе жить)

- `NumGC`
- `NumForcedGC`
- `GCCPUFraction`
- `PauseTotalNs`
- `LastGC`
- `NextGC`
- `GCSys`

### 4. Внутренние структуры Go (внутренние детали аллокатора)

- `MSpanInuse`
- `MSpanSys`
- `MCacheInuse`
- `MCacheSys`
- `BuckHashSys`
- `OtherSys`
- `Lookups`

### 5. Стек (память под горутины)

- `StackInuse`
- `StackSys`

### 6. Аллокатор (счётчики активности)

- `Frees`
- `Mallocs`

## Системные метрики

Кроме runtime-метрик Агент собирает системные метрики через библиотеку `gopsutil`:

- `TotalMemory` - общий объём оперативной памяти
- `FreeMemory` - объём свободной оперативной памяти
- `CPUutilization1`
- `CPUutilization2`
- `...`
- `CPUutilizationN` - загрузка каждого доступного CPU в процентах
