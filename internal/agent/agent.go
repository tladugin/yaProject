package agent

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
)

type Agent struct {
	flags       *Flags
	logger      *zap.SugaredLogger
	storage     *repository.MemStorage
	workerPool  *WorkerPool
	pollCounter int64
	stopPoll    chan struct{}
	stopReport  chan struct{}
}

func NewAgent(flags *Flags, logger *zap.SugaredLogger) *Agent {
	return &Agent{
		flags:      flags,
		logger:     logger,
		storage:    repository.NewMemStorage(),
		workerPool: NewWorkerPool(flags.FlagRateLimit),
		stopPoll:   make(chan struct{}),
		stopReport: make(chan struct{}),
	}
}

func (a *Agent) Start() error {
	a.logger.Info("Starting agent...")

	// Инициализация счетчика
	a.storage.AddCounter("PollCount", 0)
	a.pollCounter = 0

	// Запуск воркеров
	go a.startPolling()
	go a.startSystemMetricsPolling()
	go a.startReporting()

	return nil
}

func (a *Agent) Stop() {
	a.logger.Info("Stopping agent...")
	close(a.stopPoll)
	close(a.stopReport)
	a.workerPool.Shutdown()
}

func (a *Agent) startPolling() {
	pollDuration, err := time.ParseDuration(a.flags.FlagPollIntervalTime + "s")
	if err != nil {
		a.logger.Fatal("Invalid poll interval:", err)
	}

	ticker := time.NewTicker(pollDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.collectRuntimeMetrics()
			a.pollCounter++
		case <-a.stopPoll:
			return
		}
	}
}

func (a *Agent) startSystemMetricsPolling() {
	pollDuration, err := time.ParseDuration(a.flags.FlagPollIntervalTime + "s")
	if err != nil {
		a.logger.Fatal("Invalid poll interval:", err)
	}

	ticker := time.NewTicker(pollDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.collectSystemMetrics()
		case <-a.stopPoll:
			return
		}
	}
}

func (a *Agent) startReporting() {
	reportDuration, err := time.ParseDuration(a.flags.FlagReportIntervalTime + "s")
	if err != nil {
		a.logger.Fatal("Invalid report interval:", err)
	}

	ticker := time.NewTicker(reportDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.sendMetrics()
		case <-a.stopReport:
			return
		}
	}
}

func (a *Agent) collectRuntimeMetrics() {
	a.logger.Info("Updating runtime metrics...")

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
		a.storage.AddGauge(name, value)
	}
}

func (a *Agent) collectSystemMetrics() {
	if v, err := mem.VirtualMemory(); err == nil {
		a.storage.AddGauge("TotalMemory", float64(v.Total))
		a.storage.AddGauge("FreeMemory", float64(v.Free))
	}

	if percents, err := cpu.Percent(0, true); err == nil {
		for i, percent := range percents {
			a.storage.AddGauge(fmt.Sprintf("CPUutilization%d", i+1), percent)
		}
	}
}

func (a *Agent) sendMetrics() {
	a.logger.Info("Sending metrics...")

	// Отправка gauge метрик
	a.workerPool.Submit(func() {
		err := repository.SendWithRetry(
			a.flags.FlagRunAddr+"/updates",
			"gauge",
			a.storage,
			len(a.storage.GaugeSlice()),
			a.flags.FlagKey,
		)
		if err != nil {
			a.logger.Error("Failed to send gauge metrics:", err)
		}
	})

	// Отправка counter метрик
	a.workerPool.Submit(func() {
		a.storage.AddCounter("PollCount", a.pollCounter)
		err := repository.SendWithRetry(
			a.flags.FlagRunAddr+"/updates",
			"counter",
			a.storage,
			len(a.storage.CounterSlice()),
			a.flags.FlagKey,
		)
		if err != nil {
			a.logger.Error("Failed to send counter metrics:", err)
		} else {
			a.pollCounter = 0
		}
	})
}
