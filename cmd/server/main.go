package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/repository"
	"net/http"
)

func main() {
	parseFlags()
	storage := repository.NewMemStorage()
	s := handler.NewServer(storage)

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", s.MainPage)
		r.Get("/value/{metric}/{name}", s.GetHandler)
		r.Post("/update/{metric}/{name}/{value}", s.PostHandler)
	})

	fmt.Println("Starting server on :", flagRunAddr)
	if err := http.ListenAndServe(flagRunAddr, r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}

}
