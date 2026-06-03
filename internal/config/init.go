package config

import (
	"flag"
)

const (
	// Название env переменной с адресом сервера
	EnvServerAddress = "SERVER_ADDRESS"
	// Название env переменной с URL сервера
	EnvBaseURL = "BASE_URL"
	// Название флага с адресом сервера
	FlagServerAddress = "a"
	// Название флага с URL сервера
	FlagBaseURL = "b"
)

var Server ServerConf

// Создаём и заполняем экземпляр конфига значениями по умолчанию
func init() {
	Server = New()
}

// Задаём правила парсинга флагов командной строки
func init() {
	// Обрабатывает флаг с адресом сервера
	flag.Func(FlagServerAddress, "Адрес прослушиваемого сервера в формате ip:port", Server.SetServerAddress)
	// Обрабатывает флаг с URL сервера
	flag.Func(FlagBaseURL, "Базовый адрес сервера в формате http://localhost:8080", Server.SetBaseURL)
}
