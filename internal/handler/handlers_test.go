package handler

import (
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
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

func TestServer_MainPage(t *testing.T) {
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
			s.MainPage(tt.args.res, tt.args.req)
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
			s.PostHandler(tt.args.res, tt.args.req)
		})
	}
}
