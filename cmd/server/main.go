package main

import (
	"context"
	"github.com/tladugin/yaProject.git/internal/server/grpc"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"github.com/tladugin/yaProject.git/internal/server"
	_ "net/http/pprof"
)

func main() {
	// Вывод информации о сборке
	server.PrintBuildInfo()

	// Инициализация логгера
	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer sugar.Sync()

	// Создаем контекст для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT)
	defer stop()

	// Получаем конфигурацию
	config, err := GetServerConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Запуск pprof сервера (если включен)
	if config.UsePprof {
		go func() {
			sugar.Info("Starting pprof server on :6060")
			if err := http.ListenAndServe(":6060", nil); err != nil && err != http.ErrServerClosed {
				sugar.Errorw("Pprof server error", "error", err)
			}
		}()
	}

	// Инициализация криптографии
	if config.CryptoKey != "" {
		if err := server.LoadPrivateKey(config.CryptoKey); err != nil {
			sugar.Fatalw("Failed to load private key", "error", err)
		}
		sugar.Info("Private key loaded successfully")
	}

	// Создание хранилища
	storage := repository.NewMemStorage()

	// Восстановление данных из бэкапа
	if config.Restore {
		if err := repository.RestoreFromBackup(storage, config.StoreFile); err != nil {
			sugar.Errorw("Failed to restore from backup", "error", err)
		} else {
			sugar.Info("Data restored from backup successfully")
		}
	}

	// Инициализация продюсера для бэкапов
	producer, err := repository.NewProducer(config.StoreFile)
	if err != nil {
		sugar.Fatalw("Could not open backup file", "error", err)
	}
	defer producer.Close()

	// Инициализация проверки IP
	ipChecker, err := server.NewIPChecker(config.TrustedSubnet)
	if err != nil {
		sugar.Fatalw("Failed to initialize IP checker", "error", err)
	}

	if config.TrustedSubnet != "" {
		sugar.Infow("IP checking enabled", "trusted_subnet", config.TrustedSubnet)
	} else {
		sugar.Info("IP checking disabled (no trusted subnet specified)")
	}

	// Канал для graceful shutdown
	stopProgram := make(chan struct{})
	var wg sync.WaitGroup

	// Запуск периодического бэкапа если не синхронный режим
	if config.StoreInterval != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repository.RunPeriodicBackupWithContext(ctx, storage, producer, time.Duration(config.StoreInterval)*time.Second, config.StoreFile)
		}()
	}

	// Запуск горутины для финального бэкапа
	wg.Add(1)
	go func() {
		defer wg.Done()
		repository.RunFinalBackupWithContext(ctx, storage, producer, config.StoreFile)
	}()

	// Запуск gRPC сервера
	wg.Add(1)
	go func() {
		defer wg.Done()
		sugar.Infow("Starting gRPC server", "address", config.GRPCAddress)
		if err := grpc.RunGRPCServer(storage, config.GRPCAddress, config.TrustedSubnet); err != nil {
			sugar.Errorw("gRPC server error", "error", err)
		}
	}()

	// Запуск HTTP сервера
	wg.Add(1)
	go server.RunHTTPServer(
		storage,
		producer,
		ctx,
		&wg,
		config.StoreInterval,
		&config.Address,
		&config.DatabaseDSN,
		&config.Key,
		&config.AuditFile,
		&config.AuditURL,
		ipChecker,
	)

	sugar.Info("Server started. Press Ctrl+C to stop.")

	// Ожидаем сигнал завершения
	<-ctx.Done()
	sugar.Info("Received shutdown signal")

	// Инициируем graceful shutdown
	close(stopProgram)

	// Сохраняем финальный бэкап
	sugar.Info("Saving final backup...")
	if err := repository.SaveBackup(storage, producer, config.StoreFile); err != nil {
		sugar.Errorw("Failed to save final backup", "error", err)
	} else {
		sugar.Info("Final backup saved successfully")
	}

	// Ждем завершения всех горутин
	wg.Wait()
	sugar.Info("Application shutdown complete")
}
