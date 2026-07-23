package inmemory

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/alexsey-popov/shorturl/internal/model"
)

var ErrURLNotFound = errors.New("url не найден")

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

// Set - Сохраняем originalURL за значением prefix
func (rep *InMemory) Set(url model.URL) error {
	// Защищаем map от одновременной записи из разных горутин
	rep.mu.Lock()
	defer rep.mu.Unlock()

	rep.data[url.Prefix] = url

	return nil
}

// SetMany - Сохраняем несколько ссылок
func (rep *InMemory) SetMany(urls []model.URL) error {
	// Записываем данные в память
	for _, url := range urls {
		err := rep.Set(url)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteManyFromUserId Массовое удаление ссылок принадлежащих пользователю
func (rep *InMemory) DeleteManyFromUserId(prefixes []string, userID string) error {
	//Получаем канал с префиксами через генератор
	inCh := func(prefixes []string) chan string {
		outCh := make(chan string)
		go func() {
			defer close(outCh)
			for _, prefix := range prefixes {
				outCh <- prefix
			}
		}()

		return outCh
	}(prefixes)

	// Обрабатывать будем в 4 потока
	chs := make([]chan error, 4)

	// Запускаем каждый поток
	for i := range chs {
		chs[i] = func(in chan string) chan error {
			outCh := make(chan error)
			go func() {
				defer close(outCh)
				for prefix := range inCh {
					if err := rep.DeleteUserUrlFromPrefix(prefix, userID); err != nil {
						outCh <- fmt.Errorf("%s: %w", prefix, err)
						continue
					}
					outCh <- nil
				}
			}()

			return outCh
		}(inCh)
	}

	var resErr error
	// Читаем все данные из канала
	for err := range fanIn(chs) {
		if err != nil && resErr == nil {
			resErr = err
		}
	}

	return resErr
}

// DeleteUserUrlFromPrefix - удаление пользовательской ссылки по префиксу (с проверкой на принадлежность пользователю)
func (rep *InMemory) DeleteUserUrlFromPrefix(prefix, userID string) error {
	// Ищем ссылку
	url, err := rep.Get(prefix)
	if err != nil {
		return err
	}

	// Проверяем, что ссылка не была удалена ранее + принадлежность пользователю
	if url.IsDeleted || url.UserID != userID {
		return errors.New("ссылка уже удалена или не принадлежит пользователю")
	}

	// Помечаем ссылку как удалённую
	url.IsDeleted = true
	if err = rep.Set(url); err != nil {
		return err
	}

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

	return model.URL{}, ErrURLNotFound
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

	return model.URL{}, ErrURLNotFound
}

// FindFromUserID - поиск записей по id пользователя
func (rep *InMemory) FindFromUserID(userID string) (urls []model.URL, err error) {
	// Защищаем map от одновременного чтения из разных горутин
	rep.mu.RLock()
	defer rep.mu.RUnlock()

	for _, item := range rep.data {
		if item.UserID == userID {

			urls = append(urls, item)
		}
	}

	return urls, nil
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

// Ping - проверка соединения (считаем, что оно всегда есть)
func (rep *InMemory) Ping() error {
	return nil
}

// fanIn принимает несколько каналов, в которых итоговые значения
func fanIn[T any](chs []chan T) chan T {
	var wg sync.WaitGroup
	outCh := make(chan T)

	// определяем функцию output для каждого канала в chs
	// функция output копирует значения из канала с в канал outCh, пока с не будет закрыт
	output := func(c chan T) {
		for n := range c {
			outCh <- n
		}
		wg.Done()
	}

	// добавляем в группу столько горутин, сколько каналов пришло в fanIn
	wg.Add(len(chs))
	// перебираем все каналы, которые пришли и отправляем каждый в отдельную горутину
	for _, c := range chs {
		go output(c)
	}

	// запускаем горутину для закрытия outCh после того, как все горутины отработают
	go func() {
		wg.Wait()
		close(outCh)
	}()

	// возвращаем общий канал
	return outCh
}
