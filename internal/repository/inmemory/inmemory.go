package inmemory

import (
	"encoding/json"
	"maps"
	"slices"
	"sync"

	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/pkg/errors"
)

// InMemory - хранение данных в памяти
type InMemory struct {
	sync.RWMutex
	data map[string]model.URL
}

// New - Конструктор
func New() *InMemory {
	return &InMemory{
		data: make(map[string]model.URL),
	}
}

// Set - Фиксируем originalURL за значением prefix
func (im *InMemory) Set(URL model.URL) error {
	// Защищаем map от одновременной записи из разных горутин
	im.Lock()
	defer im.Unlock()

	im.data[URL.Prefix] = URL

	return nil
}

// Get - получение ссылки на редирект по префиксу
func (im *InMemory) Get(prefix string) (model.URL, error) {
	// Защищаем map от одновременного чтения из разных горутин
	im.RLock()
	defer im.RUnlock()

	if item, ok := im.data[prefix]; ok {
		return item, nil
	}

	return model.URL{}, errors.ErrURLNotFound
}

// FindFromOriginal - Поиск prefix по originalURL
func (im *InMemory) FindFromOriginal(originalURL string) (model.URL, error) {
	// Защищаем map от одновременного чтения из разных горутин
	im.RLock()
	defer im.RUnlock()

	for _, item := range im.data {
		if item.OriginalURL == originalURL {
			return item, nil
		}
	}

	return model.URL{}, errors.ErrURLNotFound
}

// String - приведение структуры к строке
func (im *InMemory) String() string {
	// Защищаем map от одновременного чтения из разных горутин
	im.RLock()
	defer im.RUnlock()

	text := "[\r\n"

	for _, item := range im.data {
		text += item.Prefix + " => " + item.OriginalURL + "\r\n"
	}

	text += "]\r\n"

	return text
}

// MarshalJSON приводит значения InMemory к формату json.
func (im *InMemory) MarshalJSON() ([]byte, error) {
	// Преобразовываем мапу в слайс
	slice := slices.Collect(maps.Values(im.data))

	return json.Marshal(slice)
}
