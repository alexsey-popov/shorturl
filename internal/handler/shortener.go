package handler

import (
	"crypto/rand"
	"errors"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/config"
)

var ErrUrlNotFound = errors.New("url не найден")

// Shortener - Репозиторий для сокращённых ссылок
//
// Внутри методов используются следующие переменные:
// originalUrl - Оригинальная ссылка, которую мы хотим сократить.
// prefix - идентификатор внутри map.
// shortUrl - сокращённая ссылка в формате http://localhost:8080/prefix.
type Shortener map[string]string

// Add - Добавление новой ссылки
func (s Shortener) Add(originalUrl string) string {
	// Сначала смотрим есть ли указанная originalUrl в общем списке
	if shortUrl, err := s.FindFromOriginal(originalUrl); err == nil {
		return shortUrl
	}

	prefix := rand.Text()
	s[prefix] = originalUrl

	return s.GetUrlFromPrefix(prefix)
}

// Get - получение ссылки на редирект по префиксу
func (s Shortener) Get(prefix string) (string, error) {
	if originalUrl, ok := s[prefix]; ok {
		return originalUrl, nil
	}

	return "", ErrUrlNotFound
}

// FindFromOriginal - Поиск сокращенного url по полному
func (s Shortener) FindFromOriginal(originalUrl string) (string, error) {
	for prefix, value := range s {
		if value == originalUrl {
			return s.GetUrlFromPrefix(prefix), nil
		}
	}

	return "", ErrUrlNotFound
}

// GetUrlFromPrefix - Получение сокращённой url по префиксу
func (s Shortener) GetUrlFromPrefix(prefix string) string {
	shortUrl, err := url.JoinPath(config.Protocol, config.ServerAddr, prefix)
	if err != nil {
		panic(err.Error())
	}

	return shortUrl
}

// String - приведение структуры к строке
func (s Shortener) String() string {
	text := "[\r\n"

	for key, value := range s {
		text += key + " => " + value + "\r\n"
	}

	text += "]\r\n"

	return text
}
