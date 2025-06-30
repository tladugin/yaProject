package handler

import (
	"fmt"
	"github.com/go-chi/chi/v5"
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

		//res.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(res).Encode(s.storage.GaugeSlice()[m])
		//res.WriteHeader(http.StatusOK)
	}

	for m := range s.storage.CounterSlice() {

		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.CounterSlice()[m].Name, s.storage.CounterSlice()[m].Value)

		//res.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(res).Encode(s.storage.CounterSlice()[m])
		//res.WriteHeader(http.StatusOK)

	}
	fmt.Fprint(res, "</ul></body></html>")
}

// функция принимает указатель на структуру Server, что позволяет обрашаться к storage
func (s *Server) PostHandler(res http.ResponseWriter, req *http.Request) {

	//params := chi.URLParam(req, "URL")
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
	}
	metric := chi.URLParam(req, "metric")
	//println(metric)
	name := chi.URLParam(req, "name")
	//println(name)
	value := chi.URLParam(req, "value")
	//println(value)

	switch metric {

	case "gauge":
		//if partFloat, err := strconv. ParseFloat(v, 64); err == nil { fmt. Printf("%T, %v\n", s, s)}
		partFloat, Error := strconv.ParseFloat(value, 64)
		//fmt.Printf("%f", partFloat)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddGauge(name, partFloat)
		//println(s.storage.GaugeSlice())
		//res.WriteHeader(http.StatusOK)

	case "counter":
		partInt, Error := strconv.ParseInt(value, 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddCounter(name, partInt)
		//res.WriteHeader(http.StatusOK)

		//fmt.Println(s.storage.CounterSlice())
	default:
		http.Error(res, "Invalid metric value", http.StatusBadRequest)
	}

}
func (s *Server) GetHandler(res http.ResponseWriter, req *http.Request) {

	metric := chi.URLParam(req, "metric")
	//println(metric)
	name := chi.URLParam(req, "name")
	//println(name)

	switch metric {
	case "gauge":
		getCheck := false
		for i, m := range s.storage.GaugeSlice() {
			if m.Name == name {
				getCheck = true
				//fmt.Fprintf(res, "%s %f", s.storage.GaugeSlice()[i].Name, s.storage.GaugeSlice()[i].Value)
				fmt.Fprint(res, s.storage.GaugeSlice()[i].Value)
				//fmt.Println(s.storage.GaugeSlice()[i].Value)
				//res.WriteHeader(http.StatusOK)

			}

		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	case "counter":
		getCheck := false
		for i, m := range s.storage.CounterSlice() {
			if m.Name == name {
				getCheck = true
				//println(m.Name)
				//println(m.Value)
				//res.Header().Set("Content-Type", "application/json")
				//json.NewEncoder(res).Encode(s.storage.GaugeSlice()[i])
				//fmt.Fprintf(res, "%s %d", s.storage.CounterSlice()[i].Name, s.storage.CounterSlice()[i].Value)
				fmt.Fprint(res, s.storage.CounterSlice()[i].Value)
				res.WriteHeader(http.StatusOK)

			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	default:
		http.Error(res, "Invalid metric value", http.StatusNotFound)

	}
}
