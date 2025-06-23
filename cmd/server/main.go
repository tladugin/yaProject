package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type gauge struct {
	Name  string
	Value float64
}
type counter struct {
	Name  string
	Value int64
}
type MemStorage struct {
	counterSlice []counter
	gaugeSlice   []gauge
}

var globalMemStorage MemStorage

func (s *MemStorage) addGauge(name string, value float64) {
	for i, m := range s.gaugeSlice {
		if m.Name == name {
			s.gaugeSlice[i].Value = value
			return
		}
	}

	s.gaugeSlice = append(s.gaugeSlice, gauge{Name: name, Value: value})
}
func (s *MemStorage) addCounter(name string, value int64) {
	for i, m := range s.counterSlice {
		if m.Name == name {
			s.counterSlice[i].Value += value
			return
		}
	}

	s.counterSlice = append(s.counterSlice, counter{Name: name, Value: value})
}

func handler(res http.ResponseWriter, req *http.Request) {

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

		globalMemStorage.addGauge(parts[3], partFloat)
		res.WriteHeader(http.StatusOK)

		//fmt.Printf("Gauge metrics=================== \n")
		output := "Gauge metrics:\r\n"
		for _, element := range globalMemStorage.gaugeSlice {
			output += fmt.Sprintf("Name: %s, Value: %.1f\n", element.Name, element.Value)
		}
		fmt.Println([]byte(output))

	} else if parts[2] == "counter" {
		partInt, Error := strconv.ParseInt(parts[4], 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		globalMemStorage.addCounter(parts[3], partInt)
		res.WriteHeader(http.StatusOK)
		//fmt.Printf("Counter metrics=================== \n")
		output := "Counter metrics:\r\n"
		for _, element := range globalMemStorage.counterSlice {

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
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, handler)

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}

}
