package main

import (
	"github.com/tladugin/yaProject.git/internal/repository"
	"testing"
)

func Test_sendMetric(t *testing.T) {
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
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{ // TODO: Add test cases.
		{
			name:    "gauge_values",
			args:    args{"gauge", testGaugeStorage, i},
			wantErr: false,
		},
		{
			name:    "counter_values",
			args:    args{"counter", testCounterStorage, i},
			wantErr: false,
		},
		{
			name:    "wrong_metric_type",
			args:    args{"unknown", testGaugeStorage, i},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		for i := 0; i < 4; i++ {
			t.Run(tt.name, func(t *testing.T) {
				if err := sendMetric(tt.args.metricType, tt.args.storage, i); (err != nil) != tt.wantErr {
					t.Errorf("sendMetric() error = %v, wantErr %v", err, tt.wantErr)
				}
			})

		}
	}
}
