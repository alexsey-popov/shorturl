package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexsey-popov/shorturl/internal/audit"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/service"
	"go.uber.org/zap"
)

// Server - сервер для REST API
type Server struct {
	server    *http.Server
	shortener *service.Shortener
	handler   *handler.Handler
	cfg       *config.Server
	sugar     *zap.SugaredLogger
	delCh     chan handler.DeleteTask
}

// NewServer - создание сервера для REST API
func NewServer(
	shortener *service.Shortener,
	cfg *config.Server,
	sugar *zap.SugaredLogger,
) (*Server, error) {
	// Создаём посредника для аудита
	auditPublisher, err := audit.NewPublisherFromConfig(cfg, sugar)
	if err != nil {
		return nil, err
	}

	// Создаём объект обработчика запросов
	h := handler.NewHandler(sugar, shortener, auditPublisher)

	// Создаём канал для асинхронного удаления ссылок
	delCh := make(chan handler.DeleteTask, 100)

	// Создаём роутер
	r := NewRouter(h, cfg, sugar, delCh)

	// Создаём HTTP-сервер
	server := &http.Server{
		Addr:    cfg.NetAddress,
		Handler: r,
	}

	return &Server{
		server:    server,
		shortener: shortener,
		cfg:       cfg,
		handler:   h,
		sugar:     sugar,
		delCh:     delCh,
	}, nil
}

// ListenAndServe Запуск сервера
func (s Server) ListenAndServe() error {
	// err - ошибка в работе сервера
	var err error

	// delWG - группа обработчиков удаления записей
	var delWG sync.WaitGroup

	// Запускаем воркеры для удаления ссылок
	startDeleteWorkers(10, s.delCh, &delWG, s.shortener, s.sugar)

	if s.cfg.EnableHTTPS {
		s.sugar.Infof("Сервер запущен по адресу https://%s", s.cfg.NetAddress)
		err = s.server.ListenAndServeTLS("cert.pem", "key.pem")
	} else {
		s.sugar.Infof("Сервер запущен по адресу http://%s", s.cfg.NetAddress)
		err = s.server.ListenAndServe()
	}

	// Закрываем канал задач удаления и ожидаем завершения воркеров
	close(s.delCh)
	delWG.Wait()

	return err
}

// Close - закрытие сервера
func (s Server) Close() error {
	// Контекст с таймаутом для плавного завершения активных соединений
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Перестаём получать новые запросы и пытаемся отработать текущие за отведённое время
	err := s.server.Shutdown(shutdownCtx)
	if err != nil {
		err = fmt.Errorf("ошибка при остановке сервера: %w", err)
	}

	// Отдаём совокупность возможной ошибки при закрытии сервера
	// и возможной ошибки при закрытии обработчика запросов
	// (под капотом handler.Close() кроется закрытие аудита и обработчиков удаления записей)
	return errors.Join(
		err,
		s.handler.Close(),
	)
}

// startDeleteWorkers - Запуск воркер-пула для удаления ссылок
func startDeleteWorkers(
	num int,
	delCh <-chan handler.DeleteTask,
	wg *sync.WaitGroup,
	shortener *service.Shortener,
	sugar *zap.SugaredLogger,
) {
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for task := range delCh {
				err := shortener.Rep.DeleteManyFromUserID(task.Prefixes, task.UserID)
				if err != nil {
					sugar.Errorf("ошибка при удалении ссылок: %v", err)
				}
			}
		}()
	}
}
