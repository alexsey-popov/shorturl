package audit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alexsey-popov/shorturl/internal/config"
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

// NewPublisherFromConfig - Создание посредника и регистрация наблюдателей исходя из данных конфига
func NewPublisherFromConfig(cfg *config.Server, sugar *zap.SugaredLogger) (*Publisher, error) {
	// Посредник
	p := NewPublisher(sugar)

	// Подключаем аудит в файл
	if cfg.AuditFile != "" {
		fileObserver, err := NewFileObserver(sugar, cfg.AuditFile)
		if err != nil {
			return nil, fmt.Errorf("ошибка при создании наблюдателя файла аудита: %w", err)
		}
		p.Register(fileObserver)
		sugar.Infoln("Аудит в файл включен")
	}

	// Подключаем аудит по апи
	if cfg.AuditURL != "" {
		urlObserver := NewURLObserver(sugar, cfg.AuditURL)
		p.Register(urlObserver)
		sugar.Infoln("Аудит по URL включен")
	}

	return p, nil
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
