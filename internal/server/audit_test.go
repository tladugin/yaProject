package server

import (
	"encoding/json"
	"github.com/tladugin/yaProject.git/internal/logger"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMain инициализирует логгер для всех тестов
func TestMain(m *testing.M) {
	// Инициализируем логгер для тестов
	var err error
	logger.Sugar, err = logger.InitLogger()
	if err != nil {
		// Если не удалось инициализировать логгер, создаем простой для тестов
		config := zap.NewProductionConfig()
		config.OutputPaths = []string{"stdout"}
		config.ErrorOutputPaths = []string{"stderr"}
		plainLogger, _ := config.Build()
		logger.Sugar = plainLogger.Sugar()
	}

	// Запускаем тесты
	code := m.Run()

	// Очищаем ресурсы
	if logger.Sugar != nil {
		_ = logger.Sugar.Sync()
	}
	os.Exit(code)
}

// TestGetIPAddress тестирует извлечение IP адреса
func TestGetIPAddress(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:     "X-Real-IP header",
			headers:  map[string]string{"X-Real-IP": "192.168.1.1"},
			expected: "192.168.1.1",
		},
		{
			name:     "X-Forwarded-For header",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1"},
			expected: "10.0.0.1",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "172.16.0.1:12345",
			expected:   "172.16.0.1:12345",
		},
		{
			name:     "X-Real-IP priority",
			headers:  map[string]string{"X-Real-IP": "192.168.1.2", "X-Forwarded-For": "10.0.0.2"},
			expected: "192.168.1.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			ip := GetIPAddress(req)
			if ip != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

// TestCleanupAuditData тестирует очистку данных аудита
func TestCleanupAuditData(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	// Добавляем данные в карту
	auditMutex.Lock()
	auditDataMap[req] = &AuditData{
		Metrics: []string{"test_metric"},
		IP:      "127.0.0.1",
	}
	auditMutex.Unlock()

	// Проверяем что данные есть
	auditMutex.Lock()
	_, exists := auditDataMap[req]
	auditMutex.Unlock()

	if !exists {
		t.Fatal("Data should exist before cleanup")
	}

	// Очищаем - теперь логгер инициализирован, не будет паники
	CleanupAuditData(req)

	// Проверяем что данные удалены
	auditMutex.Lock()
	_, exists = auditDataMap[req]
	auditMutex.Unlock()

	if exists {
		t.Error("Data should be cleaned up")
	}
}

// TestFileObserver тестирует файлового наблюдателя
func TestFileObserver(t *testing.T) {
	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "audit_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	observer := &FileObserver{file: tmpfile}
	defer observer.Close()

	event := AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   []string{"metric1", "metric2"},
		IPAddress: "127.0.0.1",
	}

	// Тестируем запись
	err = observer.Notify(event)
	if err != nil {
		t.Errorf("Notify failed: %v", err)
	}

	// Проверяем что файл не пустой
	info, err := tmpfile.Stat()
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if info.Size() == 0 {
		t.Error("File should not be empty after Notify")
	}
}

// TestHTTPObserver тестирует HTTP наблюдателя
func TestHTTPObserver(t *testing.T) {
	// Создаем тестовый сервер
	var receivedEvent AuditEvent
	var received bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event AuditEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("JSON decode error: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		receivedEvent = event
		received = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := &HTTPObserver{
		url:    server.URL,
		client: &http.Client{Timeout: 5 * time.Second},
	}
	defer observer.Close()

	event := AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"test_metric"},
		IPAddress: "192.168.1.1",
	}

	// Тестируем отправку
	err := observer.Notify(event)
	if err != nil {
		t.Errorf("Notify failed: %v", err)
	}

	// Даем время на отправку
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !received {
		t.Error("Event was not received by test server")
	}

	if receivedEvent.IPAddress != event.IPAddress {
		t.Errorf("Expected IP %s, got %s", event.IPAddress, receivedEvent.IPAddress)
	}
}

// TestAuditManager тестирует менеджер аудита
func TestAuditManager(t *testing.T) {
	manager := NewAuditManager(true)
	defer manager.Close()

	// Тестируем уведомление без наблюдателей (не должно падать)
	event := AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   []string{"manager_test"},
		IPAddress: "127.0.0.1",
	}
	manager.NotifyAll(event)

	// Тестируем добавление наблюдателя
	tmpfile, err := os.CreateTemp("", "audit_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	fileObserver := &FileObserver{file: tmpfile}
	manager.AddObserver(fileObserver)

	// Тестируем уведомление с наблюдателем
	manager.NotifyAll(event)
}

// TestAuditEventJSON тестирует JSON сериализацию
func TestAuditEventJSON(t *testing.T) {
	event := AuditEvent{
		TS:        1234567890,
		Metrics:   []string{"cpu", "memory"},
		IPAddress: "192.168.1.1",
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Проверяем наличие полей
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "ts") {
		t.Error("JSON should contain 'ts' field")
	}
	if !strings.Contains(jsonStr, "metrics") {
		t.Error("JSON should contain 'metrics' field")
	}
	if !strings.Contains(jsonStr, "ip_address") {
		t.Error("JSON should contain 'ip_address' field")
	}

	// Демаршалим обратно
	var decodedEvent AuditEvent
	err = json.Unmarshal(jsonData, &decodedEvent)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decodedEvent.TS != event.TS {
		t.Errorf("Expected TS %d, got %d", event.TS, decodedEvent.TS)
	}
}

// TestConcurrentAccess тестирует конкурентный доступ
func TestConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup

	// Многопоточное добавление данных
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)

			auditMutex.Lock()
			auditDataMap[req] = &AuditData{
				Metrics: []string{string(rune(65 + index))}, // A, B, C...
				IP:      "127.0.0.1",
			}
			auditMutex.Unlock()

			// Очищаем - логгер инициализирован, не будет паники
			CleanupAuditData(req)
		}(i)
	}

	wg.Wait()

	// Проверяем что карта пуста
	auditMutex.Lock()
	mapSize := len(auditDataMap)
	auditMutex.Unlock()

	if mapSize != 0 {
		t.Errorf("Map should be empty, but has %d elements", mapSize)
	}
}

// TestLoggingMiddleware тестирует middleware логирования
func TestLoggingMiddleware(t *testing.T) {
	// Создаем тестовый обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Тестируем LoggingRequest
	loggingRequest := logger.LoggingRequest(logger.Sugar)
	wrappedHandler := loggingRequest(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Тестируем LoggingAnswer
	loggingAnswer := logger.LoggingAnswer(logger.Sugar)
	wrappedHandler = loggingAnswer(handler)

	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestFileObserverClose тестирует закрытие файлового наблюдателя
func TestFileObserverClose(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "audit_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	observer := &FileObserver{file: tmpfile}

	// Закрытие должно работать без ошибок
	err = observer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestHTTPObserverClose тестирует закрытие HTTP наблюдателя
func TestHTTPObserverClose(t *testing.T) {
	observer := &HTTPObserver{
		client: &http.Client{Timeout: 5 * time.Second},
	}

	// Закрытие должно работать без ошибок
	err := observer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestAuditManagerWithObservers тестирует менеджер с наблюдателями
func TestAuditManagerWithObservers(t *testing.T) {
	manager := NewAuditManager(true)
	defer manager.Close()

	// Добавляем файлового наблюдателя
	tmpfile, err := os.CreateTemp("", "audit_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	fileObserver := &FileObserver{file: tmpfile}
	manager.AddObserver(fileObserver)

	// Добавляем HTTP наблюдателя с тестовым сервером
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpObserver := &HTTPObserver{
		url:    server.URL,
		client: &http.Client{Timeout: 5 * time.Second},
	}
	manager.AddObserver(httpObserver)

	// Тестируем уведомление всех наблюдателей
	event := AuditEvent{
		TS:        time.Now().Unix(),
		Metrics:   []string{"test1", "test2"},
		IPAddress: "127.0.0.1",
	}

	manager.NotifyAll(event)
}
