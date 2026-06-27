package infile

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
)

var ErrOriginalURLConflict = errors.New("эта ссылка уже сокращалась ранее")

// InFile - хранение данных в файле.
// inmemory.InMemory вынесен в отдельный атрибут для того, чтобы у файла был свой мьютекс
type InFile struct {
	mu       sync.RWMutex
	filename string
	data     *inmemory.InMemory
}

// Прокизываем MarshalJSON внутрь структуры InMemory
func (rep *InFile) MarshalJSON() ([]byte, error) {
	return json.Marshal(rep.data)
}

// Set - Сохраняем originalURL за значением prefix
func (rep *InFile) Set(URL model.URL) error {
	// Сначала пытаемся найти оригинальную ссылку в файле
	// Если нашли - возвращаем специфическую ошибку с данными по существующей ссылке
	diffURL, err := rep.FindFromOriginal(URL.OriginalURL)
	if err == nil {
		return errors2.NewErrOriginalURLConflict(diffURL, ErrOriginalURLConflict)
	}

	// Защищаем файл от конкурентного доступа
	rep.mu.Lock()
	defer rep.mu.Unlock()

	// Записываем данные в память
	err = rep.data.Set(URL)
	if err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(rep.data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(rep.filename, jsonData, 0666)
	if err != nil {
		return err
	}

	return nil
}

// SetMany - Сохраняем несколько ссылок
func (rep *InFile) SetMany(URLs []model.URL) error {
	// Защищаем файл от конкурентного доступа
	rep.mu.Lock()
	defer rep.mu.Unlock()

	for _, URL := range URLs {
		// Записываем данные в память
		err := rep.data.Set(URL)
		if err != nil {
			return err
		}
	}

	jsonData, err := json.MarshalIndent(rep.data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(rep.filename, jsonData, 0666)
	if err != nil {
		return err
	}

	return nil
}

// Get - получение ссылки на редирект по префиксу
func (rep *InFile) Get(prefix string) (model.URL, error) {
	return rep.data.Get(prefix)
}

// FindFromOriginal - Поиск среди загруженных в память данных
func (rep *InFile) FindFromOriginal(originalURL string) (item model.URL, err error) {
	return rep.data.FindFromOriginal(originalURL)
}

// New - Конструктор
func New(filename string) (*InFile, error) {
	rep := InFile{
		data:     inmemory.New(),
		filename: filename,
	}

	// Читаем данные из файла
	bytesData, err := os.ReadFile(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// Файл не пустой - заполняем репозиторий
	if len(bytesData) > 0 {
		data := make([]model.URL, 0)
		if err = json.Unmarshal(bytesData, &data); err != nil {
			return nil, err
		}

		for _, item := range data {
			// Используем rep.data.Set, а не rep.Set для того, чтобы с каждой итерацией не перезаписывать файл
			if err = rep.data.Set(item); err != nil {
				return nil, err
			}
		}
	}

	return &rep, nil
}

// Ping - проверка соединения (считаем, что оно всегда есть)
func (rep *InFile) Ping() error {
	return nil
}
