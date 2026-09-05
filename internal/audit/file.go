package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"go.uber.org/zap"
)

// FileObserver - Наблюдатель для записи аудита в файл
type FileObserver struct {
	mu       sync.Mutex
	filePath string
	log      *zap.SugaredLogger
	file     *os.File
}

// NewFileObserver - Конструктор наблюдателя файла
func NewFileObserver(l *zap.SugaredLogger, filePath string) (*FileObserver, error) {
	// Открываем файл при создании наблюдателя
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.Errorf("ошибка при открытии файла: %v", err.Error())

		return nil, err
	}

	return &FileObserver{
		filePath: filePath,
		log:      l,
		file:     file,
	}, nil
}

// Update - Запись события в файл
func (f *FileObserver) Update(ctx context.Context, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		f.log.Errorf("ошибка при сериализации в json: %v", err.Error())

		return
	}

	f.mu.Lock()
	_, err = f.file.Write(data)
	if err == nil {
		_, err = f.file.WriteString("\n")
	}
	f.mu.Unlock()
	if err != nil {
		f.log.Errorf("ошибка при записи в файл: %v", err.Error())

		return
	}
}

// Close - Закрытие файла
func (f *FileObserver) Close() error {
	if err := f.file.Close(); err != nil {
		f.log.Errorf("ошибка при закрытии файла: %v", err.Error())

		return err
	}

	return nil
}
