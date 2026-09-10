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
	mu       sync.Mutex
	workers  []*observerWorker
	log      *zap.SugaredLogger
	wg       sync.WaitGroup
	notifyWg sync.WaitGroup
}

type auditTask struct {
	ctx   context.Context
	event Event
}

type observerWorker struct {
	observer Observer
	eventCh  chan auditTask
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPublisher - Конструктор менеджера аудита
func NewPublisher(l *zap.SugaredLogger) *Publisher {
	return &Publisher{
		workers: make([]*observerWorker, 0),
		log:     l,
	}
}

// Register - Регистрация наблюдателя
func (p *Publisher) Register(obs Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	worker := &observerWorker{
		observer: obs,
		eventCh:  make(chan auditTask, 100),
		ctx:      ctx,
		cancel:   cancel,
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for task := range worker.eventCh {
			worker.observer.Update(task.ctx, task.event)
			p.notifyWg.Done()
		}
	}()

	p.workers = append(p.workers, worker)
}

// Notify - Уведомление всех наблюдателей о событии
func (p *Publisher) Notify(ctx context.Context, event Event) {
	p.mu.Lock()
	workers := make([]*observerWorker, len(p.workers))
	copy(workers, p.workers)
	p.mu.Unlock()

	for _, w := range workers {
		select {
		case w.eventCh <- auditTask{ctx: ctx, event: event}:
			p.notifyWg.Add(1)
		default:
			p.log.Info("Канал обработки данных аудита заполнен, запрос пропустил аудит")
		}
	}
}

// Wait - Ожидание завершения всех текущих уведомлений
func (p *Publisher) Wait() {
	p.notifyWg.Wait()
}

// Close Закрытие наблюдателей
func (p *Publisher) Close() error {
	p.notifyWg.Wait()

	p.mu.Lock()
	workers := make([]*observerWorker, len(p.workers))
	copy(workers, p.workers)
	p.mu.Unlock()

	for _, w := range workers {
		w.cancel()
		close(w.eventCh)
	}

	p.wg.Wait()

	errs := make([]error, 0, len(workers))

	for _, w := range workers {
		if err := w.observer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
