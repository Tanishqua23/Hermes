package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tanishqua/hermes/models"
)

// Queue key names
const (
	KeyHigh       = "hermes:queue:high"
	KeyNormal     = "hermes:queue:normal"
	KeyLow        = "hermes:queue:low"
	KeyProcessing = "hermes:queue:processing"
	KeyDead       = "hermes:queue:dead"
	KeyJobPrefix  = "hermes:job:"
)

// Queue wraps Redis operations for job queuing.
type Queue struct {
	rdb *redis.Client
}

func New(addr string) (*Queue, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Queue{rdb: rdb}, nil
}

func (q *Queue) Close() error { return q.rdb.Close() }

// Enqueue pushes a job onto the appropriate priority queue.
// It also stores the full job blob in a Redis hash for fast lookup.
func (q *Queue) Enqueue(ctx context.Context, job *models.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	pipe := q.rdb.Pipeline()
	// Store full job data
	pipe.Set(ctx, KeyJobPrefix+job.ID, data, 48*time.Hour)
	// Push ID onto priority queue
	pipe.LPush(ctx, priorityKey(job.Priority), job.ID)
	_, err = pipe.Exec(ctx)
	return err
}

// Dequeue blocks waiting for the next job across all priority queues
// (high → normal → low) using BRPOPLPUSH for at-least-once delivery.
// The job ID is moved to the processing list atomically.
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*models.Job, error) {
	// Try high → normal → low in order (non-blocking RPOPLPUSH, then block on normal)
	queues := []string{KeyHigh, KeyNormal, KeyLow}

	// First try non-blocking pop from high priority
	for _, key := range queues {
		id, err := q.rdb.RPopLPush(ctx, key, KeyProcessing).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		return q.getJob(ctx, id)
	}

	// Block on normal queue (catches most traffic)
	result, err := q.rdb.BRPopLPush(ctx, KeyNormal, KeyProcessing, timeout).Result()
	if err == redis.Nil {
		return nil, nil // timeout, no jobs
	}
	if err != nil {
		return nil, err
	}
	return q.getJob(ctx, result)
}

// Ack removes the job from the processing list after successful completion.
func (q *Queue) Ack(ctx context.Context, jobID string) error {
	return q.rdb.LRem(ctx, KeyProcessing, 1, jobID).Err()
}

// Nack puts the job back onto its priority queue (for retry)
// or onto the dead-letter queue (exhausted retries).
func (q *Queue) Nack(ctx context.Context, job *models.Job) error {
	pipe := q.rdb.Pipeline()
	// Remove from processing
	pipe.LRem(ctx, KeyProcessing, 1, job.ID)

	if job.RetryCount >= job.MaxRetries {
		// Dead letter
		pipe.LPush(ctx, KeyDead, job.ID)
	} else {
		// Re-enqueue with a delay stored in the job blob
		data, _ := json.Marshal(job)
		pipe.Set(ctx, KeyJobPrefix+job.ID, data, 48*time.Hour)
		pipe.LPush(ctx, priorityKey(job.Priority), job.ID)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// QueueLengths returns lengths of all queues for the stats endpoint.
func (q *Queue) QueueLengths(ctx context.Context) (map[string]int64, error) {
	pipe := q.rdb.Pipeline()
	cmds := map[string]*redis.IntCmd{
		"high":       pipe.LLen(ctx, KeyHigh),
		"normal":     pipe.LLen(ctx, KeyNormal),
		"low":        pipe.LLen(ctx, KeyLow),
		"processing": pipe.LLen(ctx, KeyProcessing),
		"dead":       pipe.LLen(ctx, KeyDead),
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(cmds))
	for k, cmd := range cmds {
		result[k] = cmd.Val()
	}
	return result, nil
}

// getJob fetches the full job blob from Redis by ID.
func (q *Queue) getJob(ctx context.Context, id string) (*models.Job, error) {
	data, err := q.rdb.Get(ctx, KeyJobPrefix+id).Bytes()
	if err != nil {
		return nil, fmt.Errorf("get job %s: %w", id, err)
	}
	var job models.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func priorityKey(p models.Priority) string {
	switch {
	case p >= models.PriorityHigh:
		return KeyHigh
	case p <= models.PriorityLow:
		return KeyLow
	default:
		return KeyNormal
	}
}
