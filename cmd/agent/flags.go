package main

import (
	"flag"
	"os"
	"strings"
)

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() []string {

	var flagRunAddr string
	var reportIntervalTime string
	var pollIntervalTime string

	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&reportIntervalTime, "r", "10", "time interval to report")
	flag.StringVar(&pollIntervalTime, "p", "2", "poll interval")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	envRunAddr, ok := os.LookupEnv("ADDRESS")
	if ok && strings.TrimSpace(envRunAddr) != "" {
		flagRunAddr = envRunAddr
	}

	envReportInter, ok := os.LookupEnv("REPORT_INTERVAL")
	if ok && strings.TrimSpace(envReportInter) != "" {
		reportIntervalTime = envReportInter
	}

	envPortInter, ok := os.LookupEnv("POLL_INTERVAL")
	if ok && strings.TrimSpace(envPortInter) != "" {
		pollIntervalTime = envPortInter
	}
	return []string{flagRunAddr, reportIntervalTime, pollIntervalTime}
}
