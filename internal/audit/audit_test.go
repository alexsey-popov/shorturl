package audit

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// fakeObserver - заглушка для тестирования
type fakeObserver struct {
	mu     sync.Mutex
	events []Event
}

// Update - Добавление события
func (o *fakeObserver) Update(event Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

// Тестирование наблюдателя
func TestPublisher(t *testing.T) {
	l := zap.NewNop().Sugar()
	p := NewPublisher(l)

	o := &fakeObserver{}
	p.Register(o)

	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user123",
		URL:       "https://example.com/long",
	}
	p.Notify(e)

	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.events) != 1 {
		t.Fatalf("ожидался один элемент, а получили: %d", len(o.events))
	}
	if o.events[0] != e {
		t.Errorf("ожидалось событие %+v, а получили %+v", e, o.events[0])
	}
}
