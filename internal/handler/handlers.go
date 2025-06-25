package handler

import (
	"encoding/json"
	"fmt"
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

func (s *Server) MainPage(res http.ResponseWriter, req *http.Request) {
	fmt.Fprint(res, "<html><body><ul>")
	for m := range s.storage.GaugeSlice() {

		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.GaugeSlice()[m].Name, s.storage.GaugeSlice()[m].Value)
		/*
			res.Header().Set("Content-Type", "application/json")
			json.NewEncoder(res).Encode(s.storage.GaugeSlice()[m])
			res.WriteHeader(http.StatusOK)

		*/
	}

	for m := range s.storage.CounterSlice() {

		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.CounterSlice()[m].Name, s.storage.CounterSlice()[m].Value)
		/*
			res.Header().Set("Content-Type", "application/json")
			json.NewEncoder(res).Encode(s.storage.CounterSlice()[m])
			res.WriteHeader(http.StatusOK)

		*/

	}
	fmt.Fprint(res, "</ul></body></html>")

}

// функция принимает указатель на структуру Server, что позволяет обрашаться к storage
func (s *Server) PostHandler(res http.ResponseWriter, req *http.Request) {

	if req.Method != "POST" {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
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
	default:
		http.Error(res, "Invalid request format", http.StatusBadRequest)
		return
	}

}
func (s *Server) GetHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(res, "Invalid request format", http.StatusBadRequest)
	}
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(res, "Invalid request format", http.StatusNotFound)
		return
	}
	switch parts[2] {
	case "gauge":
		getCheck := false
		for i, m := range s.storage.GaugeSlice() {
			if m.Name == parts[3] {
				getCheck = true
				res.Header().Set("Content-Type", "application/json")
				json.NewEncoder(res).Encode(s.storage.GaugeSlice()[i])
				res.WriteHeader(http.StatusOK)

			}

		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	case "counter":
		getCheck := false
		for i, m := range s.storage.CounterSlice() {
			if m.Name == parts[3] {
				getCheck = true
				res.Header().Set("Content-Type", "application/json")
				json.NewEncoder(res).Encode(s.storage.GaugeSlice()[i])
				res.WriteHeader(http.StatusOK)

			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	default:
		http.Error(res, "Invalid request format", http.StatusBadRequest)
		return

	}
}
