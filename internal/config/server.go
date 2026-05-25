package config

import (
	"flag"
	"net"
	"net/url"

	"github.com/alexsey-popov/shorturl/pkg/errors"
)

type ServerConf struct {
	Scheme      string
	Host        string
	NetAddress  string
	ContentType string
}

var Server ServerConf

func init() {
	Server = New()
}

func init() {
	// Обрабатывает флаг a
	flag.Func("a", "Адрес прослушиваемого сервера в формате ip:port", func(value string) error {
		// Проверка корректности указанного ip
		serverAddr, err := net.ResolveTCPAddr("tcp", value)
		if err != nil {
			return errors.ErrInvalidAddress
		}

		Server.NetAddress = serverAddr.String()

		return nil
	})

	// Обрабатывает флаг b
	flag.Func("b", "Базовый адрес сервера в формате http://localhost:8080", func(value string) error {
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
	})
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
