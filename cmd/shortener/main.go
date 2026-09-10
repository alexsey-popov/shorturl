package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

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
	cfg, err := config.NewParsed()
	if err != nil {
		sugar.Fatalf("ошибка при пирсинге конфига: %v", err.Error())
	}

	// Объект сокращателя ссылок
	var shortener service.Shortener

	// Пытаемся подключить различные виды хранилищ (по умолчанию используется хранение в памяти)
	switch {
	// Если указаны данные для подключения в БД - используем БД
	case cfg.DSN != "":
		// Создаём объект взаимодействия с базой
		db, err := connectDB(cfg.DSN)
		if err != nil {
			sugar.Fatalf("ошибка при подключении к БД :%v", err.Error())
		}
		defer db.Close()

		shortener = service.NewDBShortener(cfg.BaseURL, db)

		sugar.Infoln("В качестве хранилища используется БД")
	// Если нет данных для подключения к БД, но есть путь до файла - используем файл
	case cfg.FilePath != "":
		shortener, err = service.NewFileShortener(cfg.BaseURL, cfg.FilePath)
		if err != nil {
			sugar.Fatal(err)
		}

		sugar.Infoln("В качестве хранилища используется файл")
	default:
		sugar.Infoln("В качестве хранилища используется ОЗУ")

		shortener = service.NewMemoryShortener(cfg.BaseURL)
	}

	// Создаём канал для асинхронного удаления ссылок
	delCh := make(chan handler.DeleteTask, 100)
	var delWG sync.WaitGroup
	// Запускаем воркеры для удаления ссылок
	startDeleteWorkers(10, delCh, &delWG, shortener, sugar)

	// Создаём наблюдатель для аудита
	var auditManager *audit.Publisher
	if cfg.HasAudit() {
		auditManager = audit.NewPublisher(sugar)
		defer auditManager.Close()

		if cfg.AuditFile != "" {
			fileObserver, err := audit.NewFileObserver(sugar, cfg.AuditFile)
			if err != nil {
				sugar.Fatalf("ошибка при создании наблюдателя файла аудита: %v", err)
			}
			auditManager.Register(fileObserver)
			sugar.Infoln("Аудит в файл включен")
		}
		if cfg.AuditURL != "" {
			urlObserver := audit.NewURLObserver(sugar, cfg.AuditURL)
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
	r.Use(auth.NewHTTPMiddleware(cfg.SecretKey, cfg.TokenExp, sugar))

	r.Post("/", h.HandlePost)
	r.Post("/api/shorten/batch", h.HandlePostBatch)
	r.Post("/api/shorten", h.HandlePostJSON)
	r.Get("/api/user/urls", h.HandleGetUserURLs)
	r.Delete("/api/user/urls", h.HandleDeleteUserURLs)
	r.Get("/ping", h.HandleGetPing)
	r.Get("/{id}", h.HandleGet)
	r.MethodNotAllowed(h.HandleFails)

	// Поднимает сервер
	err = http.ListenAndServe(cfg.NetAddress, r)
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
func startDeleteWorkers(
	num int,
	delCh <-chan handler.DeleteTask,
	wg *sync.WaitGroup,
	shortener service.Shortener,
	logger *zap.SugaredLogger,
) {
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for task := range delCh {
				err := shortener.Rep.DeleteManyFromUserID(task.Prefixes, task.UserID)
				if err != nil {
					logger.Errorf("ошибка при удалении ссылок: %v", err)
				}
			}
		}()
	}
}
