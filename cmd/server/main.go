package main

import (
	"context"
	"errors"
	"log/slog"
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
	jwtpkg "krishblog/pkg/jwt"
	"krishblog/pkg/logger"
	"krishblog/pkg/validator"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("config: " + err.Error())
	}

	log := logger.New(cfg.App.Env)
	startedAt := time.Now()

	log.Info("starting", slog.String("version", version), slog.String("env", cfg.App.Env))

	// ── Postgres ──────────────────────────────────────────────────────────────
	pg, err := database.NewPostgres(cfg.Database.URL)
	if err != nil {
		log.Error("postgres", "error", err)
		os.Exit(1)
	}
	defer pg.Close()
	log.Info("postgres connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb, err := database.NewRedis(cfg.Redis.URL)
	if err != nil {
		log.Error("redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("redis connected")

	// ── Ent ───────────────────────────────────────────────────────────────────
	entClient, err := database.NewEntClient(pg)
	if err != nil {
		log.Error("ent client", "error", err)
		os.Exit(1)
	}
	defer entClient.Close()

	if cfg.IsDevelopment() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := database.RunMigrations(ctx, entClient); err != nil {
			log.Error("migration", "error", err)
			cancel()
			os.Exit(1)
		}
		cancel()
		log.Info("migrations applied")
	}

	// ── JWT ───────────────────────────────────────────────────────────────────
	jwtManager := jwtpkg.NewManager(cfg.JWT.Secret, cfg.JWT.ExpiryHours, cfg.JWT.RefreshExpiryHours)

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc := auth.NewService(entClient, rdb, jwtManager, cfg, log)

	sectionRepo := sections.NewRepository(entClient)
	sectionSvc := sections.NewService(sectionRepo)

	postRepo := posts.NewRepository(entClient)
	postSvc := posts.NewService(postRepo)

	analyticsSvc := analytics.NewService(pg, rdb, log)
	mediaSvc := media.NewService(pg, rdb)

	// ── Analytics processor ───────────────────────────────────────────────────────
	analyticsProcessor := analytics.NewProcessor(
		analyticsSvc.Repository(),
		rdb,
		log,
	)
	processorCtx, processorCancel := context.WithCancel(context.Background())
	defer processorCancel()
	go analyticsProcessor.Run(processorCtx)
	log.Info("analytics processor started")

	// ── Handlers ──────────────────────────────────────────────────────────────
	handlers := api.Handlers{
		Health: health.New(health.Dependencies{
			Postgres:  pg,
			Redis:     rdb,
			Version:   version,
			StartedAt: startedAt,
		}),
		Auth:      auth.NewHandler(authSvc),
		Posts:     posts.NewHandler(postSvc),
		Sections:  sections.NewHandler(sectionSvc),
		Analytics: analytics.NewHandler(analyticsSvc),
		Media:     media.NewHandler(mediaSvc),
	}

	// ── Echo ──────────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = validator.New()
	e.HTTPErrorHandler = httpErrorHandler

	api.Register(e, handlers, api.RouterConfig{
		AllowedOrigins: cfg.CORS.AllowedOrigins,
		RPS:            cfg.RateLimit.RPS,
		Burst:          cfg.RateLimit.Burst,
		JWTManager:     jwtManager,
		Redis:          rdb,
		Logger:         log,
	})

	// ── Serve ─────────────────────────────────────────────────────────────────
	serverErr := make(chan error, 1)
	go func() {
		addr := ":" + cfg.App.Port
		log.Info("listening", slog.String("addr", addr))
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		log.Info("shutdown", slog.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "error", err)
		os.Exit(1)
	}
	log.Info("stopped")
}

func httpErrorHandler(err error, c echo.Context) {
	var he *echo.HTTPError
	if errors.As(err, &he) {
		_ = c.JSON(he.Code, map[string]interface{}{
			"success": false,
			"error":   map[string]interface{}{"code": http.StatusText(he.Code), "message": he.Message},
		})
		return
	}
	_ = c.JSON(http.StatusInternalServerError, map[string]interface{}{
		"success": false,
		"error":   map[string]interface{}{"code": "INTERNAL_ERROR", "message": "an unexpected error occurred"},
	})
}

var _ = mw.GetRequestID
