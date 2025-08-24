package main

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/tladugin/yaProject.git/internal/agent"
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

	storage := repository.NewMemStorage()
	var pollCounter int64 = 0
	storage.AddCounter("PollCount", 0)

	pollTicker := time.NewTicker(pollDuration)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(reportDuration)
	defer reportTicker.Stop()
	go func() {

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
		for {
			select {
			case <-pollTicker.C:
				if v, err := mem.VirtualMemory(); err == nil {
					storage.AddGauge("TotalMemory", float64(v.Total))
					storage.AddGauge("FreeMemory", float64(v.Free))
				}
				if percents, err := cpu.Percent(0, true); err == nil {
					for i, percent := range percents {
						storage.AddGauge(fmt.Sprintf("CPUutilization%d", i+1), percent)
					}
				}
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

				workerPool.Submit(func() {
					err = repository.SendWithRetry(serverURL+"/updates", "gauge", storage, len(storage.GaugeSlice()), flags.FlagKey)
					if err != nil {
						sugar.Error(err)
					}
				})
				workerPool.Submit(func() {
					storage.AddCounter("PollCount", pollCounter) // Добавляем счетчик обновления метрик
					err = repository.SendWithRetry(serverURL+"/updates", "counter", storage, len(storage.CounterSlice()), flags.FlagKey)
					if err != nil {
						sugar.Error(err)
					} else {
						pollCounter = 0
					}
				})

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

}
