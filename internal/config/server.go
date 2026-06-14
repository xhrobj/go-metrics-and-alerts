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

	// CryptoKey - путь к файлу приватного ключа для расшифровки запросов.
	// Если не задан, шифрование не используется.
	CryptoKey string

	// AuditFile - путь к файлу аудита.
	// Если не задан, аудит в файл отключён.
	AuditFile string

	// AuditURL - URL удаленного приёмника аудита.
	// Если не задан, удалённый аудит отключен.
	AuditURL string

	// ConfigPath - путь к JSON-файлу конфигурации.
	ConfigPath string
}

// GetServerConfig возвращает конфигурацию HTTP-сервера.
//
// Значения параметров могут быть заданы через:
//   - флаги: -a -i -f -r -d -k --crypto-key --audit-file --audit-url -c/--config
//   - переменные окружения: ADDRESS, STORE_INTERVAL, FILE_STORAGE_PATH, RESTORE, DATABASE_DSN, KEY, CRYPTO_KEY, AUDIT_FILE, AUDIT_URL, CONFIG
//
// Приоритет источников: env > flag > json > default.
func GetServerConfig() (ServerConfig, error) {
	return parseServerConfig(os.Args[1:], os.LookupEnv)
}

func parseServerConfig(args []string, lookupEnv lookupEnvFunc) (ServerConfig, error) {
	cfg := ServerConfig{}
	flags := flag.NewFlagSet("server", flag.ContinueOnError)

	flags.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flags.IntVar(&cfg.StoreIntervalInSec, "i", 300, "store interval in seconds")
	flags.StringVar(&cfg.FileStoragePath, "f", "metrics-db.json", "path to metrics storage file")
	flags.BoolVar(&cfg.Restore, "r", false, "restore metrics from file on startup")
	flags.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")
	flags.StringVar(&cfg.Key, "k", "", "hash key for request signing")
	flags.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to private crypto key")
	flags.StringVar(&cfg.AuditFile, "audit-file", "", "path to audit log file")
	flags.StringVar(&cfg.AuditURL, "audit-url", "", "audit receiver URL")
	flags.StringVar(&cfg.ConfigPath, "c", "", "path to JSON configuration file")
	flags.StringVar(&cfg.ConfigPath, "config", "", "path to JSON configuration file")

	if err := flags.Parse(args); err != nil {
		return cfg, err
	}

	if serverAddr, ok := lookupEnv("ADDRESS"); ok {
		cfg.ServerAddr = serverAddr
	}

	if storeIntervalInSec, ok, err := getEnvInt(lookupEnv, "STORE_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.StoreIntervalInSec = storeIntervalInSec
	}

	if fileStoragePath, ok := lookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = fileStoragePath
	}

	if restore, ok, err := getEnvBool(lookupEnv, "RESTORE"); err != nil {
		return cfg, err
	} else if ok {
		cfg.Restore = restore
	}

	if databaseDSN, ok := lookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = databaseDSN
	}

	if key, ok := lookupEnv("KEY"); ok {
		cfg.Key = key
	}

	if cryptoKey, ok := lookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = cryptoKey
	}

	if auditFile, ok := lookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = auditFile
	}

	if auditURL, ok := lookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = auditURL
	}

	if configPath, ok := lookupEnv("CONFIG"); ok {
		cfg.ConfigPath = configPath
	}

	return cfg, nil
}
