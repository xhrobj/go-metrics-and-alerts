package config

// Config содержит параметры конфигурации Агента.
type Config struct {
	// ServerAddr — адрес и порт HTTP-сервера сбора метрик.
	ServerAddr string

	// PollIntervalInSec — интервал опроса runtime-метрик в секундах.
	PollIntervalInSec int

	// ReportIntervalInSec — интервал отправки метрик на сервер в секундах.
	ReportIntervalInSec int

	// Key — "секретный" ключ для вычисления и проверки подписи HTTP-запросов/ответов.
	// Если не задан, подпись не используется.
	Key string
}
