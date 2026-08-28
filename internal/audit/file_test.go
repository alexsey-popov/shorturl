package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestFileObserver - тестирование аудита в файл
func TestFileObserver(t *testing.T) {
	// Подготавливаем папку для файла с аудитом. После теста удаляем содержимое
	tmpDir, err := os.MkdirTemp("", "audit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Создаём наблюдатель
	l := zap.NewNop().Sugar()
	f := filepath.Join(tmpDir, "audit.log")
	obs, err := NewFileObserver(l, f)
	defer obs.Close()

	if err != nil {
		t.Fatal(err)
	}

	e := Event{
		Timestamp: 12345678,
		Action:    "shorten",
		UserID:    "1234",
		URL:       "https://mylongdomain.com/my/long/path/to/shorten/",
	}

	obs.Update(e)

	c, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}

	var e2 Event
	if err = json.Unmarshal(c, &e2); err != nil {
		t.Fatalf("ошибка при десериализации json: %v, text: %s", err, string(c))
	}

	assert.Equal(t, e, e2, "ожидалось событие %+v, но в файле оказалось %+v", e, e2)
}
