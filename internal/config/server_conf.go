package config

import (
	"errors"
	"net"
	"net/url"
	"os"

	errorsPkg "github.com/alexsey-popov/shorturl/pkg/errors"
)

// ServerConf - Структура для хранения конфигурации
type ServerConf struct {
	Scheme          string
	Host            string
	NetAddress      string
	ContentType     string
	ContentTypeJson string
	FilePath        string
}

// SetServerAddress - изменение адреса сервера
func (s ServerConf) SetServerAddress(value string) error {
	// Проверка корректности указанного ip
	serverAddr, err := net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return errorsPkg.ErrInvalidAddress
	}

	Server.NetAddress = serverAddr.String()

	return nil
}

// SetBaseURL - изменение URL сервера
func (s ServerConf) SetBaseURL(value string) error {
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return err
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errorsPkg.ErrInvalidAddress
	}

	Server.Scheme = parsedURL.Scheme
	Server.Host = parsedURL.Host

	return nil
}

// SetFilePath - изменение файла хранения данных
func (s ServerConf) SetFilePath(value string) error {
	// Проверка корректности пути файла
	_, err := os.Stat(value)
	// Исключаем ошибку ErrNotExist. Если файла на существует - мы его создадим
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errorsPkg.ErrInvalidFilePath
	}

	Server.FilePath = value

	return nil
}
