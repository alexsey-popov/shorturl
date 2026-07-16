package indb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexsey-popov/shorturl/internal/model"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type InDB struct {
	db *sql.DB
}

// Set - Сохраняем originalURL за значением prefix
func (rep *InDB) Set(URL model.URL) error {
	_, err := rep.db.Exec(
		"INSERT INTO urls (prefix, original_url, user_id) VALUES ($1, $2, $3)",
		URL.Prefix,
		URL.OriginalURL,
		URL.UserID,
	)

	if err != nil {
		// Если произошла ошибка при выполнении запроса - пытаемся её классифицировать
		// Если произошла ошибка уникальности по полю original_url - пытаемся найти подходящую запись по OriginalURL
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == "idx_urls_original_url_unique" {
			diffURL, err2 := rep.FindFromOriginal(URL.OriginalURL)
			if err2 == nil {
				return errors2.NewErrOriginalURLConflict(diffURL, err)
			}
		}

		return fmt.Errorf("ошибка при выполнении запроса к БД: %w", err)
	}

	return nil
}

// SetMany - Сохраняем несколько ссылок
func (rep *InDB) SetMany(URLs []model.URL) error {
	// Создаём подготовленный запрос
	stmt, err := rep.db.Prepare("INSERT INTO urls (prefix, original_url, user_id) SELECT * FROM UNNEST($1::text[], $2::text[], $3::uuid[])")
	if err != nil {
		return fmt.Errorf("ошибка при создании подготовленного запроса к БД: %w", err)
	}
	defer stmt.Close()

	pref := make([]string, len(URLs))
	orig := make([]string, len(URLs))
	users := make([]string, len(URLs))

	for i, url := range URLs {
		pref[i] = url.Prefix
		orig[i] = url.OriginalURL
		users[i] = url.UserID
	}

	_, err = stmt.Exec(pref, orig, users)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса к БД: %w", err)
	}

	return nil
}

// DeleteManyFromUserId Массовое удаление ссылок принадлежащих пользователю
func (rep *InDB) DeleteManyFromUserId(prefixes []string, userID string) error {
	_, err := rep.db.Exec("UPDATE urls SET is_deleted = true WHERE user_id = $1 AND prefix IN (SELECT UNNEST($2::text[]))", userID, prefixes)
	if err != nil {
		err = fmt.Errorf("ошибка при массовом обновлении is_deleted в БД: %w", err)
	}

	return err
}

// Get - получение ссылки на редирект по префиксу
func (rep *InDB) Get(prefix string) (URL model.URL, err error) {
	row := rep.db.QueryRow(
		"SELECT id, prefix, original_url, user_id, is_deleted FROM urls where prefix = $1",
		prefix,
	)

	err = row.Scan(&URL.UUID, &URL.Prefix, &URL.OriginalURL, &URL.UserID, &URL.IsDeleted)

	return
}

// FindFromOriginal - Поиск среди загруженных в память данных
func (rep *InDB) FindFromOriginal(originalURL string) (URL model.URL, err error) {
	row := rep.db.QueryRow(
		"SELECT id, prefix, original_url, user_id, is_deleted FROM urls where original_url = $1",
		originalURL,
	)

	err = row.Scan(&URL.UUID, &URL.Prefix, &URL.OriginalURL, &URL.UserID, &URL.IsDeleted)

	return
}

// FindFromUserID - поиск записей по id пользователя
func (rep *InDB) FindFromUserID(userID string) (URLs []model.URL, err error) {
	rows, err := rep.db.Query(
		"SELECT id, prefix, original_url, user_id, is_deleted FROM urls where user_id = $1",
		userID,
	)
	if err != nil {
		return URLs, fmt.Errorf("ошибка при выполнении запроса на поиск записей по id пользователя: %w", err)
	}

	for rows.Next() {
		URL := model.URL{}

		err = rows.Scan(&URL.UUID, &URL.Prefix, &URL.OriginalURL, &URL.UserID, &URL.IsDeleted)
		if err != nil {
			return URLs, fmt.Errorf("ошибка во время парсинга данных: %w", err)
		}

		URLs = append(URLs, URL)
	}

	if rows.Err() != nil {
		return URLs, fmt.Errorf("ошибка после парсинга данных: %w", err)
	}

	return URLs, nil
}

// Ping - проверка соединения (считаем, что оно всегда есть)
func (rep *InDB) Ping() error {
	return rep.db.Ping()
}

func New(db *sql.DB) *InDB {
	return &InDB{
		db: db,
	}
}
