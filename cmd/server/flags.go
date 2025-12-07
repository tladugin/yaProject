package main

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Address       string `mapstructure:"address"`
	Restore       bool   `mapstructure:"restore"`
	StoreInterval int    `mapstructure:"store_interval"`
	StoreFile     string `mapstructure:"store_file"`
	DatabaseDSN   string `mapstructure:"database_dsn"`
	CryptoKey     string `mapstructure:"crypto_key"`
	Key           string `mapstructure:"key"`
	AuditFile     string `mapstructure:"audit_file"`
	AuditURL      string `mapstructure:"audit_url"`
	UsePprof      bool   `mapstructure:"use_pprof"`
	TrustedSubnet string `mapstructure:"trusted_subnet"`
}

func GetServerConfig() (*ServerConfig, error) {
	// Инициализируем Viper
	v := viper.New()

	// Устанавливаем значения по умолчанию
	setDefaults(v)

	// Настраиваем флаги
	setupFlags(v)

	// Настраиваем переменные окружения
	setupEnv(v)

	// Загружаем конфигурацию из файла (если указан)
	if configPath := v.GetString("config.json"); configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Создаем и заполняем структуру конфигурации
	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(v *viper.Viper) {
	v.SetDefault("address", "localhost:8080")
	v.SetDefault("store_interval", 300)
	v.SetDefault("store_file", "server_backup")
	v.SetDefault("restore", false)
	v.SetDefault("use_pprof", false)
	v.SetDefault("trusted_subnet", "")
}

// setupFlags настраивает флаги
func setupFlags(v *viper.Viper) {
	// Создаем pflag set
	pflag.StringP("address", "a", "localhost:8080", "address and port to run server")
	pflag.IntP("store_interval", "i", 300, "saving server data interval in seconds") // Исправлено: Int вместо String
	pflag.StringP("store_file", "f", "server_backup", "path for server backup file")
	pflag.BoolP("restore", "r", false, "restore server data")
	pflag.StringP("database_dsn", "d", "", "database DSN")
	pflag.StringP("key", "k", "", "key")
	pflag.String("audit-file", "", "path for server audit file")
	pflag.String("audit-url", "", "audit URL")
	pflag.Bool("pprof", false, "use benchmark")
	pflag.String("crypto-key", "", "path to private key for decryption")
	pflag.StringP("config", "c", "", "path to config file")
	pflag.StringP("trusted-subnet", "t", "", "trusted subnet in CIDR notation")

	// Привязываем флаги к Viper
	v.BindPFlags(pflag.CommandLine)

	// Парсим флаги
	pflag.Parse()
}

// setupEnv настраивает переменные окружения
func setupEnv(v *viper.Viper) {
	// Автоматическое связывание переменных окружения
	v.AutomaticEnv()
	v.SetEnvPrefix("METRICS") // Префикс для переменных окружения
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Явные привязки для сложных случаев
	v.BindEnv("address", "ADDRESS")
	v.BindEnv("restore", "RESTORE")
	v.BindEnv("store_interval", "STORE_INTERVAL")
	v.BindEnv("store_file", "STORE_FILE")
	v.BindEnv("database_dsn", "DATABASE_DSN")
	v.BindEnv("crypto_key", "CRYPTO_KEY")
	v.BindEnv("key", "KEY")
	v.BindEnv("audit_file", "AUDIT_FILE")
	v.BindEnv("audit_url", "AUDIT_URL")
	v.BindEnv("use_pprof", "USE_PPROF")
	v.BindEnv("config", "CONFIG")
	v.BindEnv("trusted_subnet", "TRUSTED_SUBNET") // Добавлена привязка для trusted_subnet
}
