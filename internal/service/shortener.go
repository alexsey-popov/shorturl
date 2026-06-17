package service

import (
	"crypto/rand"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/repository/infile"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
)

// Repository - Интерфейс для хранилищ
type Repository interface {
	// Set - Фиксирование originalURL за значением prefix
	Set(URL model.URL) error
	// Get - Получение originalURL по значению prefix
	Get(prefix string) (URL model.URL, err error)
	// FindFromOriginal - Поиск prefix по значению originalURL
	FindFromOriginal(originalURL string) (item model.URL, err error)
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

// Add - Добавление новой ссылки
func (s Shortener) Add(originalURL string) (string, error) {
	// Проверяем является ли переданная строка корректной ссылкой
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return "", err
	}

	// Проверяем существование originalURL в базе
	URL, err := s.Rep.FindFromOriginal(originalURL)

	// Если originalURL не нашли - добавляем новый элемент
	if err != nil {
		URL = model.New(s.getNewPrefix(), originalURL)

		err = s.Rep.Set(URL)

		if err != nil {
			return "", err
		}
	}

	return s.GetURLFromPrefix(URL.Prefix)
}

// getNewPrefix - Получение нового префикса
func (s Shortener) getNewPrefix() string {
	return rand.Text()
}

// GetURLFromPrefix - Получение сокращённого url по префиксу
func (s Shortener) GetURLFromPrefix(prefix string) (string, error) {
	return url.JoinPath(config.Server.Scheme+"://", config.Server.Host, prefix)
}
