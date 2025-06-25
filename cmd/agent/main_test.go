package main

import (
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createTestServer() *httptest.Server {
	storage := repository.NewMemStorage()
	s := handler.NewServer(storage)
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, s.PostHandler)

	return httptest.NewServer(mux)
}
func Test_sendMetric(t *testing.T) {

	server := createTestServer()
	defer server.Close()

	testServerURL := server.URL
	testServerURL += "/update/"

	testGaugeStorage := new(repository.MemStorage)

	testGaugeStorage.AddGauge("Alloc", 100.1111)
	testGaugeStorage.AddGauge("TotalAlloc", 101.1111)
	testGaugeStorage.AddGauge("Free", 102.1111)
	testGaugeStorage.AddGauge("Used", 103.1111)

	testCounterStorage := new(repository.MemStorage)

	testCounterStorage.AddCounter("Alloc", 12)
	testCounterStorage.AddCounter("TotalAlloc", 44)
	testCounterStorage.AddCounter("Free", 13)
	testCounterStorage.AddCounter("Used", 25)

	var i int

	type args struct {
		metricType string
		storage    *repository.MemStorage
		i          int
		url        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{ // TODO: Add test cases.
		{
			name:    "gauge_values",
			args:    args{"gauge", testGaugeStorage, i, testServerURL},
			wantErr: false,
		},
		{
			name:    "counter_values",
			args:    args{"counter", testCounterStorage, i, testServerURL},
			wantErr: false,
		},
		{
			name:    "wrong_metric_type",
			args:    args{"unknown", testGaugeStorage, i, testServerURL},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		for i := 0; i < 4; i++ {
			t.Run(tt.name, func(t *testing.T) {
				if err := sendMetric(testServerURL, tt.args.metricType, tt.args.storage, i); (err != nil) != tt.wantErr {
					t.Errorf("sendMetric() error = %v, wantErr %v", err, tt.wantErr)

				}
			})

		}
	}
}
