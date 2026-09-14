package pool

import (
	"sync"
)

// Resettable определяет интерфейс для объектов, поддерживающих сброс состояния.
type Resettable interface {
	Reset()
}

// Pool представляет пул объектов с поддержкой дженериков.
type Pool[T Resettable] struct {
	pool sync.Pool
}

// New создает и возвращает новый экземпляр Pool.
func New[T Resettable](newFunc ...func() T) *Pool[T] {
	p := &Pool[T]{}
	if len(newFunc) > 0 && newFunc[0] != nil {
		p.pool.New = func() any {
			return newFunc[0]()
		}
	}
	return p
}

// Get извлекает объект из пула. Если пул пуст и задана функция создания, возвращает новый объект.
func (p *Pool[T]) Get() T {
	v := p.pool.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// Put помещает объект в пул, предварительно выполняя сброс его состояния.
func (p *Pool[T]) Put(obj T) {
	if any(obj) == nil {
		return
	}
	obj.Reset()
	p.pool.Put(obj)
}
