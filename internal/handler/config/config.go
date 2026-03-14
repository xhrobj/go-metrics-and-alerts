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

	// DatabaseDSN - строка подключения к базе данных
	DatabaseDSN string
}
