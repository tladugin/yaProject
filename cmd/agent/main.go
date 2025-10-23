package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/tladugin/yaProject.git/internal/agent"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
)

func main() {
	// Инициализация структурированного логгера
	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = sugar.Sync()
	}()

	// Создаем контекст, который отменится при получении сигнала
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Парсинг флагов командной строки
	flags := agent.ParseFlags()

	// Создание пула воркеров для ограничения скорости отправки запросов
	workerPool, err := agent.NewWorkerPool(flags.FlagRateLimit)
	if err != nil {
		sugar.Fatal("Failed to create worker pool: ", err)
	}
	defer workerPool.Shutdown() // Гарантированное завершение пула воркеров

	// Настройка URL сервера для отправки метрик
	serverURL := flags.FlagRunAddr

	// Парсинг интервалов опроса и отправки метрик
	pollDuration, err := time.ParseDuration(flags.FlagPollIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid poll interval:", err)
	}

	reportDuration, err := time.ParseDuration(flags.FlagReportIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid report interval:", err)
	}

	// Создание хранилища для метрик
	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	// Инициализация счетчика опросов
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
		return agent.ReportMetricsWithContext(ctx, storage, serverURL, flags.FlagKey, reportDuration, workerPool, sugar, &pollCounter)
	})

	// Ожидаем завершения всех горутин
	if err := g.Wait(); err != nil {
		if err == context.Canceled {
			sugar.Info("Service shutdown by signal")
		} else {
			sugar.Fatalw("Service shutdown with error", "error", err)
		}
	}

	sugar.Info("Service shutdown completed successfully")
}
