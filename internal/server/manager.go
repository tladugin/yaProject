package server

import (
	"github.com/tladugin/yaProject.git/internal/logger"
	"log"
	"sync"
)

// initAuditObservers инициализирует наблюдатели на основе конфигурации
func initAuditObservers(manager *AuditManager, flagAuditFile, flagAuditURL *string) {
	// Файловый наблюдатель
	if *flagAuditFile != "" {
		fileObserver, err := NewFileObserver(*flagAuditFile)
		if err != nil {
			logger.Sugar.Errorf("Failed to create file audit observer: %v", err)
		} else {
			manager.AddObserver(fileObserver)
			logger.Sugar.Infof("File audit enabled: %s", *flagAuditFile)
		}
	}

	// HTTP наблюдатель
	if *flagAuditURL != "" {
		httpObserver := NewHTTPObserver(*flagAuditURL)
		manager.AddObserver(httpObserver)
		logger.Sugar.Infof("HTTP audit enabled: %s", *flagAuditURL)
	}

	if manager.IsEnabled() {
		logger.Sugar.Info("Audit system initialized")
	} else {
		logger.Sugar.Info("Audit is disabled - no observers configured")
	}
}

// NewAuditManager создает новый менеджер аудита
func NewAuditManager() *AuditManager {
	return &AuditManager{
		observers: make([]Observer, 0),
		enabled:   false,
	}
}

// AddObserver добавляет наблюдателя
func (m *AuditManager) AddObserver(observer Observer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.observers = append(m.observers, observer)
	m.enabled = true
}

// NotifyAll уведомляет всех наблюдателей о событии
func (m *AuditManager) NotifyAll(event AuditEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return
	}

	var wg sync.WaitGroup

	for _, observer := range m.observers {
		wg.Add(1)
		go func(obs Observer) {
			defer wg.Done()
			if err := obs.Notify(event); err != nil {
				log.Printf("Audit observer error: %v", err)
			}
		}(observer)
	}

	wg.Wait()
}

// Close закрывает всех наблюдателей
func (m *AuditManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.observers {
		if err := observer.Close(); err != nil {
			log.Printf("Error closing audit observer: %v", err)
		}
	}

	m.observers = nil
	m.enabled = false
}

// IsEnabled проверяет включен ли аудит
func (m *AuditManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}
