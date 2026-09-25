package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Переменные окружения
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
	// Поддержка HTTPS
	EnvEnableHTTPS = "ENABLE_HTTPS"
	// Файл с конфигом
	EncConfigFile = "CONFIG"
)

// Консольные флаги
const (
	// Название флага с адресом сервера
	FlagNetAddress = "a"
	// Название флага с URL сервера
	FlagBaseURL = "b"
	// Название флага для пути файла (используется в репозитории InFile)
	FlagFilePath = "f"
	// Название флага параметров подключения к БД
	FlagDSN = "d"
	// Название флага для срока жизни токена аутентификации пользователя (Nanosecond)
	FlagTokenExp = "token-exp"
	// Название флага для приватного ключа JWT
	FlagSecretKey = "s"
	// Название флага для логов аудита
	FlagAuditFile = "audit-file"
	// Ссылка для логов аудита
	FlagAuditURL = "audit-url"
	// Поддержка HTTPS
	FlagEnableHTTPS = "e"
	// Файл с конфигом
	FlagConfigFile = "c"
)

// Параметры по умолчанию
const (
	// DefaultBaseURL - URL сервиса
	DefaultBaseURL = "http://localhost:8080"
	// DefaultNetAddress - Адрес сервиса (хост:порт)
	DefaultNetAddress = "localhost:8080"
	// DefaultTokenExp - Время жизни JWT токена
	DefaultTokenExp = time.Hour * 3
	// Поддержка HTTPS
	DefaultEnableHTTPS = false
)

// Server - Структура для хранения конфигурации
type Server struct {
	BaseURL     string        `json:"base_url"`
	NetAddress  string        `json:"address"`
	FilePath    string        `json:"store_file"`
	DSN         string        `json:"database_dsn"`
	TokenExp    time.Duration `json:"token_exp"`
	SecretKey   string        `json:"secret_key"`
	AuditFile   string        `json:"audit_file"`
	AuditURL    string        `json:"audit_url"`
	EnableHTTPS bool          `json:"enable_https"`
}

// serverCfg - Объект конфига для сервера
var serverCfg *Server

// NewEmpty - Конфиг заполненный дефолтными значениями
func NewEmpty() *Server {
	return &Server{
		BaseURL:     DefaultBaseURL,
		NetAddress:  DefaultNetAddress,
		FilePath:    "",
		DSN:         "",
		TokenExp:    DefaultTokenExp,
		SecretKey:   "",
		AuditFile:   "",
		AuditURL:    "",
		EnableHTTPS: DefaultEnableHTTPS,
	}
}

// NewParsed - Конфиг заполненный обработанными значениями (singleton)
func NewParsed() (*Server, error) {
	if serverCfg != nil {
		return serverCfg, nil
	}

	serverCfg = NewEmpty()

	// Передаём аргументы командной строки и функцию получения данных из переменных окружения
	err := serverCfg.Parse(os.Args[1:], os.LookupEnv)

	if err != nil {
		err = fmt.Errorf("ошибка при парсинге конфига в конструкторе: %w", err)
	}

	return serverCfg, err
}

// HasAudit - производится ли аудит запросов
func (s Server) HasAudit() bool {
	return s.AuditFile != "" || s.AuditURL != ""
}

// Parse - Парсим флаги и переменные окружения
func (s *Server) Parse(args []string, lookupFunc func(string) (string, bool)) error {
	// Каждый следующий шаг может перезаписать данные из предыдущего шага.

	fs := flag.NewFlagSet("app", flag.ContinueOnError)

	// Шаг 1. Пытаемся подгрузить дефолтные значения из файла переданного через флаги
	var flagCfgFile string
	fs.StringVar(&flagCfgFile, FlagConfigFile, "", "Файл с конфигом")

	// Шаг 2. Пытаемся подгрузить дефолтные значения из файла переданного через env
	if envCfgFile, ok := lookupFunc(EncConfigFile); ok {
		if err := s.ParseFromFile(envCfgFile); err != nil {
			return fmt.Errorf("ошибка при парсинге конфига из файла(env): %w", err)
		}
	}

	// Шаг 3. Заполняем конфиг значениями из консольных флагов
	fs.StringVar(&s.NetAddress, FlagNetAddress, s.NetAddress, "Адрес прослушиваемого сервера в формате ip:port")
	fs.StringVar(&s.BaseURL, FlagBaseURL, s.BaseURL, "Базовый адрес сервера в формате http://localhost:8080")
	fs.StringVar(&s.FilePath, FlagFilePath, "", "Путь до файла в котором будут храниться данные (если используется тип хранения \"В файле\")")
	fs.StringVar(&s.DSN, FlagDSN, "", "Параметры подключения к БД (если используется тип хранения \"База данных\")")
	fs.DurationVar(&s.TokenExp, FlagTokenExp, s.TokenExp, "Срок жизни токена аутентификации пользователя (Nanosecond)")
	fs.StringVar(&s.SecretKey, FlagSecretKey, "", "Приватный ключ JWT")

	fs.StringVar(&s.AuditFile, FlagAuditFile, "", "Файл для логов аудита")
	fs.StringVar(&s.AuditURL, FlagAuditURL, "", "URL для логов аудита")

	fs.BoolVar(&s.EnableHTTPS, FlagEnableHTTPS, s.EnableHTTPS, "Поддержка HTTPS")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("ошибка при парсинге аргументов командной строки: %w", err)
	}

	if flagCfgFile != "" {
		if err := s.ParseFromFile(flagCfgFile); err != nil {
			return fmt.Errorf("ошибка при парсинге конфига из файла(flag): %w", err)
		}
	}

	// Шаг 4. Заполняем значения из переменных окружения
	// Адрес сервера
	if netAddress, ok := lookupFunc(EnvNetAddress); ok {
		s.NetAddress = netAddress
	}
	// URL сервера
	if baseURL, ok := lookupFunc(EnvBaseURL); ok {
		s.BaseURL = baseURL
	}
	// Путь до файла хранения (при использовании репозитория InFile)
	if filePath, ok := lookupFunc(EnvFilePath); ok {
		s.FilePath = filePath
	}
	// Параметры подключения к БД (при использовании репозитория InDB)
	if dsn, ok := lookupFunc(EnvDSN); ok {
		s.DSN = dsn
	}
	// Срок жизни токена аутентификации пользователя (Nanosecond)
	if tokenExpStr, ok := lookupFunc(EnvTokenExp); ok {
		// Форматируем строку в число
		tokenExp, err := strconv.Atoi(tokenExpStr)
		if err != nil {
			return fmt.Errorf("ошибка при обработке срока жизни токена аутентификации: %w", err)
		}

		s.TokenExp = time.Duration(tokenExp)
	}
	// Приватный ключ JWT
	if secretKey, ok := lookupFunc(EnvSecretKey); ok {
		s.SecretKey = secretKey
	}

	// Файл для логов аудита
	if auditFile, ok := lookupFunc(EnvAuditFile); ok {
		s.AuditFile = auditFile
	}

	// URL для логов аудита
	if auditURL, ok := lookupFunc(EnvAuditURL); ok {
		s.AuditURL = auditURL
	}

	// Поддержка HTTPS
	if enableHTTPS, ok := lookupFunc(EnvEnableHTTPS); ok {
		answer, err := strconv.ParseBool(enableHTTPS)

		if err != nil {
			return fmt.Errorf("ошибка при парсинге переменной EnvEnableHTTPS: %w", err)
		}

		s.EnableHTTPS = answer
	}

	return nil
}

// ParseFromFile - Парсинг конфига из файла
func (s *Server) ParseFromFile(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	if err = json.Unmarshal(file, &s); err != nil {
		return fmt.Errorf("ошибка десериализации данных файла: %w", err)
	}

	return nil
}
