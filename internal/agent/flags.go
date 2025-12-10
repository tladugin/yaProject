package agent

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type AgentConfig struct {
	Address        string `mapstructure:"address"`
	GRPCAddress    string `mapstructure:"grpc_address"`
	UseGRPC        bool   `mapstructure:"use_grpc"`
	ReportInterval string `mapstructure:"report_interval"`
	PollInterval   string `mapstructure:"poll_interval"`
	Key            string `mapstructure:"key"`
	RateLimit      int    `mapstructure:"rate_limit"`
	UsePprof       bool   `mapstructure:"use_pprof"`
	CryptoKey      string `mapstructure:"crypto_key"`
	LocalIP        string `mapstructure:"local_ip"`
}

func GetAgentConfig() (*AgentConfig, error) {
	// Инициализируем Viper
	v := viper.New()

	// Устанавливаем значения по умолчанию
	setDefaults(v)

	// Настраиваем флаги
	setupFlags(v)

	// Настраиваем переменные окружения
	setupEnv(v)

	// Загружаем конфигурацию из файла (если указан)
	configPath := v.GetString("config.json")
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Создаем и заполняем структуру конфигурации
	var config AgentConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(v *viper.Viper) {
	v.SetDefault("address", "localhost:8080")
	v.SetDefault("grpc_address", ":3200")
	v.SetDefault("use_grpc", false)
	v.SetDefault("report_interval", "10")
	v.SetDefault("poll_interval", "2")
	v.SetDefault("rate_limit", 1)
	v.SetDefault("use_pprof", false)
	v.SetDefault("local_ip", "")
}

// setupFlags настраивает флаги
func setupFlags(v *viper.Viper) {
	// Создаем pflag set
	pflag.StringP("address", "a", "localhost:8080", "address and port to run server")
	pflag.String("grpc-address", ":3200", "gRPC server address")
	pflag.Bool("use-grpc", false, "use gRPC instead of HTTP")
	pflag.StringP("report_interval", "r", "10", "time interval to report")
	pflag.StringP("poll_interval", "p", "2", "poll interval")
	pflag.StringP("key", "k", "", "key")
	pflag.IntP("rate_limit", "l", 1, "rate limit (max concurrent requests)")
	pflag.Bool("pprof", false, "use benchmark")
	pflag.String("crypto-key", "", "path to public key for encryption")
	pflag.StringP("config", "c", "", "path to config file")
	pflag.String("local-ip", "", "local IP address to send in X-Real-IP header")

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
	v.BindEnv("grpc_address", "GRPC_ADDRESS")
	v.BindEnv("use_grpc", "USE_GRPC")
	v.BindEnv("report_interval", "REPORT_INTERVAL")
	v.BindEnv("poll_interval", "POLL_INTERVAL")
	v.BindEnv("key", "KEY")
	v.BindEnv("rate_limit", "RATE_LIMIT")
	v.BindEnv("use_pprof", "USE_PPROF")
	v.BindEnv("crypto_key", "CRYPTO_KEY")
	v.BindEnv("config", "CONFIG")
	v.BindEnv("local_ip", "LOCAL_IP")
}
