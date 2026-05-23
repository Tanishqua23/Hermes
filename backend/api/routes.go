package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/tanishqua/hermes/queue"
	"github.com/tanishqua/hermes/store"
)

func NewRouter(s *store.Store, q *queue.Queue) http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Recoverer)
	r.Use(Logger)

	h := NewHandler(s, q)

	r.Get("/health", h.Health)
	r.Get("/stats", h.GetStats)
	r.Get("/job-types", h.GetJobTypes)

	r.Route("/jobs", func(r chi.Router) {
		r.Post("/", h.EnqueueJob)
		r.Get("/", h.ListJobs)
		r.Get("/{id}", h.GetJob)
	})

	return r
}
