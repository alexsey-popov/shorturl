package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	"github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/go-chi/chi/v5"
)

// shortener - Хранилище ссылок (по умолчанию в памяти)
var shortener service.Shortener = service.NewMemoryShortener()

// UseFileRepository использование файлового хранилища
func UseFileRepository() (err error) {
	shortener, err = service.NewFileShortener()

	return
}

// UseMemoryRepository использование хранилища в памяти
func UseMemoryRepository() {
	shortener = service.NewMemoryShortener()
}

// HandleGet - Обработчик Get запроса
func HandleGet(rw http.ResponseWriter, r *http.Request) {
	prefix := chi.URLParam(r, "id")

	URL, err := shortener.Rep.Get(prefix)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(rw, r, URL.OriginalURL, http.StatusTemporaryRedirect)
}

// HandlePost - Обработчик Post запроса
func HandlePost(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.Plain {
		http.Error(rw, errors.ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем тело запроса (ожидается ссылка)
	originalURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем сокращённую ссылку
	shortURL, err := shortener.Add(string(originalURL))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", contentType.Plain)
	rw.WriteHeader(http.StatusCreated)
	rw.Write([]byte(shortURL))
}

// HandlePostJson - Обработчик для запроса Post /api/shorten
func HandlePostJson(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.JSON {
		http.Error(rw, errors.ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем URL из json
	request := struct {
		Url string `json:"url"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем сокращённую ссылку
	shortURL, err := shortener.Add(request.Url)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Подготавливаем json ответ
	response, err := json.Marshal(
		struct {
			Result string `json:"result"`
		}{
			Result: shortURL,
		},
	)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", contentType.JSON)
	rw.WriteHeader(http.StatusCreated)
	rw.Write(response)
}

// HandleFails - обработчик для ошибочных запросов
func HandleFails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType.Plain)
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(errors.ErrInvalidRequest.Error()))
}
