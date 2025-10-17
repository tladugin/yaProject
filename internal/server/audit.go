package server

import (
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
