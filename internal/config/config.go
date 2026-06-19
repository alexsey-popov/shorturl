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
	// Название env для параметров подключения к БД
	EnvDSN = "DATABASE_DSN"
	// Название флага с адресом сервера
	FlagServerAddress = "a"
	// Название флага с URL сервера
	FlagBaseURL = "b"
	// Название флага для пути файла (используется в репозитории InFile)
	FlagFilePath = "f"
	// Название флага параметров подключения к БД
	FlagDSN = "d"
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
		DSN:        "host=localhost user=videos password=userpassword dbname=videos sslmode=disable",
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
	// Путь до файла хранения (при использовании репозитория InFile)
	flag.Func(FlagFilePath, "Путь до файла в котором будут храниться данные (если используется тип хранения \"В файле\")", Server.SetFilePath)
	// Путь до файла хранения (при использовании репозитория InFile)
	flag.Func(FlagDSN, "Параметры подключения к БД (если используется тип хранения \"База данных\")", Server.SetDSN)

	flag.Parse()

	// Шаг 2. Заполняем значения из переменных окружения
	// Адрес сервера
	if serverAddress, ok := os.LookupEnv(EnvServerAddress); ok {
		if err := Server.SetServerAddress(serverAddress); err != nil {
			return err
		}
	}
	// URL сервера
	if baseURL, ok := os.LookupEnv(EnvBaseURL); ok {
		if err := Server.SetBaseURL(baseURL); err != nil {
			return err
		}
	}
	// Путь до файла хранения (при использовании репозитория InFile)
	if filePath, ok := os.LookupEnv(EnvFilePath); ok {
		if err := Server.SetFilePath(filePath); err != nil {
			return err
		}
	}
	// Параметры подключения к БД (при использовании репозитория InDB)
	if DSN, ok := os.LookupEnv(EnvDSN); ok {
		if err := Server.SetDSN(DSN); err != nil {
			return err
		}
	}

	return nil
}
