package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type flags struct {
	flagRunAddr         string
	flagStoreInterval   string
	flagFileStoragePath string
	flagRestore         bool
	flagDatabaseDSN     string
	flagKey             string
	flagAuditFile       string
	flagAuditURL        string
	flagUsePprof        bool
	flagCryptoKey       string
	flagConfigFile      string
}

// "host=localhost user=postgres password=543218 dbname=metrics sslmode=disable"

// LoadServerConfig загружает конфигурацию сервера из файла
func LoadServerConfig(configPath string) (*ServerConfig, error) {
	if configPath == "" {
		return &ServerConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// parseFlagsServer обновленная функция парсинга флагов сервера
func parseFlagsServer() *flags {
	f := &flags{}

	// Основные флаги
	flag.StringVar(&f.flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&f.flagStoreInterval, "i", "300", "saving server data interval")
	flag.StringVar(&f.flagFileStoragePath, "f", "server_backup", "path for server backup file")
	flag.BoolVar(&f.flagRestore, "r", false, "restore server data")
	flag.StringVar(&f.flagDatabaseDSN, "d", "", "database DSN")
	flag.StringVar(&f.flagKey, "k", "", "key")
	flag.StringVar(&f.flagAuditFile, "audit-file", "", "path for server audit file")
	flag.StringVar(&f.flagAuditURL, "audit-url", "", "audit URL")
	flag.BoolVar(&f.flagUsePprof, "pprof", false, "use benchmark")
	flag.StringVar(&f.flagCryptoKey, "crypto-key", "", "path to private key for decryption")

	flag.StringVar(&f.flagConfigFile, "config", "", "path to config file")

	flag.Parse()

	return f
}

// ServerConfig структура для конфигурации сервера
type ServerConfig struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	Key           string `json:"key"`
	AuditFile     string `json:"audit_file"`
	AuditURL      string `json:"audit_url"`
	UsePprof      bool   `json:"use_pprof"`
}

// GetServerConfig возвращает финальную конфигурацию сервера с учетом приоритетов
func GetServerConfig() (*ServerConfig, error) {
	flags := parseFlagsServer()

	// Получаем путь к конфигурационному файлу (флаг или переменная окружения)
	configPath := flags.flagConfigFile
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// Загружаем конфигурацию из файла
	fileConfig, err := LoadServerConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Создаем финальную конфигурацию с учетом приоритетов
	config := &ServerConfig{}

	// Устанавливаем значения из файла конфигурации
	if fileConfig != nil {
		*config = *fileConfig
	}

	// Переопределяем значения из флагов (высший приоритет)
	if flags.flagRunAddr != "localhost:8080" || config.Address == "" {
		config.Address = flags.flagRunAddr
	}
	if flags.flagRestore || !config.Restore && flags.flagRestore {
		config.Restore = flags.flagRestore
	}
	if flags.flagStoreInterval != "300" || config.StoreInterval == "" {
		config.StoreInterval = flags.flagStoreInterval
	}
	if flags.flagFileStoragePath != "server_backup" || config.StoreFile == "" {
		config.StoreFile = flags.flagFileStoragePath
	}
	if flags.flagDatabaseDSN != "" || config.DatabaseDSN == "" {
		config.DatabaseDSN = flags.flagDatabaseDSN
	}
	if flags.flagCryptoKey != "" {
		config.CryptoKey = flags.flagCryptoKey
	}
	if flags.flagKey != "" {
		config.Key = flags.flagKey
	}
	if flags.flagAuditFile != "" {
		config.AuditFile = flags.flagAuditFile
	}
	if flags.flagAuditURL != "" {
		config.AuditURL = flags.flagAuditURL
	}
	if flags.flagUsePprof {
		config.UsePprof = flags.flagUsePprof
	}

	// Проверяем переменные окружения (средний приоритет)
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" && flags.flagRunAddr == "localhost:8080" {
		config.Address = envAddr
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" && !flags.flagRestore {
		config.Restore = envRestore == "true"
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" && flags.flagStoreInterval == "300" {
		config.StoreInterval = envStoreInterval
	}
	if envStoreFile := os.Getenv("STORE_FILE"); envStoreFile != "" && flags.flagFileStoragePath == "server_backup" {
		config.StoreFile = envStoreFile
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" && flags.flagDatabaseDSN == "" {
		config.DatabaseDSN = envDatabaseDSN
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" && flags.flagCryptoKey == "" {
		config.CryptoKey = envCryptoKey
	}

	return config, nil
}
