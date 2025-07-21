package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type gauge struct {
	Name  string
	Value float64
}

func TestServer_GetHandler(t *testing.T) {

	gaugeTest := *repository.NewMemStorage()
	gaugeTest.AddGauge("test_gauge", 123.45)
	//gaugeTest.AddGauge("heap", 678.90)
	counterTest := *repository.NewMemStorage()
	counterTest.AddCounter("test_counter", 100)
	//counterTest.AddCounter("misses", 5)
	//bothTest := *repository.NewMemStorage()
	//bothTest.AddGauge("alloc", 123.45)
	//bothTest.AddCounter("hits", 100)
	tests := []struct {
		name       string
		url        string
		storage    *repository.MemStorage
		wantStatus int
		wantBody   string
	}{
		{
			name:       "Get existing gauge",
			url:        "/value/gauge/test_gauge",
			storage:    &gaugeTest,
			wantStatus: http.StatusOK,
			wantBody:   "123.45",
		},
		{
			name:       "Get existing counter",
			url:        "/value/counter/test_counter",
			storage:    &counterTest,
			wantStatus: http.StatusOK,
			wantBody:   "100",
		},

		/*
			{
					name:       "Get non-existent gauge",
					url:        "/value/gauge/unknown",
					storage:    &gaugeTest,
					wantStatus: http.StatusNotFound,
					wantBody:   "No metric found",
				},
			{
				name:       "Get non-existent counter",
				url:        "/value/counter/unknown",
				storage:    &counterTest,
				wantStatus: http.StatusNotFound,
				wantBody:   "No metric found",
			},
			{
				name:       "Invalid metric type",
				url:        "/value/invalid/test",
				storage:    &gaugeTest,
				wantStatus: http.StatusNotFound,
				wantBody:   "Invalid metric value",
			},

		*/
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем сервер с mock хранилищем
			s := &Server{
				storage: tt.storage,
			}

			// Создаем запрос
			req := httptest.NewRequest("GET", tt.url, nil)

			// Создаем роутер chi и добавляем параметры маршрута
			r := chi.NewRouter()
			r.Get("/value/{metric}/{name}", s.GetHandler)

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Вызываем обработчик через роутер
			r.ServeHTTP(rr, req)

			// Проверяем статус код
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			// Проверяем тело ответа
			body := rr.Body.String()
			if tt.wantBody != "" && body != tt.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					body, tt.wantBody)
			}
		})
	}
}

func TestServer_MainPage(t *testing.T) {

	emptyTest := *repository.NewMemStorage()
	gaugeTest := *repository.NewMemStorage()
	gaugeTest.AddGauge("alloc", 123.45)
	gaugeTest.AddGauge("heap", 678.90)
	counterTest := *repository.NewMemStorage()
	counterTest.AddCounter("hits", 100)
	counterTest.AddCounter("misses", 5)
	bothTest := *repository.NewMemStorage()
	bothTest.AddGauge("alloc", 123.45)
	bothTest.AddCounter("hits", 100)

	//s.AddGauge()
	//s := &repository.MemStorage{}

	tests := []struct {
		name            string
		storage         *repository.MemStorage
		wantContains    []string // строки, которые должны быть в ответе
		wantNotContains []string // строки, которых не должно быть
		wantStatus      int
	}{
		{
			name:            "Empty storage",
			storage:         &emptyTest,
			wantContains:    []string{"<html>", "<body>", "<ul>", "</ul>"},
			wantNotContains: []string{"<li>"},
			wantStatus:      http.StatusOK,
		},
		{
			name:    "With gauge metrics",
			storage: &gaugeTest,
			wantContains: []string{
				"<li>alloc: 123.45</li>",
				"<li>heap: 678.9</li>",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "With counter metrics",
			storage: &counterTest,
			wantContains: []string{
				"<li>hits: 100</li>",
				"<li>misses: 5</li>",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "With both types of metrics",
			storage: &bothTest,
			wantContains: []string{
				"<li>alloc: 123.45</li>",
				"<li>hits: 100</li>",
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем сервер с mock хранилищем
			s := &Server{
				storage: tt.storage,
			}

			// Создаем запрос и рекордер
			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()

			// Вызываем тестируемый метод
			s.MainPage(rr, req)

			// Проверяем статус код
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			// Проверяем Content-Type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "text/html; charset=utf-8" {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, "text/html; charset=utf-8")
			}

			// Получаем тело ответа
			body := rr.Body.String()

			// Проверяем наличие обязательных строк
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response body does not contain expected string: %q", want)
				}
			}

			// Проверяем отсутствие нежелательных строк
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(body, notWant) {
					t.Errorf("response body contains unexpected string: %q", notWant)
				}
			}

			// Проверяем валидность HTML (базовая проверка)
			if !strings.HasPrefix(body, "<html>") || !strings.HasSuffix(body, "</html>") {
				t.Error("response is not valid HTML")
			}
		})
	}
}

func TestServer_PostHandler(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantBody   string
		storage    *repository.MemStorage
	}{
		{
			name:       "Valid gauge update",
			url:        "/update/gauge/test_metric/123.45",
			wantStatus: http.StatusOK,
			storage:    &repository.MemStorage{},
		},
		{
			name:       "Valid counter update",
			url:        "/update/counter/test_counter/100",
			wantStatus: http.StatusOK,
			storage:    &repository.MemStorage{},
		},
		{
			name:       "Invalid URL format",
			url:        "/update/invalid",
			wantStatus: http.StatusNotFound,
			wantBody:   "404 page not found",
			storage:    &repository.MemStorage{},
		},
		{
			name:       "Invalid metric type",
			url:        "/update/invalid_type/test/123",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid metric value",
			storage:    &repository.MemStorage{},
		},
		{
			name:       "Invalid gauge value",
			url:        "/update/gauge/test/invalid_value",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid metric value",
			storage:    &repository.MemStorage{},
		},
		{
			name:       "Invalid counter value",
			url:        "/update/counter/test/invalid_value",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid metric value",
			storage:    &repository.MemStorage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем сервер с mock хранилищем
			s := &Server{
				storage: tt.storage,
			}

			// Создаем запрос
			req := httptest.NewRequest("POST", tt.url, nil)

			// Создаем роутер chi и добавляем параметры маршрута
			r := chi.NewRouter()
			r.Post("/update/{metric}/{name}/{value}", s.PostHandler)

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Вызываем обработчик через роутер
			r.ServeHTTP(rr, req)

			// Проверяем статус код
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			// Проверяем тело ответа для ошибок
			if tt.wantBody != "" {
				if !strings.Contains(rr.Body.String(), tt.wantBody) {
					t.Errorf("handler returned unexpected body: got %v want %v",
						rr.Body.String(), tt.wantBody)
				}
			}

			// Для успешных запросов проверяем обновление хранилища
			if tt.wantStatus == http.StatusOK {
				parts := strings.Split(tt.url, "/")
				metricType := parts[2]
				metricName := parts[3]
				metricValue := parts[4]

				switch metricType {
				case "gauge":
					val, _ := strconv.ParseFloat(metricValue, 64)
					found := false
					for _, m := range tt.storage.GaugeSlice() {
						if m.Name == metricName && m.Value == val {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("gauge metric was not updated in storage")
					}
				case "counter":
					val, _ := strconv.ParseInt(metricValue, 0, 64)
					found := false
					for _, m := range tt.storage.CounterSlice() {
						if m.Name == metricName && m.Value == val {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("counter metric was not updated in storage")
					}
				}
			}
		})
	}
}
