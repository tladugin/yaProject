package repository

import (
	"bytes"
	"compress/gzip"
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

func SendMetricsBatch(URL string, metricType string, storage *MemStorage, batchSize int, key string, pollCounter int64) error {
	// 1. Подготовка URL
	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	// 2. Подготовка метрик
	var metrics []models.Metrics
	switch metricType {
	case "gauge":
		if len(storage.GaugeSlice()) == 0 {
			return nil
		}

		for i := 0; i < batchSize; i++ {
			value := storage.GaugeSlice()[i].Value // Создаем копию значения
			metrics = append(metrics, models.Metrics{
				MType: "gauge",
				ID:    storage.GaugeSlice()[i].Name,
				Value: &value,
			})
		}
	case "counter":
		if len(storage.CounterSlice()) == 0 {
			return nil
		}

		for i := 0; i < batchSize; i++ {
			//delta := storage.CounterSlice()[i].Value
			delta := pollCounter
			metrics = append(metrics, models.Metrics{
				MType: "counter",
				ID:    storage.CounterSlice()[i].Name,
				Delta: &delta,
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
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress data error: %w", err)
	}

	// 5. Создание запроса
	req, err := http.NewRequest("POST", URL, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// 6. Добавление хеша, если есть ключ
	if key != "" {
		hash := sha256.Sum256(append([]byte(key), jsonData...)) // Хешируем исходные данные
		req.Header.Set("HashSHA256", hex.EncodeToString(hash[:]))
	}

	// 7. Отправка запроса
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 8. Проверка ответа
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read error response: %w", err)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isRetriableError(err error) bool {
	// Считаем ошибку временной, если это:
	// - ошибка сети/соединения
	// - таймаут
	// - 5xx ошибка сервера
	var netErr net.Error
	return errors.As(err, &netErr)
}

func SendWithRetry(url string, storage *MemStorage, key string, pollCounter int64) error {
	maxRetries := 3
	retryDelays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelays[attempt-1])
		}

		errG := SendMetricsBatch(url, "gauge", storage, len(storage.gaugeSlice), key, pollCounter)
		if errG != nil {
			lastErr = errG
		}

		lastErr = errG

		errC := SendMetricsBatch(url, "counter", storage, len(storage.counterSlice), key, pollCounter)
		if errC == nil {
			return nil
		}

		lastErr = errC

		if !isRetriableError(errG) || !isRetriableError(errC) {
			break
		}
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
