package service

import (
	"crypto/rand"
	"database/sql"
	"net/url"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/repository/indb"
	"github.com/alexsey-popov/shorturl/internal/repository/infile"
	"github.com/alexsey-popov/shorturl/internal/repository/inmemory"
	"go.uber.org/zap"
)

// Repository - Интерфейс для хранилищ
type Repository interface {
	// Set - Сохранение originalURL за значением prefix
	Set(url model.URL) error
	// SetMany - Сохранение нескольких ссылок
	SetMany(urls []model.URL) error
	// Get - Получение originalURL по значению prefix
	Get(prefix string) (url model.URL, err error)
	// FindFromUserID - Получение списка ссылок закреплённых за пользователем
	FindFromUserID(userID string) (urls []model.URL, err error)
	// FindFromOriginal - Поиск значений по значению originalURL
	FindFromOriginal(originalURL string) (url model.URL, err error)
	// DeleteManyFromUserID - Массовое удаление ссылок принадлежащих пользователю
	DeleteManyFromUserID(prefixes []string, userID string) error
	// Ping - Проверка соединения
	Ping() error
	// GetUrlsCount - Получение общего количества сокращённых ссылок
	GetUrlsCount() (int, error)
	// GetUsersCount - Получение общего количества пользователей
	GetUsersCount() (int, error)
	// Закрытие соединений
	Close() error
}

// Shortener - Сервис для сокращения ссылок
type Shortener struct {
	Rep     Repository
	BaseURL string
}

// NewFromConfig - создание сервиса для сокращения ссылок по файлу конфигурации
func NewFromConfig(cfg *config.Server, sugar *zap.SugaredLogger) (*Shortener, error) {
	// Объект бизнес-логики
	var shortener *Shortener

	// Пытаемся подключить различные виды хранилищ (по умолчанию используется хранение в памяти)
	switch {
	// Если указаны данные для подключения в БД - используем БД
	case cfg.DSN != "":
		// Создаём объект взаимодействия с базой
		// Закрытие соединения с бд происходит в Shortener.Close()
		db, err := indb.ConnectDB(cfg.DSN)
		if err != nil {
			return nil, err
		}

		shortener = NewDBShortener(cfg.BaseURL, db)

		sugar.Infoln("В качестве хранилища используется БД")
	// Если нет данных для подключения к БД, но есть путь до файла - используем файл
	case cfg.FilePath != "":
		var err error
		shortener, err = NewFileShortener(cfg.BaseURL, cfg.FilePath)
		if err != nil {
			return nil, err
		}

		sugar.Infoln("В качестве хранилища используется файл")
	default:
		shortener = NewMemoryShortener(cfg.BaseURL)

		sugar.Infoln("В качестве хранилища используется ОЗУ")
	}

	return shortener, nil
}

// NewMemoryShortener - Конструктор Shortener для хранения данных в памяти
func NewMemoryShortener(baseURL string) *Shortener {
	return &Shortener{
		Rep:     inmemory.New(),
		BaseURL: baseURL,
	}
}

// NewFileShortener - Конструктор Shortener для хранения данных внутри файла
func NewFileShortener(baseURL string, filepath string) (*Shortener, error) {
	rep, err := infile.New(filepath)
	if err != nil {
		return nil, err
	}

	return &Shortener{
		Rep:     rep,
		BaseURL: baseURL,
	}, nil
}

// NewDBShortener - Конструктор Shortener для хранения данных в базе данных
func NewDBShortener(baseURL string, db *sql.DB) *Shortener {
	return &Shortener{
		Rep:     indb.New(db),
		BaseURL: baseURL,
	}
}

// CheckValidURL - валидация ссылки
func (s Shortener) CheckValidURL(originalURL string) (err error) {
	// Проверяем корректность URL
	_, err = url.ParseRequestURI(originalURL)

	return err
}

// Add - Добавление новой ссылки
func (s Shortener) Add(originalURL string, userID string) (string, error) {
	// Валидация ссылки
	if err := s.CheckValidURL(originalURL); err != nil {
		return "", err
	}

	item := model.New(s.getNewPrefix(), originalURL, userID)

	err := s.Rep.Set(item)
	if err != nil {
		return "", err
	}

	return s.GetURLFromPrefix(item.Prefix)
}

// AddMany - множественное создание сокращённых ссылок
func (s Shortener) AddMany(originalURLs []string, userID string) (mapURLs map[string]string, err error) {
	// Проверяем все ссылки на валидность и заполняем URLs
	urls := make([]model.URL, 0, len(originalURLs))
	for _, originalURL := range originalURLs {
		if err = s.CheckValidURL(originalURL); err != nil {
			return
		}

		urls = append(urls, model.New(s.getNewPrefix(), originalURL, userID))
	}

	// Пытаемся сохранить данные в базе
	err = s.Rep.SetMany(urls)
	if err != nil {
		return
	}

	mapURLs = make(map[string]string)
	for _, item := range urls {
		shortURL, err := s.GetURLFromPrefix(item.Prefix)
		if err != nil {
			return nil, err
		}

		mapURLs[item.OriginalURL] = shortURL
	}

	return
}

// getNewPrefix - Получение нового префикса
func (s Shortener) getNewPrefix() string {
	return rand.Text()
}

// GetURLFromPrefix - Получение сокращённого url по префиксу
func (s Shortener) GetURLFromPrefix(prefix string) (string, error) {
	return url.JoinPath(s.BaseURL, prefix)
}

// Close - закрытие соединений
func (s Shortener) Close() error {
	return s.Rep.Close()
}
