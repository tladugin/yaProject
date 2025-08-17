package main

import (
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"github.com/tladugin/yaProject.git/internal/server"

	"go.uber.org/zap"

	"log"
	"sync"
	"time"
)

// logger
var Sugar *zap.SugaredLogger
var err error

func main() {
	parseFlags()

	// Инициализация логгера
	Sugar, err = logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = Sugar.Sync() // Безопасное закрытие логгера
	}()

	// Убедимся, что sugar не nil
	if Sugar == nil {
		log.Fatal("Logger initialization failed")
	}

	storage := repository.NewMemStorage()

	// Восстановление данных из бэкапа
	if flagRestore {
		repository.RestoreFromBackup(storage, flagFileStoragePath)
	}

	// Инициализация продюсера
	producer, err := repository.NewProducer(flagFileStoragePath)
	if err != nil {
		Sugar.Fatal("Could not open backup file: ", err)
	}
	//defer safeCloseProducer(producer)

	// Парсинг интервала сохранения
	storeInterval, err := time.ParseDuration(flagStoreInterval + "s")
	if err != nil {
		Sugar.Fatal("Invalid store interval: ", err)
	}

	// Канал для graceful shutdown
	stopProgram := make(chan struct{})
	var wg sync.WaitGroup

	// Запуск фоновых задач

	if flagStoreInterval != "0" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			repository.RunPeriodicBackup(storage, producer, storeInterval, stopProgram, flagFileStoragePath)
		}()
	}

	wg.Add(1)
	go repository.RunFinalBackup(storage, producer, stopProgram, &wg, flagFileStoragePath)

	wg.Add(1)
	go server.RunHTTPServer(storage, producer, stopProgram, &wg, flagStoreInterval, &flagRunAddr, &flagDatabaseDSN)

	// Ожидание сигнала завершения
	repository.WaitForShutdown(stopProgram)
	wg.Wait()
	Sugar.Info("Application shutdown complete")
}
