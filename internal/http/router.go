package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/config"
	"github.com/masoniclounge/masoniccore/internal/db"
	"github.com/masoniclounge/masoniccore/internal/handlers"
	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/services"
	"github.com/masoniclounge/masoniccore/internal/storage"
	"github.com/masoniclounge/masoniccore/internal/store"
	"github.com/masoniclounge/masoniccore/internal/ws"
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

	groupsStore := store.NewGroupStore(deps.Database.Pool())
	threadsStore := store.NewThreadStore(deps.Database.Pool())
	postsStore := store.NewPostStore(deps.Database.Pool())
	groupSvc := services.NewGroupService(groupsStore, threadsStore)

	s3, err := storage.New(context.Background(), deps.Config)
	if err != nil {
		return nil, err
	}
	attachmentsStore := store.NewAttachmentStore(deps.Database.Pool())
	mediaSvc := services.NewMediaService(attachmentsStore, users, postsStore, s3, deps.Config)
	media := handlers.NewMedia(mediaSvc, s3)
	moderationStore := store.NewModerationStore(deps.Database.Pool())
	postSvc := services.NewPostService(postsStore, threadsStore, attachmentsStore, moderationStore)
	threadSvc := services.NewThreadService(groupsStore, threadsStore, attachmentsStore, moderationStore)

	settingsStore := store.NewSettingsStore(deps.Database.Pool())
	adminSvc := services.NewAdminService(users, attachmentsStore, settingsStore, moderationStore)
	admin := handlers.NewAdmin(adminSvc)
	publicSettings := handlers.NewPublicSettings(settingsStore)

	installSvc := services.NewInstallService(users, settingsStore, hasher)
	install := handlers.NewInstall(installSvc)

	health := handlers.NewHealth(deps.Database)
	versionHandler := handlers.NewVersion(deps.Database, deps.Version)
	auth := handlers.NewAuth(authSvc, deps.Config)
	groups := handlers.NewGroups(groupSvc)
	threads := handlers.NewThreads(threadSvc)
	posts := handlers.NewPosts(postSvc)

	hub := ws.NewHub()
	go hub.Run(context.Background())
	messagesStore := store.NewMessageStore(deps.Database.Pool())
	notificationsStore := store.NewNotificationStore(deps.Database.Pool())
	messageSvc := services.NewMessageService(messagesStore, users)
	notificationSvc := services.NewNotificationService(notificationsStore)
	messages := handlers.NewMessages(messageSvc, notificationSvc, hub)

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

		r.Get("/install", install.GetStatus)
		r.Post("/install", install.Create)
		r.Get("/settings", publicSettings.Get)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", auth.Register)
			r.Post("/login", auth.Login)
			r.Post("/refresh", auth.Refresh)
			r.Post("/logout", auth.Logout)

			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Get("/me", auth.Me)
				r.Patch("/me", auth.UpdateMe)
				r.Post("/change-password", auth.ChangePassword)
			})
		})

		r.Route("/groups", func(r chi.Router) {
			r.Get("/", groups.List)

			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Use(handlers.RequireRole(models.RoleAdmin))
				r.Post("/", groups.Create)
				r.Patch("/{groupID}", groups.Update)
				r.Delete("/{groupID}", groups.Delete)
			})

			r.Get("/{groupID}", groups.Get)

			r.Route("/{groupID}/threads", func(r chi.Router) {
				r.Get("/", threads.ListByGroup)
				r.Group(func(r chi.Router) {
					r.Use(handlers.RequireAuth(jwtm))
					r.Post("/", threads.Create)
				})
			})
		})

		r.Route("/threads", func(r chi.Router) {
			r.Get("/{threadID}", threads.Get)

			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Patch("/{threadID}", threads.Update)
				r.Delete("/{threadID}", threads.Delete)
			})

			r.Route("/{threadID}/posts", func(r chi.Router) {
				r.Get("/", posts.ListByThread)
				r.Group(func(r chi.Router) {
					r.Use(handlers.RequireAuth(jwtm))
					r.Post("/", posts.Create)
				})
			})
		})

		r.Route("/posts", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Patch("/{postID}", posts.Update)
				r.Delete("/{postID}", posts.Delete)
			})
		})

		r.Route("/media", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Post("/avatar", media.UploadAvatar)
				r.Post("/attachments", media.UploadAttachment)
				r.Delete("/attachments/{attachmentID}", media.DeleteAttachment)
			})
		})

		r.Route("/admin", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Use(handlers.RequireRole(models.RoleAdmin))
				r.Get("/users", admin.ListUsers)
				r.Patch("/users/{userID}", admin.UpdateUser)
				r.Get("/settings", admin.GetSettings)
				r.Put("/settings", admin.UpdateSettings)
				r.Get("/media", admin.ListMedia)
				r.Get("/moderation-log", admin.Log)
			})
		})

		r.Route("/pms", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Get("/", messages.Inbox)
				r.Get("/with", messages.Conversation)
				r.Post("/", messages.Send)
				r.Get("/unread-count", messages.UnreadCount)
				r.Patch("/{messageID}/read", messages.MarkRead)
			})
		})

		r.Route("/notifications", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireAuth(jwtm))
				r.Get("/", messages.Notifications)
				r.Patch("/{notificationID}/read", messages.MarkNotificationRead)
				r.Patch("/read-all", messages.MarkAllNotificationsRead)
			})
		})

		r.Route("/presence", func(r chi.Router) {
			r.Get("/", messages.Online)
			r.Get("/{userID}", messages.Presence)
		})
	})

	r.Get("/media/{bucket}/*", media.Serve)
	r.Get("/ws", handlers.WS(hub, jwtm))

	return r, nil
}
