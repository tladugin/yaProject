package main

import (
	"flag"
	"os"
)

/*type Config struct {
	AD []string `env:"ADDRESS"`
	PI string   `env:"POLL_INTERVAL"`
	RI string   `env:"REPORT_INTERVAL"`
}*/

var flagRunAddr string
var reportIntervalTime string
var pollIntervalTime string

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&reportIntervalTime, "r", "10", "time interval to report")
	flag.StringVar(&pollIntervalTime, "p", "2", "poll interval")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
	if envReportInter := os.Getenv("REPORT_INTERVAL"); envReportInter != "" {
		reportIntervalTime = envReportInter
	}
	if envPortInter := os.Getenv("POLL_INTERVAL"); envPortInter != "" {
		pollIntervalTime = envPortInter
	}
}
