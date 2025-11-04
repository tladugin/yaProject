package agent

import (
	"sync"
	"time"
)

// generate:reset
type Task struct {
	ID       string
	URL      string
	Method   string
	Body     []byte
	Headers  map[string]string
	Attempts int
	Timeout  time.Duration
	Result   interface{}
	Error    error
}

// WorkerPool представляет пул воркеров с поддержкой функций
type WorkerPool struct {
	workers   int
	taskQueue chan func() // Изменено на func()
	wg        sync.WaitGroup
	once      sync.Once
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(workers int) (*WorkerPool, error) {
	if workers <= 0 {
		workers = 1
	}

	pool := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), workers*10), // Буферизованный канал для функций
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
			task() // Просто выполняем функцию
		}
	}
}

// Submit добавляет задачу-функцию в очередь
func (p *WorkerPool) Submit(task func()) {
	if p == nil || p.taskQueue == nil {
		return
	}

	select {
	case p.taskQueue <- task:
		// Задача добавлена в очередь
	default:
		// Если очередь заполнена, задача будет пропущена
	}
}

// SubmitTask добавляет HTTP задачу в очередь (для обратной совместимости)
func (p *WorkerPool) SubmitTask(url, method string, body []byte) {
	p.Submit(func() {
		// Здесь ваша логика отправки HTTP запроса
		// repository.SendWithRetry(url, body, method, ...)
	})
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
