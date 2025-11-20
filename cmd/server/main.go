package main

import (
	"fmt"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"github.com/tladugin/yaProject.git/internal/server"
	"log"
	"net/http"
	_ "net/http/pprof" // подключаем пакет pprof для профилирования
	"sync"
	"time"
)

// Глобальная переменная для ошибок
var err error

// main - основная функция приложения, точка входа
func main() {

	// Вывод информации о сборке
	server.PrintBuildInfo()

	config, err := GetServerConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Парсинг флагов командной строки
	//flags := parseFlags()

	// Запуск pprof сервера для профилирования если включен
	if config.UsePprof {
		go func() {
			fmt.Println("Starting pprof server on :6060")
			// Запуск HTTP сервера для сбора профилей производительности
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Sugar.Error("Pprof server error: ", err)
			}
		}()
	}

	// Инициализация логгера для структурированного логирования
	logger.Sugar, err = logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		// Безопасное закрытие логгера при завершении программы
		_ = logger.Sugar.Sync()
	}()

	// Проверка успешной инициализации логгера
	if logger.Sugar == nil {
		log.Fatal("Logger initialization failed")
	}

	// Инициализация криптографии - загрузка приватного ключа для расшифровки
	if config.CryptoKey != "" {
		err := server.LoadPrivateKey(config.CryptoKey)
		if err != nil {
			logger.Sugar.Fatal("Failed to load private key: ", err)
		}
		logger.Sugar.Info("Private key loaded successfully")
	}

	// Создание in-memory хранилища для метрик
	storage := repository.NewMemStorage()

	// Восстановление данных из файла бэкапа если включена опция restore
	if config.Restore {
		repository.RestoreFromBackup(storage, config.StoreFile)
	}

	// Инициализация продюсера для записи бэкапов
	producer, err := repository.NewProducer(config.StoreFile)
	if err != nil {
		logger.Sugar.Fatal("Could not open backup file: ", err)
	}

	// Парсинг интервала сохранения из строки в Duration
	storeInterval, err := time.ParseDuration(config.StoreInterval + "s")
	if err != nil {
		logger.Sugar.Fatal("Invalid store interval: ", err)
	}

	// Канал для graceful shutdown - уведомляет все горутины о необходимости завершения
	stopProgram := make(chan struct{})
	var wg sync.WaitGroup // WaitGroup для ожидания завершения всех горутин

	// Запуск фоновых задач в отдельных горутинах

	// Запуск периодического бэкапа если интервал не равен 0 (не синхронный режим)
	if config.StoreInterval != "0" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Запуск периодического создания бэкапов с заданным интервалом
			repository.RunPeriodicBackup(storage, producer, storeInterval, stopProgram, config.StoreFile)
		}()
	}

	// Запуск горутины для финального бэкапа при завершении программы
	wg.Add(1)
	go repository.RunFinalBackup(storage, producer, stopProgram, &wg, config.StoreFile)

	// Запуск основного HTTP сервера для обработки запросов метрик
	wg.Add(1)
	go server.RunHTTPServer(storage, producer, stopProgram, &wg, config.StoreInterval, &config.Address, &config.DatabaseDSN, &config.Key, &config.AuditFile, &config.AuditURL)

	// Ожидание сигнала завершения (SIGTERM, SIGINT)
	repository.WaitForShutdown(stopProgram)

	// Ожидание завершения всех горутин
	wg.Wait()
	logger.Sugar.Info("Application shutdown complete")
}
