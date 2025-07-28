package server

import (
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/repository"
	"go.uber.org/zap"
	"sync"
	"testing"
)

func TestRunHTTPServer(t *testing.T) {
	type args struct {
		storage           *repository.MemStorage
		producer          *handler.Producer
		stop              <-chan struct{}
		wg                *sync.WaitGroup
		flagStoreInterval string
		sugar             *zap.SugaredLogger
		flagRunAddr       *string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunHTTPServer(tt.args.storage, tt.args.producer, tt.args.stop, tt.args.wg, tt.args.flagStoreInterval, tt.args.sugar, tt.args.flagRunAddr)
		})
	}
}
