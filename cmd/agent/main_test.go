package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_sendMetric(t *testing.T) {

	gaugeTest := *repository.NewMemStorage()
	gaugeTest.AddGauge("Alloc", 123.45)
	//gaugeTest.AddGauge("heap", 678.90)
	counterTest := *repository.NewMemStorage()
	counterTest.AddCounter("PollCount", 42)
	//counterTest.AddCounter("misses", 5)
	//bothTest := *repository.NewMemStorage()
	//bothTest.AddGauge("alloc", 123.45)
	//bothTest.AddCounter("hits", 100)
	tests := []struct {
		name        string
		metricType  string
		storage     *repository.MemStorage
		index       int
		setupServer func(r chi.Router) // Настройка роутера chi
		wantURL     string
		wantErr     bool
		wantErrText string
	}{
		{
			name:       "Success gauge metric",
			metricType: "gauge",
			storage:    &gaugeTest,
			index:      0,
			setupServer: func(r chi.Router) {
				r.Post("/update/gauge/{name}/{value}", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "name")
					value := chi.URLParam(r, "value")
					if name != "Alloc" || value != "123.450000" {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusOK)
				})
			},
			wantURL: "/update/gauge/Alloc/123.450000",
		},
		{
			name:       "Success counter metric",
			metricType: "counter",
			storage:    &counterTest,
			index:      0,
			setupServer: func(r chi.Router) {
				r.Post("/update/counter/{name}/{value}", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "name")
					value := chi.URLParam(r, "value")
					if name != "PollCount" || value != "42" {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusOK)
				})
			},
			wantURL: "/update/counter/PollCount/42",
		},
		{
			name:       "Server returns error",
			metricType: "gauge",
			storage:    &gaugeTest,
			index:      0,
			setupServer: func(r chi.Router) {
				r.Post("/update/gauge/{name}/{value}", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, "invalid metric")
				})
			},
			wantURL:     "/update/gauge/Alloc/123.45",
			wantErr:     true,
			wantErrText: "metric send failed with status 400: invalid metric",
		},
		{
			name:       "Invalid metric type",
			metricType: "invalid",
			storage:    &repository.MemStorage{},
			index:      0,
			setupServer: func(r chi.Router) {
				// Ничего не настраиваем - должен вернуться 404
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем chi роутер
			r := chi.NewRouter()
			tt.setupServer(r)

			// Запускаем тестовый сервер
			ts := httptest.NewServer(r)
			defer ts.Close()

			// Вызываем тестируемую функцию
			err := sendMetric(ts.URL+"/update/", tt.metricType, tt.storage, tt.index)
			println(ts.URL + "/update/" + tt.metricType)

			// Проверяем ошибки
			if (err != nil) != tt.wantErr {
				//println(tt.)
				t.Errorf("sendMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErrText != "" && (err == nil || err.Error() != tt.wantErrText) {
				t.Errorf("got error %v, want %s", err, tt.wantErrText)
			}
		})
	}
}
