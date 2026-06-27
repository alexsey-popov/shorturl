package config

import (
	"net"
	"net/url"

	errorsPkg "github.com/alexsey-popov/shorturl/pkg/errors"
)

// ServerConf - Структура для хранения конфигурации
type ServerConf struct {
	Scheme     string
	Host       string
	NetAddress string
	FilePath   string
	DSN        string
}

// SetServerAddress - изменение адреса сервера
func (s *ServerConf) SetServerAddress(value string) error {
	// Проверка корректности указанного ip
	serverAddr, err := net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return errorsPkg.ErrInvalidAddress
	}

	s.NetAddress = serverAddr.String()

	return nil
}

// SetBaseURL - изменение URL сервера
func (s *ServerConf) SetBaseURL(value string) error {
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return err
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return errorsPkg.ErrInvalidAddress
	}

	s.Scheme = parsedURL.Scheme
	s.Host = parsedURL.Host

	return nil
}

// SetFilePath - изменение файла хранения данных
func (s *ServerConf) SetFilePath(value string) error {
	s.FilePath = value

	return nil
}

// SetDBDsn - изменение файла хранения данных
func (s *ServerConf) SetDSN(value string) error {
	s.DSN = value

	return nil
}
