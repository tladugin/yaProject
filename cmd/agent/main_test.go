package main

import (
	"encoding/json"
	models "github.com/tladugin/yaProject.git/internal/model"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_sendMetric(t *testing.T) { // Создаем тестовое хранилище
	testGauge := repository.NewMemStorage()
	testGauge.AddGauge("Alloc", 123.45)
	testCounter := repository.NewMemStorage()
	testCounter.AddCounter("PollCounter", 42)
	/*{
		GaugeSliceVal: []repository.Gauge{
			{Name: "testGauge", Value: 123.45},
		},
		CounterSliceVal: []repository.Counter{
			{Name: "testCounter", Value: 42},
		},
	}

	*/

	tests := []struct {
		name        string
		URL         string
		metricType  string
		storage     *repository.MemStorage
		i           int
		handler     http.HandlerFunc
		wantErr     bool
		expectedErr string
	}{
		{
			name:       "successful gauge send",
			URL:        "http://example.com",
			metricType: "gauge",
			storage:    testGauge,
			i:          0,
			handler: func(w http.ResponseWriter, r *http.Request) {
				var metric models.Metrics
				if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
					t.Errorf("failed to decode request: %v", err)
				}
				if metric.ID != "Alloc" || metric.MType != "gauge" || *metric.Value != 123.45 {
					t.Errorf("unexpected metric data")
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:       "successful counter send",
			URL:        "http://example.com",
			metricType: "counter",
			storage:    testCounter,
			i:          0,
			handler: func(w http.ResponseWriter, r *http.Request) {
				var metric models.Metrics
				if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
					t.Errorf("failed to decode request: %v", err)
				}
				if metric.ID != "PollCounter" || metric.MType != "counter" || *metric.Delta != 42 {
					t.Errorf("unexpected metric data")
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:       "invalid metric type",
			URL:        "http://example.com",
			metricType: "invalid",
			storage:    testGauge,
			i:          0,
			handler:    nil,
			wantErr:    true,
		},
		{
			name:       "server error response",
			URL:        "http://example.com",
			metricType: "gauge",
			storage:    testGauge,
			i:          0,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("server error"))
			},
			wantErr:     true,
			expectedErr: "metric send failed with status 500: server error",
		},
		{
			name:       "invalid URL",
			URL:        "invalid-url",
			metricType: "gauge",
			storage:    testGauge,
			i:          0,
			handler:    nil,
			wantErr:    true,
		},
		{
			name:       "URL without scheme",
			URL:        "example.com",
			metricType: "gauge",
			storage:    testGauge,
			i:          0,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.handler != nil {
				ts = httptest.NewServer(tt.handler)
				defer ts.Close()

				// Если тест требует тестового сервера, подменяем URL
				if strings.HasPrefix(tt.URL, "http://") {
					tt.URL = ts.URL
				}
			}

			err := sendMetric(tt.URL, tt.metricType, tt.storage, tt.i)

			if (err != nil) != tt.wantErr {
				t.Errorf("sendMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedErr != "" && err.Error() != tt.expectedErr {
				t.Errorf("sendMetric() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
