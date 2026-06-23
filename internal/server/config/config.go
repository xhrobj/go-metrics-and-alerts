package config

import (
	"flag"
	"os"
)

// ServerConfig содержит параметры конфигурации Сервера.
type ServerConfig struct {
	// HTTPAddr - адрес и порт запуска HTTP-Сервера.
	HTTPAddr string

	// GRPCAddr - адрес и порт запуска gRPC-Сервера.
	GRPCAddr string

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

	// TrustedSubnet - доверенная подсеть в формате CIDR.
	// Если не задана, проверка IP-адреса Агента отключена.
	TrustedSubnet string

	// AuditFile - путь к файлу аудита.
	// Если не задан, аудит в файл отключён.
	AuditFile string

	// AuditURL - URL удаленного приёмника аудита.
	// Если не задан, удалённый аудит отключен.
	AuditURL string
}

// GetServerConfig возвращает конфигурацию Сервера.
//
// Значения параметров могут быть заданы через:
//   - флаги: -a -g -i -f -r -d -k --crypto-key -t --audit-file --audit-url -c/--config
//   - переменные окружения:
//     ADDRESS,
//     GRPC_ADDRESS,
//     STORE_INTERVAL,
//     FILE_STORAGE_PATH / STORE_FILE,
//     RESTORE,
//     DATABASE_DSN,
//     KEY,
//     CRYPTO_KEY,
//     TRUSTED_SUBNET,
//     AUDIT_FILE,
//     AUDIT_URL,
//     CONFIG
//   - JSON-файл конфигурации
//
// Приоритет источников: env > flag > json > default.
func GetServerConfig() (ServerConfig, error) {
	return parseServerConfig(os.Args[1:], os.LookupEnv)
}

func parseServerConfig(args []string, lookupEnv lookupEnvFunc) (ServerConfig, error) {
	cfg := defaultServerConfig()
	flagCfg := cfg

	flags := flag.NewFlagSet("server", flag.ContinueOnError)

	flags.StringVar(&flagCfg.HTTPAddr, "a", flagCfg.HTTPAddr, "address and port to run HTTP server")
	flags.StringVar(&flagCfg.GRPCAddr, "g", flagCfg.GRPCAddr, "address and port to run gRPC server")
	flags.IntVar(&flagCfg.StoreIntervalInSec, "i", flagCfg.StoreIntervalInSec, "store interval in seconds")
	flags.StringVar(&flagCfg.FileStoragePath, "f", flagCfg.FileStoragePath, "path to metrics storage file")
	flags.BoolVar(&flagCfg.Restore, "r", flagCfg.Restore, "restore metrics from file on startup")
	flags.StringVar(&flagCfg.DatabaseDSN, "d", flagCfg.DatabaseDSN, "database connection string")
	flags.StringVar(&flagCfg.Key, "k", flagCfg.Key, "hash key for request signing")
	flags.StringVar(&flagCfg.CryptoKey, "crypto-key", flagCfg.CryptoKey, "path to private crypto key")
	flags.StringVar(&flagCfg.TrustedSubnet, "t", flagCfg.TrustedSubnet, "trusted subnet in CIDR notation")
	flags.StringVar(&flagCfg.AuditFile, "audit-file", flagCfg.AuditFile, "path to audit log file")
	flags.StringVar(&flagCfg.AuditURL, "audit-url", flagCfg.AuditURL, "audit receiver URL")

	var configPath string
	flags.StringVar(&configPath, "c", "", "path to JSON configuration file")
	flags.StringVar(&configPath, "config", "", "path to JSON configuration file")

	if err := flags.Parse(args); err != nil {
		return cfg, err
	}

	setFlags := make(map[string]bool)

	flags.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if value, ok := lookupEnv("CONFIG"); ok {
		configPath = value
	}

	if configPath != "" {
		if err := loadServerConfigFile(configPath, &cfg); err != nil {
			return cfg, err
		}
	}

	applyServerFlags(&cfg, flagCfg, setFlags)

	if err := applyServerEnvironment(&cfg, lookupEnv); err != nil {
		return cfg, err
	}

	if err := validateServerConfig(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		HTTPAddr:           "localhost:8080",
		GRPCAddr:           "localhost:50051",
		StoreIntervalInSec: 300,
		FileStoragePath:    "metrics-db.json",
	}
}

func applyServerFlags(cfg *ServerConfig, flagCfg ServerConfig, setFlags map[string]bool) {
	if setFlags["a"] {
		cfg.HTTPAddr = flagCfg.HTTPAddr
	}

	if setFlags["g"] {
		cfg.GRPCAddr = flagCfg.GRPCAddr
	}

	if setFlags["i"] {
		cfg.StoreIntervalInSec = flagCfg.StoreIntervalInSec
	}

	if setFlags["f"] {
		cfg.FileStoragePath = flagCfg.FileStoragePath
	}

	if setFlags["r"] {
		cfg.Restore = flagCfg.Restore
	}

	if setFlags["d"] {
		cfg.DatabaseDSN = flagCfg.DatabaseDSN
	}

	if setFlags["k"] {
		cfg.Key = flagCfg.Key
	}

	if setFlags["crypto-key"] {
		cfg.CryptoKey = flagCfg.CryptoKey
	}

	if setFlags["t"] {
		cfg.TrustedSubnet = flagCfg.TrustedSubnet
	}

	if setFlags["audit-file"] {
		cfg.AuditFile = flagCfg.AuditFile
	}

	if setFlags["audit-url"] {
		cfg.AuditURL = flagCfg.AuditURL
	}
}

func applyServerEnvironment(cfg *ServerConfig, lookupEnv lookupEnvFunc) error {
	if httpAddr, ok := lookupEnv("ADDRESS"); ok {
		cfg.HTTPAddr = httpAddr
	}

	if grpcAddr, ok := lookupEnv("GRPC_ADDRESS"); ok {
		cfg.GRPCAddr = grpcAddr
	}

	if storeIntervalInSec, ok, err := getEnvInt(lookupEnv, "STORE_INTERVAL"); err != nil {
		return err
	} else if ok {
		cfg.StoreIntervalInSec = storeIntervalInSec
	}

	if fileStoragePath, ok := lookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = fileStoragePath
	}

	// NOTE: STORE_FILE указан в задании С8И25 (см. корневой README) как новое имя для FILE_STORAGE_PATH
	if fileStoragePath, ok := lookupEnv("STORE_FILE"); ok {
		cfg.FileStoragePath = fileStoragePath
	}

	if restore, ok, err := getEnvBool(lookupEnv, "RESTORE"); err != nil {
		return err
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

	if trustedSubnet, ok := lookupEnv("TRUSTED_SUBNET"); ok {
		cfg.TrustedSubnet = trustedSubnet
	}

	if auditFile, ok := lookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = auditFile
	}

	if auditURL, ok := lookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = auditURL
	}

	return nil
}
