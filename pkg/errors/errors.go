package errors

import (
	"errors"

	"github.com/alexsey-popov/shorturl/internal/model"
)

var (
	ErrURLNotFound         = errors.New("url не найден")
	ErrInvalidContentType  = errors.New("некорректный тип содержимого запроса")
	ErrInvalidRequest      = errors.New("некорректный запрос")
	ErrInvalidAddress      = errors.New("некорректный адрес")
	ErrInvalidFilePath     = errors.New("некорректный путь до файла")
	ErrEmptyBatch          = errors.New("пустая пачка данных")
	ErrOriginalURLConflict = errors.New("эта ссылка уже сокращалась ранее")
)

// OriginalURLConflictError Кастомная ошибка при ошибке уникальности в момент записи данных. Хранит данные дубликата
type OriginalURLConflictError struct {
	Err     error
	DiffURL model.URL
}

// Error - вывод сообщение об ошибке
func (o OriginalURLConflictError) Error() string {
	return o.Err.Error()
}

// Unwrap - разворачивание ошибки
func (o OriginalURLConflictError) Unwrap() error {
	return o.Err
}

// NewErrOriginalURLConflict - Создание экземпляра ошибки OriginalURLConflictError
func NewErrOriginalURLConflict(diffURL model.URL, err error) error {
	return &OriginalURLConflictError{
		Err:     err,
		DiffURL: diffURL,
	}
}
