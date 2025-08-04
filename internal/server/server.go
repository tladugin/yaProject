package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"

	"net/http"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func RunHTTPServer(storage *repository.MemStorage, producer *handler.Producer, stop <-chan struct{}, wg *sync.WaitGroup, flagStoreInterval string, sugar *zap.SugaredLogger, flagRunAddr *string, flagConnectString *string) {
	defer wg.Done()

	s := handler.NewServer(storage)
	c := handler.NewServerDB(storage, flagConnectString)
	sSync := handler.NewServerSync(storage, producer)

	r := chi.NewRouter()
	r.Use(repository.GzipMiddleware,
		logger.LoggingAnswer(sugar),
		logger.LoggingRequest(sugar),
	)
	r.Route("/", func(r chi.Router) {

		r.Get("/", s.MainPage)
		r.Get("/ping", c.GetPing)
		r.Get("/value/{metric}/{name}", s.GetHandler)
		r.Post("/update/{metric}/{name}/{value}", s.PostHandler)

		if flagStoreInterval == "0" {
			sugar.Info("Running in sync backup mode")
			r.Post("/update", sSync.PostUpdateSyncBackup)
			r.Post("/update/", sSync.PostUpdateSyncBackup)
		} else {
			sugar.Info("Running in async backup mode")
			r.Post("/update", s.PostUpdate)
			r.Post("/update/", s.PostUpdate)
		}

		r.Post("/value", s.PostValue)
		r.Post("/value/", s.PostValue)
	})

	server := &http.Server{
		Addr:    *flagRunAddr,
		Handler: r,
	}

	go func() {
		<-stop
		sugar.Info("Shutting down HTTP server...")
		if err := server.Close(); err != nil {
			sugar.Error("HTTP server shutdown error: ", err)
		}
	}()

	sugar.Infof("Starting server on %s", *flagRunAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		sugar.Error("Server failed: ", err)
	}
}
