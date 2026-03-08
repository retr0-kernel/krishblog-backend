package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"krishblog/internal/analytics"
	"krishblog/internal/api"
	"krishblog/internal/api/health"
	"krishblog/internal/auth"
	"krishblog/internal/config"
	"krishblog/internal/database"
	"krishblog/internal/media"
	mw "krishblog/internal/middleware"
	"krishblog/internal/posts"
	"krishblog/internal/sections"
	"krishblog/internal/subscribers"
	jwtpkg "krishblog/pkg/jwt"
	"krishblog/pkg/logger"
	"krishblog/pkg/validator"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	log := logger.New(cfg.App.Env)
	startedAt := time.Now()

	log.Info("starting server",
		"version", version,
		"env", cfg.App.Env,
		"port", cfg.App.Port,
	)

	// ── Postgres (raw sql.DB — used by posts, analytics, subscribers, media) ──
	pg, err := database.NewPostgres(cfg.Database.URL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pg.Close()
	log.Info("postgres connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	redis, err := database.NewRedis(cfg.Redis.URL)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()
	log.Info("redis connected")

	// ── Ent client (shares the same sql.DB pool — used by auth and sections) ──
	entClient, err := database.NewEntClient(pg)
	if err != nil {
		log.Error("failed to create ent client", "error", err)
		os.Exit(1)
	}
	defer entClient.Close()

	jwtManager := jwtpkg.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.ExpiryHours,
		cfg.JWT.RefreshExpiryHours,
	)

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc := auth.NewService(entClient, redis, jwtManager, cfg, log)
	postsSvc := posts.NewService(pg, redis)
	sectionsRepo := sections.NewRepository(entClient)
	sectionsSvc := sections.NewService(sectionsRepo)
	analyticsSvc := analytics.NewService(pg, redis, log)
	mediaSvc := media.NewService(pg, redis)

	subRepo := subscribers.NewRepository(pg)
	subSvc := subscribers.NewService(subRepo, subscribers.EmailConfig{
		Host:     cfg.Email.Host,
		Port:     cfg.Email.Port,
		Username: cfg.Email.Username,
		Password: cfg.Email.Password,
		From:     cfg.Email.From,
		SiteURL:  cfg.Site.URL,
		SiteName: cfg.Site.Name,
	})

	// ── Handlers ──────────────────────────────────────────────────────────────
	handlers := api.Handlers{
		Health: health.New(health.Dependencies{
			Postgres:  pg,
			Redis:     redis,
			Version:   version,
			StartedAt: startedAt,
		}),
		Auth:        auth.NewHandler(authSvc),
		Posts:       posts.NewHandler(postsSvc),
		Sections:    sections.NewHandler(sectionsSvc),
		Analytics:   analytics.NewHandler(analyticsSvc),
		Media:       media.NewHandler(mediaSvc),
		Subscribers: subscribers.NewHandler(subSvc),
	}

	// ── Analytics background processor ────────────────────────────────────────
	analyticsProcessor := analytics.NewProcessor(analyticsSvc.Repository(), redis, log)
	processorCtx, cancelProcessor := context.WithCancel(context.Background())
	go analyticsProcessor.Run(processorCtx)

	// ── Echo ──────────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = validator.New()

	_ = mw.RequestID

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var he *echo.HTTPError
		if errors.As(err, &he) {
			_ = c.JSON(he.Code, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    http.StatusText(he.Code),
					"message": he.Message,
				},
			})
			return
		}
		// Log the actual error for debugging
		log.Error("unhandled error",
			"error", err,
			"path", c.Request().URL.Path,
			"method", c.Request().Method,
			"request_id", mw.GetRequestID(c),
		)
		_ = c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": "an unexpected error occurred",
			},
		})
	}

	api.Register(e, handlers, api.RouterConfig{
		AllowedOrigins: cfg.CORS.AllowedOrigins,
		RPS:            cfg.RateLimit.RPS,
		Burst:          cfg.RateLimit.Burst,
		JWTManager:     jwtManager,
		Redis:          redis,
		Logger:         log,
	})

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	serverErr := make(chan error, 1)
	go func() {
		addr := ":" + cfg.App.Port
		log.Info("http server listening", "addr", addr)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Error("server error", "error", err)
		cancelProcessor()
		os.Exit(1)
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	cancelProcessor()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped gracefully")
}
