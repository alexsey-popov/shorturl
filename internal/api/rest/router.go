package rest

import (
	"github.com/alexsey-popov/shorturl/internal/auth"
	"github.com/alexsey-popov/shorturl/internal/compact"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/logger"
	"github.com/alexsey-popov/shorturl/internal/subnet"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// NewRouter - создание роутера и закрепление обработчиков за эндпоинтами
func NewRouter(
	h *handler.Handler,
	cfg *config.Server,
	sugar *zap.SugaredLogger,
	delCh chan handler.DeleteTask,
) *chi.Mux {
	// Объявляем роуты
	r := chi.NewRouter()

	// Логируем результаты запросов LogMiddleware
	r.Use(logger.NewHTTPMiddleware(sugar))

	// Разворачиваем и сокращаём данные
	r.Use(compact.HTTPMiddleware(sugar))

	// Аутентифицируем пользователя
	r.Use(auth.NewHTTPMiddleware(cfg.SecretKey, cfg.TokenExp, sugar))

	r.Post("/", h.HandlePost)
	r.Post("/api/shorten/batch", h.HandlePostBatch)
	r.Post("/api/shorten", h.HandlePostJSON)
	r.Get("/api/user/urls", h.HandleGetUserURLs)
	r.Delete("/api/user/urls", h.HandleDeleteUserURLs(delCh))
	r.Get("/ping", h.HandleGetPing)
	r.Get("/{id}", h.HandleGet)

	r.With(subnet.HTTPMiddleware(sugar, cfg.TrustedSubnet)).Get("/api/internal/stats", h.HandleGetStats)

	r.MethodNotAllowed(h.HandleFails)

	return r
}
