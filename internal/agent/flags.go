package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tladugin/yaProject.git/internal/logger"
	"os"
	"strconv"
	"strings"
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
}

type AgentConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	Key            string `json:"key"`
	RateLimit      int    `json:"rate_limit"`
	UsePprof       bool   `json:"use_pprof"`
	CryptoKey      string `json:"crypto_key"`
}

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func ParseFlags() *Flags {
	var f Flags
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&f.FlagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&f.FlagReportIntervalTime, "r", "10", "time interval to report")
	flag.StringVar(&f.FlagPollIntervalTime, "p", "2", "poll interval")
	flag.StringVar(&f.FlagKey, "k", "", "key")
	flag.IntVar(&f.FlagRateLimit, "l", 1, "rate limit (max concurrent requests)")
	flag.BoolVar(&f.FlagUsePprof, "pprof", false, "use benchmark")
	flag.StringVar(&f.FlagCryptoKey, "crypto-key", "", "path to public key for encryption")
	flag.StringVar(&f.FlagConfigFile, "config", "", "path to config file")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	envRunAddr, ok := os.LookupEnv("ADDRESS")
	if ok && strings.TrimSpace(envRunAddr) != "" {
		f.FlagRunAddr = envRunAddr
	}

	envReportInter, ok := os.LookupEnv("REPORT_INTERVAL")
	if ok && strings.TrimSpace(envReportInter) != "" {
		f.FlagReportIntervalTime = envReportInter
	}

	envPortInter, ok := os.LookupEnv("POLL_INTERVAL")
	if ok && strings.TrimSpace(envPortInter) != "" {
		f.FlagPollIntervalTime = envPortInter
	}
	envKey, ok := os.LookupEnv("KEY")
	if ok && strings.TrimSpace(envKey) != "" {
		f.FlagKey = envKey
	}
	envRateLimit, ok := os.LookupEnv("RATE_LIMIT")
	if ok && strings.TrimSpace(envRateLimit) != "" {
		flagRateLimit, err := strconv.Atoi(envRateLimit)
		if err == nil {
			logger.Sugar.Info("Using rate limit: " + strconv.Itoa(flagRateLimit))
		} else {
			logger.Sugar.Info("Can't parse envRateLimit")
		}
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		f.FlagCryptoKey = envCryptoKey
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
		configPath = os.Getenv("CONFIG")
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
	if flags.FlagReportIntervalTime != "10" {
		config.ReportInterval = flags.FlagReportIntervalTime
	}
	if flags.FlagPollIntervalTime != "2" {
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
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" && flags.FlagRunAddr == "localhost:8080" {
		config.Address = envAddr
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" && flags.FlagReportIntervalTime == "10" {
		config.ReportInterval = envReportInterval
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" && flags.FlagPollIntervalTime == "2" {
		config.PollInterval = envPollInterval
	}
	if envKey := os.Getenv("KEY"); envKey != "" && flags.FlagKey == "" {
		config.Key = envKey
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" && flags.FlagRateLimit == 1 {
		if rateLimit, err := strconv.Atoi(envRateLimit); err == nil {
			config.RateLimit = rateLimit
		}
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" && flags.FlagCryptoKey == "" {
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
