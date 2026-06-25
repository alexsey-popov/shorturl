package indb

import (
	"database/sql"
	"errors"

	"github.com/alexsey-popov/shorturl/internal/model"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type InDB struct {
	DB *sql.DB
}

// Set - Сохраняем originalURL за значением prefix
func (rep *InDB) Set(URL model.URL) error {
	_, err := rep.DB.Exec(
		"INSERT INTO urls (prefix, original_url) VALUES ($1, $2)",
		URL.Prefix,
		URL.OriginalURL,
	)

	if err != nil {
		// Если произошла ошибка при выполнении запроса - пытаемся её классифицировать
		// Если произошла ошибка уникальности - пытаемся найти подходящую запись по OriginalURL
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			diffURL, err2 := rep.FindFromOriginal(URL.OriginalURL)
			if err2 == nil {
				return errors2.NewErrOriginalURLConflict(diffURL, err)
			}
		}

		return err
	}

	return nil
}

// SetMany - Сохраняем несколько ссылок
func (rep *InDB) SetMany(URLs []model.URL) error {
	// Создаём транзакцию и
	tx, err := rep.DB.Begin()
	if err != nil {
		return err
	}

	for _, URL := range URLs {
		_, err := tx.Exec(
			"INSERT INTO urls (prefix, original_url) VALUES ($1, $2)",
			URL.Prefix,
			URL.OriginalURL,
		)

		if err != nil {
			tx.Rollback()

			return err
		}
	}

	return tx.Commit()
}

// Get - получение ссылки на редирект по префиксу
func (rep *InDB) Get(prefix string) (URL model.URL, err error) {
	row := rep.DB.QueryRow(
		"SELECT id, prefix, original_url FROM urls where prefix = $1",
		prefix,
	)

	err = row.Scan(&URL.UUID, &URL.Prefix, &URL.OriginalURL)

	return
}

// FindFromOriginal - Поиск среди загруженных в память данных
func (rep *InDB) FindFromOriginal(originalURL string) (URL model.URL, err error) {
	row := rep.DB.QueryRow(
		"SELECT id, prefix, original_url FROM urls where original_url = $1",
		originalURL,
	)

	err = row.Scan(&URL.UUID, &URL.Prefix, &URL.OriginalURL)

	return
}

func New(db *sql.DB) *InDB {
	return &InDB{
		DB: db,
	}
}
