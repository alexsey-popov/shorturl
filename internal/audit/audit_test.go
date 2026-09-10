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

type blockingObserver struct {
	mu     sync.Mutex
	events []Event
	block  chan struct{}
}

func (b *blockingObserver) Update(ctx context.Context, event Event) {
	<-b.block
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

func (b *blockingObserver) Close() error {
	select {
	case <-b.block:
	default:
		close(b.block)
	}
	return nil
}

func TestPublisherSlowObserver(t *testing.T) {
	l := zap.NewNop().Sugar()
	p := NewPublisher(l)
	defer p.Close()

	obs := &blockingObserver{
		block: make(chan struct{}),
	}
	p.Register(obs)

	// Buffer size is 100. Send 300 events while worker is blocked.
	for i := 0; i < 300; i++ {
		p.Notify(t.Context(), Event{
			Timestamp: int64(i),
			Action:    "test",
			UserID:    "user",
			URL:       "https://example.com",
		})
	}

	// Unblock worker
	close(obs.block)

	// Wait for processing
	p.Wait()

	obs.mu.Lock()
	defer obs.mu.Unlock()

	if len(obs.events) >= 300 {
		t.Errorf("ожидалось отбрасывание событий при переполнении буфера, но получено все 300")
	}
	if len(obs.events) == 0 {
		t.Errorf("ожидалось получение хотя бы части событий, но получено 0")
	}
}
