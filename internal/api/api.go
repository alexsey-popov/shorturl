package api

import (
	"errors"
	"fmt"

	"github.com/alexsey-popov/shorturl/internal/api/grpcsrv"
	"github.com/alexsey-popov/shorturl/internal/api/rest"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
	"go.uber.org/zap"
)

// Api - фасад для управления серверами
type Api struct {
	shortener *service.Shortener
	sugar     *zap.SugaredLogger
	rest      *rest.Server
	grpc      *grpcsrv.Server
}

// NewFromConfig - создание объекта по данным из конфига
func NewFromConfig(cfg *config.Server, sugar *zap.SugaredLogger) (*Api, error) {
	// Создаём слой бизнес-логики
	shortener, err := service.NewFromConfig(cfg, sugar)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании слоя бизнес-логики: %w", err)
	}

	// Создаём сервер для REST API
	r, err := rest.NewServer(shortener, cfg, sugar)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании REST сервера: %w", err)
	}

	//
	g, err := grpcsrv.NewServer()
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании gRPC сервера: %w", err)
	}

	return &Api{
		shortener: shortener,
		sugar:     sugar,
		rest:      r,
		grpc:      g,
	}, nil
}

// Close - Закрытие всех подключений
func (a *Api) Close() error {
	// TODO: ДОПИШИ
	return errors.Join(
		// Закрываем REST сервер
		a.rest.Close(),
		// Закрываем gRPC сервер
		a.grpc.Close(),
		// Закрываем слой бизнес-логики
		a.shortener.Close(),
	)
}

func (a *Api) ListenAndServe() error {
	// Канал для ошибок запуска серверов с буфером на 2 горутины
	srvErr := make(chan error, 2)

	// Запускаем REST сервер
	go func() {
		srvErr <- a.rest.ListenAndServe()
	}()

	// Запускаем gRPC сервер
	go func() {
		srvErr <- grpcsrv.ListenAndServe(a.grpc)
	}()

	return <-srvErr
}
