# Hermes — Distributed Task Queue

A production-grade distributed task queue built with Go, Redis, PostgreSQL, and React.

## Architecture

```
React Dashboard (5173)
        │  REST API
        ▼
  API Server (Go :8080)
  ├── POST /jobs        — enqueue
  ├── GET  /jobs        — list + filter + paginate
  ├── GET  /jobs/:id    — single job
  ├── GET  /stats       — db counts + redis queue lengths
  └── GET  /job-types   — registered handler names

  Worker Pool (5 goroutines, configurable)
  ├── Pulls from Redis: high → normal → low queues
  ├── Executes typed handlers
  ├── Retry logic with exponential intent
  └── Dead-letter queue after max_retries exhausted

  Redis (queues + in-flight state)
  PostgreSQL (durable job history + results)
```

## Quick Start

### 1. Prerequisites
- Docker + Docker Compose
- Go 1.22+
- Node.js 18+

### 2. Start infrastructure
```bash
docker compose up -d
# Wait ~5 seconds for postgres/redis to be ready
```

### 3. Start the backend
```bash
cd backend
go mod tidy        # downloads dependencies
go run .           # starts API + worker pool on :8080
```

### 4. Start the frontend (new terminal)
```bash
cd frontend
npm install
npm run dev        # starts Vite dev server on :5173
```

### 5. Open the dashboard
Visit http://localhost:5173

## API Reference

### Enqueue a job
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "priority": 5,
    "max_retries": 3,
    "payload": {
      "to": "alice@example.com",
      "subject": "Hello!"
    }
  }'
```

### List jobs
```bash
curl "http://localhost:8080/jobs?status=completed&page=1&limit=20"
```

### Get a specific job
```bash
curl http://localhost:8080/jobs/<job-id>
```

### Stats
```bash
curl http://localhost:8080/stats
```

## Job Types

| Type | Required payload fields |
|------|------------------------|
| `send_email` | `to`, `subject` |
| `resize_image` | `source_url`, `width`, `height` |
| `send_notification` | `user_id`, `message` |
| `generate_report` | `report_type` |
| `process_payment` | `amount`, `currency` (fails 20% of the time to demo retries) |

## Adding a new job type

1. Add a handler function in `backend/worker/handlers.go`:
```go
func handleMyJob(ctx context.Context, payload map[string]any) (map[string]any, error) {
    // your logic here
    return map[string]any{"status": "done"}, nil
}
```

2. Register it in `Registry`:
```go
var Registry = map[string]HandlerFunc{
    ...
    "my_job": handleMyJob,
}
```

That's it. The API, dashboard, and worker all pick it up automatically.

## Priority Levels

| Value | Name |
|-------|------|
| 1 | Low |
| 5 | Normal (default) |
| 10 | High |

## Configuration (.env)

```env
POSTGRES_DSN=postgres://hermes:hermes_secret@localhost:5432/hermes?sslmode=disable
REDIS_ADDR=localhost:6379
API_PORT=8080
WORKER_CONCURRENCY=5
```
