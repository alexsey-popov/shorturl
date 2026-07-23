package service

import (
	"crypto/rand"
	"database/sql"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/repository/indb"
	"github.com/alexsey-popov/shorturl/internal/repository/infile"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
)

// Repository - Интерфейс для хранилищ
type Repository interface {
	// Set - Сохранение originalURL за значением prefix
	Set(url model.URL) error
	// SetMany - Сохранение нескольких ссылок
	SetMany(urls []model.URL) error
	// Get - Получение originalURL по значению prefix
	Get(prefix string) (url model.URL, err error)
	// FindFromUserID - Получение списка ссылок закреплённых за пользователем
	FindFromUserID(userID string) (urls []model.URL, err error)
	// FindFromOriginal - Поиск значений по значению originalURL
	FindFromOriginal(originalURL string) (url model.URL, err error)
	// DeleteManyFromUserId - Массовое удаление ссылок принадлежащих пользователю
	DeleteManyFromUserId(prefixes []string, userID string) error
	// Ping - Проверка соединения
	Ping() error
}

// Shortener - Сервис для сокращения ссылок
type Shortener struct {
	Rep     Repository
	BaseURL string
}

// NewMemoryShortener - Конструктор Shortener для хранения данных в памяти
func NewMemoryShortener(baseURL string) Shortener {
	return Shortener{
		Rep:     inmemory.New(),
		BaseURL: baseURL,
	}
}

// NewFileShortener - Конструктор Shortener для хранения данных внутри файла
func NewFileShortener(baseURL string, filepath string) (Shortener, error) {
	rep, err := infile.New(filepath)
	if err != nil {
		return Shortener{BaseURL: baseURL}, err
	}

	return Shortener{
		Rep:     rep,
		BaseURL: baseURL,
	}, nil
}

// NewDBShortener - Конструктор Shortener для хранения данных в базе данных
func NewDBShortener(baseURL string, db *sql.DB) Shortener {
	return Shortener{
		Rep:     indb.New(db),
		BaseURL: baseURL,
	}
}

// CheckValidURL - валидация ссылки
func (s Shortener) CheckValidURL(originalURL string) (err error) {
	// Проверяем корректность URL
	_, err = url.ParseRequestURI(originalURL)

	return err
}

// Add - Добавление новой ссылки
func (s Shortener) Add(originalURL string, userID string) (string, error) {
	// Валидация ссылки
	if err := s.CheckValidURL(originalURL); err != nil {
		return "", err
	}

	item := model.New(s.getNewPrefix(), originalURL, userID)

	err := s.Rep.Set(item)
	if err != nil {
		return "", err
	}

	return s.GetURLFromPrefix(item.Prefix)
}

// AddMany - множественное создание сокращённых ссылок
func (s Shortener) AddMany(originalURLs []string, userID string) (mapURLs map[string]string, err error) {
	// Проверяем все ссылки на валидность и заполняем URLs
	urls := make([]model.URL, 0, len(originalURLs))
	for _, originalURL := range originalURLs {
		if err = s.CheckValidURL(originalURL); err != nil {
			return
		}

		urls = append(urls, model.New(s.getNewPrefix(), originalURL, userID))
	}

	// Пытаемся сохранить данные в базе
	err = s.Rep.SetMany(urls)
	if err != nil {
		return
	}

	mapURLs = make(map[string]string)
	for _, item := range urls {
		shortURL, err := s.GetURLFromPrefix(item.Prefix)
		if err != nil {
			return nil, err
		}

		mapURLs[item.OriginalURL] = shortURL
	}

	return
}

// getNewPrefix - Получение нового префикса
func (s Shortener) getNewPrefix() string {
	return rand.Text()
}

// GetURLFromPrefix - Получение сокращённого url по префиксу
func (s Shortener) GetURLFromPrefix(prefix string) (string, error) {
	return url.JoinPath(s.BaseURL, prefix)
}
