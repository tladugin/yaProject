package main

import (
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/repository"
	"sync"
	"testing"
	"time"
)

func Test_initLogger(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initLogger()
		})
	}
}

func Test_performBackup(t *testing.T) {
	type args struct {
		storage  *repository.MemStorage
		producer *handler.Producer
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
			if err := performBackup(tt.args.storage, tt.args.producer); (err != nil) != tt.wantErr {
				t.Errorf("performBackup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_restoreFromBackup(t *testing.T) {
	type args struct {
		storage *repository.MemStorage
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreFromBackup(tt.args.storage)
		})
	}
}

func Test_runFinalBackup(t *testing.T) {
	type args struct {
		storage  *repository.MemStorage
		producer *handler.Producer
		stop     <-chan struct{}
		wg       *sync.WaitGroup
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runFinalBackup(tt.args.storage, tt.args.producer, tt.args.stop, tt.args.wg)
		})
	}
}

func Test_runHTTPServer(t *testing.T) {
	type args struct {
		storage  *repository.MemStorage
		producer *handler.Producer
		stop     <-chan struct{}
		wg       *sync.WaitGroup
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runHTTPServer(tt.args.storage, tt.args.producer, tt.args.stop, tt.args.wg)
		})
	}
}

func Test_runPeriodicBackup(t *testing.T) {
	type args struct {
		storage  *repository.MemStorage
		producer *handler.Producer
		interval time.Duration
		stop     <-chan struct{}
		wg       *sync.WaitGroup
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runPeriodicBackup(tt.args.storage, tt.args.producer, tt.args.interval, tt.args.stop, tt.args.wg)
		})
	}
}

func Test_waitForShutdown(t *testing.T) {
	type args struct {
		stop chan<- struct{}
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waitForShutdown(tt.args.stop)
		})
	}
}
