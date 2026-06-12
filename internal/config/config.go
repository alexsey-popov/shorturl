package config

import (
	"flag"
	"log"
	"os"
)

const (
	// Название env переменной с адресом сервера
	EnvServerAddress = "SERVER_ADDRESS"
	// Название env переменной с URL сервера
	EnvBaseURL = "BASE_URL"
	// Название env для пути файла (используется в репозитории InFile)
	EnvFilePath = "FILE_STORAGE_PATH"
	// Название флага с адресом сервера
	FlagServerAddress = "a"
	// Название флага с URL сервера
	FlagBaseURL = "b"
	// Название флага для пути файла (используется в репозитории InFile)
	FlagFilePath = "f"
)

// Задаём правила парсинга флагов командной строки
func init() {
	// Обрабатывает флаг с адресом сервера
	flag.Func(FlagServerAddress, "Адрес прослушиваемого сервера в формате ip:port", Server.SetServerAddress)
	// Обрабатывает флаг с URL сервера
	flag.Func(FlagBaseURL, "Базовый адрес сервера в формате http://localhost:8080", Server.SetBaseURL)
	// Обрабатывает флаг с URL сервера
	flag.Func(FlagFilePath, "Путь до файла в котором будут храниться данные (если используется тип хранения \"В файле\")", Server.SetFilePath)
}

// Server - Объект для хранения документация
var Server ServerConf

// При инициализации создаём и заполняем экземпляр конфига значениями по умолчанию
func init() {
	Server = New()
}

// New - Конструктор со значениями по умолчанию
func New() ServerConf {
	return ServerConf{
		Scheme:     "http",
		Host:       "localhost:8080",
		NetAddress: "localhost:8080",
		FilePath:   "infile.json",
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
	// Путь до файла хранения (при использовании репозитория InFile)
	if filePath := os.Getenv(EnvFilePath); filePath != "" {
		if err := Server.SetFilePath(filePath); err != nil {
			log.Fatal(err)
		}
	}
}
