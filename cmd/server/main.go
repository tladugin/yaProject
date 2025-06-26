package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
)

func main() {

	storage := repository.NewMemStorage()
	s := handler.NewServer(storage)

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", s.MainPage)
		r.Get("/value/{metric}/{name}", s.GetHandler)
		r.Post("/update/{metric}/{name}/{value}", s.PostHandler)
	})
	//r.Route("/update", func(r chi.Router) {

	//})
	//r.Route("/value", func(r chi.Router) {
	//	r.Get("/{metric}/{name}", s.GetHandler)
	//})
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}

	/*s := handler.NewServer(storage)

	mux := http.NewServeMux()

	mux.HandleFunc(`/`, s.MainPage)
	mux.HandleFunc(`/update/`, s.PostHandler)
	mux.HandleFunc(`/value/`, s.GetHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
	*/
}
