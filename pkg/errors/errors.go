package errors

import "errors"

var (
	ErrURLNotFound        = errors.New("url не найден")
	ErrInvalidContentType = errors.New("Некорректный тип содержимого запроса")
	ErrInvalidRequest     = errors.New("Некорректный запрос")
	ErrInvalidAddress     = errors.New("некорректный адрес")
)
