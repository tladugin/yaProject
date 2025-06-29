package main

import "flag"

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

}
