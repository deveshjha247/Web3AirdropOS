package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Handler is a function that processes a job
type Handler func(ctx context.Context, job *Job) error

// Worker processes jobs from a queue
type Worker struct {
	queue        *Queue
	workerID     string
	handlers     map[string]Handler
	concurrency  int
	pollInterval time.Duration
	lockDuration time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	isRunning    bool
}

// WorkerConfig configures a worker
type WorkerConfig struct {
	Concurrency  int
	PollInterval time.Duration
	LockDuration time.Duration
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Concurrency:  5,
		PollInterval: 100 * time.Millisecond,
		LockDuration: 5 * time.Minute,
	}
}

// NewWorker creates a new queue worker
func NewWorker(queue *Queue, workerID string, config WorkerConfig) *Worker {
	return &Worker{
		queue:        queue,
		workerID:     workerID,
		handlers:     make(map[string]Handler),
		concurrency:  config.Concurrency,
		pollInterval: config.PollInterval,
		lockDuration: config.LockDuration,
		stopChan:     make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a job type
func (w *Worker) RegisterHandler(jobType string, handler Handler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[jobType] = handler
}

// Start begins processing jobs
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return fmt.Errorf("worker is already running")
	}
	w.isRunning = true
	w.mu.Unlock()

	log.Printf("🚀 Worker %s starting with concurrency %d", w.workerID, w.concurrency)

	// Start worker goroutines
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.processLoop(ctx, i)
	}

	return nil
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	log.Printf("🛑 Worker %s stopping...", w.workerID)
	close(w.stopChan)
	w.wg.Wait()

	w.mu.Lock()
	w.isRunning = false
	w.stopChan = make(chan struct{})
	w.mu.Unlock()

	log.Printf("✅ Worker %s stopped", w.workerID)
}

func (w *Worker) processLoop(ctx context.Context, workerNum int) {
	defer w.wg.Done()

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-w.stopChan:
			return
		case <-workerCtx.Done():
			return
		default:
			job, err := w.queue.Dequeue(workerCtx, fmt.Sprintf("%s-%d", w.workerID, workerNum), w.lockDuration)
			if err != nil {
				log.Printf("⚠️ Worker %s-%d dequeue error: %v", w.workerID, workerNum, err)
				time.Sleep(w.pollInterval)
				continue
			}

			if job == nil {
				// No jobs available
				time.Sleep(w.pollInterval)
				continue
			}

			w.processJob(workerCtx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *Job) {
	w.mu.RLock()
	handler, exists := w.handlers[job.Type]
	w.mu.RUnlock()

	if !exists {
		log.Printf("⚠️ No handler for job type: %s", job.Type)
		w.queue.Fail(ctx, job.ID, fmt.Errorf("no handler for job type: %s", job.Type))
		return
	}

	log.Printf("🔄 Processing job %s (type: %s, attempt: %d/%d)",
		job.ID, job.Type, job.RetryCount+1, job.MaxRetries)

	// Create timeout context
	jobCtx, cancel := context.WithTimeout(ctx, w.lockDuration-30*time.Second)
	defer cancel()

	// Execute handler
	startTime := time.Now()
	err := handler(jobCtx, job)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ Job %s failed in %v: %v", job.ID, duration, err)
		if failErr := w.queue.Fail(ctx, job.ID, err); failErr != nil {
			log.Printf("⚠️ Failed to mark job as failed: %v", failErr)
		}
		return
	}

	log.Printf("✅ Job %s completed in %v", job.ID, duration)
	if completeErr := w.queue.Complete(ctx, job.ID, nil); completeErr != nil {
		log.Printf("⚠️ Failed to mark job as complete: %v", completeErr)
	}
}

// WorkerPool manages multiple workers across queues
type WorkerPool struct {
	workers []*Worker
	mu      sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: make([]*Worker, 0),
	}
}

// AddWorker adds a worker to the pool
func (p *WorkerPool) AddWorker(worker *Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers = append(p.workers, worker)
}

// Start starts all workers
func (p *WorkerPool) Start(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, worker := range p.workers {
		if err := worker.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops all workers
func (p *WorkerPool) Stop() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, worker := range p.workers {
		worker.Stop()
	}
}
