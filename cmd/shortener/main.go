package main

import (
	"database/sql"
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
		log.Fatal(err)
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
		db, err := sql.Open("pgx", config.Server.DSN)
		if err != nil {
			sugar.Fatal(err)
		}
		defer db.Close()

		// Запуск миграций
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			sugar.Fatal(err)
		}
		m, err := migrate.NewWithDatabaseInstance(
			"file://migrations",
			"postgres", driver)
		if err != nil {
			sugar.Fatal(err)
		}

		m.Up() // or m.Steps(2) if you want to explicitly set the number of migrations to run

		handler.UseDBRepository(db)

		sugar.Infoln("Use database repository")
	// Если нет данных для подключения к БД, но есть путь до файла - используем файл
	case config.Server.FilePath != "":
		if err = handler.UseFileRepository(); err != nil {
			sugar.Fatal(err)
		}

		sugar.Infoln("Use file repository")
	default:
		sugar.Infoln("Use database repository")
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
	r.Get("/ping", handler.HandleGetPing)
	r.Get("/{id}", handler.HandleGet)
	r.MethodNotAllowed(handler.HandleFails)

	// Поднимает сервер
	err = http.ListenAndServe(config.Server.NetAddress, r)
	if err != nil {
		sugar.Fatal(err)
	}
}
