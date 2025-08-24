package agent

import (
	"flag"
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
	return &f
}
