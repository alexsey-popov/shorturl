package main

import (
	"log"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/compact"
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
		log.Fatal(err)
	}
	defer l.Sync()

	// Передаём созданный логер в пакет
	logger.Sugar = l.Sugar()

	// Объявляем роуты
	r := chi.NewRouter()

	// Логируем результаты запросов LogMiddleware
	r.Use(logger.HTTPMiddleware)

	// Разворачиваем и сокращаём данные
	r.Use(compact.HTTPMiddleware)

	r.Post("/", handler.HandlePost)
	r.Post("/api/shorten", handler.HandlePostJson)
	r.Get("/{id}", handler.HandleGet)
	r.MethodNotAllowed(handler.HandleFails)

	// Поднимает сервер
	err = http.ListenAndServe(config.Server.NetAddress, r)
	if err != nil {
		log.Fatal(err)
	}
}
