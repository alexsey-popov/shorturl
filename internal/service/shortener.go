package service

import (
	"crypto/rand"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
)

// Repository - Интерфейс для хранилищ
type Repository interface {
	// Set - Фиксирование originalURL за значением prefix
	Set(prefix string, originalURL string) (err error)
	// Get - Получение originalURL по значению prefix
	Get(prefix string) (originalURL string, err error)
	// FindFromOriginal - Поиск prefix по значению originalURL
	FindFromOriginal(originalURL string) (prefix string, err error)
}

// Shortener - Сервис для сокращения ссылок
type Shortener struct {
	Repository
}

// NewShortener - Конструктор для Shortener
func NewShortener() Shortener {
	return Shortener{
		Repository: inmemory.New(),
	}
}

// Add - Добавление новой ссылки
func (s Shortener) Add(originalURL string) (string, error) {
	// Проверяем является ли переданная строка корректной ссылкой
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return "", err
	}

	// Проверяем существование originalURL в базе
	prefix, err := s.FindFromOriginal(originalURL)

	// Если originalURL не нашли - добавляем новый элемент
	if err != nil {
		prefix = s.getNewPrefix()

		if err = s.Set(prefix, originalURL); err != nil {
			return "", err
		}
	}

	return s.GetURLFromPrefix(prefix), nil
}

// getNewPrefix - Получение нового префикса
func (s Shortener) getNewPrefix() string {
	return rand.Text()
}

// GetURLFromPrefix - Получение сокращённого url по префиксу
func (s Shortener) GetURLFromPrefix(prefix string) string {
	shortURL, err := url.JoinPath(config.Server.Scheme+"://", config.Server.Host, prefix)
	if err != nil {
		panic(err.Error())
	}

	return shortURL
}
