package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tladugin/yaProject.git/internal/models"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func SendMetric(URL string, metricType string, storage *MemStorage, i int, key string) error {
	// 1. Подготовка метрики
	var metric models.Metrics

	switch metricType {
	case "gauge":
		metric = models.Metrics{
			MType: "gauge",
			ID:    storage.GaugeSlice()[i].Name,
			Value: &storage.GaugeSlice()[i].Value,
		}
	case "counter":
		metric = models.Metrics{
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
	buf, err := CompressData(jsonData)
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

	// 5.1 Проверяем наличие ключа, если он есть, отправляем в заголовке хеш
	if key != "" {
		bytesBuf := buf.Bytes()
		bytesKey := []byte(key)
		hash := sha256.Sum256(append(bytesKey, bytesBuf...))
		hashHeader := hex.EncodeToString(hash[:])
		req.Header.Set("HashSHA256", hashHeader)
	}

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

func SendMetricsBatch(URL string, metricType string, storage *MemStorage, batchSize int, key string) error {
	// 1. Подготовка URL
	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	// 2. Подготовка метрик в зависимости от типа
	var metrics []models.Metrics
	switch metricType {
	case "gauge":
		if len(storage.GaugeSlice()) == 0 {
			return nil
		}
		for i := 0; i < min(len(storage.GaugeSlice()), batchSize); i++ {
			metrics = append(metrics, models.Metrics{
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
			metrics = append(metrics, models.Metrics{
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
	buf, err := CompressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress data error: %w", err)
	}

	// 5. Создание и настройка запроса
	req, err := http.NewRequest("POST", URL, buf)
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// 5.1 Проверяем наличие ключа, если он есть, отправляем в заголовке хеш
	if key != "" {
		bytesBuf := buf.Bytes()
		bytesKey := []byte(key)
		hash := sha256.Sum256(append(bytesKey, bytesBuf...))
		hashHeader := hex.EncodeToString(hash[:])
		req.Header.Set("HashSHA256", hashHeader)
	}
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

func isRetriableError(err error) bool {
	// Считаем ошибку временной, если это:
	// - ошибка сети/соединения
	// - таймаут
	// - 5xx ошибка сервера
	var netErr net.Error
	return errors.As(err, &netErr)
}

func SendWithRetry(url, metricType string, storage *MemStorage, i int, key string) error {
	maxRetries := 3
	retryDelays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelays[attempt-1])
		}

		err := SendMetricsBatch(url, metricType, storage, i, key)
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetriableError(err) {
			break
		}
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
