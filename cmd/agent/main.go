package main

import (
	"github.com/tladugin/yaProject.git/internal/agent"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"log"

	"time"
)

func main() {
	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = sugar.Sync() // Безопасное закрытие логгера
		if err != nil {
			log.Fatal(err)
		}
	}()

	flags := agent.ParseFlags()
	workerPool := agent.NewWorkerPool(flags.FlagRateLimit)
	defer workerPool.Shutdown()

	serverURL := flags.FlagRunAddr
	pollDuration, err := time.ParseDuration(flags.FlagPollIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid poll interval:", err)
	}

	reportDuration, err := time.ParseDuration(flags.FlagReportIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid report interval:", err)
	}
	stopPoll := make(chan struct{})
	stopReport := make(chan struct{})

	// Канал для обработки фатальных ошибок
	fatalErrors := make(chan error, 10)

	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	storage.AddCounter("PollCount", 0)

	pollTicker1 := time.NewTicker(pollDuration)
	defer pollTicker1.Stop()
	pollTicker2 := time.NewTicker(pollDuration)
	defer pollTicker2.Stop()
	reportTicker := time.NewTicker(reportDuration)
	defer reportTicker.Stop()

	go func() {
		for {
			select {
			case <-pollTicker1.C:
				sugar.Infoln("Updating metrics...")
				agent.CollectRuntimeMetrics(storage)
				pollCounter++

			case <-stopPoll:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-pollTicker2.C:
				agent.CollectSystemMetrics(storage)
			case <-stopPoll:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-reportTicker.C:
				sugar.Infoln("Sending metrics...")
				reportTicker.Stop()
				workerPool.Submit(func() {

					err = repository.SendWithRetry(serverURL+"/updates", storage, flags.FlagKey, pollCounter)
					if err != nil {

						sugar.Errorf("Error sending metrics: %v", err)
						reportTicker.Reset(reportDuration)
					} else {
						pollCounter = 0

						reportTicker.Reset(reportDuration)
					}
				})

			case <-stopReport:
				return
			}
		}
	}()

	agent.WaitForShutdownSignal(stopPoll, stopReport, fatalErrors, sugar)
}
