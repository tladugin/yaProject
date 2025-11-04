package pool

import (
	"sync"
)

// Resetter ограничение типа для объектов, имеющих метод Reset()
type Resetter interface {
	Reset()
}

// Pool - generic-пул для хранения объектов с методом Reset()
type Pool[T any] struct {
	pool sync.Pool
}

// New создает и возвращает указатель на новую структуру Pool
// T должен быть указателем на тип с методом Reset()
func New[T Resetter]() *Pool[T] {
	p := &Pool[T]{}
	p.pool.New = func() interface{} {
		var zero T
		return zero
	}
	return p
}

// Get возвращает объект из пула
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put помещает объект в пул, предварительно сбрасывая его состояние
func (p *Pool[T]) Put(x T) {
	x.Reset() // Теперь это работает, так как x имеет метод Reset()
	p.pool.Put(x)
}
