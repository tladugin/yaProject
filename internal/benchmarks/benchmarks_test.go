package benchmark

import (
	"bytes"

	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"net/http/httptest"
	"testing"

	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/models"
	"github.com/tladugin/yaProject.git/internal/repository"
)

// Бенчмарк для добавления метрик в хранилище
func BenchmarkStorageAddGauge(b *testing.B) {
	storage := repository.NewMemStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.AddGauge("test_metric", 123.45)
	}
}

func BenchmarkStorageAddCounter(b *testing.B) {
	storage := repository.NewMemStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.AddCounter("test_counter", 1)
	}
}

// Бенчмарк для обработки HTTP запросов
func BenchmarkHandlerPostUpdate(b *testing.B) {
	storage := repository.NewMemStorage()
	s := handler.NewServer(storage)

	metric := models.Metrics{
		ID:    "test_metric",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	body, _ := json.Marshal(metric)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.PostUpdate(rr, req)
	}
}

// Бенчмарк для пакетного обновления метрик
func BenchmarkHandlerUpdatesBatch(b *testing.B) {
	storage := repository.NewMemStorage()
	s := handler.NewServer(storage)

	metrics := make([]models.Metrics, 100)
	for i := 0; i < 100; i++ {
		value := float64(i) + 0.5
		metrics[i] = models.Metrics{
			ID:    string(rune('a' + i%26)),
			MType: "gauge",
			Value: &value,
		}
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		b.Fatalf("Failed to marshal: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.UpdatesGaugesBatch(rr, req)
	}
}

// Бенчмарк для вычисления хеша
func BenchmarkHashCalculation(b *testing.B) {
	key := "test_key"
	data := []byte("test_data_for_hashing")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesKey := []byte(key)
		hash := sha256.Sum256(append(bytesKey, data...))
		_ = hex.EncodeToString(hash[:])
	}
}

// Бенчмарк для JSON маршалинга
func BenchmarkJSONMarshal(b *testing.B) {
	metric := models.Metrics{
		ID:    "test_metric",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(metric)
	}
}

// Бенчмарк для JSON анмаршалинга
func BenchmarkJSONUnmarshal(b *testing.B) {
	data := []byte(`{"id":"test_metric","type":"gauge","value":123.45}`)
	var metric models.Metrics

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(data, &metric)
	}
}
