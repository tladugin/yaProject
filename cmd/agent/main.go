package main

import (
	"github.com/tladugin/yaProject.git/internal/agent"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"log"
	"time"
)

func main() {
	// Инициализация структурированного логгера
	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		// Безопасное закрытие логгера при завершении программы
		err = sugar.Sync()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Парсинг флагов командной строки
	flags := agent.ParseFlags()

	// Создание пула воркеров для ограничения скорости отправки запросов
	workerPool := agent.NewWorkerPool(flags.FlagRateLimit)
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

	// Каналы для управления жизненным циклом горутин
	stopPoll := make(chan struct{})   // Сигнал остановки сбора метрик
	stopReport := make(chan struct{}) // Сигнал остановки отправки метрик

	// Канал для обработки фатальных ошибок (буферизованный для избежания блокировок)
	fatalErrors := make(chan error, 10)

	// Создание хранилища для метрик
	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	// Инициализация счетчика опросов
	storage.AddCounter("PollCount", 0)

	// Создание тикеров для периодического выполнения задач
	pollTicker1 := time.NewTicker(pollDuration) // Для сбора runtime метрик
	defer pollTicker1.Stop()
	pollTicker2 := time.NewTicker(pollDuration) // Для сбора системных метрик
	defer pollTicker2.Stop()
	reportTicker := time.NewTicker(reportDuration) // Для отправки метрик
	defer reportTicker.Stop()

	// Горутина для сбора runtime метрик Go
	go func() {
		for {
			select {
			case <-pollTicker1.C:
				sugar.Infoln("Updating metrics...")
				agent.CollectRuntimeMetrics(storage)
				pollCounter++ // Увеличение счетчика опросов

			case <-stopPoll:
				return // Завершение горутины при получении сигнала остановки
			}
		}
	}()

	// Горутина для сбора системных метрик (память, CPU)
	go func() {
		for {
			select {
			case <-pollTicker2.C:
				agent.CollectSystemMetrics(storage)
			case <-stopPoll:
				return // Завершение горутины при получении сигнала остановки
			}
		}
	}()

	// Горутина для отправки метрик на сервер
	go func() {
		for {
			select {
			case <-reportTicker.C:
				sugar.Infoln("Sending metrics...")
				reportTicker.Stop() // Временная остановка тикера до завершения отправки

				// Отправка метрик через пул воркеров с ограничением скорости
				workerPool.Submit(func() {
					err = repository.SendWithRetry(serverURL+"/updates", storage, flags.FlagKey, pollCounter)
					if err != nil {
						// Логирование ошибки и повторная установка тикера
						sugar.Errorf("Error sending metrics: %v", err)
						reportTicker.Reset(reportDuration)
					} else {
						// Сброс счетчика опросов после успешной отправки
						pollCounter = 0
						reportTicker.Reset(reportDuration)
					}
				})

			case <-stopReport:
				return // Завершение горутины при получении сигнала остановки
			}
		}
	}()

	// Ожидание сигналов завершения работы приложения
	agent.WaitForShutdownSignal(stopPoll, stopReport, fatalErrors, sugar)
}
