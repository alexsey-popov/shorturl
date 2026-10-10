package grpcsrv

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	UnimplementedShortenerServiceServer

	server    *grpc.Server
	shortener *service.Shortener
	cfg       *config.Server
	sugar     *zap.SugaredLogger
}

// NewServer - создание сервера для gRPC API
func NewServer(
	shortener *service.Shortener,
	cfg *config.Server,
	sugar *zap.SugaredLogger,
) (*Server, error) {
	return &Server{
		shortener: shortener,
		cfg:       cfg,
		sugar:     sugar,
	}, nil
}

// ListenAndServe - запуск gRPC сервера
func ListenAndServe(s *Server) error {
	// Определяем порт для сервера
	// В задании инкремента не было указано конкретного порта, поэтому захардкодил тот, что был в последних уроках
	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		return fmt.Errorf("не удалось прослушать порт для gRPC сервера: %w", err)
	}

	// создаём gRPC-сервер без зарегистрированной службы
	s.server = grpc.NewServer()
	// регистрируем сервис
	RegisterShortenerServiceServer(s.server, s)

	s.sugar.Infof("Сервер(gRPC) запущен")
	if err = s.server.Serve(listen); err != nil {
		return fmt.Errorf("ошибка на gRPC сервере: %w", err)
	}

	return nil
}

// Close - закрытие сервера
func (s *Server) Close() error {
	if s.server == nil {
		return nil
	}
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("gRPC сервер успешно остановлен")
	case <-time.After(15 * time.Second):
		log.Println("gRPC сервер не успел обработать запросы до таймаута и завершен принудительно")
		s.server.Stop()
	}

	return nil
}

// ShortenURL - сокращение ссылки
func (s *Server) ShortenURL(ctx context.Context, in *URLShortenRequest) (*URLShortenResponse, error) {
	// response - объект ответа
	var response URLShortenResponse
	// shortURL - сокращенный URL
	var shortURL string

	// originalURL - ссылка, которую нужно сократить
	originalURL := in.GetUrl()

	// Получаем сокращённую ссылку
	shortURL, err := s.shortener.Add(originalURL, "userID") // TODO: ЗАМЕНИ USERID на реальный USERID
	if err != nil {
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		var conflictErr *errors2.OriginalURLConflictError
		if errors.As(err, &conflictErr) {
			// Получаем полную ссылку по префиксу
			shortURL, err = s.shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err != nil {
				//TODO ОШИБКА
			}
		}
	}

	// Отмечаем событие аудита
	//s.auditPublisher.Notify(context.TODO(), audit.Event{
	//	Timestamp: time.Now().Unix(),
	//	Action:    "shorten",
	//	UserID:    "userID", // TODO: ЗАМЕНИ USERID на реальный USERID
	//	URL:       originalURL,
	//})

	// Добавляем ссылку в тело ответа
	response.SetResult(shortURL)

	return &response, nil
}

func (s *Server) ExpandURL(ctx context.Context, in *URLExpandRequest) (*URLExpandResponse, error) {
	var response URLExpandResponse

	return &response, nil
}

func (s *Server) ListUserURLs(ctx context.Context, in *emptypb.Empty) (*UserURLsResponse, error) {
	var response UserURLsResponse

	return &response, nil
}
