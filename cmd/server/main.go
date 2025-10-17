package main

import (
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"github.com/tladugin/yaProject.git/internal/server"

	"log"
	"sync"
	"time"
)

// logger

var err error

func main() {
	flags := parseFlags()

	// Инициализация логгера
	logger.Sugar, err = logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = logger.Sugar.Sync() // Безопасное закрытие логгера
	}()

	// Убедимся, что sugar не nil
	if logger.Sugar == nil {
		log.Fatal("Logger initialization failed")
	}

	storage := repository.NewMemStorage()

	// Восстановление данных из бэкапа
	if flags.flagRestore {
		repository.RestoreFromBackup(storage, flags.flagFileStoragePath)
	}

	// Инициализация продюсера
	producer, err := repository.NewProducer(flags.flagFileStoragePath)
	if err != nil {
		logger.Sugar.Fatal("Could not open backup file: ", err)
	}

	// Парсинг интервала сохранения
	storeInterval, err := time.ParseDuration(flags.flagStoreInterval + "s")
	if err != nil {
		logger.Sugar.Fatal("Invalid store interval: ", err)
	}

	// Канал для graceful shutdown
	stopProgram := make(chan struct{})
	var wg sync.WaitGroup

	// Запуск фоновых задач

	if flags.flagStoreInterval != "0" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			repository.RunPeriodicBackup(storage, producer, storeInterval, stopProgram, flags.flagFileStoragePath)
		}()
	}

	wg.Add(1)
	go repository.RunFinalBackup(storage, producer, stopProgram, &wg, flags.flagFileStoragePath)

	wg.Add(1)
	go server.RunHTTPServer(storage, producer, stopProgram, &wg, flags.flagStoreInterval, &flags.flagRunAddr, &flags.flagDatabaseDSN, &flags.flagKey, &flags.flagAuditFile, &flags.flagAuditUrl)

	// Ожидание сигнала завершения
	repository.WaitForShutdown(stopProgram)
	wg.Wait()
	logger.Sugar.Info("Application shutdown complete")
}
