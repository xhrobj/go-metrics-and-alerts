package config

import (
	"flag"
	"os"
)

// ServerConfig содержит параметры конфигурации HTTP-сервера.
type ServerConfig struct {
	// ServerAddr - адрес и порт запуска HTTP-сервера.
	ServerAddr string

	// StoreIntervalInSec - интервал сохранения метрик на диск в секундах.
	StoreIntervalInSec int

	// FileStoragePath - путь к файлу хранения метрик.
	FileStoragePath string

	// Restore - определяет, нужно ли загружать метрики из файла при старте.
	Restore bool

	// DatabaseDSN - строка подключения к базе данных PostgreSQL.
	DatabaseDSN string

	// Key - "секретный" ключ для вычисления и проверки подписи HTTP-запросов/ответов.
	// Если не задан, подпись не используется.
	Key string

	// AuditFile - путь к файлу аудита.
	// Если не задан, аудит в файл отключён.
	AuditFile string

	// AuditURL - URL удаленного приёмника аудита.
	// Если не задан, удалённый аудит отключен.
	AuditURL string

	// CryptoKey - путь к файлу приватного ключа для расшифровки запросов.
	// Если не задан, шифрование не используется.
	CryptoKey string
}

// GetServerConfig возвращает конфигурацию HTTP-сервера.
//
// Значения параметров могут быть заданы через:
//   - флаги: -a -i -f -r -d -k --crypto-key --audit-file --audit-url
//   - переменные окружения: ADDRESS, STORE_INTERVAL, FILE_STORAGE_PATH, RESTORE, DATABASE_DSN, KEY, CRYPTO_KEY, AUDIT_FILE, AUDIT_URL
//
// Приоритет источников: env > flag > default.
func GetServerConfig() (ServerConfig, error) {
	cfg := ServerConfig{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&cfg.StoreIntervalInSec, "i", 300, "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", "metrics-db.json", "path to metrics storage file")
	flag.BoolVar(&cfg.Restore, "r", false, "restore metrics from file on startup")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")
	flag.StringVar(&cfg.Key, "k", "", "hash key for request signing")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to private crypto key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit receiver URL")

	flag.Parse()

	if serverAddr, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddr = serverAddr
	}

	if storeIntervalInSec, ok, err := getEnvInt("STORE_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.StoreIntervalInSec = storeIntervalInSec
	}

	if fileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = fileStoragePath
	}

	if restore, ok, err := getEnvBool("RESTORE"); err != nil {
		return cfg, err
	} else if ok {
		cfg.Restore = restore
	}

	if databaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = databaseDSN
	}

	if key, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = key
	}

	if cryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = cryptoKey
	}

	if auditFile, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = auditFile
	}

	if auditURL, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = auditURL
	}

	return cfg, nil
}
