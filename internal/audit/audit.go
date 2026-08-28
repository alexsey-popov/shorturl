package audit

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Timeout - Максимальное время выполнения http запроса
var Timeout = 5 * time.Second

// Event - Структура события аудита
type Event struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
}

// Observer - Интерфейс наблюдателя
type Observer interface {
	Update(ctx context.Context, event Event)
	Close() error
}

// Publisher - Менеджер аудита
type Publisher struct {
	mu        sync.Mutex
	observers []Observer
	log       *zap.SugaredLogger
	sem       chan struct{}
	wg        sync.WaitGroup
}

// NewPublisher - Конструктор менеджера аудита
func NewPublisher(l *zap.SugaredLogger) *Publisher {
	return &Publisher{
		observers: make([]Observer, 0),
		log:       l,
		sem:       make(chan struct{}, 10),
	}
}

// Register - Регистрация наблюдателя
func (p *Publisher) Register(obs Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, obs)
}

// Notify - Уведомление всех наблюдателей о событии
func (p *Publisher) Notify(ctx context.Context, event Event) {
	p.mu.Lock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mu.Unlock()

	for _, obs := range observers {
		if p.sem != nil {
			p.sem <- struct{}{}
		}
		p.wg.Add(1)
		go func(o Observer) {
			if p.sem != nil {
				defer func() { <-p.sem }()
			}
			defer p.wg.Done()
			o.Update(ctx, event)
		}(obs)
	}
}

// Wait - Ожидание завершения всех текущих уведомлений
func (p *Publisher) Wait() {
	p.wg.Wait()
}

// Close Закрытие наблюдателей
func (p *Publisher) Close() error {
	p.wg.Wait()
	errs := make([]error, len(p.observers))

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, o := range p.observers {
		errs = append(errs, o.Close())
	}

	return errors.Join(errs...)
}
