package main

import (
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/server"

	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
	"log"

	"sync"

	"time"
)

// logger
var sugar *zap.SugaredLogger
var err error

func main() {
	parseFlags()

	// Инициализация логгера
	sugar, err = logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = sugar.Sync() // Безопасное закрытие логгера
	}()

	// Убедимся, что sugar не nil
	if sugar == nil {
		log.Fatal("Logger initialization failed")
	}

	storage := repository.NewMemStorage()

	// Восстановление данных из бэкапа
	if flagRestore {
		handler.RestoreFromBackup(storage, flagFileStoragePath, sugar)
	}

	// Инициализация продюсера
	producer, err := handler.NewProducer(flagFileStoragePath)
	if err != nil {
		sugar.Fatal("Could not open backup file: ", err)
	}
	//defer safeCloseProducer(producer)

	// Парсинг интервала сохранения
	storeInterval, err := time.ParseDuration(flagStoreInterval + "s")
	if err != nil {
		sugar.Fatal("Invalid store interval: ", err)
	}

	// Канал для graceful shutdown
	stopProgram := make(chan struct{})
	var wg sync.WaitGroup

	// Запуск фоновых задач

	if flagStoreInterval != "0" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			handler.RunPeriodicBackup(storage, producer, storeInterval, stopProgram, flagFileStoragePath, sugar)
		}()
	}

	wg.Add(1)
	go handler.RunFinalBackup(storage, producer, stopProgram, &wg, flagFileStoragePath, sugar)

	wg.Add(1)
	go server.RunHTTPServer(storage, producer, stopProgram, &wg, flagStoreInterval, sugar, &flagRunAddr, &flagDatabaseDSN)

	// Ожидание сигнала завершения
	handler.WaitForShutdown(stopProgram, *sugar)
	wg.Wait()
	sugar.Info("Application shutdown complete")
}
