package models

import (
	"encoding/json"
	"fmt"
	"github.com/tladugin/yaProject.git/internal/repository"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func SendMetric(URL string, metricType string, storage *repository.MemStorage, i int) error {
	// 1. Подготовка метрики
	var metric Metrics

	switch metricType {
	case "gauge":
		metric = Metrics{
			MType: "gauge",
			ID:    storage.GaugeSlice()[i].Name,
			Value: &storage.GaugeSlice()[i].Value,
		}
	case "counter":
		metric = Metrics{
			MType: "counter",
			ID:    storage.CounterSlice()[i].Name,
			Delta: &storage.CounterSlice()[i].Value,
		}
	default:
		return fmt.Errorf("unknown metric type: %s", metricType)
	}

	// 2. Сериализация в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	// 3. Сжатие данных
	buf, err := repository.CompressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress data error: %w", err)
	}

	// 4. Нормализация URL
	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	// 5. Создание и настройка запроса
	req, err := http.NewRequest("POST", URL, buf)
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// 6. Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	// 7. Проверка ответа
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read error response: %w", err)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func SendMetricsBatch(URL string, metricType string, storage *repository.MemStorage, batchSize int) error {
	// 1. Подготовка URL
	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	// 2. Подготовка метрик в зависимости от типа
	var metrics []Metrics
	switch metricType {
	case "gauge":
		if len(storage.GaugeSlice()) == 0 {
			return nil
		}
		for i := 0; i < min(len(storage.GaugeSlice()), batchSize); i++ {
			metrics = append(metrics, Metrics{
				MType: "gauge",
				ID:    storage.GaugeSlice()[i].Name,
				Value: &storage.GaugeSlice()[i].Value,
			})
		}
	case "counter":
		if len(storage.CounterSlice()) == 0 {
			return nil
		}
		for i := 0; i < min(len(storage.CounterSlice()), batchSize); i++ {
			metrics = append(metrics, Metrics{
				MType: "counter",
				ID:    storage.CounterSlice()[i].Name,
				Delta: &storage.CounterSlice()[i].Value,
			})
		}
	default:
		return fmt.Errorf("unknown metric type: %s", metricType)
	}

	// 3. Сериализация в JSON
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	// 4. Сжатие данных
	buf, err := repository.CompressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress data error: %w", err)
	}

	// 5. Создание и настройка запроса
	req, err := http.NewRequest("POST", URL+"/updates/", buf)
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// 6. Отправка запроса
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	// 7. Проверка ответа
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read error response: %w", err)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
