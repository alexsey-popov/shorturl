package config

import (
	"errors"
	"flag"
	"net"
	"net/url"
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
			return errors.New("Некорректный ip")
		}

		NetAddress = serverAddr.String()

		return nil
	})

	// Обрабатывает флаг b
	flag.Func("b", "Базовый адрес сервера в формате https://example.com", func(value string) error {
		parsedURL, err := url.ParseRequestURI(value)
		if err != nil {
			return err
		}

		Scheme = parsedURL.Scheme
		Host = parsedURL.Host

		return nil
	})
}
