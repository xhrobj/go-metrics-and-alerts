package config

// Config содержит параметры конфигурации HTTP-сервера.
type Config struct {
	// ServerAddr — адрес и порт запуска HTTP-сервера.
	ServerAddr string

	// StoreIntervalInSec - интервал сохранения метрик на диск в секундах.
	StoreIntervalInSec int

	// FileStoragePath - путь к файлу хранения метрик.
	FileStoragePath string

	// Restore - определяет, нужно ли загружать метрики из файла при старте.
	Restore bool

	// DatabaseDSN — строка подключения к базе данных PostgreSQL.
	DatabaseDSN string

	// Key — "секретный" ключ для вычисления и проверки подписи HTTP-запросов/ответов.
	// Если не задан, подпись не используется.
	Key string

	// AuditFile - путь к файлу аудита.
	// Если не задан, аудит в файл отключён.
	AuditFile string

	// AuditURL - URL удаленного приёмника аудита.
	// Если не задан, удалённый аудит отключен.
	AuditURL string
}
