package agent

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"

	"github.com/tladugin/yaProject.git/internal/repository"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// CollectRuntimeMetricsWithContext собирает runtime метрики с учетом контекста
func CollectRuntimeMetricsWithContext(ctx context.Context, storage *repository.MemStorage, pollDuration time.Duration, sugar *zap.SugaredLogger, pollCounter *int64) error {
	sugar.Info("Starting runtime metrics collection")
	defer sugar.Info("Runtime metrics collection stopped")

	ticker := time.NewTicker(pollDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sugar.Debug("Updating runtime metrics...")
			collectRuntimeMetrics(storage)
			(*pollCounter)++ // Увеличение счетчика опросов
		}
	}
}

// CollectSystemMetricsWithContext собирает системные метрики с учетом контекста
func CollectSystemMetricsWithContext(ctx context.Context, storage *repository.MemStorage, pollDuration time.Duration, sugar *zap.SugaredLogger) error {
	sugar.Info("Starting system metrics collection")
	defer sugar.Info("System metrics collection stopped")

	ticker := time.NewTicker(pollDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sugar.Debug("Updating system metrics...")
			collectSystemMetrics(storage)
		}
	}
}

func ReportMetricsWithContext(ctx context.Context, storage *repository.MemStorage, serverURL, key string, reportDuration time.Duration, workerPool *WorkerPool, sugar *zap.SugaredLogger, pollCounter *int64, FlagCryptoKey string, localIP string) error {
	sugar.Info("Starting metrics reporting")
	defer sugar.Info("Metrics reporting stopped")

	ticker := time.NewTicker(reportDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sugar.Debug("Sending metrics...")

			// Простая отправка через worker pool
			workerPool.Submit(func() {
				// Проверяем контекст внутри задачи
				select {
				case <-ctx.Done():
					return
				default:
					err := SendWithRetry(serverURL+"/updates", storage, key, *pollCounter, FlagCryptoKey, localIP)
					if err != nil && err != context.Canceled {
						sugar.Errorf("Error sending metrics: %v", err)
					} else if err == nil {
						*pollCounter = 0
						sugar.Debug("Metrics sent successfully")
					}
				}
			})
		}
	}
}

// collectRuntimeMetrics собирает метрики runtime Go и сохраняет их в хранилище
func collectRuntimeMetrics(storage *repository.MemStorage) {
	// Получаем статистику runtime Go
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Словарь метрик runtime с их значениями
	metrics := map[string]float64{
		"Alloc":         float64(m.Alloc),         // Выделено байт кучи
		"BuckHashSys":   float64(m.BuckHashSys),   // Байты в хэш-таблицах
		"Frees":         float64(m.Frees),         // Количество освобождений памяти
		"GCCPUFraction": float64(m.GCCPUFraction), // Доля CPU времени на GC
		"GCSys":         float64(m.GCSys),         // Байты в системах GC
		"HeapAlloc":     float64(m.HeapAlloc),     // Байты выделено в куче
		"HeapIdle":      float64(m.HeapIdle),      // Байты в неиспользуемой куче
		"HeapInuse":     float64(m.HeapInuse),     // Байты в используемой куче
		"HeapObjects":   float64(m.HeapObjects),   // Количество объектов в куче
		"HeapReleased":  float64(m.HeapReleased),  // Байты возвращенные ОС
		"HeapSys":       float64(m.HeapSys),       // Байты полученные от ОС для кучи
		"LastGC":        float64(m.LastGC),        // Время последней GC (наносекунды)
		"Lookups":       float64(m.Lookups),       // Количество поисков указателей
		"MCacheInuse":   float64(m.MCacheInuse),   // Байты в локальных кэшах
		"MCacheSys":     float64(m.MCacheSys),     // Байты полученные для кэшей
		"MSpanInuse":    float64(m.MSpanInuse),    // Байты в span структурах
		"MSpanSys":      float64(m.MSpanSys),      // Байты полученные для span
		"Mallocs":       float64(m.Mallocs),       // Количество выделений памяти
		"NextGC":        float64(m.NextGC),        // Цель следующей GC
		"NumForcedGC":   float64(m.NumForcedGC),   // Количество принудительных GC
		"NumGC":         float64(m.NumGC),         // Количество выполненных GC
		"OtherSys":      float64(m.OtherSys),      // Байты в других системах
		"PauseTotalNs":  float64(m.PauseTotalNs),  // Суммарное время пауз GC (нс)
		"StackInuse":    float64(m.StackInuse),    // Байты в стеках
		"StackSys":      float64(m.StackSys),      // Байты полученные для стеков
		"Sys":           float64(m.Sys),           // Всего байт получено от ОС
		"TotalAlloc":    float64(m.TotalAlloc),    // Всего байт выделено за время работы
		"RandomValue":   rand.Float64(),           // Случайное значение для тестирования
	}

	// Сохраняем все метрики в хранилище
	for name, value := range metrics {
		storage.AddGauge(name, value)
	}
}

// collectSystemMetrics собирает системные метрики (память, CPU) и сохраняет их в хранилище
func collectSystemMetrics(storage *repository.MemStorage) {
	// Получаем информацию о виртуальной памяти
	if v, err := mem.VirtualMemory(); err == nil {
		storage.AddGauge("TotalMemory", float64(v.Total)) // Общий объем памяти
		storage.AddGauge("FreeMemory", float64(v.Free))   // Свободный объем памяти
	}
	// Если произошла ошибка - метрики просто не будут добавлены

	// Получаем загрузку CPU по ядрам
	if percents, err := cpu.Percent(0, true); err == nil {
		// Для каждого CPU ядра сохраняем его утилизацию
		for i, percent := range percents {
			storage.AddGauge(fmt.Sprintf("CPUutilization%d", i+1), percent)
		}
	}
	// Если произошла ошибка - метрики CPU не будут добавлены
}

// printBuildInfo выводит информацию о сборке
func PrintBuildInfo() {
	// Устанавливаем "N/A" если значения не заданы
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	// Вывод в формате согласно требованиям
	log.Printf("Build version: %s", buildVersion)
	log.Printf("Build date: %s", buildDate)
	log.Printf("Build commit: %s", buildCommit)
}
