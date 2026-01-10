package repository

type AgentStorage interface {
	UpdateGauge(string, float64)
	UpdateCounter(string, int64)
	Snapshot() (map[string]float64, map[string]int64)
}

type ServerStorage interface {
	UpdateGauge(string, float64)
	UpdateCounter(string, int64)
	Snapshot() (map[string]float64, map[string]int64)
	GetGauge(string) (float64, error)
	GetCounter(string) (int64, error)
}
