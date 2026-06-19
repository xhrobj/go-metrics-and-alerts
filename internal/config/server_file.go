package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type serverFileConfig struct {
	Address       *string `json:"address"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	Restore       *bool   `json:"restore"`
	DatabaseDSN   *string `json:"database_dsn"`
	Key           *string `json:"key"`
	CryptoKey     *string `json:"crypto_key"`
	AuditFile     *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`
}

func loadServerConfigFile(path string, cfg *ServerConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read server config file %q: %w", path, err)
	}

	var fileCfg serverFileConfig

	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("decode server config file %q: %w", path, err)
	}

	if fileCfg.Address != nil {
		cfg.ServerAddr = *fileCfg.Address
	}

	if fileCfg.StoreInterval != nil {
		storeInterval, err := time.ParseDuration(*fileCfg.StoreInterval)
		if err != nil {
			return fmt.Errorf(
				"parse store_interval %q: %w",
				*fileCfg.StoreInterval,
				err,
			)
		}

		if storeInterval%time.Second != 0 {
			return fmt.Errorf(
				"parse store_interval %q: duration must contain whole seconds",
				*fileCfg.StoreInterval,
			)
		}

		cfg.StoreIntervalInSec = int(storeInterval / time.Second)
	}

	if fileCfg.StoreFile != nil {
		cfg.FileStoragePath = *fileCfg.StoreFile
	}

	if fileCfg.Restore != nil {
		cfg.Restore = *fileCfg.Restore
	}

	if fileCfg.DatabaseDSN != nil {
		cfg.DatabaseDSN = *fileCfg.DatabaseDSN
	}

	if fileCfg.Key != nil {
		cfg.Key = *fileCfg.Key
	}

	if fileCfg.CryptoKey != nil {
		cfg.CryptoKey = *fileCfg.CryptoKey
	}

	if fileCfg.AuditFile != nil {
		cfg.AuditFile = *fileCfg.AuditFile
	}

	if fileCfg.AuditURL != nil {
		cfg.AuditURL = *fileCfg.AuditURL
	}

	return nil
}
