package service

import (
	"crypto/rand"
	"database/sql"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/repository/indb"
	"github.com/alexsey-popov/shorturl/internal/repository/infile"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
)

// Repository - Интерфейс для хранилищ
type Repository interface {
	// Set - Сохранение originalURL за значением prefix
	Set(URL model.URL) error
	// SetMany - Сохранение нескольких ссылок
	SetMany(URLs []model.URL) error
	// Get - Получение originalURL по значению prefix
	Get(prefix string) (URL model.URL, err error)
	// FindFromOriginal - Поиск значений по значению originalURL
	FindFromOriginal(originalURL string) (URL model.URL, err error)
	// Ping - Проверка соединения
	Ping() error
}

// Shortener - Сервис для сокращения ссылок
type Shortener struct {
	Rep Repository
}

// NewMemoryShortener - Конструктор Shortener для хранения данных в памяти
func NewMemoryShortener() Shortener {
	return Shortener{
		Rep: inmemory.New(),
	}
}

// NewFileShortener - Конструктор Shortener для хранения данных внутри файла
func NewFileShortener() (Shortener, error) {
	rep, err := infile.New(config.Server.FilePath)
	if err != nil {
		return Shortener{}, err
	}

	return Shortener{
		Rep: rep,
	}, nil
}

// NewDBShortener - Конструктор Shortener для хранения данных в базе данных
func NewDBShortener(db *sql.DB) Shortener {
	return Shortener{
		Rep: indb.New(db),
	}
}

// CheckValidURL - валидация ссылки
func (s Shortener) CheckValidURL(originalURL string) (err error) {
	// Проверяем корректность URL
	_, err = url.ParseRequestURI(originalURL)

	return err
}

// Add - Добавление новой ссылки
func (s Shortener) Add(originalURL string) (string, error) {
	// Валидация ссылки
	if err := s.CheckValidURL(originalURL); err != nil {
		return "", err
	}

	URL := model.New(s.getNewPrefix(), originalURL)

	err := s.Rep.Set(URL)
	if err != nil {
		return "", err
	}

	return s.GetURLFromPrefix(URL.Prefix)
}

// AddMany - множественное создание сокращённых ссылок
func (s Shortener) AddMany(originalURLs []string) (mapURLs map[string]string, err error) {
	// Проверяем все ссылки на валидность и заполняем URLs
	URLs := make([]model.URL, 0, len(originalURLs))
	for _, originalURL := range originalURLs {
		if err = s.CheckValidURL(originalURL); err != nil {
			return
		}

		URLs = append(URLs, model.New(s.getNewPrefix(), originalURL))
	}

	// Пытаемся сохранить данные в базе
	err = s.Rep.SetMany(URLs)
	if err != nil {
		return
	}

	mapURLs = make(map[string]string)
	for _, URL := range URLs {
		shortURL, err := s.GetURLFromPrefix(URL.Prefix)
		if err != nil {
			return nil, err
		}

		mapURLs[URL.OriginalURL] = shortURL
	}

	return
}

// getNewPrefix - Получение нового префикса
func (s Shortener) getNewPrefix() string {
	return rand.Text()
}

// GetURLFromPrefix - Получение сокращённого url по префиксу
func (s Shortener) GetURLFromPrefix(prefix string) (string, error) {
	return url.JoinPath(config.Server.Scheme+"://", config.Server.Host, prefix)
}
