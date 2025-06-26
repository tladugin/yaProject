package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func createTestServer() *httptest.Server {
	storage := repository.NewMemStorage()
	s := NewServer(storage)

	r := chi.NewRouter()
	//r.Route("/", func(r chi.Router) {
	//	r.Get("/", s.MainPage)
	//	r.Get("/value/{metric}/{name}", s.GetHandler)
	r.Post("/update/{metric}/{name}/{value}", s.PostHandler)

	//})

	return httptest.NewServer(r)
}
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

	storage := repository.NewMemStorage()
	s := NewServer(storage)
	server := createTestServer()
	defer server.Close()
	testServerURL := server.URL

	r := chi.NewRouter()

	//testServerURL += "/update/"

	testStruct := []struct {
		name string
		url  string
		want int
	}{
		// TODO: Add test cases.
		{name: "gauge", url: "/update/gauge/Alloc/12.34", want: http.StatusOK},
		{name: "counter", url: "/update/counter/PollCount/567890", want: http.StatusOK},
		{name: "invalid_metric_type", url: "/update/unknown/metrics/123", want: http.StatusBadRequest},
		{name: "invalid_path_format", url: "/update/gauge/Alloc", want: http.StatusNotFound},
	}
	for _, tt := range testStruct {
		t.Run(tt.name, func(t *testing.T) {

			//req, err := http.NewRequest("POST", testServerURL+tt.url, nil)
			//println(testServerURL + tt.url)
			//if err != nil {
			//	t.Fatal(err)
			//}
			res := httptest.NewRecorder()

			r.Post(tt.url, s.PostHandler)

			req, err := http.NewRequest("POST", testServerURL+tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.want {
				t.Errorf("Expected status code %d, but got %d", tt.want, res.Code)

			}
		})
	}
}
