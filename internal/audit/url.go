package audit

import (
	"bytes"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// URLObserver - Наблюдатель для отправки аудита по HTTP URL
type URLObserver struct {
	url    string
	client *http.Client
	log    *zap.SugaredLogger
}

// NewURLObserver - Конструктор наблюдателя URL
func NewURLObserver(l *zap.SugaredLogger, url string) *URLObserver {
	return &URLObserver{
		url:    url,
		client: &http.Client{Timeout: Timeout},
		log:    l,
	}
}

// Update - Отправка POST запроса с событием аудита
func (u *URLObserver) Update(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		u.log.Errorf("ошибка при сериализации json: %v", err.Error())

		return
	}

	req, err := http.NewRequest(http.MethodPost, u.url, bytes.NewBuffer(data))
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
