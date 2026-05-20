package inmemory

import "github.com/alexsey-popov/shorturl/pkg/errors"

// InMemory - хранение данных в памяти
type InMemory map[string]string

// Set - Фиксируем originalURL за значением prefix
func (im InMemory) Set(prefix string, originalURL string) (err error) {
	im[prefix] = originalURL

	return nil
}

// Get - получение ссылки на редирект по префиксу
func (im InMemory) Get(prefix string) (string, error) {
	if originalURL, ok := im[prefix]; ok {
		return originalURL, nil
	}

	return "", errors.ErrURLNotFound
}

// FindFromOriginal - Поиск prefix по originalURL
func (im InMemory) FindFromOriginal(originalURL string) (string, error) {
	for prefix, value := range im {
		if value == originalURL {
			return prefix, nil
		}
	}

	return "", errors.ErrURLNotFound
}

// String - приведение структуры к строке
func (im InMemory) String() string {
	text := "[\r\n"

	for prefix, originalURL := range im {
		text += prefix + " => " + originalURL + "\r\n"
	}

	text += "]\r\n"

	return text
}
