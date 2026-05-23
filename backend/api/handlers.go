package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tanishqua/hermes/models"
	"github.com/tanishqua/hermes/queue"
	"github.com/tanishqua/hermes/store"
	"github.com/tanishqua/hermes/worker"
)

type Handler struct {
	store *store.Store
	queue *queue.Queue
}

func NewHandler(s *store.Store, q *queue.Queue) *Handler {
	return &Handler{store: s, queue: q}
}

// POST /jobs — enqueue a new job
func (h *Handler) EnqueueJob(w http.ResponseWriter, r *http.Request) {
	var req models.EnqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "field 'type' is required")
		return
	}
	if _, ok := worker.Registry[req.Type]; !ok {
		writeError(w, http.StatusBadRequest, "unknown job type: "+req.Type)
		return
	}
	if req.Priority == 0 {
		req.Priority = models.PriorityNormal
	}
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}

	scheduledAt := time.Now()
	if req.ScheduledAt != nil {
		scheduledAt = *req.ScheduledAt
	}

	job := &models.Job{
		ID:          uuid.New().String(),
		Type:        req.Type,
		Payload:     req.Payload,
		Status:      models.StatusPending,
		Priority:    req.Priority,
		MaxRetries:  req.MaxRetries,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ScheduledAt: scheduledAt,
	}

	if err := h.store.Insert(r.Context(), job); err != nil {
		writeError(w, http.StatusInternalServerError, "db insert: "+err.Error())
		return
	}
	if err := h.queue.Enqueue(r.Context(), job); err != nil {
		writeError(w, http.StatusInternalServerError, "enqueue: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

// GET /jobs — list jobs with optional filters
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	jobType := r.URL.Query().Get("type")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	jobs, total, err := h.store.List(r.Context(), status, jobType, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db list: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ListResponse{
		Jobs:  jobs,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GET /jobs/{id} — get a single job
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// GET /stats — aggregate counts from Postgres + queue lengths from Redis
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	dbStats, err := h.store.Stats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	qLengths, err := h.queue.QueueLengths(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"db":    dbStats,
		"redis": qLengths,
	})
}

// GET /job-types — list all registered handler types
func (h *Handler) GetJobTypes(w http.ResponseWriter, r *http.Request) {
	types := make([]string, 0, len(worker.Registry))
	for k := range worker.Registry {
		types = append(types, k)
	}
	writeJSON(w, http.StatusOK, map[string]any{"types": types})
}

// GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}