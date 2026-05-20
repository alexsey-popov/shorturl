package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
)

// links - Объект для работы со ссылками
var links = service.NewShortener()

// RouterFunc - Определение хендлера в зависимости от метода запроса
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

// handleGet - Обработчик Get запроса
func handleGet(rw http.ResponseWriter, r *http.Request) {
	// Если не передан id - выводим ошибку
	if r.URL.Path == "/" {
		http.Error(rw, "Отсутствует идентификатор ссылки", http.StatusBadRequest)
		return
	}

	originalURL, err := links.Get(strings.TrimLeft(r.URL.Path, "/"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(rw, r, originalURL, http.StatusTemporaryRedirect)
}

// handlePost - Обработчик Post запроса
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
	originalURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем сокращённую ссылку
	shortURL, err := links.Add(string(originalURL))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", config.ContentType)

	rw.WriteHeader(http.StatusCreated)

	rw.Write([]byte(shortURL))
}
