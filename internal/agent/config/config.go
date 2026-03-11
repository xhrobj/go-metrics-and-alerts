package config

// Config содержит параметры конфигурации Агента.
type Config struct {
	// ServerAddr — адрес и порт HTTP-сервера сбора метрик.
	ServerAddr string

	// PollIntervalInSec — интервал опроса runtime-метрик в секундах.
	PollIntervalInSec int

	// ReportIntervalInSec — интервал отправки метрик на сервер в секундах.
	ReportIntervalInSec int
}
