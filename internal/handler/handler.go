package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alexsey-popov/shorturl/internal/audit"
	"github.com/alexsey-popov/shorturl/internal/auth"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var (
	// ErrInvalidContentType - ошибка некорректного типа содержимого запроса.
	ErrInvalidContentType = errors.New("некорректный тип содержимого запроса")
	// ErrInvalidRequest - ошибка некорректного запроса.
	ErrInvalidRequest = errors.New("некорректный запрос")
	// ErrEmptyBatch - ошибка пустого пакета данных.
	ErrEmptyBatch = errors.New("пустая пачка данных")
)

// DeleteTask - Задача на удаление ссылок пользователя.
type DeleteTask struct {
	UserID   string
	Prefixes []string
}

// Handler - структура HTTP-хендеров приложения.
type Handler struct {
	sugar        *zap.SugaredLogger
	shortener    service.Shortener
	delCh        chan DeleteTask
	auditManager *audit.Publisher
}

// NewHandler - конструктор Handler.
func NewHandler(
	sugar *zap.SugaredLogger,
	shortener service.Shortener,
	delCh chan DeleteTask,
	auditManager *audit.Publisher,
) Handler {
	return Handler{
		sugar:        sugar,
		shortener:    shortener,
		delCh:        delCh,
		auditManager: auditManager,
	}
}

// getUserID - Получаем id пользователя из контекста запроса
func (h Handler) getUserID(r *http.Request) (userID string, ok bool) {
	return auth.GetUserId(r.Context())
}

// HandleGet - Обработчик Get запроса
func (h Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	prefix := chi.URLParam(r, "id")

	url, err := h.shortener.Rep.Get(prefix)
	if err != nil {
		h.sugar.Error(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Если ссылка помечена как удалённая - вместо редиректа выдаём 410 статус
	if url.IsDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}

	userID, _ := h.getUserID(r)
	if h.auditManager != nil {
		h.auditManager.Notify(context.TODO(), audit.Event{
			Timestamp: time.Now().Unix(),
			Action:    "follow",
			UserID:    userID,
			URL:       url.OriginalURL,
		})
	}

	http.Redirect(w, r, url.OriginalURL, http.StatusTemporaryRedirect)
}

// HandlePost - Обработчик Post запроса
func (h Handler) HandlePost(w http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.Plain {
		h.sugar.Error(ErrInvalidContentType)
		http.Error(w, ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем тело запроса (ожидается ссылка)
	originalURL, err := io.ReadAll(r.Body)
	if err != nil {
		h.sugar.Errorf("ошибка при чтении тела запроса: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Получаем id пользователя
	userId, ok := h.getUserID(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем сокращённую ссылку
	shortURL, err := h.shortener.Add(string(originalURL), userId)
	if err != nil {
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		var conflictErr *errors2.OriginalURLConflictError
		if errors.As(err, &conflictErr) {
			diffShortURL, err2 := h.shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err2 == nil {
				if h.auditManager != nil {
					h.auditManager.Notify(context.TODO(), audit.Event{
						Timestamp: time.Now().Unix(),
						Action:    "shorten",
						UserID:    userId,
						URL:       string(originalURL),
					})
				}

				w.Header().Set("Content-Type", contentType.Plain)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(diffShortURL))

				return
			}
		}

		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if h.auditManager != nil {
		h.auditManager.Notify(context.TODO(), audit.Event{
			Timestamp: time.Now().Unix(),
			Action:    "shorten",
			UserID:    userId,
			URL:       string(originalURL),
		})
	}

	w.Header().Set("Content-Type", contentType.Plain)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// HandlePostJSON - Обработчик для запроса Post /api/shorten
func (h Handler) HandlePostJSON(w http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.JSON {
		h.sugar.Error(ErrInvalidContentType)
		http.Error(w, ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	// Читаем URL из json
	request := struct {
		URL string `json:"url"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		err = fmt.Errorf("ошибка парсинга json: %w", err)
		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// response - Структура ответа
	type response struct {
		Result string `json:"result"`
	}

	// Получаем id пользователя
	userId, ok := h.getUserID(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем сокращённую ссылку
	shortURL, err := h.shortener.Add(request.URL, userId)
	if err != nil {
		// Если при добавлении сокр. ссылки мы получили ошибку - возможно это была ошибка уникальности
		// и мы можем отдать пользователю уже существующую shortURL
		if conflictErr, ok := errors.AsType[*errors2.OriginalURLConflictError](err); ok {
			diffShortURL, err2 := h.shortener.GetURLFromPrefix(conflictErr.DiffURL.Prefix)
			if err2 == nil {

				// Подготавливаем json ответ
				responseJSON, err3 := json.Marshal(
					response{
						Result: diffShortURL,
					},
				)
				if err3 == nil {
					if h.auditManager != nil {
						h.auditManager.Notify(context.TODO(), audit.Event{
							Timestamp: time.Now().Unix(),
							Action:    "shorten",
							UserID:    userId,
							URL:       request.URL,
						})
					}

					w.Header().Set("Content-Type", contentType.JSON)
					w.WriteHeader(http.StatusConflict)
					w.Write(responseJSON)

					return
				}
			}
		}

		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if h.auditManager != nil {
		h.auditManager.Notify(context.TODO(), audit.Event{
			Timestamp: time.Now().Unix(),
			Action:    "shorten",
			UserID:    userId,
			URL:       request.URL,
		})
	}

	// Подготавливаем json ответ
	responseJSON, err := json.Marshal(
		response{
			Result: shortURL,
		},
	)
	if err != nil {
		err = fmt.Errorf("ошибка при сериализации в json: %w", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType.JSON)
	w.WriteHeader(http.StatusCreated)
	w.Write(responseJSON)
}

// HandlePostBatch - Обработчик для запроса Post /api/shorten/batch (массовое создание)
func (h Handler) HandlePostBatch(w http.ResponseWriter, r *http.Request) {
	// Некорректный content-type - ошибка
	if r.Header.Get("Content-Type") != contentType.JSON {
		h.sugar.Error(ErrInvalidContentType)
		http.Error(w, ErrInvalidContentType.Error(), http.StatusBadRequest)
		return
	}

	type requestItem struct {
		Prefix      string `json:"correlation_id"`
		OriginalURL string `json:"original_url"`
	}

	// Читаем URL из json
	requestItems := make([]requestItem, 0)
	if err := json.NewDecoder(r.Body).Decode(&requestItems); err != nil {
		err = fmt.Errorf("ошибка при парсинге json: %w", err)
		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(requestItems) == 0 {
		h.sugar.Error(ErrEmptyBatch)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Создаём слайс оригинальных ссылок и заполняем его
	originalURLs := make([]string, 0, len(requestItems))
	for _, item := range requestItems {
		originalURLs = append(originalURLs, item.OriginalURL)
	}

	// Получаем id пользователя
	userId, ok := h.getUserID(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Передаём слайс оригинальных ссылок на создание
	mapURLs, err := h.shortener.AddMany(originalURLs, userId)
	if err != nil {
		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Подготавливаем ответ
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
		err = fmt.Errorf("ошибка при сериализации в json: %w", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType.JSON)
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}

// HandleFails - обработчик для ошибочных запросов
func (h Handler) HandleFails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType.Plain)
	http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
}

// HandleGetPing - Обработчик Get запроса /ping
func (h Handler) HandleGetPing(w http.ResponseWriter, r *http.Request) {
	// Проверяем соединение и в случае ошибки - пишем в чём дело
	if err := h.shortener.Rep.Ping(); err != nil {
		h.sugar.Error(err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleGetUserURLs - обработчик для Get запроса api/user/urls (массовое создание ссылок)
func (h Handler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, ok := h.getUserID(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	urls, err := h.shortener.Rep.FindFromUserID(userId)
	if err != nil {
		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Если записей не нашли - возвращаем StatusNoContent
	if len(urls) == 0 {
		http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		return
	}

	type responseItem struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	responseItems := make([]responseItem, len(urls))

	for i, url := range urls {

		shortURL, err := h.shortener.GetURLFromPrefix(url.Prefix)
		if err != nil {
			h.sugar.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		responseItems[i] = responseItem{
			ShortURL:    shortURL,
			OriginalURL: url.OriginalURL,
		}
	}

	// Подготавливаем json ответ
	response, err := json.Marshal(responseItems)
	if err != nil {
		h.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType.JSON)
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// HandleDeleteUserURLs - обработчик для Delete запроса api/user/urls (массовое удаление ссылок)
func (h Handler) HandleDeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	prefixes := make([]string, 0)
	if err := json.NewDecoder(r.Body).Decode(&prefixes); err != nil {
		h.sugar.Errorf("ошибка при декодировании json: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(prefixes) == 0 {
		http.Error(w, ErrEmptyBatch.Error(), http.StatusBadRequest)
		return
	}

	// Получаем id пользователя
	userId, ok := h.getUserID(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Отправляем задачу в канал для асинхронного удаления (fan-in паттерн)
	go func() {
		h.delCh <- DeleteTask{
			UserID:   userId,
			Prefixes: prefixes,
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
