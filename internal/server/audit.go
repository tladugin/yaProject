package server

import (
	"github.com/tladugin/yaProject.git/internal/logger"
	"net/http"
	"os"
	"sync"
)

// AuditEvent представляет событие аудита
type AuditEvent struct {
	TS        int64    `json:"ts"`         // unix timestamp события
	Metrics   []string `json:"metrics"`    // наименование полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}

// AuditData хранит данные для аудита
type AuditData struct {
	Metrics []string
	IP      string
}

var (
	auditDataMap = make(map[*http.Request]*AuditData)
	auditMutex   sync.RWMutex
)

// Observer интерфейс для наблюдателей аудита
type Observer interface {
	Notify(event AuditEvent) error
	Close() error
}

// FileObserver записывает события в файл
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// HTTPObserver отправляет события по HTTP
type HTTPObserver struct {
	url    string
	client *http.Client
	mu     sync.Mutex
}

// AuditManager управляет наблюдателями
type AuditManager struct {
	observers []Observer
	mu        sync.RWMutex
	enabled   bool
}

// CleanupAuditData очищает данные аудита после обработки
func CleanupAuditData(r *http.Request) {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	if _, exists := auditDataMap[r]; exists {
		delete(auditDataMap, r)
		logger.Sugar.Debug("Audit data cleaned up for request")
	}
}

// GetIPAddress извлекает реальный IP адрес клиента
func GetIPAddress(r *http.Request) string {
	// Пробуем получить IP из X-Real-IP
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Пробуем получить IP из X-Forwarded-For
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}

	// Используем RemoteAddr (может содержать порт)
	return r.RemoteAddr
}
