package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Connect(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db: cannot open: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("db: cannot ping: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	return db
}

// Migrate creates tables if they don't exist.
func Migrate(db *sql.DB) {
	schema := `
CREATE TABLE IF NOT EXISTS jobs (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    priority     TEXT NOT NULL DEFAULT 'default',
    status       TEXT NOT NULL DEFAULT 'pending',
    retries      INT  NOT NULL DEFAULT 0,
    max_retries  INT  NOT NULL DEFAULT 3,
    result       TEXT,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_jobs_status   ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority);
CREATE INDEX IF NOT EXISTS idx_jobs_created  ON jobs(created_at DESC);
`
	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("db: migrate failed: %v", err)
	}
	log.Println("db: schema ready")
}
