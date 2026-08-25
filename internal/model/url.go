package model

import "github.com/google/uuid"

// URL - модель данных для сокращенного URL.
type URL struct {
	UUID        string `json:"uuid"`
	Prefix      string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

// New - создает новый экземпляр URL с сгенерированным UUID.
func New(prefix, originalURL, userID string) URL {
	return URL{
		UUID:        uuid.NewString(),
		Prefix:      prefix,
		OriginalURL: originalURL,
		UserID:      userID,
	}
}
