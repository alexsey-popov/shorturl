package main

import (
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/go-chi/chi/v5"
)

func main() {

	r := chi.NewRouter()

	r.Post("/", handler.HandlePost)

	r.Get("/{id}", handler.HandleGet)

	err := http.ListenAndServe(config.ServerAddr, r)
	if err != nil {
		panic(err)
	}
}
