package api

import (
	"errors"
	"fmt"

	"github.com/alexsey-popov/shorturl/internal/api/rest"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Api - фасад для управления серверами
type Api struct {
	shortener *service.Shortener
	sugar     *zap.SugaredLogger
	rest      *rest.Server
	grpc      *grpc.Server
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
		return nil, fmt.Errorf("ошибка при создании REST эндпоинтов: %w", err)
	}

	//

	return &Api{
		shortener: shortener,
		sugar:     sugar,
		rest:      r,
	}, nil
}

func (a *Api) Close() error {
	// TODO: ДОПИШИ
	return errors.Join(
		// Закрываем REST сервер
		a.rest.Close(),
		//
		//a.grpc.Close(),
		//
		a.shortener.Close(),
	)
}

func (a *Api) ListenAndServe() error {

	//TODO: ТУТ НАДО ЗАПУСТИТЬ 2 сервера через errgroup
	return a.rest.ListenAndServe()
}
