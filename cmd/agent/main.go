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
	"strings"
	"time"
)

const (
	contentType = "Content-Type: text/plain"
)

func sendMetric(URL string, metricType string, storage *repository.MemStorage, i int) error {

	var sendAddr string
	var req *http.Request
	var err error

	if metricType == "gauge" {
		sendAddr = fmt.Sprintf("%s%s/%s/%f", URL, metricType, storage.GaugeSlice()[i].Name, storage.GaugeSlice()[i].Value)

	}
	if metricType == "counter" {
		sendAddr = fmt.Sprintf("%s%s/%s/%d", URL, metricType, storage.CounterSlice()[i].Name, storage.CounterSlice()[i].Value)
		//println(url)
	}
	//println("sendAddr:", sendAddr)
	//println("URL:", URL)

	if strings.HasPrefix(URL, "http://") {
		req, err = http.NewRequest("POST", sendAddr, nil)
		if err != nil {
			return err
		}
	} else {
		req, err = http.NewRequest("POST", "http://"+sendAddr, nil)
		if err != nil {
			return err
		}
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

	var err error
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	parseFlags()
	serverURL := flagRunAddr

	pollIntervalTime, error := strconv.Atoi(pollIntervalTime)
	if error != nil {
		log.Fatal("Invalid value for pollInterval")
	}

	if pollIntervalTime < 0 {
		log.Fatal("Invalid value for pollInterval")
	}

	reportIntervalTime, error := strconv.Atoi(reportIntervalTime)
	if error != nil {
		log.Fatal("Invalid value for reportInterval")
	}
	if reportIntervalTime < 0 {
		log.Fatal("Invalid value for reportInterval")
	}

	storage := repository.NewMemStorage()

	pollCounter := 0
	reportCounter := 0

	for {
		time.Sleep(1 * time.Second)
		pollCounter += 1   // счетчик секунд для обновления метрик
		reportCounter += 1 // счетчик секунд для отправки метрик
		if pollIntervalTime == pollCounter {
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

			//println(storage.CounterSlice()[0].Value)

			pollCounter = 0
		}

		if reportIntervalTime == reportCounter {
			fmt.Println("Sending metrics...")
			for i := range storage.GaugeSlice() {
				err = sendMetric(serverURL+"/update/", "gauge", storage, i)
				if err != nil {
					fmt.Println(storage.GaugeSlice()[i])
					fmt.Println("Error sending metric:", err)
				}

			}
			for i := range storage.CounterSlice() {
				err = sendMetric(serverURL+"/update/", "counter", storage, i)
				if err != nil {
					fmt.Println(storage.CounterSlice()[i])
					fmt.Println("Error sending metric:", err)
				}
			}
			if err == nil {
				storage.CounterSlice()[0].Value = 0
				reportCounter = 0
			}
		}
	}
}
