package server

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"sync"
)

func RunHTTPServer(storage *repository.MemStorage, producer *repository.Producer, stop <-chan struct{}, wg *sync.WaitGroup, flagStoreInterval string, flagRunAddr *string, flagDatabaseDSN *string, flagKey *string) {
	defer wg.Done()

	s := handler.NewServer(storage)
	sSync := handler.NewServerSync(storage, producer)

	var db *handler.ServerDB
	var ping *handler.ServerPing

	if *flagDatabaseDSN != "" {
		//проверка миграций
		p, _, err := repository.NewPostgresRepository(*flagDatabaseDSN)
		if err != nil {
			logger.Sugar.Error("Failed to initialize storage: %v", err.Error())
		}
		defer p.Close()
		//соединение с БД
		pool, _, _, err := repository.GetConnection(*flagDatabaseDSN)
		if err != nil {
			logger.Sugar.Error("Failed to get connection!: %v", err.Error())
		}
		defer pool.Close()
		ping = handler.NewServerPingDB(storage, flagDatabaseDSN)
		db = handler.NewServerDB(storage, pool, flagKey)

	}

	r := chi.NewRouter()
	r.Use(repository.GzipMiddleware,
		logger.LoggingAnswer(logger.Sugar),
		logger.LoggingRequest(logger.Sugar),
	)
	r.Route("/", func(r chi.Router) {

		if *flagDatabaseDSN != "" {

			r.Get("/ping", ping.GetPing)
			r.Post("/update", db.PostUpdatePostgres)
			r.Post("/update/", db.PostUpdatePostgres)
			r.Post("/value", db.PostValue)
			r.Post("/value/", db.PostValue)
			r.Post("/updates", db.UpdatesGaugesBatchPostgres)
			r.Post("/updates/", db.UpdatesGaugesBatchPostgres)

		} else {
			r.Get("/", s.MainPage)
			r.Get("/value/{metric}/{name}", s.GetHandler)
			r.Post("/update/{metric}/{name}/{value}", s.PostHandler)

			if flagStoreInterval == "0" {
				logger.Sugar.Info("Running in sync backup mode")
				r.Post("/update", sSync.PostUpdateSyncBackup)
				r.Post("/update/", sSync.PostUpdateSyncBackup)
			} else {
				logger.Sugar.Info("Running in async backup mode")
				r.Post("/update", s.PostUpdate)
				r.Post("/update/", s.PostUpdate)
			}

			r.Post("/value", s.PostValue)
			r.Post("/value/", s.PostValue)
		}

	})

	server := &http.Server{
		Addr:    *flagRunAddr,
		Handler: r,
	}

	go func() {
		<-stop
		logger.Sugar.Info("Shutting down HTTP server...")
		if err := server.Close(); err != nil {
			logger.Sugar.Error("HTTP server shutdown error: ", err)
		}
	}()

	logger.Sugar.Infof("Starting server on %s", *flagRunAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Sugar.Error("Server failed: ", err)
	}
}
