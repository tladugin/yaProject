package server

import (
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"net/http"
	"time"
)

// AuditMiddleware создает middleware для аудита запросов
func AuditMiddleware(auditManager *AuditManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если аудит отключен, пропускаем
			if !auditManager.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			logger.Sugar.Debugf("Audit middleware started for: %s %s", r.Method, r.URL.Path)

			// Создаем wrapper для ResponseWriter чтобы перехватить статус
			rw := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			// Обрабатываем запрос
			next.ServeHTTP(rw, r)

			logger.Sugar.Debugf("Request completed with status: %d", rw.status)

			// После обработки проверяем успешность и отправляем аудит
			if rw.status >= http.StatusOK && rw.status < http.StatusBadRequest {
				go func(req *http.Request, status int) {
					// Получаем данные аудита
					metrics, ip := handler.GetAuditData(req)

					logger.Sugar.Debugf("Audit data retrieved - Metrics: %v, IP: %s", metrics, ip)

					if len(metrics) > 0 { // Отправляем только если есть метрики
						// Создаем событие аудита
						event := AuditEvent{
							TS:        time.Now().Unix(),
							Metrics:   metrics,
							IPAddress: ip,
						}

						logger.Sugar.Infof("Sending audit event: %+v", event)

						// Отправляем событие наблюдателям
						auditManager.NotifyAll(event)
						logger.Sugar.Infof("Audit event sent successfully: %d metrics from %s", len(metrics), ip)
					} else {
						logger.Sugar.Warn("No metrics found for audit - event will not be sent")
					}

					// Очищаем данные
					CleanupAuditData(req)
				}(r, rw.status)
			} else {
				// Очищаем данные даже при неуспешном ответе
				CleanupAuditData(r)
				logger.Sugar.Debugf("Request failed with status %d, audit skipped", rw.status)
			}
		})
	}
}

// statusRecorder записывает статус ответа
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(data)
}
