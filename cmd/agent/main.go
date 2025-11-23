package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tladugin/yaProject.git/internal/agent"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"golang.org/x/sync/errgroup"
	_ "net/http/pprof"
)

func main() {
	// Вывод информации о сборке
	agent.PrintBuildInfo()

	// Инициализация структурированного логгера
	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = sugar.Sync()
	}()

	// Создаем контекст, который отменится при получении сигнала
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT)
	defer stop()

	config, err := agent.GetAgentConfig()
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
		err := repository.LoadPublicKey(config.CryptoKey)
		if err != nil {
			sugar.Fatal("Failed to load public key: ", err)
		}
		sugar.Info("Public key loaded successfully")
	}

	// Создание пула воркеров для ограничения скорости отправки запросов
	workerPool, err := agent.NewWorkerPool(config.RateLimit)
	if err != nil {
		sugar.Fatal("Failed to create worker pool: ", err)
	}
	defer workerPool.Shutdown()

	// Настройка URL сервера для отправки метрик
	serverURL := config.Address

	// Парсинг интервалов опроса и отправки метрик
	pollDuration, err := time.ParseDuration(config.PollInterval + "s")
	if err != nil {
		sugar.Fatal("Invalid poll interval:", err)
	}

	reportDuration, err := time.ParseDuration(config.ReportInterval + "s")
	if err != nil {
		sugar.Fatal("Invalid report interval:", err)
	}

	// Создание хранилища для метрик
	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	storage.AddCounter("PollCount", 0)

	// Создаем errgroup с нашим контекстом
	g, ctx := errgroup.WithContext(ctx)

	// Горутина для сбора runtime метрик Go
	g.Go(func() error {
		return agent.CollectRuntimeMetricsWithContext(ctx, storage, pollDuration, sugar, &pollCounter)
	})

	// Горутина для сбора системных метрик (память, CPU)
	g.Go(func() error {
		return agent.CollectSystemMetricsWithContext(ctx, storage, pollDuration, sugar)
	})

	// Горутина для отправки метрик на сервер
	g.Go(func() error {
		return agent.ReportMetricsWithContext(ctx, storage, serverURL, config.Key, reportDuration, workerPool, sugar, &pollCounter, config.CryptoKey)
	})

	// Ожидаем сигнал завершения
	sugar.Info("Agent started. Press Ctrl+C to stop.")

	// Ожидаем завершения всех горутин
	if err := g.Wait(); err != nil {
		if err == context.Canceled {
			sugar.Info("Service shutdown by signal")
		} else {
			sugar.Errorw("Service shutdown with error", "error", err)
		}
	}

	// Даем время для завершения отправки оставшихся метрик
	sugar.Info("Waiting for pending requests to complete...")
	time.Sleep(1 * time.Second)

	sugar.Info("Service shutdown completed successfully")
}
