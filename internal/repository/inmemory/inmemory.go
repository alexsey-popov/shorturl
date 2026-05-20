package inmemory

import "github.com/alexsey-popov/shorturl/pkg/errors"

// InMemory - хранение данных в памяти
type InMemory map[string]string

// Set - Фиксируем originalUrl за значением prefix
func (im InMemory) Set(prefix string, originalUrl string) (err error) {
	im[prefix] = originalUrl

	return nil
}

// Get - получение ссылки на редирект по префиксу
func (im InMemory) Get(prefix string) (string, error) {
	if originalUrl, ok := im[prefix]; ok {
		return originalUrl, nil
	}

	return "", errors.ErrUrlNotFound
}

// FindFromOriginal - Поиск prefix по originalUrl
func (im InMemory) FindFromOriginal(originalUrl string) (string, error) {
	for prefix, value := range im {
		if value == originalUrl {
			return prefix, nil
		}
	}

	return "", errors.ErrUrlNotFound
}

// String - приведение структуры к строке
func (im InMemory) String() string {
	text := "[\r\n"

	for prefix, originalUrl := range im {
		text += prefix + " => " + originalUrl + "\r\n"
	}

	text += "]\r\n"

	return text
}
