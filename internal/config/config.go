package config

import (
	"flag"
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

// Server - Объект конфига (по умолчанию заполнен дефолтными значениями)
var Server ServerConf = New()

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
func Parse() error {
	// Каждый следующий шаг может перезаписать данные из предыдущего шага.
	// Шаг 1. Заполняем конфиг значениями из консольных флагов
	// Адрес сервера
	flag.Func(FlagServerAddress, "Адрес прослушиваемого сервера в формате ip:port", Server.SetServerAddress)
	// URL сервера
	flag.Func(FlagBaseURL, "Базовый адрес сервера в формате http://localhost:8080", Server.SetBaseURL)
	// Обрабатывает флаг с URL сервера
	flag.Func(FlagFilePath, "Путь до файла в котором будут храниться данные (если используется тип хранения \"В файле\")", Server.SetFilePath)
	// Путь до файла хранения (при использовании репозитория InFile)
	flag.Parse()

	// Шаг 2. Заполняем значения из переменных окружения
	// Адрес сервера
	if serverAddress, exists := os.LookupEnv(EnvServerAddress); exists {
		if err := Server.SetServerAddress(serverAddress); err != nil {
			return err
		}
	}
	// URL сервера
	if baseURL, exists := os.LookupEnv(EnvBaseURL); exists {
		if err := Server.SetBaseURL(baseURL); err != nil {
			return err
		}
	}
	// Путь до файла хранения (при использовании репозитория InFile)
	if filePath, exists := os.LookupEnv(EnvFilePath); exists {
		if err := Server.SetFilePath(filePath); err != nil {
			return err
		}
	}

	return nil
}
