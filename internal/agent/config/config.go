package config

// Config содержит параметры конфигурации Агента.
type Config struct {
	// ServerAddr — адрес и порт HTTP-сервера сбора метрик.
	ServerAddr string

	// PollIntervalInSec — интервал опроса runtime-метрик в секундах.
	PollIntervalInSec int

	// ReportIntervalInSec — интервал отправки метрик на сервер в секундах.
	ReportIntervalInSec int

	// RateLimit — максимальное количество одновременно исходящих запросов от Агента к Серверу.
	RateLimit int

	// Key — "секретный" ключ для вычисления и проверки подписи HTTP-запросов/ответов.
	// Если не задан, подпись не используется.
	Key string

	// CryptoKey - путь к файлу публичного ключа для шифрования запросов.
	// Если не задан, шифрование не используется.
	CryptoKey string
}
