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
}

// WorkerPool представляет пул воркеров с пулом объектов
type WorkerPool struct {
	workers   int
	taskQueue chan *Task
	wg        sync.WaitGroup
	once      sync.Once
	taskPool  *pool.Pool[*Task] // Изменено на *Task (указатель)
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(workers int) (*WorkerPool, error) {
	if workers <= 0 {
		workers = 1
	}

	pool := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan *Task, workers*10),
		taskPool:  pool.New[*Task](), // Создаем пул для указателей на Task
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
			// Выполняем задачу
			p.processTask(task)

			// Возвращаем задачу в пул после выполнения
			p.taskPool.Put(task)
		}
	}
}

// processTask обрабатывает одну задачу
func (p *WorkerPool) processTask(task *Task) {
	// Ваша логика обработки HTTP запросов
	// Например:
	// result, err := p.makeRequest(task)
	// task.Result = result
	// task.Error = err
	// task.Attempts++
}

// SubmitTask добавляет новую задачу в очередь
func (p *WorkerPool) SubmitTask(url, method string, body []byte) {
	if p == nil || p.taskQueue == nil {
		return
	}

	// Берем Task из пула (указатель)
	task := p.taskPool.Get()

	// Заполняем поля задачи
	task.URL = url
	task.Method = method
	task.Body = body
	task.Attempts = 0

	// Отправляем задачу в очередь воркеров
	select {
	case p.taskQueue <- task:
		// Задача добавлена в очередь
	default:
		// Если очередь заполнена, возвращаем задачу в пул
		p.taskPool.Put(task)
	}
}

// Submit добавляет готовую задачу в очередь
func (p *WorkerPool) Submit(task *Task) {
	if p == nil || p.taskQueue == nil || task == nil {
		return
	}

	select {
	case p.taskQueue <- task:
		// Задача добавлена в очередь
	default:
		// Если очередь заполнена, возвращаем задачу в пул
		p.taskPool.Put(task)
	}
}

// CreateTask создает новую задачу (использует пул)
func (p *WorkerPool) CreateTask() *Task {
	return p.taskPool.Get()
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
