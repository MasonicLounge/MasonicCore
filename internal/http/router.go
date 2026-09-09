package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/config"
	"github.com/masoniclounge/masoniccore/internal/db"
	"github.com/masoniclounge/masoniccore/internal/handlers"
	"github.com/masoniclounge/masoniccore/internal/services"
	"github.com/masoniclounge/masoniccore/internal/store"
)

// Dependencies carries the services the router wires together.
type Dependencies struct {
	Database *db.DB
	Version  string
	Config   config.Config
}

// NewRouter assembles the HTTP router with all middleware and routes.
func NewRouter(deps Dependencies) (http.Handler, error) {
	jwtm := auth.NewManager(deps.Config.JWTSecret, deps.Config.JWTIssuer, deps.Config.AccessTTL)
	hasher := auth.NewPasswordHasher()

	users := store.NewUserStore(deps.Database.Pool())
	sessions := store.NewSessionStore(deps.Database.Pool())
	authSvc := services.NewAuthService(users, sessions, hasher, jwtm, deps.Config)

	health := handlers.NewHealth(deps.Database)
	versionHandler := handlers.NewVersion(deps.Database, deps.Version)
	auth := handlers.NewAuth(authSvc, deps.Config)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/healthz", health.Health)
	r.Get("/readyz", health.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", health.Health)
		r.Get("/version", versionHandler.Get)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", auth.Register)
			r.Post("/login", auth.Login)
			r.Post("/refresh", auth.Refresh)
			r.Post("/logout", auth.Logout)

			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Get("/me", auth.Me)
			})
		})
	})

	return r, nil
}
