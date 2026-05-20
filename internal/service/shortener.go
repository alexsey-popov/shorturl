package service

import (
	"crypto/rand"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
)

// Repository - Интерфейс для хранилищ
type Repository interface {
	// Set - Фиксирование originalUrl за значением prefix
	Set(prefix string, originalUrl string) (err error)
	// Get - Получение originalUrl по значению prefix
	Get(prefix string) (originalUrl string, err error)
	// FindFromOriginal - Поиск prefix по значению originalUrl
	FindFromOriginal(originalUrl string) (prefix string, err error)
}

// Shortener - Сервис для сокращения ссылок
type Shortener struct {
	Repository
}

// NewShortener - Конструктор для Shortener
func NewShortener() Shortener {
	return Shortener{
		Repository: make(inmemory.InMemory),
	}
}

// Add - Добавление новой ссылки
func (s Shortener) Add(originalUrl string) (string, error) {
	// Проверяем является ли переданная строка корректной ссылкой
	if _, err := url.ParseRequestURI(originalUrl); err != nil {
		return "", err
	}

	// Проверяем существование originalUrl в базе
	prefix, err := s.FindFromOriginal(originalUrl)

	// Если originalUrl не нашли - добавляем новый элемент
	if err != nil {
		prefix = s.getNewPrefix()

		if err = s.Set(prefix, originalUrl); err != nil {
			return "", err
		}
	}

	return s.GetUrlFromPrefix(prefix), nil
}

// getNewPrefix - Получение нового префикса
func (s Shortener) getNewPrefix() string {
	return rand.Text()
}

// GetUrlFromPrefix - Получение сокращённого url по префиксу
func (s Shortener) GetUrlFromPrefix(prefix string) string {
	shortUrl, err := url.JoinPath(config.Protocol, config.ServerAddr, prefix)
	if err != nil {
		panic(err.Error())
	}

	return shortUrl
}
