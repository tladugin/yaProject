package main

import (
	"fmt"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
)

func main() {

	storage := repository.NewMemStorage()
	/*s := handler.NewServer(storage)

	r := chi.NewRouter()
	r.Post("/update/", s.PostHandler)
	r.Get("/value/", s.GetHandler)
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}*/

	s := handler.NewServer(storage)

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, s.PostHandler)
	mux.HandleFunc(`/value/`, s.GetHandler)
	mux.HandleFunc(`/`, s.MainPage)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}

}
