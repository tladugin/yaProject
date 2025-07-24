package logger

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

func LoggingRequest(h http.HandlerFunc, sugar zap.SugaredLogger) http.HandlerFunc {

	logFn := func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		// эндпоинт /ping
		uri := r.RequestURI
		// метод запроса
		method := r.Method
		// функция Now() возвращает текущее время

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

	}
	// возвращаем функционально расширенный хендлер
	return http.HandlerFunc(logFn)
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
func LoggingAnswer(h http.HandlerFunc, sugar zap.SugaredLogger) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {

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
	}
	// возвращаем функционально расширенный хендлер
	return http.HandlerFunc(logFn)
}
