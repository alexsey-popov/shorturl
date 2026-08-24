package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

const (
	// Название env переменной с адресом сервера
	EnvNetAddress = "SERVER_ADDRESS"
	// Название env переменной с URL сервера
	EnvBaseURL = "BASE_URL"
	// Название env для пути файла (используется в репозитории InFile)
	EnvFilePath = "FILE_STORAGE_PATH"
	// Название env для параметров подключения к БД
	EnvDSN = "DATABASE_DSN"
	// Срок жизни токена аутентификации пользователя (Nanosecond)
	EnvTokenExp = "TOKEN_EXP"
	// Приватный ключ JWT
	EnvSecretKey = "SECRET_KEY"
	// Файл для хранения логов создания и прохождения по сокращённым ссылкам
	EnvAuditFile = "AUDIT_FILE"
	// Ссылка для передачи логов создания и прохождения по сокращённым ссылкам
	EnvAuditURL = "AUDIT_URL"
	// Название флага с адресом сервера
	FlagNetAddress = "a"
	// Название флага с URL сервера
	FlagBaseURL = "b"
	// Название флага для пути файла (используется в репозитории InFile)
	FlagFilePath = "f"
	// Название флага параметров подключения к БД
	FlagDSN = "d"
	// Название флага для срока жизни токена аутентификации пользователя (Nanosecond)
	FlagTokenExp = "e"
	// Название флага для приватного ключа JWT
	FlagSecretKey = "s"
	// Название флага для логов аудита
	FlagAuditFile = "audit-file"
	// Ссылка для логов аудита
	FlagAuditURL = "audit-url"
)

// ServerConf - Структура для хранения конфигурации
type ServerConf struct {
	BaseURL    string
	NetAddress string
	FilePath   string
	DSN        string
	TokenExp   time.Duration
	SecretKey  string
	AuditFile  string
	AuditURL   string
}

// Server - Объект конфига (по умолчанию заполнен дефолтными значениями)
var Server ServerConf = New()

// New - Конструктор со значениями по умолчанию
func New() ServerConf {
	return ServerConf{
		BaseURL:    "http://localhost:8080",
		NetAddress: "localhost:8080",
		FilePath:   "",
		DSN:        "",
		TokenExp:   time.Hour * 3,
		SecretKey:  "",
		AuditFile:  "",
		AuditURL:   "",
	}
}

// HasAudit - производится ли аудит запросов
func (s ServerConf) HasAudit() bool {
	return s.AuditFile != "" || s.AuditURL != ""
}

// Parse - Парсим флаги и переменные окружения
func Parse() error {
	// Каждый следующий шаг может перезаписать данные из предыдущего шага.
	// Шаг 1. Заполняем конфиг значениями из консольных флагов
	flag.StringVar(&Server.NetAddress, FlagNetAddress, "localhost:8080", "Адрес прослушиваемого сервера в формате ip:port")
	flag.StringVar(&Server.BaseURL, FlagBaseURL, "http://localhost:8080", "Базовый адрес сервера в формате http://localhost:8080")
	flag.StringVar(&Server.FilePath, FlagFilePath, "", "Путь до файла в котором будут храниться данные (если используется тип хранения \"В файле\")")
	flag.StringVar(&Server.DSN, FlagDSN, "", "Параметры подключения к БД (если используется тип хранения \"База данных\")")
	flag.DurationVar(&Server.TokenExp, FlagTokenExp, time.Hour*3, "Срок жизни токена аутентификации пользователя (Nanosecond)")
	flag.StringVar(&Server.SecretKey, FlagSecretKey, "", "Приватный ключ JWT")

	flag.StringVar(&Server.AuditFile, FlagAuditFile, "", "Файл для логов аудита")
	flag.StringVar(&Server.AuditURL, FlagAuditURL, "", "URL для логов аудита")

	flag.Parse()

	// Шаг 2. Заполняем значения из переменных окружения
	// Адрес сервера
	if netAddress, ok := os.LookupEnv(EnvNetAddress); ok {
		Server.NetAddress = netAddress
	}
	// URL сервера
	if baseURL, ok := os.LookupEnv(EnvBaseURL); ok {
		Server.BaseURL = baseURL
	}
	// Путь до файла хранения (при использовании репозитория InFile)
	if filePath, ok := os.LookupEnv(EnvFilePath); ok {
		Server.FilePath = filePath
	}
	// Параметры подключения к БД (при использовании репозитория InDB)
	if dsn, ok := os.LookupEnv(EnvDSN); ok {
		Server.DSN = dsn
	}
	// Срок жизни токена аутентификации пользователя (Nanosecond)
	if tokenExpStr, ok := os.LookupEnv(EnvTokenExp); ok {
		// Форматируем строку в число
		tokenExp, err := strconv.Atoi(tokenExpStr)
		if err != nil {
			return err
		}

		Server.TokenExp = time.Duration(tokenExp)
	}
	// Приватный ключ JWT
	if secretKey, ok := os.LookupEnv(EnvSecretKey); ok {
		Server.SecretKey = secretKey
	}

	// Файл для логов аудита
	if auditFile, ok := os.LookupEnv(EnvAuditFile); ok {
		Server.AuditFile = auditFile
	}

	// URL для логов аудита
	if auditURL, ok := os.LookupEnv(EnvAuditURL); ok {
		Server.AuditURL = auditURL
	}

	return nil
}
