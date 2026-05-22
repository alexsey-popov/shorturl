package config

import (
	"flag"
	"net"
	"net/url"

	"github.com/alexsey-popov/shorturl/pkg/errors"
)

var (
	Scheme     = "http"
	Host       = "localhost:8080"
	NetAddress = Host
)

func init() {
	// Обрабатывает флаг a
	flag.Func("a", "Адрес прослушиваемого сервера в формате ip:port", func(value string) error {
		// Проверка корректности указанного ip
		serverAddr, err := net.ResolveTCPAddr("tcp", value)
		if err != nil {
			return errors.ErrInvalidAddress
		}

		NetAddress = serverAddr.String()

		return nil
	})

	// Обрабатывает флаг b
	flag.Func("b", "Базовый адрес сервера в формате http://localhost:8080", func(value string) error {
		parsedURL, err := url.ParseRequestURI(value)
		if err != nil {
			return err
		}

		Scheme = parsedURL.Scheme
		Host = parsedURL.Host

		return nil
	})
}
