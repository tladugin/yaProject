package main

import (
	"flag"
	"github.com/tladugin/yaProject.git/internal/logger"
	"os"
	"strconv"
	"strings"
)

var (
	flagRunAddr            string
	flagReportIntervalTime string
	flagPollIntervalTime   string
	flagKey                string
	flagRateLimit          int
)

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {

	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&flagReportIntervalTime, "r", "10", "time interval to report")
	flag.StringVar(&flagPollIntervalTime, "p", "2", "poll interval")
	flag.StringVar(&flagKey, "k", "", "key")
	flag.IntVar(&flagRateLimit, "l", 1, "rate limit (max concurrent requests)")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	envRunAddr, ok := os.LookupEnv("ADDRESS")
	if ok && strings.TrimSpace(envRunAddr) != "" {
		flagRunAddr = envRunAddr
	}

	envReportInter, ok := os.LookupEnv("REPORT_INTERVAL")
	if ok && strings.TrimSpace(envReportInter) != "" {
		flagReportIntervalTime = envReportInter
	}

	envPortInter, ok := os.LookupEnv("POLL_INTERVAL")
	if ok && strings.TrimSpace(envPortInter) != "" {
		flagPollIntervalTime = envPortInter
	}
	envKey, ok := os.LookupEnv("KEY")
	if ok && strings.TrimSpace(envKey) != "" {
		flagKey = envKey
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
}
