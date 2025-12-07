package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Flags struct {
	FlagRunAddr            string
	FlagReportIntervalTime string
	FlagPollIntervalTime   string
	FlagKey                string
	FlagRateLimit          int
	FlagUsePprof           bool
	FlagCryptoKey          string
	FlagConfigFile         string
	FlagLocalIP            string
}

type AgentConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	Key            string `json:"key"`
	RateLimit      int    `json:"rate_limit"`
	UsePprof       bool   `json:"use_pprof"`
	CryptoKey      string `json:"crypto_key"`
	LocalIP        string `json:"local_ip"`
}

// parseFlags обрабатывает аргументы командной строки
func ParseFlags() *Flags {
	var f Flags

	// Регистрируем флаги
	flag.StringVar(&f.FlagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&f.FlagReportIntervalTime, "r", "10", "time interval to report")
	flag.StringVar(&f.FlagPollIntervalTime, "p", "2", "poll interval")
	flag.StringVar(&f.FlagKey, "k", "", "key")
	flag.IntVar(&f.FlagRateLimit, "l", 1, "rate limit (max concurrent requests)")
	flag.BoolVar(&f.FlagUsePprof, "pprof", false, "use benchmark")
	flag.StringVar(&f.FlagCryptoKey, "crypto-key", "", "path to public key for encryption")
	flag.StringVar(&f.FlagConfigFile, "c", "", "path to config file")
	flag.StringVar(&f.FlagConfigFile, "config", "", "path to config file")
	flag.StringVar(&f.FlagLocalIP, "ip", "", "local IP address to send in X-Real-IP header")

	flag.Parse()

	// Обрабатываем переменные окружения с помощью LookupEnv
	if envRunAddr, ok := os.LookupEnv("ADDRESS"); ok {
		f.FlagRunAddr = envRunAddr
	}

	if envReportInter, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		f.FlagReportIntervalTime = envReportInter
	}

	if envPollInterval, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		f.FlagPollIntervalTime = envPollInterval
	}

	if envKey, ok := os.LookupEnv("KEY"); ok {
		f.FlagKey = envKey
	}

	if envRateLimit, ok := os.LookupEnv("RATE_LIMIT"); ok {
		if rateLimit, err := strconv.Atoi(envRateLimit); err == nil {
			f.FlagRateLimit = rateLimit
		}
		// Если не удалось распарсить, оставляем значение по умолчанию
	}

	if envCryptoKey, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		f.FlagCryptoKey = envCryptoKey
	}

	// Новая переменная окружения для IP
	if envLocalIP, ok := os.LookupEnv("LOCAL_IP"); ok {
		f.FlagLocalIP = envLocalIP
	}

	return &f
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
	if flags.FlagLocalIP != "" {
		config.LocalIP = flags.FlagLocalIP
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
	if envLocalIP, ok := os.LookupEnv("LOCAL_IP"); ok && flags.FlagLocalIP == "" {
		config.LocalIP = envLocalIP
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
