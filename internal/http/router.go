package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/masoniclounge/masoniccore/internal/db"
	"github.com/masoniclounge/masoniccore/internal/handlers"
)

// NewRouter assembles the HTTP router with all middleware and routes.
func NewRouter(database *db.DB, version string, _ *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	health := handlers.NewHealth(database)
	versionHandler := handlers.NewVersion(database, version)

	r.Get("/healthz", health.Health)
	r.Get("/readyz", health.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", health.Health)
		r.Get("/version", versionHandler.Get)
	})

	return r
}
