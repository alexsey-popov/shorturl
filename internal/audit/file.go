package audit

import (
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
}

// NewFileObserver - Конструктор наблюдателя файла
func NewFileObserver(l *zap.SugaredLogger, filePath string) (*FileObserver, error) {
	return &FileObserver{
		filePath: filePath,
		log:      l,
	}, nil
}

// Update - Запись события в файл
func (f *FileObserver) Update(event Event) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		f.log.Errorf("ошибка при сериализации в json: %v", err.Error())

		return
	}

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.log.Errorf("ошибка при открытии файла: %v", err.Error())

		return
	}
	defer file.Close()

	file.Write(data)
	file.WriteString("\n")
}
