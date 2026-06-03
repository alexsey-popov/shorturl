package config

import (
	"flag"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/alexsey-popov/shorturl/pkg/errors"
)

type ServerConf struct {
	Scheme      string
	Host        string
	NetAddress  string
	ContentType string
}

// SetServerAddress - изменение адреса сервера
func (s ServerConf) SetServerAddress(value string) error {
	// Проверка корректности указанного ip
	serverAddr, err := net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return errors.ErrInvalidAddress
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
		return errors.ErrInvalidAddress
	}

	Server.Scheme = parsedURL.Scheme
	Server.Host = parsedURL.Host

	return nil
}

// New - Конструктор со значениями по умолчанию
func New() ServerConf {
	return ServerConf{
		Scheme:      "http",
		Host:        "localhost:8080",
		NetAddress:  "localhost:8080",
		ContentType: "text/plain",
	}
}

// Parse - Парсим флаги и переменные окружения
func Parse() {
	// Каждый следующий шаг может перезаписать данные из предыдущего шага.
	// Шаг 1. Заполняем конфиг значения из консольных флагов
	flag.Parse()

	// Шаг 2. Заполняем значения из переменных окружения
	// Адрес сервера
	if serverAddress := os.Getenv(EnvServerAddress); serverAddress != "" {
		if err := Server.SetServerAddress(serverAddress); err != nil {
			log.Fatal(err)
		}
	}
	// URL сервера
	if baseURL := os.Getenv(EnvBaseURL); baseURL != "" {
		if err := Server.SetBaseURL(baseURL); err != nil {
			log.Fatal(err)
		}
	}
}
