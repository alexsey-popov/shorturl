package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/go-chi/chi/v5"
)

// links - Объект для работы со ссылками
var links = service.NewShortener()

// HandleGet - Обработчик Get запроса
func HandleGet(rw http.ResponseWriter, r *http.Request) {
	prefix := chi.URLParam(r, "id")

	originalURL, err := links.Get(prefix)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(rw, r, originalURL, http.StatusTemporaryRedirect)
}

// HandlePost - Обработчик Post запроса
func HandlePost(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != config.Server.ContentType {
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
	shortURL, err := links.Add(string(originalURL))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", config.Server.ContentType)
	rw.WriteHeader(http.StatusCreated)
	rw.Write([]byte(shortURL))
}

// HandlePostJson - Обработчик для запроса Post /api/shorten
func HandlePostJson(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != config.Server.ContentTypeJson {
		http.Error(rw, errors.ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем URL из json
	request := struct {
		Url string `json:"url"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}

	// Получаем сокращённую ссылку
	shortURL, err := links.Add(request.Url)
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
	}

	rw.Header().Set("Content-Type", config.Server.ContentTypeJson)
	rw.WriteHeader(http.StatusCreated)
	rw.Write(response)
}

// HandleFails - обработчик для ошибочных запросов
func HandleFails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", config.Server.ContentType)
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(errors.ErrInvalidRequest.Error()))
}
