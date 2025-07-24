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
	"syscall"
	"time"
)

var sugar zap.SugaredLogger

func main() {
	parseFlags()

	log, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer func(log *zap.Logger) {
		err = log.Sync()
		if err != nil {
			panic(err)
		}
	}(log)

	sugar = *log.Sugar()

	//sugar.Infoln(flagFileStoragePath)
	//sugar.Infoln(flagStoreInterval)
	storage := repository.NewMemStorage()

	if flagRestore {
		consumer, err := handler.NewConsumer(flagFileStoragePath)
		if err != nil {
			return
		}

		event, err := consumer.ReadEvent()
		if err != nil {
			return
		}
		for event != nil {

			if event.MType == "gauge" {
				storage.AddGauge(event.ID, *event.Value)
			} else if event.MType == "counter" {
				storage.AddCounter(event.ID, *event.Delta)
			}

			event, err = consumer.ReadEvent()
			if err != nil {
				sugar.Infoln("Backup restored")
			}

		}
		for i := range storage.GaugeSlice() {
			println(storage.GaugeSlice()[i].Name, storage.GaugeSlice()[i].Value)
		}
		for i := range storage.CounterSlice() {
			println(storage.CounterSlice()[i].Name, storage.CounterSlice()[i].Value)
		}
	}

	storeInterval, err := time.ParseDuration(flagStoreInterval + "s")
	if err != nil {
		sugar.Fatal("wrong store_interval value", err)
	}

	//sugar.Infoln(storeInterval)
	if err != nil {
		sugar.Fatal("Invalid flagStoreInterval:", err)
	}
	stopProgram := make(chan struct{})

	producer, err := handler.NewProducer(flagFileStoragePath)
	if err != nil {
		sugar.Fatal("could not open backup file", err)
	}

	if flagStoreInterval != "0" {

		go func() {
			for {
				select {
				default:
					storeTicker := time.NewTicker(storeInterval)
					defer storeTicker.Stop()

					<-storeTicker.C
					/*if _, err = os.Stat(flagFileStoragePath); !os.IsNotExist(err) {

						//fmt.Println(!os.IsNotExist(err))
						fmt.Println("Файл существует")
						err = producer.Close()
						if err != nil {
							fmt.Println(err)
						}
						err = os.Remove(flagFileStoragePath)
						if err != nil {
							sugar.Fatal(err)
						} else {
							fmt.Println("Файл удален")
						}
						producer, err = handler.NewProducer(flagFileStoragePath)
						if err != nil {
							sugar.Fatal("could not open backup file", err)
						}
						fmt.Println("Новый бэкап файл создан")

					}


					*/
					for i := range storage.GaugeSlice() {

						backup := models.Metrics{}
						backup.ID = storage.GaugeSlice()[i].Name
						backup.MType = "gauge"
						backup.Value = &storage.GaugeSlice()[i].Value
						//sugar.Infoln(backup.ID)
						err := producer.WriteEvent(&backup)
						if err != nil {
							return
						}

					}
					for i := range storage.CounterSlice() {

						backup := models.Metrics{}
						backup.ID = storage.CounterSlice()[i].Name
						backup.MType = "counter"
						backup.Delta = &storage.CounterSlice()[i].Value
						//sugar.Infoln(backup.ID)
						err := producer.WriteEvent(&backup)
						if err != nil {
							return
						}
					}
				case <-stopProgram:
					return
				}
			}

		}()
	}
	go func() {

		<-stopProgram
		sugar.Info("Exiting program backup")
		/*if _, err = os.Stat(flagFileStoragePath); !os.IsNotExist(err) {
			err = producer.Close()
			if err != nil {
				return
			}

			err = os.Remove(flagFileStoragePath)
			if err != nil {
				sugar.Fatal(err)
			}
		}

		producer, err := handler.NewProducer(flagFileStoragePath)
		if err != nil {
			sugar.Fatal("could not open backup file", err)
		}

		*/
		for i := range storage.GaugeSlice() {

			backup := models.Metrics{}
			backup.ID = storage.GaugeSlice()[i].Name
			backup.MType = "gauge"
			backup.Value = &storage.GaugeSlice()[i].Value
			//sugar.Infoln(backup.ID)
			err := producer.WriteEvent(&backup)
			if err != nil {
				sugar.Errorw("Error writing event", "error", err)
			}

		}
		for i := range storage.CounterSlice() {

			backup := models.Metrics{}
			backup.ID = storage.CounterSlice()[i].Name
			backup.MType = "counter"
			backup.Delta = &storage.CounterSlice()[i].Value
			//sugar.Infoln(backup.ID)
			err = producer.WriteEvent(&backup)
			if err != nil {
				sugar.Errorw("Error writing event", "error", err)
			}
		}
		sugar.Info("Backup finished")
		err = producer.Close()
		if err != nil {
			sugar.Errorw("Error closing producer", "error", err)
		}

	}()
	/*if flagStoreInterval == "0" {
		file, err := os.Open(flagFileStoragePath)
		if err != nil {
			sugar.Fatal(err)
		}
		lineCount := 0
		go func() {
			for {

				scanner := bufio.NewScanner(file)

				for scanner.Scan() {
					lineCount++
				}

				if err = scanner.Err(); err != nil {
					sugar.Fatal(err)
				}
				//fmt.Println("lineCount:", lineCount)
				if lineCount > 29 {
					err = file.Close()
					if err != nil {
						return
					}
					err = producer.Close()
					if err != nil {
						fmt.Println(err)
					}
					err = os.Remove(flagFileStoragePath)
					if err != nil {
						sugar.Fatal(err)
					}
					fmt.Println("Файл удален (sync mode)")
					producer, err = handler.NewProducer(flagFileStoragePath)
					if err != nil {
						sugar.Fatal("could not open backup file", err)
					}
					fmt.Println("Файл создан (sync mode)")
					lineCount = 0

					file, err = os.Open(flagFileStoragePath)
					if err != nil {
						sugar.Fatal(err)
					}
				}

			}

		}()
	}

	*/
	go func() {

		select {
		default:
			s := handler.NewServer(storage)
			sSync := handler.NewServerSync(storage, producer)

			r := chi.NewRouter()
			r.Route("/", func(r chi.Router) {

				r.Get("/", logger.LoggingAnswer(gzipMiddleware(s.MainPage), sugar))
				r.Get("/value/{metric}/{name}", logger.LoggingAnswer(s.GetHandler, sugar))
				r.Post("/update/{metric}/{name}/{value}", logger.LoggingRequest(s.PostHandler, sugar))
				if flagStoreInterval == "0" { //Sync backup mode
					sugar.Infoln("Sync backup mode")
					r.Post("/update", logger.LoggingRequest(gzipMiddleware(sSync.PostUpdateSyncBackup), sugar))
					r.Post("/update/", logger.LoggingRequest(gzipMiddleware(sSync.PostUpdateSyncBackup), sugar))
				} else {
					sugar.Infoln("aSync backup mode")
					r.Post("/update", logger.LoggingRequest(gzipMiddleware(s.PostUpdate), sugar))
					r.Post("/update/", logger.LoggingRequest(gzipMiddleware(s.PostUpdate), sugar))
				}
				r.Post("/value", logger.LoggingRequest(gzipMiddleware(s.PostValue), sugar))
				r.Post("/value/", logger.LoggingRequest(gzipMiddleware(s.PostValue), sugar))
			})
			sugar.Infoln("Starting server on :", flagRunAddr)
			if err := http.ListenAndServe(flagRunAddr, r); err != nil {
				sugar.Errorln("Server failed: %v\n", err)
			}
		case <-stopProgram:

		}

	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	close(stopProgram)
	sugar.Infoln("Shutting down...")

}

/*

 */
