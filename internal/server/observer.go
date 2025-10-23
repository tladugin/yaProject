package server

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Notify записывает событие в файл
func (o *FileObserver) Notify(event AuditEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Сериализуем событие в JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Записываем в файл с новой строки
	if _, err := o.file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	return nil
}

// Close закрывает файл
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.file != nil {
		return o.file.Close()
	}
	return nil
}

// Notify отправляет событие по HTTP
func (o *HTTPObserver) Notify(event AuditEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	resp, err := o.client.Post(o.url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send audit event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("audit server returned status: %d", resp.StatusCode)
	}

	return nil
}

// Close закрывает HTTP наблюдатель
func (o *HTTPObserver) Close() error {
	// Можно закрыть HTTP клиент если нужно
	return nil
}
