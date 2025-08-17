package main

import (
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	contentType = "Content-Type: application/json"
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
	var m runtime.MemStats

	parseFlags()

	serverURL := flagRunAddr
	pollDuration, err := time.ParseDuration(flagPollIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid poll interval:", err)
	}

	reportDuration, err := time.ParseDuration(flagReportIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid report interval:", err)
	}
	stopPoll := make(chan struct{})
	stopReport := make(chan struct{})

	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	storage.AddCounter("PollCount", 0)
	go func() {

		pollTicker := time.NewTicker(pollDuration)
		defer pollTicker.Stop()

		for {
			select {
			case <-pollTicker.C:
				runtime.ReadMemStats(&m)
				sugar.Infoln("Updating metrics...")
				//fmt.Println("Updating metrics...")
				storage.AddGauge("Alloc", float64(m.Alloc))
				storage.AddGauge("BuckHashSys", float64(m.BuckHashSys))
				storage.AddGauge("Frees", float64(m.Frees))
				storage.AddGauge("GCCPUFraction", float64(m.GCCPUFraction))
				storage.AddGauge("GCSys", float64(m.GCSys))
				storage.AddGauge("HeapAlloc", float64(m.HeapAlloc))
				storage.AddGauge("HeapIdle", float64(m.HeapIdle))
				storage.AddGauge("HeapInuse", float64(m.HeapInuse))
				storage.AddGauge("HeapObjects", float64(m.HeapObjects))
				storage.AddGauge("HeapReleased", float64(m.HeapReleased))
				storage.AddGauge("HeapSys", float64(m.HeapSys))
				storage.AddGauge("LastGC", float64(m.LastGC))
				storage.AddGauge("Lookups", float64(m.Lookups))
				storage.AddGauge("MCacheInuse", float64(m.MCacheInuse))
				storage.AddGauge("MCacheSys", float64(m.MCacheSys))
				storage.AddGauge("MSpanInuse", float64(m.MSpanInuse))
				storage.AddGauge("MSpanSys", float64(m.MSpanSys))
				storage.AddGauge("Mallocs", float64(m.Mallocs))
				storage.AddGauge("NextGC", float64(m.NextGC))
				storage.AddGauge("NumForcedGC", float64(m.NumForcedGC))
				storage.AddGauge("NumGC", float64(m.NumGC))
				storage.AddGauge("OtherSys", float64(m.OtherSys))
				storage.AddGauge("PauseTotalNs", float64(m.PauseTotalNs))
				storage.AddGauge("StackInuse", float64(m.StackInuse))
				storage.AddGauge("StackSys", float64(m.StackSys))
				storage.AddGauge("Sys", float64(m.Sys))
				storage.AddGauge("TotalAlloc", float64(m.TotalAlloc))
				storage.AddGauge("RandomValue", rand.Float64())
				pollCounter++

			case <-stopPoll:
				return
			}
		}
	}()
	go func() {
		reportTicker := time.NewTicker(reportDuration)
		defer reportTicker.Stop()

		for {
			select {
			case <-reportTicker.C:
				sugar.Infoln("Sending metrics...")
				//fmt.Println("Sending metrics...")

				/*
					// Отправка gauge метрик
					for i := range storage.GaugeSlice() {
						err = repository.SendWithRetry(serverURL+"/update", "gauge", storage, i, flagKey)

						if err != nil {
							sugar.Infoln(storage.GaugeSlice()[i])

							sugar.Infoln("Error sending metric:", err)
						}

					}
					// Отправка counter метрик
					for i := range storage.CounterSlice() {
						storage.AddCounter("PollCount", pollCounter)
						err = repository.SendWithRetry(serverURL+"/update", "counter", storage, i, flagKey)
						if err != nil {
							sugar.Infow("Failed to send counter metric after retries",
								"metric", storage.CounterSlice()[i],
								"error", err)
						} else {
							pollCounter = 0
						}
					}
				*/

				err = repository.SendWithRetry(serverURL+"/updates", "gauge", storage, 28, flagKey)
				if err != nil {
					sugar.Error(err)
				}

				storage.AddCounter("PollCount", pollCounter) // Добавляем счетчик обновления метрик
				err = repository.SendWithRetry(serverURL+"/updates", "counter", storage, 1, flagKey)
				if err != nil {
					sugar.Error(err)
				} else {
					pollCounter = 0
				}

			case <-stopReport:
				return
			}
		}
	}()
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown

	// Останавливаем горутины
	close(stopPoll)
	close(stopReport)
	sugar.Infoln("Shutting down...")
	//println(storage.CounterSlice()[0].Value)

	//if reportIntervalTime == reportCounter {
	//	reportCounter = 0

}

//}
//}
