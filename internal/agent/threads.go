package agent

import "sync"

type WorkerPool struct {
	taskQueue chan func()
	wg        sync.WaitGroup
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	pool := &WorkerPool{
		taskQueue: make(chan func(), 100),
	}

	for i := 0; i < maxWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for task := range p.taskQueue {
		task()
	}
}

func (p *WorkerPool) Submit(task func()) {
	p.taskQueue <- task
}

func (p *WorkerPool) Shutdown() {
	close(p.taskQueue)
	p.wg.Wait()
}
