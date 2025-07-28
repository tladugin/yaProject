package repository

import (
	"encoding/json"
	"fmt"

	models "github.com/tladugin/yaProject.git/internal/model"

	"io"
	"log"
	"net/http"

	"strings"
	"sync"
)

type gauge struct {
	Name  string
	Value float64
}
type counter struct {
	Name  string
	Value int64
}
type MemStorage struct {
	counterSlice []counter
	gaugeSlice   []gauge
}

func (s *MemStorage) GaugeSlice() []gauge {
	return s.gaugeSlice
}

func (s *MemStorage) CounterSlice() []counter {
	return s.counterSlice
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		counterSlice: make([]counter, 0),
		gaugeSlice:   make([]gauge, 0),
	}
}

var mutex sync.Mutex

func (s *MemStorage) AddGauge(name string, value float64) {
	//fmt.Println(name, value)
	mutex.Lock()
	defer mutex.Unlock()
	for i, m := range s.gaugeSlice {
		if m.Name == name {
			s.gaugeSlice[i].Value = value
			return
		}
	}

	s.gaugeSlice = append(s.gaugeSlice, gauge{Name: name, Value: value})
}
func (s *MemStorage) AddCounter(name string, value int64) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, m := range s.counterSlice {
		if m.Name == name {
			s.counterSlice[i].Value += value
			return
		}
	}

	s.counterSlice = append(s.counterSlice, counter{Name: name, Value: value})
}

func SendMetric(URL string, metricType string, storage *MemStorage, i int) error {
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
		fmt.Errorf("compress data error: %w", err)
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
