package errors

import "errors"

var (
	ErrURLNotFound        = errors.New("url не найден")
	ErrInvalidContentType = errors.New("некорректный тип содержимого запроса")
	ErrInvalidRequest     = errors.New("некорректный запрос")
	ErrInvalidAddress     = errors.New("некорректный адрес")
	ErrInvalidFilePath    = errors.New("некорректный путь до файла")
)
