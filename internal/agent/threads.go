package agent

import (
	"sync"
)

// WorkerPool представляет пул воркеров для ограничения скорости отправки запросов
type WorkerPool struct {
	workers   int
	taskQueue chan func()
	wg        sync.WaitGroup
	once      sync.Once
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(workers int) (*WorkerPool, error) {
	if workers <= 0 {
		workers = 1 // Минимум 1 воркер
	}

	pool := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), workers*10), // Буферизованный канал
	}

	// Запускаем воркеры
	pool.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go pool.worker()
	}

	return pool, nil
}

// worker обрабатывает задачи из очереди
func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for task := range p.taskQueue {
		if task != nil {
			task()
		}
	}
}

// Submit добавляет задачу в очередь
func (p *WorkerPool) Submit(task func()) {
	if p == nil || p.taskQueue == nil {
		return
	}

	select {
	case p.taskQueue <- task:
		// Задача добавлена в очередь
	default:
		// Если очередь заполнена, задача будет пропущена
		// В реальном приложении можно добавить логирование
	}
}

// Shutdown останавливает пул воркеров
func (p *WorkerPool) Shutdown() {
	if p == nil {
		return
	}

	p.once.Do(func() {
		close(p.taskQueue)
		p.wg.Wait()
	})
}
