package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/go-chi/chi/v5"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// shortener - Хранилище ссылок (по умолчанию в памяти)
var shortener service.Shortener = service.NewMemoryShortener()

// UseDBRepository использование базы данных
func UseDBRepository(db *sql.DB) {
	shortener = service.NewDBShortener(db)
}

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
		http.Error(rw, errors2.ErrInvalidContentType.Error(), http.StatusBadRequest)
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
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		var conflictErr *errors2.OriginalURLConflictError
		if errors.As(err, &conflictErr) {
			diffShortURL, err2 := shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err2 == nil {
				rw.Header().Set("Content-Type", contentType.Plain)
				rw.WriteHeader(http.StatusConflict)
				rw.Write([]byte(diffShortURL))

				return
			}
		}

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
		http.Error(rw, errors2.ErrInvalidContentType.Error(), http.StatusBadRequest)
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
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		var conflictErr *errors2.OriginalURLConflictError
		if errors.As(err, &conflictErr) {
			diffShortURL, err2 := shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err2 == nil {
				rw.Header().Set("Content-Type", contentType.Plain)
				rw.WriteHeader(http.StatusConflict)
				rw.Write([]byte(diffShortURL))

				return
			}
		}

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

// HandlePostBatch - Обработчик для запроса Post /api/shorten/batch (массовое создание)
func HandlePostBatch(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.JSON {
		http.Error(rw, errors2.ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	type requestItem struct {
		Prefix      string `json:"correlation_id"`
		OriginalURL string `json:"original_url"`
	}

	// Читаем URL из json
	requestItems := make([]requestItem, 0)
	if err := json.NewDecoder(r.Body).Decode(&requestItems); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if len(requestItems) == 0 {
		http.Error(rw, errors2.ErrEmptyBatch.Error(), http.StatusBadRequest)
		return
	}

	// Создаём слайс оригинальных ссылок и заполняем его
	originalURLs := make([]string, 0, len(requestItems))
	for _, item := range requestItems {
		originalURLs = append(originalURLs, item.OriginalURL)
	}

	// Передаём слайс оригинальных ссылок на создание
	mapURLs, err := shortener.AddMany(originalURLs)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	type responseItem struct {
		Prefix   string `json:"correlation_id"`
		ShortURL string `json:"short_url"`
	}
	responseItems := make([]responseItem, 0, len(requestItems))

	for _, item := range requestItems {
		responseItems = append(responseItems, responseItem{Prefix: item.Prefix, ShortURL: mapURLs[item.OriginalURL]})
	}

	// Подготавливаем json ответ
	response, err := json.Marshal(responseItems)
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
	http.Error(w, errors2.ErrInvalidRequest.Error(), http.StatusBadRequest)
}

// HandleGetPing - Обработчик Get запроса /ping
func HandleGetPing(w http.ResponseWriter, r *http.Request) {
	// Создаём объект взаимодействия с базой
	db, err := sql.Open("pgx", config.Server.DSN)

	// Если мы не встретили ошибку, то обещаем закрыть соединение и пингуем его
	if err == nil {
		defer db.Close()

		err = db.Ping()

		// Если при мы не получили ошибку, то возвращаем статус 200
		if err == nil {
			w.WriteHeader(http.StatusOK)

			return
		}
	}

	// Если мы дошли до этого этапа - значит где то была ошибка
	w.WriteHeader(http.StatusInternalServerError)
	log.Println("error is", err)
	w.Write([]byte(err.Error()))

	return
}
