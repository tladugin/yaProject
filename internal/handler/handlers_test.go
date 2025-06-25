package handler

import (
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewServer(t *testing.T) {
	type args struct {
		s *repository.MemStorage
	}
	tests := []struct {
		name string
		args args
		want *Server
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewServer(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServer_GetHandler(t *testing.T) {
	type fields struct {
		storage *repository.MemStorage
	}
	type args struct {
		res http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				storage: tt.fields.storage,
			}
			s.GetHandler(tt.args.res, tt.args.req)
		})
	}
}

func TestServer_PostHandler(t *testing.T) {
	type fields struct {
		storage *repository.MemStorage
	}
	type args struct {
		res http.ResponseWriter
		req *http.Request
	}
	storage := repository.NewMemStorage()
	s := NewServer(storage)

	testStruct := []struct {
		name string
		url  string
		want int
	}{
		// TODO: Add test cases.
		{name: "gauge", url: "/update/gauge/Alloc/12.34", want: http.StatusOK},
		{name: "counter", url: "/update/counter/PollCount/567890", want: http.StatusOK},
		{name: "invalid_metric_type", url: "/update/unknown/metrics/type/123", want: http.StatusNotFound},
		{name: "invalid_path_format", url: "/update/gauge/Alloc", want: http.StatusNotFound},
	}
	for _, tt := range testStruct {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest("POST", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			res := httptest.NewRecorder()
			s.PostHandler(res, req)
			if res.Code != tt.want {
				t.Errorf("Expected status code %d, but got %d", tt.want, res.Code)
			}
		})
	}
}
