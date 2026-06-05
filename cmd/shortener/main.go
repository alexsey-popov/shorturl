package main

import (
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Парсим конфиг значениями из флагов и переменных окружения
	config.Parse()

	// Создаём новый логгер
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	// Передаём его версию в нет logger
	logger.Sugar = l.Sugar()

	// Объявляем роуты
	r := chi.NewRouter()

	// Оборачиваем все запросы в LogMiddleware
	r.Use(logger.LogMiddleware)

	r.Post("/", handler.HandlePost)
	r.Get("/{id}", handler.HandleGet)
	r.MethodNotAllowed(handler.HandleFails)

	// Поднимает сервер
	err = http.ListenAndServe(config.Server.NetAddress, r)
	if err != nil {
		panic(err)
	}
}
