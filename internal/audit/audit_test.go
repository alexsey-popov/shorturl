package audit

import (
	"context"
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
func (f *fakeObserver) Update(ctx context.Context, event Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

// Close - Закрытие наблюдателя
func (f *fakeObserver) Close() error {
	return nil
}

// Тестирование наблюдателя
func TestPublisher(t *testing.T) {
	l := zap.NewNop().Sugar()
	p := NewPublisher(l)
	defer p.Close()

	o := &fakeObserver{}
	p.Register(o)

	e := Event{
		Timestamp: time.Now().Unix(),
		Action:    "shorten",
		UserID:    "user123",
		URL:       "https://example.com/long",
	}
	p.Notify(t.Context(), e)
	p.Wait()

	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.events) != 1 {
		t.Fatalf("ожидался один элемент, а получили: %d", len(o.events))
	}
	if o.events[0] != e {
		t.Errorf("ожидалось событие %+v, а получили %+v", e, o.events[0])
	}
}

// Тестирование ограничения количества горутин и параллельного уведомления
func TestPublisherConcurrency(t *testing.T) {
	l := zap.NewNop().Sugar()
	p := NewPublisher(l)
	defer p.Close()

	o := &fakeObserver{}
	p.Register(o)

	const totalEvents = 50
	for i := 0; i < totalEvents; i++ {
		p.Notify(t.Context(), Event{
			Timestamp: time.Now().Unix(),
			Action:    "test",
			UserID:    "user",
			URL:       "https://example.com",
		})
	}
	p.Wait()

	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.events) != totalEvents {
		t.Fatalf("ожидалось %d событий, а получили: %d", totalEvents, len(o.events))
	}
}
