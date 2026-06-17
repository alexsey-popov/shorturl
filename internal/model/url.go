package model

import "github.com/google/uuid"

type URL struct {
	UUID        string `json:"uuid"`
	Prefix      string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func New(prefix string, originalURL string) URL {
	return URL{
		UUID:        uuid.NewString(),
		Prefix:      prefix,
		OriginalURL: originalURL,
	}
}
