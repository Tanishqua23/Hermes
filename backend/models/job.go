package models

import (
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusDead       JobStatus = "dead" // exhausted all retries
)

type Priority int

const (
	PriorityLow    Priority = 1
	PriorityNormal Priority = 5
	PriorityHigh   Priority = 10
)

// Job is the core domain object.
type Job struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`         // e.g. "email", "resize_image", "send_notification"
	Payload     map[string]any    `json:"payload"`      // arbitrary JSON payload
	Status      JobStatus         `json:"status"`
	Priority    Priority          `json:"priority"`
	MaxRetries  int               `json:"max_retries"`
	RetryCount  int               `json:"retry_count"`
	Error       string            `json:"error,omitempty"`
	Result      map[string]any    `json:"result,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ScheduledAt time.Time         `json:"scheduled_at"` // run-at time (can be future)
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	FinishedAt  *time.Time        `json:"finished_at,omitempty"`
}

// EnqueueRequest is the JSON body for POST /jobs
type EnqueueRequest struct {
	Type        string         `json:"type"`
	Payload     map[string]any `json:"payload"`
	Priority    Priority       `json:"priority"`
	MaxRetries  int            `json:"max_retries"`
	ScheduledAt *time.Time     `json:"scheduled_at,omitempty"`
}

// Stats is returned by GET /stats
type Stats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Dead       int64 `json:"dead"`
	Total      int64 `json:"total"`
}

// ListResponse wraps paginated job list
type ListResponse struct {
	Jobs  []*Job `json:"jobs"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
