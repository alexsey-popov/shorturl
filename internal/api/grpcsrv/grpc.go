package grpcsrv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/alexsey-popov/shorturl/internal/auth"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
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

	var opts []grpc.ServerOption

	// Поддержка TLS в соответствии с конфигурацией
	if s.cfg.EnableHTTPS {
		creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
		if err != nil {
			return fmt.Errorf("ошибка при загрузке TLS сертификата для gRPC сервера: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Подключаем интерцептор аутентификации через metadata
	opts = append(opts, grpc.UnaryInterceptor(auth.NewGRPCUnaryInterceptor(s.cfg.SecretKey, s.cfg.TokenExp, s.sugar)))

	// создаём gRPC-сервер с опциями
	s.server = grpc.NewServer(opts...)
	// регистрируем сервис
	RegisterShortenerServiceServer(s.server, s)

	if s.cfg.EnableHTTPS {
		s.sugar.Infof("Сервер(gRPC) запущен с TLS")
	} else {
		s.sugar.Infof("Сервер(gRPC) запущен")
	}

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
		s.sugar.Info("Сервер(gRPC) успешно остановлен")
	case <-time.After(15 * time.Second):
		s.sugar.Error("Сервер(gRPC) не успел обработать запросы до таймаута и завершен принудительно")
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

	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "пользователь не авторизован")
	}

	// Получаем сокращённую ссылку
	shortURL, err := s.shortener.Add(originalURL, userID)
	if err != nil {
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		var conflictErr *errors2.OriginalURLConflictError
		if errors.As(err, &conflictErr) {
			// Получаем полную ссылку по префиксу
			shortURL, err = s.shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "ошибка при получении URL: %v", err)
			}
		} else {
			return nil, status.Errorf(codes.Internal, "ошибка при добавлении URL: %v", err)
		}
	}

	// Добавляем ссылку в тело ответа
	response.SetResult(shortURL)

	return &response, nil
}

// ExpandURL - Переход по сокращённой ссылке
func (s *Server) ExpandURL(ctx context.Context, in *URLExpandRequest) (*URLExpandResponse, error) {
	var response URLExpandResponse

	id := in.GetId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "пустой id")
	}

	url, err := s.shortener.Rep.Get(id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "ссылка не найдена: %v", err)
	}

	if url.IsDeleted {
		return nil, status.Error(codes.NotFound, "ссылка удалена")
	}

	response.SetResult(url.OriginalURL)

	return &response, nil
}

// ListUserURLs - список сокращенных ссылок пользовтеля
func (s *Server) ListUserURLs(ctx context.Context, in *emptypb.Empty) (*UserURLsResponse, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "пользователь не авторизован")
	}

	urls, err := s.shortener.Rep.FindFromUserID(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка при получении ссылок: %v", err)
	}

	var response UserURLsResponse
	urlDataList := make([]*URLData, 0, len(urls))
	for _, u := range urls {
		shortURL, err := s.shortener.GetURLFromPrefix(u.Prefix)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ошибка при формировании short URL: %v", err)
		}
		item := &URLData{}
		item.SetShortUrl(shortURL)
		item.SetOriginalUrl(u.OriginalURL)
		urlDataList = append(urlDataList, item)
	}
	response.SetUrl(urlDataList)

	return &response, nil
}
