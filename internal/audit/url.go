package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

// URLObserver - Наблюдатель для отправки аудита по HTTP URL
type URLObserver struct {
	url    string
	client *retryablehttp.Client
	log    *zap.SugaredLogger
}

// NewURLObserver - Конструктор наблюдателя URL
func NewURLObserver(l *zap.SugaredLogger, url string) *URLObserver {
	client := retryablehttp.NewClient()
	client.HTTPClient.Timeout = Timeout
	client.Logger = nil

	return &URLObserver{
		url:    url,
		client: client,
		log:    l,
	}
}

// Update - Отправка POST запроса с событием аудита
func (u *URLObserver) Update(ctx context.Context, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		u.log.Errorf("ошибка при сериализации json: %v", err.Error())

		return
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, u.url, bytes.NewBuffer(data))
	if err != nil {
		u.log.Errorf("ошибка при создании запроса: %v", err.Error())

		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		u.log.Errorf("ошибка при отправке запроса: %v", err.Error())

		return
	}
	defer resp.Body.Close()
}

// Закрытие наблюдателя
func (u URLObserver) Close() error {
	return nil
}
