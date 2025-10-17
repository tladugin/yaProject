package server

import (
	"context"

	"net/http"
	"time"
)

type contextKey string

const (
	metricsContextKey = contextKey("metrics")
	ipContextKey      = contextKey("ip")
)

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

// AuditMiddleware создает middleware для аудита запросов
func AuditMiddleware(auditManager *AuditManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если аудит отключен, пропускаем
			if !auditManager.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			// Создаем канал для сбора метрик
			metricsChan := make(chan []string, 1)

			// Оборачиваем ResponseWriter для перехвата данных
			rw := &auditResponseWriter{
				ResponseWriter: w,
				metricsChan:    metricsChan,
				request:        r, // Сохраняем ссылку на запрос
			}

			// Обрабатываем запрос
			next.ServeHTTP(rw, r)

			// Ждем сбор метрик (асинхронно)
			go func(req *http.Request) {
				select {
				case metrics := <-metricsChan:
					// Получаем IP адрес
					ip := getIPAddress(req)

					// Создаем событие аудита
					event := AuditEvent{
						TS:        time.Now().Unix(),
						Metrics:   metrics,
						IPAddress: ip,
					}

					// Отправляем событие наблюдателям
					auditManager.NotifyAll(event)
				case <-time.After(1 * time.Second):
					// Таймаут сбора метрик
				}
			}(r) // Передаем оригинальный запрос
		})
	}
}

// auditResponseWriter перехватывает данные ответа
type auditResponseWriter struct {
	http.ResponseWriter
	metricsChan chan<- []string
	request     *http.Request
	wroteHeader bool
}

func (w *auditResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		// Если статус успешный, собираем метрики
		if code >= 200 && code < 300 {
			w.collectMetrics()
		} else {
			// Для неуспешных ответов отправляем пустой список
			w.metricsChan <- []string{}
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *auditResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *auditResponseWriter) collectMetrics() {
	// Получаем метрики из контекста запроса
	metrics, _ := GetAuditData(w.request)
	w.metricsChan <- metrics
}

// getIPAddress извлекает реальный IP адрес клиента
func getIPAddress(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}

	return r.RemoteAddr
}
