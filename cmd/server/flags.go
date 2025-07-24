package main


import (
	"flag"
	"os"
)

var (
	flagRunAddr         string
	flagStoreInterval   string
	flagFileStoragePath string
	flagRestore         bool
)


// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")

	flag.StringVar(&flagStoreInterval, "i", "300", "saving server data interval")
	flag.StringVar(&flagFileStoragePath, "f", "server_backup", "path for server backup file")
	flag.BoolVar(&flagRestore, "r", false, "restore server data")
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
	if flagRestoreKey := os.Getenv("RESTORE"); flagRestoreKey != "" {
		flagRestore = true
	}


}
