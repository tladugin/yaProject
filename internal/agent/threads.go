package agent

import (
	"github.com/tladugin/yaProject.git/internal/pool"
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
	Fn       func() // Функция для выполнения
}

// WorkerPool представляет пул воркеров с поддержкой пула объектов
type WorkerPool struct {
	workers   int
	taskQueue chan *Task // Канал для задач
	wg        sync.WaitGroup
	once      sync.Once
	taskPool  *pool.Pool[*Task] // Пул для переиспользования Task объектов
	mu        sync.RWMutex      // Защита от гонок данных
	closed    bool              // Флаг закрытия пула
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(workers int) (*WorkerPool, error) {
	if workers <= 0 {
		workers = 1
	}

	wp := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan *Task, workers*10),
		taskPool:  pool.New[*Task](),
		closed:    false,
	}

	// Запускаем воркеры
	wp.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go wp.worker()
	}

	return wp, nil
}

// worker обрабатывает задачи из очереди
func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for task := range p.taskQueue {
		if task != nil && task.Fn != nil {
			// Выполняем функцию
			task.Fn()

			// Возвращаем задачу в пул
			p.mu.RLock()
			if !p.closed {
				p.taskPool.Put(task)
			}
			p.mu.RUnlock()
		}
	}
}

// Submit добавляет задачу-функцию в очередь
func (p *WorkerPool) Submit(task func()) {
	if p == nil || task == nil {
		return
	}

	p.mu.RLock()
	if p.closed || p.taskQueue == nil {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	// Берем задачу из пула
	taskObj := p.taskPool.Get()
	if taskObj == nil {
		// Если пул вернул nil, создаем новую задачу
		taskObj = &Task{}
	}
	taskObj.Fn = task // Сохраняем функцию

	select {
	case p.taskQueue <- taskObj:
		// Задача добавлена в очередь
	default:
		// Если очередь заполнена, возвращаем задачу в пул
		p.taskPool.Put(taskObj)
	}
}

// Shutdown останавливает пул воркеров
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
