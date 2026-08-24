package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/audit"
	"github.com/alexsey-popov/shorturl/internal/auth"
	"github.com/alexsey-popov/shorturl/internal/compact"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/logger"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Создаём логгер
	l, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("ошибка при создании логгера: %v", err)
	}
	defer l.Sync()
	sugar := l.Sugar()

	// Парсим конфиг значениями из флагов и переменных окружения
	if err = config.Parse(); err != nil {
		sugar.Fatal(err)
	}

	// Объект сокращателя ссылок
	var shortener service.Shortener

	// Пытаемся подключить различные виды хранилищ (по умолчанию используется хранение в памяти)
	switch {
	// Если указаны данные для подключения в БД - используем БД
	case config.Server.DSN != "":
		// Создаём объект взаимодействия с базой
		db, err := connectDB(config.Server.DSN)
		if err != nil {
			sugar.Fatal(err)
		}
		defer db.Close()

		shortener = service.NewDBShortener(config.Server.BaseURL, db)

		sugar.Infoln("В качестве хранилища используется БД")
	// Если нет данных для подключения к БД, но есть путь до файла - используем файл
	case config.Server.FilePath != "":
		shortener, err = service.NewFileShortener(config.Server.BaseURL, config.Server.FilePath)
		if err != nil {
			sugar.Fatal(err)
		}

		sugar.Infoln("В качестве хранилища используется файл")
	default:
		sugar.Infoln("В качестве хранилища используется ОЗУ")

		shortener = service.NewMemoryShortener(config.Server.BaseURL)
	}

	// Создаём канал для асинхронного удаления ссылок
	delCh := make(chan handler.DeleteTask, 100)
	// Запускаем воркеры для удаления ссылок
	startDeleteWorkers(10, delCh, shortener, sugar)

	// Создаём наблюдатель для аудита
	var auditManager *audit.Publisher
	if config.Server.HasAudit() {
		auditManager = audit.NewPublisher(sugar)
		if config.Server.AuditFile != "" {
			fileObserver, err := audit.NewFileObserver(sugar, config.Server.AuditFile)
			if err != nil {
				sugar.Fatalf("ошибка при создании наблюдателя файла аудита: %v", err)
			}
			auditManager.Register(fileObserver)
			sugar.Infoln("Аудит в файл включен")
		}
		if config.Server.AuditURL != "" {
			urlObserver := audit.NewURLObserver(sugar, config.Server.AuditURL)
			auditManager.Register(urlObserver)
			sugar.Infoln("Аудит по URL включен")
		}
	}

	// Создаём объект обработчика запросов
	h := handler.NewHandler(sugar, shortener, delCh, auditManager)

	// Объявляем роуты
	r := chi.NewRouter()

	// Логируем результаты запросов LogMiddleware
	r.Use(logger.NewHTTPMiddleware(sugar))

	// Разворачиваем и сокращаём данные
	r.Use(compact.HTTPMiddleware(sugar))

	// Аутентифицируем пользователя
	r.Use(auth.NewHTTPMiddleware(config.Server.SecretKey, config.Server.TokenExp, sugar))

	r.Post("/", h.HandlePost)
	r.Post("/api/shorten/batch", h.HandlePostBatch)
	r.Post("/api/shorten", h.HandlePostJSON)
	r.Get("/api/user/urls", h.HandleGetUserURLs)
	r.Delete("/api/user/urls", h.HandleDeleteUserURLs)
	r.Get("/ping", h.HandleGetPing)
	r.Get("/{id}", h.HandleGet)
	r.MethodNotAllowed(h.HandleFails)

	// Поднимает сервер
	err = http.ListenAndServe(config.Server.NetAddress, r)
	if err != nil {
		sugar.Fatalf("ошибка в работе сервера: %v", err)
	}
}

// connectDB - Подключение в БД и выполнение миграций
func connectDB(serverDSN string) (*sql.DB, error) {
	// Создаём объект взаимодействия с базой
	db, err := sql.Open("pgx", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("ошибка при подключении к БД: %w", err)
	}

	// Создаём драйвер для миграций
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return db, fmt.Errorf("ошибка при создании драйвера БД: %w", err)
	}

	//   Создаём объект миграции на основе файлов с миграциями и подключения
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return db, fmt.Errorf("ошибка при подготовке к миграций БД: %w", err)
	}

	// Проводим миграции
	err = m.Up()
	// Ошибку migrate.ErrNoChange пропускаем
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return db, fmt.Errorf("ошибка при запуске миграций БД: %w", err)
	}

	return db, nil
}

// startDeleteWorkers - Запуск воркер-пула для удаления ссылок
func startDeleteWorkers(num int, delCh <-chan handler.DeleteTask, shortener service.Shortener, logger *zap.SugaredLogger) {
	for i := 0; i < num; i++ {
		go func() {
			for task := range delCh {
				err := shortener.Rep.DeleteManyFromUserId(task.Prefixes, task.UserID)
				if err != nil {
					logger.Errorf("ошибка при удалении ссылок: %v", err)
				}
			}
		}()
	}
}
