package main

import (
	"flag"
	"os"
)

const defaultDSN = ""

var (
	flagRunAddr         string
	flagStoreInterval   string
	flagFileStoragePath string
	flagRestore         bool
	flagDatabaseDSN     string
	flagKey             string
)

// "host=localhost user=postgres password=543218 dbname=metrics sslmode=disable"

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")

	flag.StringVar(&flagStoreInterval, "i", "300", "saving server data interval")
	flag.StringVar(&flagFileStoragePath, "f", "server_backup", "path for server backup file")
	flag.BoolVar(&flagRestore, "r", false, "restore server data")
	flag.StringVar(&flagDatabaseDSN, "d", defaultDSN, "database DSN")
	flag.StringVar(&flagKey, "k", "", "key")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		flagStoreInterval = envStoreInterval
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		flagFileStoragePath = envFileStoragePath
	}
	if envRestoreKey := os.Getenv("RESTORE"); envRestoreKey != "" {
		flagRestore = true
	}
	if envConnectString := os.Getenv("DATABASE_DSN"); envConnectString != "" {
		flagDatabaseDSN = envConnectString
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		flagKey = envKey
	}

}
