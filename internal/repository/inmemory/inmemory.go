package inmemory

import (
	"sync"

	"github.com/alexsey-popov/shorturl/pkg/errors"
)

// InMemory - хранение данных в памяти
type InMemory struct {
	sync.RWMutex
	data map[string]string
}

// New - Конструктор
func New() *InMemory {
	return &InMemory{
		data: make(map[string]string),
	}
}

// Set - Фиксируем originalURL за значением prefix
func (im *InMemory) Set(prefix string, originalURL string) (err error) {
	// Защищаем map от одновременной записи из разных горутин
	im.Lock()
	defer im.Unlock()

	im.data[prefix] = originalURL

	return nil
}

// Get - получение ссылки на редирект по префиксу
func (im *InMemory) Get(prefix string) (string, error) {
	// Защищаем map от одновременного чтения из разных горутин
	im.RLock()
	defer im.RUnlock()

	if originalURL, ok := im.data[prefix]; ok {
		return originalURL, nil
	}

	return "", errors.ErrURLNotFound
}

// FindFromOriginal - Поиск prefix по originalURL
func (im *InMemory) FindFromOriginal(originalURL string) (string, error) {
	// Защищаем map от одновременного чтения из разных горутин
	im.RLock()
	defer im.RUnlock()

	for prefix, value := range im.data {
		if value == originalURL {
			return prefix, nil
		}
	}

	return "", errors.ErrURLNotFound
}

// String - приведение структуры к строке
func (im *InMemory) String() string {
	// Защищаем map от одновременного чтения из разных горутин
	im.RLock()
	defer im.RUnlock()

	text := "[\r\n"

	for prefix, originalURL := range im.data {
		text += prefix + " => " + originalURL + "\r\n"
	}

	text += "]\r\n"

	return text
}
