package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tanishqua/hermes/models"
)

type Store struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	s := &Store{db: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS jobs (
			id           TEXT PRIMARY KEY,
			type         TEXT NOT NULL,
			payload      JSONB NOT NULL DEFAULT '{}',
			status       TEXT NOT NULL DEFAULT 'pending',
			priority     INT  NOT NULL DEFAULT 5,
			max_retries  INT  NOT NULL DEFAULT 3,
			retry_count  INT  NOT NULL DEFAULT 0,
			error        TEXT NOT NULL DEFAULT '',
			result       JSONB,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			started_at   TIMESTAMPTZ,
			finished_at  TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS jobs_status_idx ON jobs(status);
		CREATE INDEX IF NOT EXISTS jobs_type_idx   ON jobs(type);
		CREATE INDEX IF NOT EXISTS jobs_created_at ON jobs(created_at DESC);
	`)
	return err
}

func (s *Store) Close() { s.db.Close() }

// Insert persists a new job.
func (s *Store) Insert(ctx context.Context, j *models.Job) error {
	payload, _ := json.Marshal(j.Payload)
	_, err := s.db.Exec(ctx, `
		INSERT INTO jobs (id, type, payload, status, priority, max_retries, retry_count,
		                  error, created_at, updated_at, scheduled_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		j.ID, j.Type, payload, j.Status, j.Priority, j.MaxRetries, j.RetryCount,
		j.Error, j.CreatedAt, j.UpdatedAt, j.ScheduledAt,
	)
	return err
}

// UpdateStatus updates a job's mutable fields after processing.
func (s *Store) UpdateStatus(ctx context.Context, j *models.Job) error {
	result, _ := json.Marshal(j.Result)
	_, err := s.db.Exec(ctx, `
		UPDATE jobs SET status=$1, retry_count=$2, error=$3, result=$4,
		               updated_at=$5, started_at=$6, finished_at=$7
		WHERE id=$8`,
		j.Status, j.RetryCount, j.Error, result,
		j.UpdatedAt, j.StartedAt, j.FinishedAt, j.ID,
	)
	return err
}

// Get returns a single job by ID.
func (s *Store) Get(ctx context.Context, id string) (*models.Job, error) {
	row := s.db.QueryRow(ctx, `SELECT * FROM jobs WHERE id=$1`, id)
	return scanJob(row)
}

// List returns paginated jobs, optionally filtered by status/type.
func (s *Store) List(ctx context.Context, status, jobType string, page, limit int) ([]*models.Job, int, error) {
	offset := (page - 1) * limit

	where := "WHERE 1=1"
	args := []any{}
	idx := 1

	if status != "" {
		where += fmt.Sprintf(" AND status=$%d", idx)
		args = append(args, status)
		idx++
	}
	if jobType != "" {
		where += fmt.Sprintf(" AND type=$%d", idx)
		args = append(args, jobType)
		idx++
	}

	// count
	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs `+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// paginated rows
	args = append(args, limit, offset)
	rows, err := s.db.Query(ctx,
		`SELECT * FROM jobs `+where+fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, nil
}

// Stats returns per-status counts.
func (s *Store) Stats(ctx context.Context) (*models.Stats, error) {
	rows, err := s.db.Query(ctx, `SELECT status, COUNT(*) FROM jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	st := &models.Stats{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		switch models.JobStatus(status) {
		case models.StatusPending:
			st.Pending = count
		case models.StatusProcessing:
			st.Processing = count
		case models.StatusCompleted:
			st.Completed = count
		case models.StatusFailed:
			st.Failed = count
		case models.StatusDead:
			st.Dead = count
		}
		st.Total += count
	}
	return st, nil
}

// DeleteCompleted removes completed jobs older than age.
func (s *Store) DeleteCompleted(ctx context.Context, age time.Duration) (int64, error) {
	t, err := s.db.Exec(ctx,
		`DELETE FROM jobs WHERE status='completed' AND finished_at < $1`,
		time.Now().Add(-age))
	if err != nil {
		return 0, err
	}
	return t.RowsAffected(), nil
}

// scanJob scans a pgx row into a Job struct.
func scanJob(row pgx.Row) (*models.Job, error) {
	j := &models.Job{}
	var payloadRaw, resultRaw []byte
	err := row.Scan(
		&j.ID, &j.Type, &payloadRaw, &j.Status, &j.Priority,
		&j.MaxRetries, &j.RetryCount, &j.Error, &resultRaw,
		&j.CreatedAt, &j.UpdatedAt, &j.ScheduledAt,
		&j.StartedAt, &j.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(payloadRaw) > 0 {
		_ = json.Unmarshal(payloadRaw, &j.Payload)
	}
	if len(resultRaw) > 0 {
		_ = json.Unmarshal(resultRaw, &j.Result)
	}
	return j, nil
}
