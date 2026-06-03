package main

import (
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Парсим конфиг значениями из флагов и переменных окружения
	config.Parse()

	// Объявляем роуты
	r := chi.NewRouter()
	r.Post("/", handler.HandlePost)
	r.Get("/{id}", handler.HandleGet)
	r.MethodNotAllowed(handler.HandleFails)

	// Поднимает сервер
	err := http.ListenAndServe(config.Server.NetAddress, r)
	if err != nil {
		panic(err)
	}
}
