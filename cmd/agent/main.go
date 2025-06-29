package main

import (
	"fmt"
	"github.com/tladugin/yaProject.git/internal/repository"
	"io"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

const (
	//pollInterval = 2 * time.Second

	//serverURL   = "http://localhost:8080/update/"
	contentType = "Content-Type: text/plain"
)

func sendMetric(URL string, metricType string, storage *repository.MemStorage, i int) error {
	var sendAddr string
	if metricType == "gauge" {
		sendAddr = fmt.Sprintf("%s%s/%s/%f", URL, metricType, storage.GaugeSlice()[i].Name, storage.GaugeSlice()[i].Value)
		//println(url)
	}
	if metricType == "counter" {
		sendAddr = fmt.Sprintf("%s%s/%s/%d", URL, metricType, storage.CounterSlice()[i].Name, storage.CounterSlice()[i].Value)
		//println(url)
	}

	req, err := http.NewRequest("POST", sendAddr, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("metric send failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
func main() {
	parseFlags()
	serverURL := flagRunAddr
	//reportInterval, error := (strconv.ParseInt(reportInterval)) * time.Second
	pollIntervalTime, error := strconv.Atoi(pollIntervalTime)
	if error != nil {
		log.Fatal("Invalid value for pollInterval")
	}
	//println("Polling interval time: ", pollIntervalTime)
	if pollIntervalTime < 0 {
		log.Fatal("Invalid value for pollInterval")
	}

	pollInterval := time.Duration(pollIntervalTime) * time.Second
	reportIntervalTime, error := strconv.Atoi(reportIntervalTime)
	if error != nil {
		log.Fatal("Invalid value for reportInterval")
	}
	if reportIntervalTime < 0 {
		log.Fatal("Invalid value for reportInterval")
	}
	reportInterval := time.Duration(reportIntervalTime) * time.Second
	storage := repository.NewMemStorage()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ticker := time.NewTicker(pollInterval)
	pauseCh := make(chan struct{})
	resumeCh := make(chan struct{})

	go func() {
		for range ticker.C {
			// Обновляем метрики здесь
			fmt.Println("Updating metrics...")
			storage.AddGauge("Alloc", float64(m.Alloc))
			storage.AddGauge("BuckHashSys", float64(m.BuckHashSys))
			storage.AddGauge("Frees", float64(m.Frees))
			storage.AddGauge("GCCPUFraction", float64(m.GCCPUFraction))
			storage.AddGauge("GCSys", float64(m.GCSys))
			storage.AddGauge("HeapAlloc", float64(m.HeapAlloc))
			storage.AddGauge("HeapIdle", float64(m.HeapIdle))
			storage.AddGauge("HeapInuse", float64(m.HeapInuse))
			storage.AddGauge("HeapObjects", float64(m.HeapObjects))
			storage.AddGauge("HeapReleased", float64(m.HeapReleased))
			storage.AddGauge("HeapSys", float64(m.HeapSys))
			storage.AddGauge("LastGC", float64(m.LastGC))
			storage.AddGauge("Lookups", float64(m.Lookups))
			storage.AddGauge("MCacheInuse", float64(m.MCacheInuse))
			storage.AddGauge("MCacheSys", float64(m.MCacheSys))
			storage.AddGauge("MSpanInuse", float64(m.MSpanInuse))
			storage.AddGauge("MSpanSys", float64(m.MSpanSys))
			storage.AddGauge("Mallocs", float64(m.Mallocs))
			storage.AddGauge("NextGC", float64(m.NextGC))
			storage.AddGauge("NumForcedGC", float64(m.NumForcedGC))
			storage.AddGauge("NumGC", float64(m.NumGC))
			storage.AddGauge("OtherSys", float64(m.OtherSys))
			storage.AddGauge("PauseTotalNs", float64(m.PauseTotalNs))
			storage.AddGauge("StackInuse", float64(m.StackInuse))
			storage.AddGauge("StackSys", float64(m.StackSys))
			storage.AddGauge("Sys", float64(m.Sys))
			storage.AddGauge("TotalAlloc", float64(m.TotalAlloc))

			storage.AddGauge("RandomValue", rand.Float64())

			storage.AddCounter("PollCount", 1)

			// После завершения обновления, возобновляем отправку метрик на сервер
			resumeCh <- struct{}{}
		}
	}()

	t := time.NewTicker(reportInterval)
	for range t.C {
		select {
		case <-resumeCh:
			// Если канал resumeCh получил значение, значит метрики были обновлены,
			// и мы можем отправлять их на сервер
			fmt.Println("Sending metrics...")
			for i := range storage.GaugeSlice() {
				err := sendMetric(serverURL+"/update/", "gauge", storage, i)
				if err != nil {
					fmt.Println(storage.GaugeSlice()[i])
					fmt.Println("Error sending metric:", err)

				}

			}
			for i := range storage.CounterSlice() {
				err := sendMetric(serverURL+"/update/", "counter", storage, i)
				if err != nil {
					fmt.Println(storage.CounterSlice()[i])
					fmt.Println("Error sending metric:", err)

				}

			} // отправляем метрики на сервер
		case <-pauseCh:
			// Если канал pauseCh получил значение, ставим отправку метрик на паузу
			fmt.Println("Pausing metric sending...")
			t.Stop()
			time.Sleep(pollInterval)
			t.Reset(reportInterval) // возобновляем отправку метрик на сервер после завершения обновления
		}
	}

	// Для остановки горутины, которая отвечает за обновление метрик,
	// мы можем закрыть канал pauseCh и дождаться ее окончания
	close(pauseCh)
	<-ticker.C
}
