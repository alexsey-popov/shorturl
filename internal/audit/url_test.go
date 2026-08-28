package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestURLObserver - тестирование url аудита
func TestURLObserver(t *testing.T) {
	// received - был ли выполнен хотя бы один запрос
	var received bool
	// receivedEvent - событие последнего запроса
	var receivedEvent Event

	var mu sync.Mutex
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		received = true
		_ = json.NewDecoder(r.Body).Decode(&receivedEvent)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	l := zap.NewNop().Sugar()
	obs := NewURLObserver(l, ts.URL)
	defer obs.Close()

	event := Event{
		Timestamp: 12345678,
		Action:    "follow",
		UserID:    "1324",
		URL:       "https://example.com/target",
	}

	obs.Update(event)

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, received, "наблюдатель не отправил запрос на сервер")

	assert.Equal(t, event, receivedEvent, "ожидалось, что наблюдатель отправит событие  %+v, но пришло  %+v", event, receivedEvent)
}
