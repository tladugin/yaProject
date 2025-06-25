package handler

import (
	"encoding/json"
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
func (s *Server) PostHandler(res http.ResponseWriter, req *http.Request) {

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "Invalid request format", http.StatusNotFound)
		return
	}

	switch parts[2] {
	case "gauge":
		partFloat, Error := strconv.ParseFloat(parts[4], 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddGauge(parts[3], partFloat)
		res.WriteHeader(http.StatusOK)
	case "counter":
		partInt, Error := strconv.ParseInt(parts[4], 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddCounter(parts[3], partInt)
		res.WriteHeader(http.StatusOK)

		//fmt.Println(s.storage.CounterSlice())
		/*default:
		http.Error(res, "Invalid request format", http.StatusBadRequest)
		return*/
	}

}
func (s *Server) GetHandler(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(res, "Invalid request format", http.StatusNotFound)
		return
	}
	switch parts[2] {
	case "gauge":
		for i, m := range s.storage.GaugeSlice() {
			if m.Name == parts[3] {
				res.Header().Set("Content-Type", "application/json")
				json.NewEncoder(res).Encode(s.storage.GaugeSlice()[i])
				res.WriteHeader(http.StatusOK)

			}
		}
	case "counter":
		for i, m := range s.storage.CounterSlice() {
			if m.Name == parts[3] {
				res.Header().Set("Content-Type", "application/json")
				json.NewEncoder(res).Encode(s.storage.GaugeSlice()[i])
				res.WriteHeader(http.StatusOK)

			}
		}
	default:
		http.Error(res, "Invalid request format", http.StatusBadRequest)

	}
}
