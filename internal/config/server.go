package config

import (
	"flag"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/alexsey-popov/shorturl/pkg/errors"
)

const (
	ENV_SERVER_ADDRESS  = "SERVER_ADDRESS"
	ENV_BASE_URL        = "BASE_URL"
	FLAG_SERVER_ADDRESS = "a"
	FLAG_BASE_URL       = "b"
)

type ServerConf struct {
	Scheme      string
	Host        string
	NetAddress  string
	ContentType string
}

var Server ServerConf

// Каждый следующий шаг может перезаписать данные из предыдущего.
// Шаг 1. Создаём и заполняем экземпляр конфига базовыми значениями
func init() {
	Server = New()
}

// Шаг 2. Заполняем конфиг значениями из консольных флагов
func init() {
	// Обрабатывает флаг с адресом сервера
	flag.Func(FLAG_SERVER_ADDRESS, "Адрес прослушиваемого сервера в формате ip:port", Server.SetServerAddress)

	// Обрабатывает флаг с URL сервера
	flag.Func(FLAG_BASE_URL, "Базовый адрес сервера в формате http://localhost:8080", Server.SetBaseURL)

	// Парсим аргументы командной строки
	flag.Parse()
}

// Шаг 3. Заполняем значения из переменных окружения
func init() {
	// Адрес сервера
	if serverAddress := os.Getenv(ENV_SERVER_ADDRESS); serverAddress != "" {
		if err := Server.SetServerAddress(serverAddress); err != nil {
			log.Fatal(err)
		}
	}

	// URL сервера
	if baseURL := os.Getenv(ENV_BASE_URL); baseURL != "" {
		if err := Server.SetBaseURL(baseURL); err != nil {
			log.Fatal(err)
		}
	}
}

// New - Конструктор
func New() ServerConf {
	return ServerConf{
		Scheme:      "http",
		Host:        "localhost:8080",
		NetAddress:  "localhost:8080",
		ContentType: "text/plain",
	}
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
