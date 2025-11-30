package agent

import (
	"sync"
	"time"
)

// Task представляет задачу для выполнения
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
	Fn       func() // Функция для выполнения
}

// WorkerPool представляет пул воркеров для выполнения задач
type WorkerPool struct {
	workers   int
	taskQueue chan *Task
	wg        sync.WaitGroup
	once      sync.Once
	mu        sync.RWMutex
	closed    bool
}

func NewWorkerPool(workers int) (*WorkerPool, error) {
	if workers <= 0 {
		workers = 1
	}

	wp := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan *Task, workers*10),
		closed:    false,
	}

	wp.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go wp.worker()
	}

	return wp, nil
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for task := range p.taskQueue {
		if task != nil && task.Fn != nil {
			task.Fn()
		}
	}
}

func (p *WorkerPool) Submit(task func()) {
	if p == nil || task == nil {
		return
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	taskObj := &Task{Fn: task}

	select {
	case p.taskQueue <- taskObj:
	default:
		// Пропускаем задачу если очередь заполнена
	}
}

func (p *WorkerPool) Shutdown() {
	if p == nil {
		return
	}

	p.once.Do(func() {
		p.mu.Lock()
		p.closed = true
		close(p.taskQueue)
		p.mu.Unlock()
		p.wg.Wait()
	})
}

func (p *WorkerPool) Flush() error {
	return nil
}
