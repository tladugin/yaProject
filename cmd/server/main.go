package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
	"net/http"
)

var sugar zap.SugaredLogger

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

		r.Get("/", logger.LoggingAnswer(gzipMiddleware(s.MainPage), sugar))
		r.Get("/value/{metric}/{name}", logger.LoggingAnswer(gzipMiddleware(s.GetHandler), sugar))
		r.Post("/update/{metric}/{name}/{value}", logger.LoggingRequest(gzipMiddleware(s.PostHandler), sugar))
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
