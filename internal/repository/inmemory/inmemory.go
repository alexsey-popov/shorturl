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
	mu   sync.RWMutex
	data map[string]model.URL
}

// New - Конструктор
func New() *InMemory {
	return &InMemory{
		data: make(map[string]model.URL),
	}
}

// Set - Фиксируем originalURL за значением prefix
func (rep *InMemory) Set(URL model.URL) error {
	// Защищаем map от одновременной записи из разных горутин
	rep.mu.Lock()
	defer rep.mu.Unlock()

	rep.data[URL.Prefix] = URL

	return nil
}

// Get - получение ссылки на редирект по префиксу
func (rep *InMemory) Get(prefix string) (model.URL, error) {
	// Защищаем map от одновременного чтения из разных горутин
	rep.mu.RLock()
	defer rep.mu.RUnlock()

	if item, ok := rep.data[prefix]; ok {
		return item, nil
	}

	return model.URL{}, errors.ErrURLNotFound
}

// FindFromOriginal - Поиск prefix по originalURL
func (rep *InMemory) FindFromOriginal(originalURL string) (model.URL, error) {
	// Защищаем map от одновременного чтения из разных горутин
	rep.mu.RLock()
	defer rep.mu.RUnlock()

	for _, item := range rep.data {
		if item.OriginalURL == originalURL {
			return item, nil
		}
	}

	return model.URL{}, errors.ErrURLNotFound
}

// String - приведение структуры к строке
func (rep *InMemory) String() string {
	// Защищаем map от одновременного чтения из разных горутин
	rep.mu.RLock()
	defer rep.mu.RUnlock()

	text := "[\r\n"

	for _, item := range rep.data {
		text += item.Prefix + " => " + item.OriginalURL + "\r\n"
	}

	text += "]\r\n"

	return text
}

// MarshalJSON приводит значения InMemory к формату json.
func (rep *InMemory) MarshalJSON() ([]byte, error) {
	// Защищаем map от одновременного чтения из разных горутин
	rep.mu.RLock()
	defer rep.mu.RUnlock()

	// Преобразовываем мапу в слайс
	slice := slices.Collect(maps.Values(rep.data))

	return json.Marshal(slice)
}
