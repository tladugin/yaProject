package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	models "github.com/tladugin/yaProject.git/internal/model"
	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	sugar     *zap.SugaredLogger
	producerM sync.Mutex
)

func main() {
	parseFlags()

	// Инициализация логгера
	initLogger()
	defer func() {
		_ = sugar.Sync() // Безопасное закрытие логгера
	}()

	storage := repository.NewMemStorage()

	// Восстановление данных из бэкапа
	if flagRestore {
		restoreFromBackup(storage)
	}

	// Инициализация продюсера
	producer, err := handler.NewProducer(flagFileStoragePath)
	if err != nil {
		sugar.Fatal("Could not open backup file: ", err)
	}
	//defer safeCloseProducer(producer)

	// Парсинг интервала сохранения
	storeInterval, err := time.ParseDuration(flagStoreInterval + "s")
	if err != nil {
		sugar.Fatal("Invalid store interval: ", err)
	}

	// Канал для graceful shutdown
	stopProgram := make(chan struct{})
	var wg sync.WaitGroup

	// Запуск фоновых задач
	if flagStoreInterval != "0" {
		wg.Add(1)
		go runPeriodicBackup(storage, producer, storeInterval, stopProgram, &wg)
	}

	wg.Add(1)
	go runFinalBackup(storage, producer, stopProgram, &wg)

	wg.Add(1)
	go runHTTPServer(storage, producer, stopProgram, &wg)

	// Ожидание сигнала завершения
	waitForShutdown(stopProgram)
	wg.Wait()
	sugar.Info("Application shutdown complete")
}

func initLogger() {
	log, err := zap.NewProduction(
		zap.ErrorOutput(os.Stdout), // Перенаправляем ошибки в stdout
	)
	if err != nil {
		panic(err)
	}
	sugar = log.Sugar()
}

func restoreFromBackup(storage *repository.MemStorage) {
	consumer, err := handler.NewConsumer(flagFileStoragePath)
	if err != nil {
		sugar.Error("Failed to create consumer: ", err)
		return
	}
	defer consumer.Close()

	event, err := consumer.ReadEvent()
	if err != nil {
		sugar.Error("Failed to read event: ", err)
		return
	}

	for event != nil {
		switch event.MType {
		case "gauge":
			storage.AddGauge(event.ID, *event.Value)
		case "counter":
			storage.AddCounter(event.ID, *event.Delta)
		}

		event, err = consumer.ReadEvent()
		if err != nil {
			sugar.Info("Backup restore completed")
			break
		}
	}
	for m := range storage.GaugeSlice() {
		sugar.Infoln(storage.GaugeSlice()[m].Name, storage.GaugeSlice()[m].Value)
	}
	for m := range storage.CounterSlice() {
		sugar.Infoln(storage.CounterSlice()[m].Name, storage.CounterSlice()[m].Value)
	}
}

func runPeriodicBackup(storage *repository.MemStorage, producer *handler.Producer, interval time.Duration, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := performBackup(storage, producer); err != nil {
				sugar.Error("Periodic backup failed: ", err)
			}
		case <-stop:
			return
		}
	}
}

func runFinalBackup(storage *repository.MemStorage, producer *handler.Producer, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	<-stop

	sugar.Info("Starting final backup...")
	producer.Close()
	if err := performBackup(storage, producer); err != nil {
		sugar.Error("Final backup failed: ", err)
	}
	sugar.Info("Final backup completed")
}

func performBackup(storage *repository.MemStorage, producer *handler.Producer) error {
	producerM.Lock()
	defer producerM.Unlock()

	if err := os.Remove(flagFileStoragePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Создаем новый продюсер (файл будет создан заново)
	newProducer, err := handler.NewProducer(flagFileStoragePath)
	if err != nil {
		return err
	}
	defer producer.Close()

	// Бэкап gauge метрик
	for _, gauge := range storage.GaugeSlice() {
		backup := models.Metrics{
			ID:    gauge.Name,
			MType: "gauge",
			Value: &gauge.Value,
		}
		if err := newProducer.WriteEvent(&backup); err != nil {
			return err
		}
	}

	// Бэкап counter метрик
	for _, counter := range storage.CounterSlice() {
		backup := models.Metrics{
			ID:    counter.Name,
			MType: "counter",
			Delta: &counter.Value,
		}
		if err := newProducer.WriteEvent(&backup); err != nil {
			return err
		}
	}

	return nil
}

func runHTTPServer(storage *repository.MemStorage, producer *handler.Producer, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	s := handler.NewServer(storage)
	sSync := handler.NewServerSync(storage, producer)

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.LoggingAnswer(gzipMiddleware(s.MainPage), *sugar))
		r.Get("/value/{metric}/{name}", logger.LoggingAnswer(s.GetHandler, *sugar))
		r.Post("/update/{metric}/{name}/{value}", logger.LoggingRequest(s.PostHandler, *sugar))

		if flagStoreInterval == "0" {
			sugar.Info("Running in sync backup mode")
			r.Post("/update", logger.LoggingRequest(gzipMiddleware(sSync.PostUpdateSyncBackup), *sugar))
			r.Post("/update/", logger.LoggingRequest(gzipMiddleware(sSync.PostUpdateSyncBackup), *sugar))
		} else {
			sugar.Info("Running in async backup mode")
			r.Post("/update", logger.LoggingRequest(gzipMiddleware(s.PostUpdate), *sugar))
			r.Post("/update/", logger.LoggingRequest(gzipMiddleware(s.PostUpdate), *sugar))
		}

		r.Post("/value", logger.LoggingRequest(gzipMiddleware(s.PostValue), *sugar))
		r.Post("/value/", logger.LoggingRequest(gzipMiddleware(s.PostValue), *sugar))
	})

	server := &http.Server{
		Addr:    flagRunAddr,
		Handler: r,
	}

	go func() {
		<-stop
		sugar.Info("Shutting down HTTP server...")
		if err := server.Close(); err != nil {
			sugar.Error("HTTP server shutdown error: ", err)
		}
	}()

	sugar.Infof("Starting server on %s", flagRunAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		sugar.Error("Server failed: ", err)
	}
}

func safeCloseProducer(p *handler.Producer) {
	if err := p.Close(); err != nil {
		sugar.Error("Failed to close producer: ", err)
	}
}

func waitForShutdown(stop chan<- struct{}) {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	sig := <-shutdown
	sugar.Infof("Received signal: %v. Shutting down...", sig)
	close(stop)
}
