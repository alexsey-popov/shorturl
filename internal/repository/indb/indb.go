package indb

import (
	"database/sql"

	"github.com/alexsey-popov/shorturl/internal/model"
)

type InDB struct {
	DB *sql.DB
}

// Set - Фиксируем originalURL за значением prefix
func (rep *InDB) Set(URL model.URL) error {
	_, err := rep.DB.Exec(
		"INSERT INTO urls (prefix, original_url) VALUES ($1, $2)",
		URL.Prefix,
		URL.OriginalURL,
	)

	if err != nil {
		return err
	}

	return nil
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
