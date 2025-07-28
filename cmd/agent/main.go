package main

import (
	"github.com/tladugin/yaProject.git/internal/logger"
	models "github.com/tladugin/yaProject.git/internal/model"
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
		_ = sugar.Sync() // Безопасное закрытие логгера
	}()
	var m runtime.MemStats

	flags := parseFlags()
	flagRunAddr := flags[0]
	reportIntervalTime := flags[1]
	pollIntervalTime := flags[2]

	serverURL := flagRunAddr
	pollDuration, err := time.ParseDuration(pollIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid poll interval:", err)
	}

	reportDuration, err := time.ParseDuration(reportIntervalTime + "s")
	if err != nil {
		sugar.Fatal("Invalid report interval:", err)
	}
	stopPoll := make(chan struct{})
	stopReport := make(chan struct{})

	storage := repository.NewMemStorage()

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

				storage.AddCounter("PollCount", 1)
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
				for i := range storage.GaugeSlice() {
					err = models.SendMetric(serverURL+"/update", "gauge", storage, i)
					if err != nil {
						sugar.Infoln(storage.GaugeSlice()[i])

						sugar.Infoln("Error sending metric:", err)
					}

				}
				for i := range storage.CounterSlice() {
					err = models.SendMetric(serverURL+"/update", "counter", storage, i)
					if err != nil {
						sugar.Infoln(storage.CounterSlice()[i])
						sugar.Infoln("Error sending metric:", err)
					}
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
