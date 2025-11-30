package pool

import (
	"sync"
)

// Resetter ограничение типа для объектов, имеющих метод Reset()
type Resetter interface {
	Reset()
}

// Pool - generic-пул для хранения объектов с методом Reset()
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создает и возвращает указатель на новую структуру Pool
func New[T Resetter]() *Pool[T] {
	p := &Pool[T]{}
	p.pool.New = func() interface{} {
		// Создаем нулевое значение типа T
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
	x.Reset() // Теперь компилятор знает, что x имеет метод Reset()
	p.pool.Put(x)
}
