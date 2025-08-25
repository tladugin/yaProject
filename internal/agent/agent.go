package agent

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func WaitForShutdownSignal(stopPoll, stopReport chan struct{}, fatalErrors chan error, sugar *zap.SugaredLogger) {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case <-shutdown:
		// Останавливаем горутины
		close(stopPoll)
		close(stopReport)
		sugar.Infoln("Shutting down...")

	case err := <-fatalErrors:
		sugar.Fatal("Fatal error occurred: ", err)
	}

}

func CollectRuntimeMetrics(storage *repository.MemStorage) {

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := map[string]float64{
		"Alloc":         float64(m.Alloc),
		"BuckHashSys":   float64(m.BuckHashSys),
		"Frees":         float64(m.Frees),
		"GCCPUFraction": float64(m.GCCPUFraction),
		"GCSys":         float64(m.GCSys),
		"HeapAlloc":     float64(m.HeapAlloc),
		"HeapIdle":      float64(m.HeapIdle),
		"HeapInuse":     float64(m.HeapInuse),
		"HeapObjects":   float64(m.HeapObjects),
		"HeapReleased":  float64(m.HeapReleased),
		"HeapSys":       float64(m.HeapSys),
		"LastGC":        float64(m.LastGC),
		"Lookups":       float64(m.Lookups),
		"MCacheInuse":   float64(m.MCacheInuse),
		"MCacheSys":     float64(m.MCacheSys),
		"MSpanInuse":    float64(m.MSpanInuse),
		"MSpanSys":      float64(m.MSpanSys),
		"Mallocs":       float64(m.Mallocs),
		"NextGC":        float64(m.NextGC),
		"NumForcedGC":   float64(m.NumForcedGC),
		"NumGC":         float64(m.NumGC),
		"OtherSys":      float64(m.OtherSys),
		"PauseTotalNs":  float64(m.PauseTotalNs),
		"StackInuse":    float64(m.StackInuse),
		"StackSys":      float64(m.StackSys),
		"Sys":           float64(m.Sys),
		"TotalAlloc":    float64(m.TotalAlloc),
		"RandomValue":   rand.Float64(),
	}

	for name, value := range metrics {
		storage.AddGauge(name, value)
	}
}

func CollectSystemMetrics(storage *repository.MemStorage) {
	if v, err := mem.VirtualMemory(); err == nil {
		storage.AddGauge("TotalMemory", float64(v.Total))
		storage.AddGauge("FreeMemory", float64(v.Free))
	}

	if percents, err := cpu.Percent(0, true); err == nil {
		for i, percent := range percents {
			storage.AddGauge(fmt.Sprintf("CPUutilization%d", i+1), percent)
		}
	}
}
