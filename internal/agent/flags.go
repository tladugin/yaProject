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
	if configPath := v.GetString("config.json"); configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Если не удалось распарсить, оставляем значение по умолчанию
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		f.FlagCryptoKey = envCryptoKey
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

// LoadAgentConfig загружает конфигурацию агента из файла
func LoadAgentConfig(configPath string) (*AgentConfig, error) {
	if configPath == "" {
		return &AgentConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetAgentConfig возвращает финальную конфигурацию агента с учетом приоритетов
func GetAgentConfig() (*AgentConfig, error) {
	flags := ParseFlags()

	// Получаем путь к конфигурационному файлу (флаг или переменная окружения)
	configPath := flags.FlagConfigFile
	if configPath == "" {
		if envConfig, ok := os.LookupEnv("CONFIG"); ok {
			configPath = envConfig
		}
	}

	// Загружаем конфигурацию из файла
	fileConfig, err := LoadAgentConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Создаем финальную конфигурацию
	config := &AgentConfig{}

	// Устанавливаем значения из файла конфигурации
	if fileConfig != nil {
		*config = *fileConfig
	}

	// Переопределяем значения из флагов (высший приоритет)
	if flags.FlagRunAddr != "localhost:8080" || config.Address == "" {
		config.Address = flags.FlagRunAddr
	}
	if flags.FlagReportIntervalTime != "10" || config.ReportInterval == "" {
		config.ReportInterval = flags.FlagReportIntervalTime
	}
	if flags.FlagPollIntervalTime != "2" || config.PollInterval == "" {
		config.PollInterval = flags.FlagPollIntervalTime
	}
	if flags.FlagKey != "" {
		config.Key = flags.FlagKey
	}
	if flags.FlagRateLimit != 1 {
		config.RateLimit = flags.FlagRateLimit
	}
	if flags.FlagUsePprof {
		config.UsePprof = flags.FlagUsePprof
	}
	if flags.FlagCryptoKey != "" {
		config.CryptoKey = flags.FlagCryptoKey
	}

	// Проверяем переменные окружения (средний приоритет)
	// Используем LookupEnv для точного контроля
	if envAddr, ok := os.LookupEnv("ADDRESS"); ok && flags.FlagRunAddr == "localhost:8080" {
		config.Address = envAddr
	}
	if envReportInterval, ok := os.LookupEnv("REPORT_INTERVAL"); ok && flags.FlagReportIntervalTime == "10" {
		config.ReportInterval = envReportInterval
	}
	if envPollInterval, ok := os.LookupEnv("POLL_INTERVAL"); ok && flags.FlagPollIntervalTime == "2" {
		config.PollInterval = envPollInterval
	}
	if envKey, ok := os.LookupEnv("KEY"); ok && flags.FlagKey == "" {
		config.Key = envKey
	}
	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok && flags.FlagRateLimit == 1 {
		if rateLimit, err := strconv.Atoi(envRateLimit); err == nil {
			config.RateLimit = rateLimit
		}
	}
	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok && flags.FlagCryptoKey == "" {
		config.CryptoKey = envCryptoKey
	}

	// Устанавливаем значения по умолчанию если не установлены
	if config.ReportInterval == "" {
		config.ReportInterval = "10"
	}
	if config.PollInterval == "" {
		config.PollInterval = "2"
	}
	if config.RateLimit == 0 {
		config.RateLimit = 1
	}

	return config, nil
}
