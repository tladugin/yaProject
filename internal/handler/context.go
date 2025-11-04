package handler

import (
	"context"
	"net/http"
)

type contextKey string

const (
	metricsContextKey = contextKey("metrics")
	ipContextKey      = contextKey("ip")
)

// getIPAddress извлекает реальный IP адрес клиента
func getIPAddress(r *http.Request) string {
	// Пробуем получить IP из X-Real-IP
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Пробуем получить IP из X-Forwarded-For
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}

	// Используем RemoteAddr
	return r.RemoteAddr
}

// WithAuditData добавляет данные для аудита в контекст
func WithAuditData(r *http.Request, metrics []string, ip string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, metricsContextKey, metrics)
	ctx = context.WithValue(ctx, ipContextKey, ip)
	return r.WithContext(ctx)
}

// GetAuditData получает данные аудита из контекста
func GetAuditData(r *http.Request) ([]string, string) {
	metrics, _ := r.Context().Value(metricsContextKey).([]string)
	ip, _ := r.Context().Value(ipContextKey).(string)
	return metrics, ip
}
