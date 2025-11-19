package main

import (
	"context"
	"fmt"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Парсинг флагов командной строки
	flags := agent.ParseFlags()

	if flags.FlagUsePprof {
		go func() {
			fmt.Println("Starting pprof server on :6060")
			// Запуск HTTP сервера для сбора профилей производительности
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Sugar.Error("Pprof server error: ", err)
			}
		}()
	}

	// Инициализация криптографии - загрузка публичного ключа для шифрования
	if flags.FlagCryptoKey != "" {
		err := repository.LoadPublicKey(flags.FlagCryptoKey)
		if err != nil {
			sugar.Fatal("Failed to load public key: ", err)
		}
		sugar.Info("Public key loaded successfully")
	}

	if flags.FlagUsePprof {
		go func() {
			fmt.Println("Starting pprof server on :6060")
			// Запуск HTTP сервера для сбора профилей производительности
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Sugar.Error("Pprof server error: ", err)
			}
		}()
	}
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
		return agent.ReportMetricsWithContext(ctx, storage, serverURL, flags.FlagKey, reportDuration, workerPool, sugar, &pollCounter, flags.FlagCryptoKey)
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
