package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Task represents a unit of work with acknowledgment and negative-acknowledgment callbacks.
type Task struct {
	Event MediaConfirmedEvent
	Ack   func()
	Nack  func()
}

// WorkerPool manages a concurrent pool of goroutines executing tasks with rate/concurrency limiting.
type WorkerPool struct {
	concurrency int
	tasksChan   chan Task
	processor   EventProcessor
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
	mu          sync.Mutex
}

// NewWorkerPool creates an initialized WorkerPool.
func NewWorkerPool(concurrency int, queueSize int, processor EventProcessor) (*WorkerPool, error) {
	if concurrency <= 0 {
		return nil, fmt.Errorf("concurrency must be greater than 0")
	}
	if queueSize <= 0 {
		queueSize = concurrency * 2
	}
	if processor == nil {
		return nil, fmt.Errorf("processor cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		concurrency: concurrency,
		tasksChan:   make(chan Task, queueSize),
		processor:   processor,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start launches worker goroutines.
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.concurrency; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case task, ok := <-wp.tasksChan:
			if !ok {
				return
			}
			wp.processTask(task)
		}
	}
}

func (wp *WorkerPool) processTask(task Task) {
	taskCtx, cancel := context.WithTimeout(wp.ctx, 10*time.Minute)
	defer cancel()

	err := wp.processor.ProcessMediaConfirmed(taskCtx, task.Event)
	if err != nil {
		log.Printf("[WorkerPool] Failed processing media %s (user: %s): %v", task.Event.MediaID, task.Event.UserID, err)
		if task.Nack != nil {
			task.Nack()
		}
		return
	}

	if task.Ack != nil {
		task.Ack()
	}
}

// Submit queues a task to be processed by worker goroutines.
func (wp *WorkerPool) Submit(ctx context.Context, task Task) error {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return fmt.Errorf("worker pool is closed")
	}
	wp.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool shutting down")
	case wp.tasksChan <- task:
		return nil
	}
}

// Stop gracefully stops the worker pool, drains queued tasks, and waits for active workers to finish.
func (wp *WorkerPool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return nil
	}
	wp.closed = true
	close(wp.tasksChan)
	wp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.cancel()
		return nil
	case <-ctx.Done():
		wp.cancel()
		return ctx.Err()
	}
}
