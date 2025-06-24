package main

import (
	"fmt"
	"github.com/tladugin/yaProject.git/internal/repository"
	"io/ioutil"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	serverURL      = "http://localhost:8080/update/"
	contentType    = "Content-Type: text/plain"
)

func sendMetric(metricType string, storage *repository.MemStorage, i int) error {

	url := fmt.Sprintf("%s%s/%s/%f", serverURL, metricType, storage.GaugeSlice()[i].Name, storage.GaugeSlice()[i].Value)
	req, err := http.NewRequest("GET", url, nil)
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
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("metric send failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
func main() {
	storage := repository.NewMemStorage()

	for i, _ := range storage.GaugeSlice() {
		err := sendMetric("gauge", storage, i)
		if err != nil {
			fmt.Println("Error sending metric:", err)
		}

	}
	for i, _ := range storage.CounterSlice() {
		err := sendMetric("counter", storage, i)
		if err != nil {
			fmt.Println("Error sending metric:", err)
		}
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			storage.AddGauge("Alloc:", float64(m.Alloc))
			storage.AddGauge("BuckHashSys:", float64(m.BuckHashSys))
			storage.AddGauge("Frees:", float64(m.Frees))
			storage.AddGauge("GCCPUFraction:", float64(m.GCCPUFraction))
			storage.AddGauge("GCSys:", float64(m.GCSys))
			storage.AddGauge("HeapAlloc:", float64(m.HeapAlloc))
			storage.AddGauge("HeapIdle:", float64(m.HeapIdle))
			storage.AddGauge("HeapInuse:", float64(m.HeapInuse))
			storage.AddGauge("HeapObjects:", float64(m.HeapObjects))
			storage.AddGauge("HeapReleased:", float64(m.HeapReleased))
			storage.AddGauge("HeapSys:", float64(m.HeapSys))
			storage.AddGauge("LastGC:", float64(m.LastGC))
			storage.AddGauge("Lookups:", float64(m.Lookups))
			storage.AddGauge("MCacheInuse:", float64(m.MCacheInuse))
			storage.AddGauge("MCacheSys:", float64(m.MCacheSys))
			storage.AddGauge("MSpanInuse:", float64(m.MSpanInuse))
			storage.AddGauge("MSpanSys:", float64(m.MSpanSys))
			storage.AddGauge("Mallocs:", float64(m.Mallocs))
			storage.AddGauge("NextGC:", float64(m.NextGC))
			storage.AddGauge("NumForcedGC:", float64(m.NumForcedGC))
			storage.AddGauge("NumGC:", float64(m.NumGC))
			storage.AddGauge("OtherSys:", float64(m.OtherSys))
			storage.AddGauge("PauseTotalNs:", float64(m.PauseTotalNs))
			storage.AddGauge("StackInuse:", float64(m.StackInuse))
			storage.AddGauge("StackSys:", float64(m.StackSys))
			storage.AddGauge("Sys:", float64(m.Sys))
			storage.AddGauge("TotalAlloc:", float64(m.TotalAlloc))

			storage.AddGauge("RandomValue:", rand.Float64())

			storage.AddCounter("PollCount:", 1)

			//fmt.Println(storage.CounterSlice())
			time.Sleep(pollInterval)
		}

	}
}
