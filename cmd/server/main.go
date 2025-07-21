package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

var sugar zap.SugaredLogger

func gzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(ow, r)
	}
}
func main() {

	log, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer log.Sync()

	sugar = *log.Sugar()

	parseFlags()

	storage := repository.NewMemStorage()
	s := handler.NewServer(storage)

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {

		r.Get("/", logger.LoggingAnswer(s.MainPage, sugar))
		r.Get("/value/{metric}/{name}", logger.LoggingAnswer(s.GetHandler, sugar))
		r.Post("/update/{metric}/{name}/{value}", logger.LoggingRequest(s.PostHandler, sugar))
		r.Post("/update", logger.LoggingRequest(gzipMiddleware(s.PostUpdate), sugar))
		r.Post("/update/", logger.LoggingRequest(gzipMiddleware(s.PostUpdate), sugar))
		r.Post("/value", logger.LoggingRequest(gzipMiddleware(s.PostValue), sugar))
		r.Post("/value/", logger.LoggingRequest(gzipMiddleware(s.PostValue), sugar))
	})
	sugar.Infoln("Starting server on :", flagRunAddr)
	if err := http.ListenAndServe(flagRunAddr, r); err != nil {
		sugar.Errorln("Server failed: %v\n", err)
	}
}
