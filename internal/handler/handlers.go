package handler

import (
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"strconv"
	"strings"
)

// создаем функцию NewServer которая возвращает указать на нашу структуру Server, которая содержит storage
func NewServer(s *repository.MemStorage) *Server {
	return &Server{
		storage: s,
	}

}

type Server struct {
	storage *repository.MemStorage
}

// функция принимает указатель на структуру Server, что позволяет обрашаться к storage
func (s *Server) Handler(res http.ResponseWriter, req *http.Request) {

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

		s.storage.AddGauge(parts[3], partFloat)
		res.WriteHeader(http.StatusOK)

		//fmt.Println(s.storage.GaugeSlice())

	} else if parts[2] == "counter" {
		partInt, Error := strconv.ParseInt(parts[4], 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddCounter(parts[3], partInt)
		res.WriteHeader(http.StatusOK)

		//fmt.Println(s.storage.CounterSlice())

	}

}
