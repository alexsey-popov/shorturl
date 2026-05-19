package handler

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexsey-popov/shorturl/internal/config"
)

// links - Место для хранения ссылок
var links = make(Shortener)

// RouterFunc - определяет хендлер в зависимости от метода запроса
func RouterFunc(rw http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGet(rw, r)
	case http.MethodPost:
		handlePost(rw, r)
	default:
		http.Error(rw, "Метод не поддерживается", http.StatusBadRequest)
	}
}

// handleGet - обработчик Get запроса
func handleGet(rw http.ResponseWriter, r *http.Request) {
	// Если не передан id - выводим ошибку
	if r.URL.Path == "/" {
		http.Error(rw, "Отсутствует идентификатор ссылки", http.StatusBadRequest)
		return
	}

	originalUrl, err := links.Get(strings.TrimLeft(r.URL.Path, "/"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(rw, r, originalUrl, http.StatusTemporaryRedirect)
}

// handlePost - обработчик Post запроса
func handlePost(rw http.ResponseWriter, r *http.Request) {
	// Если передан id - выводим ошибку
	if r.URL.Path != "/" {
		http.Error(rw, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != config.ContentType {
		http.Error(rw, "Некорректный тип содержимого запроса", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса (ожидается ссылка)
	originalUrl, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверка url на корректность
	if _, err = url.ParseRequestURI(string(originalUrl)); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем сокращённую ссылку
	shortUrl := links.Add(string(originalUrl))

	rw.Header().Set("Content-Type", config.ContentType)

	rw.WriteHeader(http.StatusCreated)

	rw.Write([]byte(shortUrl))
}
