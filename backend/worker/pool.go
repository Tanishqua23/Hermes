package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/tanishqua/hermes/models"
	"github.com/tanishqua/hermes/queue"
	"github.com/tanishqua/hermes/store"
)

// Pool manages a fixed set of goroutines that pull and execute jobs.
type Pool struct {
	concurrency int
	q           *queue.Queue
	store       *store.Store
	wg          sync.WaitGroup
	quit        chan struct{}
}

func NewPool(concurrency int, q *queue.Queue, s *store.Store) *Pool {
	return &Pool{
		concurrency: concurrency,
		q:           q,
		store:       s,
		quit:        make(chan struct{}),
	}
}

// Start launches worker goroutines and blocks until Stop() is called.
func (p *Pool) Start() {
	slog.Info("worker pool starting", "concurrency", p.concurrency)
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.runWorker(i)
	}
	p.wg.Wait()
	slog.Info("worker pool stopped")
}

// Stop signals all workers to exit gracefully.
func (p *Pool) Stop() {
	slog.Info("shutting down worker pool...")
	close(p.quit)
}

func (p *Pool) runWorker(id int) {
	defer p.wg.Done()
	log := slog.With("worker", id)
	log.Info("worker started")

	for {
		select {
		case <-p.quit:
			log.Info("worker stopping")
			return
		default:
			job, err := p.q.Dequeue(context.Background(), 2*time.Second)
			if err != nil {
				log.Error("dequeue error", "err", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if job == nil {
				// No job available (timeout), loop back
				continue
			}
			p.process(id, job)
		}
	}
}

func (p *Pool) process(workerID int, job *models.Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log := slog.With("worker", workerID, "job_id", job.ID, "type", job.Type)
	log.Info("processing job")

	// Mark as processing in Postgres
	now := time.Now()
	job.Status = models.StatusProcessing
	job.StartedAt = &now
	job.UpdatedAt = now
	if err := p.store.UpdateStatus(ctx, job); err != nil {
		log.Error("failed to update status to processing", "err", err)
	}

	// Find handler
	handler, ok := Registry[job.Type]
	if !ok {
		p.fail(ctx, job, fmt.Errorf("no handler registered for type %q", job.Type))
		return
	}

	// Execute
	result, err := handler(ctx, job.Payload)
	if err != nil {
		log.Warn("job failed", "err", err, "retry_count", job.RetryCount)
		p.fail(ctx, job, err)
		return
	}

	// Success
	finished := time.Now()
	job.Status = models.StatusCompleted
	job.Result = result
	job.FinishedAt = &finished
	job.UpdatedAt = finished

	if err := p.store.UpdateStatus(ctx, job); err != nil {
		log.Error("failed to update status to completed", "err", err)
	}
	if err := p.q.Ack(ctx, job.ID); err != nil {
		log.Error("ack failed", "err", err)
	}
	log.Info("job completed", "duration_ms", finished.Sub(*job.StartedAt).Milliseconds())
}

func (p *Pool) fail(ctx context.Context, job *models.Job, err error) {
	job.RetryCount++
	job.Error = err.Error()
	job.UpdatedAt = time.Now()

	if job.RetryCount > job.MaxRetries {
		job.Status = models.StatusDead
		finished := time.Now()
		job.FinishedAt = &finished
	} else {
		job.Status = models.StatusFailed
	}

	if dbErr := p.store.UpdateStatus(ctx, job); dbErr != nil {
		slog.Error("failed to persist failure", "job_id", job.ID, "err", dbErr)
	}
	if qErr := p.q.Nack(ctx, job); qErr != nil {
		slog.Error("nack failed", "job_id", job.ID, "err", qErr)
	}
}
