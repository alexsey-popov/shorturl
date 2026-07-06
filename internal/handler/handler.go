package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrInvalidContentType = errors.New("некорректный тип содержимого запроса")
	ErrInvalidRequest     = errors.New("некорректный запрос")
	ErrEmptyBatch         = errors.New("пустая пачка данных")
)

// shortener - Хранилище ссылок
var shortener service.Shortener

// UseDBRepository использование базы данных
func UseDBRepository(baseURL string, db *sql.DB) {
	shortener = service.NewDBShortener(baseURL, db)
}

// UseFileRepository использование файлового хранилища
func UseFileRepository(baseURL string, filepath string) (err error) {
	shortener, err = service.NewFileShortener(baseURL, filepath)

	return
}

// UseMemoryRepository использование хранилища в памяти
func UseMemoryRepository(baseURL string) {
	shortener = service.NewMemoryShortener(baseURL)
}

// getUserID - Получаем id пользователя из контекста запроса
func getUserID(r *http.Request) (userID string) {
	if ctxUserID, ok := r.Context().Value("user_id").(string); ok {
		userID = ctxUserID
	}

	return userID
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
		http.Error(rw, ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем тело запроса (ожидается ссылка)
	originalURL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем сокращённую ссылку
	shortURL, err := shortener.Add(string(originalURL), getUserID(r))
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

// HandlePostJSON - Обработчик для запроса Post /api/shorten
func HandlePostJSON(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.JSON {
		http.Error(rw, ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем URL из json
	request := struct {
		URL string `json:"url"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// response - Структура ответа
	type response struct {
		Result string `json:"result"`
	}

	// Получаем сокращённую ссылку
	shortURL, err := shortener.Add(request.URL, getUserID(r))
	if err != nil {
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		if conflictErr, ok := errors.AsType[*errors2.OriginalURLConflictError](err); ok {
			diffShortURL, err2 := shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err2 == nil {

				// Подготавливаем json ответ
				responseJSON, err3 := json.Marshal(
					response{
						Result: diffShortURL,
					},
				)
				if err3 == nil {
					rw.Header().Set("Content-Type", contentType.JSON)
					rw.WriteHeader(http.StatusConflict)
					rw.Write(responseJSON)

					return
				}
			}
		}

		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Подготавливаем json ответ
	responseJSON, err := json.Marshal(
		response{
			Result: shortURL,
		},
	)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", contentType.JSON)
	rw.WriteHeader(http.StatusCreated)
	rw.Write(responseJSON)
}

// HandlePostBatch - Обработчик для запроса Post /api/shorten/batch (массовое создание)
func HandlePostBatch(rw http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.JSON {
		http.Error(rw, ErrInvalidContentType.Error(), http.StatusBadRequest)
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
		http.Error(rw, ErrEmptyBatch.Error(), http.StatusBadRequest)
		return
	}

	// Создаём слайс оригинальных ссылок и заполняем его
	originalURLs := make([]string, 0, len(requestItems))
	for _, item := range requestItems {
		originalURLs = append(originalURLs, item.OriginalURL)
	}

	// Передаём слайс оригинальных ссылок на создание
	mapURLs, err := shortener.AddMany(originalURLs, getUserID(r))
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
	http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
}

// HandleGetPing - Обработчик Get запроса /ping
func HandleGetPing(sugar *zap.SugaredLogger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем соединение и в случае ошибки - пишем в чём дело
		if err := shortener.Rep.Ping(); err != nil {
			sugar.Error(err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// HandleGetUserURLs - обработчик для запроса api/user/urls
func HandleGetUserURLs(sugar *zap.SugaredLogger) func(rw http.ResponseWriter, r *http.Request) {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Некорректный content-type - ошибка
		if r.Header.Get("Content-Type") != contentType.JSON {
			sugar.Error(ErrInvalidContentType)
			http.Error(rw, ErrInvalidContentType.Error(), http.StatusBadRequest)
			return
		}

		URLs, err := shortener.Rep.FindFromUserID(getUserID(r))
		if err != nil {
			sugar.Error(err)
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		// Если записей не нашли - возвращаем StatusNoContent
		if len(URLs) == 0 {
			sugar.Error("no content")
			http.Error(rw, http.StatusText(http.StatusNoContent), http.StatusNoContent)
			return
		}

		type responseItem struct {
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
		}
		responseItems := make([]responseItem, len(URLs))

		for i, URL := range URLs {

			shortURL, err := shortener.GetURLFromPrefix(URL.Prefix)
			if err != nil {
				sugar.Error(err)
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}

			responseItems[i] = responseItem{
				ShortURL:    shortURL,
				OriginalURL: URL.OriginalURL,
			}
		}

		// Подготавливаем json ответ
		response, err := json.Marshal(responseItems)
		if err != nil {
			sugar.Error(err)
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		rw.Header().Set("Content-Type", contentType.JSON)
		rw.WriteHeader(http.StatusOK)
		rw.Write(response)
	})
}
