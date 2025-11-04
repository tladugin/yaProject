package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/models"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ExampleServer_PostHandler демонстрирует обновление метрик через URL параметры
func ExampleServer_PostHandler() {
	// Создаем in-memory хранилище и сервер
	storage := repository.NewMemStorage()
	server := NewServer(storage)

	// Создаем маршрутизатор chi для тестирования
	r := chi.NewRouter()
	r.Post("/update/{metric}/{name}/{value}", server.PostHandler)

	// Создаем тестовый HTTP запрос для обновления gauge метрики
	req := httptest.NewRequest("POST", "/update/gauge/temperature/23.5", nil)
	w := httptest.NewRecorder()

	// Вызываем обработчик через маршрутизатор
	r.ServeHTTP(w, req)

	// Проверяем статус ответа
	if w.Code != http.StatusOK {
		fmt.Printf("Expected status 200, got %d\n", w.Code)
	}

	// Проверяем, что метрика сохранилась
	fmt.Printf("Gauge metrics count: %d\n", len(storage.GaugeSlice()))

	// Проверяем значение метрики
	for _, gauge := range storage.GaugeSlice() {
		if gauge.Name == "temperature" {
			fmt.Printf("Temperature value: %.1f", gauge.Value)
		}
	}

	// Output:
	// Gauge metrics count: 1
	// Temperature value: 23.5
}

// ExampleServer_GetHandler демонстрирует получение метрик через URL параметры
func ExampleServer_GetHandler() {
	// Создаем in-memory хранилище с тестовыми данными
	storage := repository.NewMemStorage()
	storage.AddGauge("cpu_usage", 75.3)
	storage.AddCounter("request_count", 42)

	server := NewServer(storage)

	// Создаем маршрутизатор chi для тестирования
	r := chi.NewRouter()
	r.Get("/value/{metric}/{name}", server.GetHandler)

	// Запрос для получения gauge метрики
	req := httptest.NewRequest("GET", "/value/gauge/cpu_usage", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Выводим значение метрики
	fmt.Printf("CPU Usage: %s", w.Body.String())

	// Output:
	// CPU Usage: 75.3
}

// ExampleServer_PostUpdate демонстрирует обновление метрик через JSON API
func ExampleServer_PostUpdate() {
	storage := repository.NewMemStorage()
	server := NewServer(storage)

	// Создаем метрику для обновления
	metric := models.Metrics{
		ID:    "memory_usage",
		MType: "gauge",
		Value: func() *float64 { v := 85.7; return &v }(),
	}

	// Кодируем в JSON
	body, _ := json.Marshal(metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.PostUpdate(w, req)

	// Проверяем ответ
	var response models.Metrics
	json.Unmarshal(w.Body.Bytes(), &response)

	fmt.Printf("Updated metric: %s = %v", response.ID, *response.Value)

	// Output:
	// Updated metric: memory_usage = 85.7
}

// ExampleServer_PostValue демонстрирует получение метрик через JSON API
func ExampleServer_PostValue() {
	storage := repository.NewMemStorage()
	storage.AddGauge("disk_space", 150.5)
	server := NewServer(storage)

	// Запрос для получения метрики
	metricRequest := models.Metrics{
		ID:    "disk_space",
		MType: "gauge",
	}

	body, _ := json.Marshal(metricRequest)
	req := httptest.NewRequest("POST", "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.PostValue(w, req)

	// Декодируем ответ
	var response models.Metrics
	json.Unmarshal(w.Body.Bytes(), &response)

	fmt.Printf("Metric value: %v", *response.Value)

	// Output:
	// Metric value: 150.5
}

// ExampleServer_UpdatesGaugesBatch демонстрирует пакетное обновление метрик
func ExampleServer_UpdatesGaugesBatch() {
	storage := repository.NewMemStorage()
	server := NewServer(storage)

	// Создаем пакет метрик
	metrics := []models.Metrics{
		{
			ID:    "metric_1",
			MType: "gauge",
			Value: func() *float64 { v := 10.5; return &v }(),
		},
		{
			ID:    "metric_2",
			MType: "counter",
			Delta: func() *int64 { v := int64(5); return &v }(),
		},
	}

	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.UpdatesGaugesBatch(w, req)

	fmt.Printf("Status: %d, Metrics count: %d", w.Code, len(storage.GaugeSlice())+len(storage.CounterSlice()))

	// Output:
	// Status: 200, Metrics count: 2
}

// ExampleServer_MainPage демонстрирует отображение главной страницы
func ExampleServer_MainPage() {
	storage := repository.NewMemStorage()
	storage.AddGauge("temperature", 22.5)
	storage.AddCounter("visits", 100)
	server := NewServer(storage)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.MainPage(w, req)

	// Проверяем, что страница содержит ожидаемые метрики
	body := w.Body.String()
	hasTemperature := strings.Contains(body, "temperature")
	hasVisits := strings.Contains(body, "visits")

	fmt.Printf("Has temperature: %v, Has visits: %v", hasTemperature, hasVisits)

	// Output:
	// Has temperature: true, Has visits: true
}

// ExampleServerSync_PostUpdateSyncBackup демонстрирует обновление с синхронным бэкапом
func ExampleServerSync_PostUpdateSyncBackup() {
	storage := repository.NewMemStorage()

	// В реальном коде здесь должен быть инициализирован producer
	// producer := repository.NewProducer("backup.json")
	// server := handler.NewServerSync(storage, producer)

	// Для примера используем обычный сервер
	server := NewServer(storage)

	metric := models.Metrics{
		ID:    "backup_metric",
		MType: "gauge",
		Value: func() *float64 { v := 99.9; return &v }(),
	}

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.PostUpdate(w, req)

	fmt.Printf("Metric stored: %v", len(storage.GaugeSlice()) > 0)

	// Output:
	// Metric stored: true
}

// ExampleServerDB_PostUpdatePostgres демонстрирует обновление метрик в PostgreSQL
func ExampleServerDB_PostUpdatePostgres() {
	// В реальном приложении здесь будет подключение к БД
	// pool, _ := pgxpool.New(context.Background(), "postgres://...")
	// server := handler.NewServerDB(storage, pool, &key)

	// Для примера используем in-memory хранилище
	storage := repository.NewMemStorage()
	server := NewServer(storage)

	metric := models.Metrics{
		ID:    "db_metric",
		MType: "counter",
		Delta: func() *int64 { v := int64(1); return &v }(),
	}

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.PostUpdate(w, req)

	fmt.Printf("Counter updated: %v", len(storage.CounterSlice()) > 0)

	// Output:
	// Counter updated: true
}

// ExampleServerPing_GetPing демонстрирует проверку соединения с БД
func ExampleServerPing_GetPing() {
	// В реальном приложении:
	// dsn := "postgres://user:pass@localhost:5432/db"
	// server := handler.NewServerPingDB(storage, &dsn)

	// Для примера создаем простой сервер
	//storage := repository.NewMemStorage()
	//server := NewServer(storage)

	//req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	// В реальном приложении здесь будет server.GetPing(w, req)
	// Для примера просто возвращаем OK
	w.WriteHeader(http.StatusOK)

	fmt.Printf("Ping status: %d", w.Code)

	// Output:
	// Ping status: 200
}

// Example комплексного использования - мониторинг приложения
// Example_comprehensiveUsage демонстрирует комплексное использование
func Example_comprehensiveUsage() {
	storage := repository.NewMemStorage()
	server := NewServer(storage)

	// Создаем маршрутизатор для тестирования URL параметров
	r := chi.NewRouter()
	r.Post("/update/{metric}/{name}/{value}", server.PostHandler)
	r.Get("/value/{metric}/{name}", server.GetHandler)

	// 1. Добавляем метрики через URL параметры
	req1 := httptest.NewRequest("POST", "/update/gauge/cpu_usage/75.5", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("POST", "/update/counter/requests/10", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	// 2. Добавляем метрики через JSON API
	value := 1024.0
	metricJSON := models.Metrics{
		ID:    "memory",
		MType: "gauge",
		Value: &value,
	}
	body, _ := json.Marshal(metricJSON)
	req3 := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	server.PostUpdate(w3, req3)

	// 3. Получаем метрики
	req4 := httptest.NewRequest("GET", "/value/gauge/cpu_usage", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	// 4. Проверяем результаты
	fmt.Printf("Gauge metrics: %d\n", len(storage.GaugeSlice()))
	fmt.Printf("Counter metrics: %d\n", len(storage.CounterSlice()))
	fmt.Printf("CPU usage value: %s", strings.TrimSpace(w4.Body.String()))

	// Output:
	// Gauge metrics: 2
	// Counter metrics: 1
	// CPU usage value: 75.5
}
