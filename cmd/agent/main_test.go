package main

import (
	"github.com/tladugin/yaProject.git/internal/repository"
	"testing"
)

func Test_sendMetric(t *testing.T) {
	type args struct {
		metricType string
		storage    *repository.MemStorage
		i          int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sendMetric(tt.args.metricType, tt.args.storage, tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("sendMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
