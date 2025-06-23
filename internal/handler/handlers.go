package handler

import (
	"fmt"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"strconv"
	"strings"
)

func Handler(res http.ResponseWriter, req *http.Request) {

	memstorage := repository.NewMemStorage()

	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "Invalid request format", http.StatusNotFound)
		return
	}

	if parts[2] != "counter" && parts[2] != "gauge" {
		http.Error(res, "Invalid request format", http.StatusBadRequest)
		return
	}

	if parts[2] == "gauge" {
		//MemStorage := &MemStorage{counterSlice: make([]counter, 0), gaugeSlice: make([]gauge, 0)}

		partFloat, Error := strconv.ParseFloat(parts[4], 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		memstorage.AddGauge(parts[3], partFloat)
		res.WriteHeader(http.StatusOK)

		//fmt.Printf("Gauge metrics=================== \n")
		output := "Gauge metrics:\r\n"
		for _, element := range memstorage.GaugeSlice() {
			output += fmt.Sprintf("Name: %s, Value: %.1f\n", element.Name, element.Value)
		}
		fmt.Println([]byte(output))

	} else if parts[2] == "counter" {
		partInt, Error := strconv.ParseInt(parts[4], 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		memstorage.AddCounter(parts[3], partInt)
		res.WriteHeader(http.StatusOK)
		//fmt.Printf("Counter metrics=================== \n")
		output := "Counter metrics:\r\n"
		for _, element := range memstorage.CounterSlice() {

			output += fmt.Sprintf("Name: %s, Value: %d\n", element.Name, element.Value)
		}
		fmt.Println([]byte(output))
	}
	/*else if parts[3] == "" {
		http.Error(res, "Invalid request path", http.StatusNotFound)
	} else if parts[2] != "gauge" && parts[2] != "counter" {
		http.Error(res, "Invalid metric type", http.StatusBadRequest)
	} else {
		http.Error(res, "Metrics received!", http.StatusOK)
	}*/
}
