package main

import (
	"flag"
	"os"
	"strings"
)

var (
	flagRunAddr            string
	flagReportIntervalTime string
	flagPollIntervalTime   string
	flagKey                string
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

}
