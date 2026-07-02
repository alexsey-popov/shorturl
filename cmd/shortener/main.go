package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/compact"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
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

		handler.UseDBRepository(config.Server.BaseURL, db)

		sugar.Infoln("В качестве хранилища используется БД")
	// Если нет данных для подключения к БД, но есть путь до файла - используем файл
	case config.Server.FilePath != "":
		if err = handler.UseFileRepository(config.Server.BaseURL, config.Server.FilePath); err != nil {
			sugar.Fatal(err)
		}

		sugar.Infoln("В качестве хранилища используется файл")
	default:
		sugar.Infoln("В качестве хранилища используется ОЗУ")

		handler.UseMemoryRepository(config.Server.BaseURL)
	}

	// Объявляем роуты
	r := chi.NewRouter()

	// Логируем результаты запросов LogMiddleware
	r.Use(logger.NewHTTPMiddleware(sugar))

	// Разворачиваем и сокращаём данные
	r.Use(compact.HTTPMiddleware)

	r.Post("/", handler.HandlePost)
	r.Post("/api/shorten/batch", handler.HandlePostBatch)
	r.Post("/api/shorten", handler.HandlePostJSON)
	r.Get("/ping", handler.HandleGetPing(sugar))
	r.Get("/{id}", handler.HandleGet)
	r.MethodNotAllowed(handler.HandleFails)

	// Поднимает сервер
	err = http.ListenAndServe(config.Server.NetAddress, r)
	if err != nil {
		sugar.Fatalf("ошибка в работе сервера: %w", err)
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
