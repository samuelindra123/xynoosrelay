package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xynoss/relay/services/api/internal/auth"
	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/database"
	"github.com/xynoss/relay/services/api/internal/health"
	"github.com/xynoss/relay/services/api/internal/media"
	"github.com/xynoss/relay/services/api/internal/messaging"
	"github.com/xynoss/relay/services/api/internal/middleware"
	"github.com/xynoss/relay/services/api/internal/response"
	"github.com/xynoss/relay/services/api/internal/session"
	"github.com/xynoss/relay/services/api/internal/storage"
	"github.com/xynoss/relay/services/api/internal/users"
	"github.com/xynoss/relay/services/api/internal/workspaces"
)

// Server holds dependencies for the HTTP server and Chi router.
type Server struct {
	cfg               *config.Config
	log               *slog.Logger
	db                database.DB
	storageProv       storage.StorageProvider
	mediaStore        *media.PGStore
	mediaService      *media.Service
	sessionService    *session.Service
	usersService      *users.Service
	authService       *auth.Service
	workspacesService *workspaces.Service
	emailSender       auth.EmailSender
	router            chi.Router
}

// New constructs a configured Server instance with modular monolith services.
func New(cfg *config.Config, log *slog.Logger, db database.DB) *Server {
	s := &Server{
		cfg: cfg,
		log: log,
		db:  db,
	}

	// 1. Initialize Storage Provider
	var storageProv storage.StorageProvider
	if cfg.StorageProvider == "s3" || cfg.StorageProvider == "r2" || cfg.StorageProvider == "neon" {
		s3Prov, err := storage.NewS3Provider(storage.S3Config{
			Endpoint:        cfg.StorageEndpoint,
			Bucket:          cfg.StorageBucket,
			Region:          cfg.StorageRegion,
			AccessKeyID:     cfg.StorageAccessKeyID,
			SecretAccessKey: cfg.StorageSecretAccessKey,
			PublicBaseURL:   cfg.StoragePublicBaseURL,
		})
		if err != nil {
			// A remote provider must never silently become ephemeral local storage.
			panic("remote storage initialization failed: check storage configuration")
		} else {
			storageProv = s3Prov
		}
	} else {
		localProv, _ := storage.NewLocalProvider(cfg.StorageLocalDir, cfg.StoragePublicBaseURL, cfg.SessionHashKey)
		storageProv = localProv
	}
	s.storageProv = storageProv

	var pool *pgxpool.Pool
	if db != nil {
		pool = db.RawPool()
	}

	// 2. Initialize Media Layer
	mediaStore := media.NewPGStore(pool)
	s.mediaStore = mediaStore
	s.mediaService = media.NewService(mediaStore, storageProv)

	// 3. Initialize Session Layer
	sessionStore := session.NewPGStore(pool)
	s.sessionService = session.NewService(sessionStore, cfg.SessionHashKey)

	// 4. Initialize Users & Profile Layer
	usersStore := users.NewPGStore(pool)
	s.usersService = users.NewService(usersStore, s.mediaService, storageProv, cfg.APIBaseURL)

	// 5. Initialize Authentication Layer
	s.emailSender = auth.NewConfiguredEmailSender(cfg, log)
	authStore := auth.NewPGStore(pool)
	s.authService = auth.NewService(cfg, authStore, usersStore, s.sessionService, s.emailSender)

	// 6. Initialize Workspaces Layer
	workspacesStore := workspaces.NewPGStore(pool)
	s.workspacesService = workspaces.NewService(workspacesStore, cfg.FrontendURL)

	s.setupRouter()
	return s
}

// Router returns the initialized Chi router.
func (s *Server) Router() chi.Router {
	return s.router
}

// EmailSender returns the active test double email sender (useful for testing).
func (s *Server) EmailSender() *auth.TestDoubleEmailSender {
	if td, ok := s.emailSender.(*auth.TestDoubleEmailSender); ok {
		return td
	}
	return nil
}

// SessionService returns the session service.
func (s *Server) SessionService() *session.Service {
	return s.sessionService
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// 1. Core defensive middlewares
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.SecurityHeaders(s.cfg.IsProduction()))
	r.Use(middleware.CORS(s.cfg.WebOrigins))
	r.Use(middleware.RequestLogger(s.log))
	r.Use(middleware.Recoverer(s.log))
	r.Use(middleware.Timeout(s.cfg.RequestTimeout))
	r.Use(middleware.Authenticate(s.sessionService))
	r.Use(middleware.RequireCSRF(s.sessionService))

	// 2. Custom 404 and 405 error handlers conforming to standard error envelope
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		response.RenderError(w, req, response.NewNotFoundError("the requested endpoint was not found"))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		response.RenderError(w, req, response.NewError(
			http.StatusMethodNotAllowed,
			response.CodeMalformedRequest,
			"method not allowed for this resource",
		))
	})

	// 3. Health & Monitoring Handlers
	healthHandler := health.NewHandler(s.db, s.cfg.AppEnv)
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)
	r.Get("/health", healthHandler.Status)

	// 4. API Handlers
	authHandler := auth.NewHandler(s.authService, s.sessionService, s.usersService, s.cfg.IsProduction(), s.cfg.FrontendURL)
	usersHandler := users.NewHandler(s.usersService)
	sessionHandler := session.NewHandler(s.sessionService, s.cfg.IsProduction())
	mediaHandler := media.NewHandler(s.mediaService, s.mediaStore)
	workspacesHandler := workspaces.NewHandler(s.workspacesService)
	var messagingPool *pgxpool.Pool
	if s.db != nil {
		messagingPool = s.db.RawPool()
	}
	messagingHandler := messaging.NewHandler(messagingPool)

	// 5. API v1 Route Group
	r.Route("/v1", func(v1 chi.Router) {
		v1.Mount("/health", healthHandler.Routes())
		v1.Mount("/auth", authHandler.Routes(middleware.RequireAuth))
		v1.Mount("/media", mediaHandler.Routes(middleware.RequireAuth))
		v1.Mount("/invites", workspacesHandler.InviteRoutes(middleware.RequireAuth))

		// Authenticated user & session routes
		v1.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth)
			protected.Mount("/me", usersHandler.MeRoutes())
			protected.Mount("/me/sessions", sessionHandler.Routes())
			protected.Mount("/users", usersHandler.UserMediaRoutes())
			protected.Mount("/workspaces", workspacesHandler.WorkspaceRoutes())
			protected.Mount("/channels", messagingHandler.Routes())
			protected.Mount("/conversations", messagingHandler.DirectRoutes())
		})

		// Internal service-to-service protected routes
		v1.Route("/internal", func(internal chi.Router) {
			internal.Use(middleware.RequireInternalAPIKey(s.cfg.InternalAPIKeys))
			internal.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
				response.JSON(w, req, http.StatusOK, map[string]string{
					"status":  "trusted_internal_service_connected",
					"message": "service identity verified via X-Internal-API-Key",
				})
			})
		})
	})

	s.router = r
}
