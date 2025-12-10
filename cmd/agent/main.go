package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tladugin/yaProject.git/internal/agent"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"golang.org/x/sync/errgroup"
)

func main() {
	agent.PrintBuildInfo()

	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer sugar.Sync()

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	config, err := agent.GetAgentConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Преобразуем интервалы
	pollDuration, _ := time.ParseDuration(config.PollInterval + "s")
	reportDuration, _ := time.ParseDuration(config.ReportInterval + "s")

	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	storage.AddCounter("PollCount", 0)

	// Получаем локальный IP-адрес
	localIP := agent.GetLocalIPConfig(config)
	if config.UseGRPC {
		if localIP == "" {
			sugar.Fatal("Local IP is required for gRPC mode")
		}
		sugar.Infow("Using gRPC mode",
			"server", config.GRPCAddress,
			"ip", localIP,
		)
	} else {
		sugar.Infow("Using HTTP mode",
			"server", config.Address,
			"ip", localIP,
		)
		// Создаем worker pool только для HTTP режима
		workerPool, err := agent.NewWorkerPool(config.RateLimit)
		if err != nil {
			sugar.Fatal("Failed to create worker pool: ", err)
		}
		defer workerPool.Shutdown()
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return agent.CollectRuntimeMetricsWithContext(ctx, storage, pollDuration, sugar, &pollCounter)
	})

	g.Go(func() error {
		return agent.CollectSystemMetricsWithContext(ctx, storage, pollDuration, sugar)
	})

	if config.UseGRPC {
		g.Go(func() error {
			return agent.ReportMetricsWithContext(ctx, storage, config, reportDuration, nil, sugar, &pollCounter, localIP)
		})
	} else {
		workerPool, _ := agent.NewWorkerPool(config.RateLimit)
		defer workerPool.Shutdown()

		g.Go(func() error {
			return agent.ReportMetricsWithContext(ctx, storage, config, reportDuration, workerPool, sugar, &pollCounter, localIP)
		})
	}

	sugar.Info("Agent started. Press Ctrl+C to stop.")

	if err := g.Wait(); err != nil && err != context.Canceled {
		sugar.Errorw("Service shutdown with error", "error", err)
	} else {
		sugar.Info("Service shutdown by signal")
	}

	workerPool.Shutdown()
	sugar.Info("Service shutdown completed successfully")
}
