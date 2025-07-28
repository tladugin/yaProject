package logger

import (
	"go.uber.org/zap"
	"net/http"
	"os"

	"time"
)

func InitLogger() (*zap.SugaredLogger, error) {

	log, err := zap.NewProduction(
		zap.ErrorOutput(os.Stdout), // Перенаправляем ошибки в stdout
	)
	if err != nil {
		return nil, err
	}
	sugar := log.Sugar()

	return sugar, nil
}

func LoggingRequest(sugar *zap.SugaredLogger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()
			// эндпоинт
			uri := r.RequestURI
			// метод запроса
			method := r.Method

			// точка, где выполняется хендлер pingHandler
			h.ServeHTTP(w, r) // обслуживание оригинального запроса

			// Since возвращает разницу во времени между start
			// и моментом вызова Since. Таким образом можно посчитать
			// время выполнения запроса.
			duration := time.Since(start)

			// отправляем сведения о запросе в zap
			sugar.Infoln(
				"URL:", uri,
				"METHOD:", method,
				"DURATION:", duration,
			)

		})

	}
}

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}
func LoggingAnswer(sugar *zap.SugaredLogger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			responseData := &responseData{
				status: 200,
				size:   0,
			}
			lw := &loggingResponseWriter{
				ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
				responseData:   responseData,
			}
			h.ServeHTTP(lw, r) // внедряем реализацию http.ResponseWriter

			sugar.Infoln(
				"STATUS CODE:", responseData.status, // получаем перехваченный код статуса ответа
				"ANSWER SIZE:", responseData.size, // получаем перехваченный размер ответа
			)
		})
	}
}
