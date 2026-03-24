package api

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"krishblog/internal/analytics"
	"krishblog/internal/api/health"
	"krishblog/internal/auth"
	"krishblog/internal/database"
	"krishblog/internal/media"
	mw "krishblog/internal/middleware"
	"krishblog/internal/posts"
	"krishblog/internal/sections"
	"krishblog/internal/subscribers"
	jwtpkg "krishblog/pkg/jwt"
)

type Handlers struct {
	Health      *health.Handler
	Auth        *auth.Handler
	Posts       *posts.Handler
	Sections    *sections.Handler
	Analytics   *analytics.Handler
	Media       *media.Handler
	Subscribers *subscribers.Handler
}

type RouterConfig struct {
	AllowedOrigins []string
	RPS            float64
	Burst          int
	JWTManager     *jwtpkg.Manager
	Redis          *database.Redis
	Logger         *slog.Logger
}

func Register(e *echo.Echo, h Handlers, cfg RouterConfig) {
	e.Use(mw.RequestID())
	e.Use(mw.Recover(cfg.Logger))
	e.Use(mw.Logger(cfg.Logger))
	e.Use(mw.SecureHeaders())
	e.Use(mw.CORS(cfg.AllowedOrigins))
	e.Use(echomw.Decompress())

	e.GET("/health", h.Health.Live)
	e.GET("/ready", h.Health.Ready)

	v1 := e.Group("/v1")

	authGroup := v1.Group("/auth")
	authGroup.Use(mw.StrictRateLimiter(cfg.Redis, 5))
	authGroup.POST("/login", h.Auth.Login)
	authGroup.POST("/refresh", h.Auth.Refresh)
	authGroup.POST("/logout", h.Auth.Logout, mw.Auth(cfg.JWTManager))
	authGroup.GET("/me", h.Auth.Me, mw.Auth(cfg.JWTManager))

	pub := v1.Group("/public")
	pub.Use(mw.RateLimiter(cfg.Redis, cfg.RPS, cfg.Burst))
	pub.GET("/posts", h.Posts.List)
	pub.GET("/posts/:slug", h.Posts.GetBySlug)
	pub.GET("/sections", h.Sections.ListPublic)
	pub.GET("/sections/:slug", h.Sections.GetBySlug)

	subGroup := v1.Group("/subscribe")
	subGroup.Use(mw.StrictRateLimiter(cfg.Redis, 10))
	subGroup.POST("", h.Subscribers.Subscribe)
	subGroup.GET("/confirm", h.Subscribers.Confirm)
	subGroup.GET("/unsubscribe", h.Subscribers.Unsubscribe)

	analyticsGroup := v1.Group("/analytics")
	analyticsGroup.Use(mw.RateLimiter(cfg.Redis, cfg.RPS*2, cfg.Burst*2))
	analyticsGroup.POST("/event", h.Analytics.RecordEvent)
	analyticsGroup.POST("/session/start", h.Analytics.SessionStart)
	analyticsGroup.POST("/session/end", h.Analytics.SessionEnd)

	admin := v1.Group("/admin")
	admin.Use(mw.Auth(cfg.JWTManager))
	admin.Use(mw.RateLimiter(cfg.Redis, cfg.RPS, cfg.Burst))

	adminPosts := admin.Group("/posts")
	adminPosts.Use(mw.RequireRole("editor"))
	adminPosts.GET("", h.Posts.AdminList)
	adminPosts.GET("/:id", h.Posts.AdminGet)
	adminPosts.POST("", h.Posts.Create)
	adminPosts.PUT("/:id", h.Posts.Update)
	adminPosts.PATCH("/:id/status", h.Posts.UpdateStatus)
	adminPosts.DELETE("/:id", h.Posts.Delete, mw.RequireRole("admin"))

	adminSections := admin.Group("/sections")
	adminSections.Use(mw.RequireRole("admin"))
	adminSections.GET("", h.Sections.AdminList)
	adminSections.POST("", h.Sections.Create)
	adminSections.PUT("/:id", h.Sections.Update)
	adminSections.DELETE("/:id", h.Sections.Delete)

	adminMedia := admin.Group("/media")
	adminMedia.Use(mw.RequireRole("editor"))
	adminMedia.GET("", h.Media.List)
	adminMedia.POST("/upload", h.Media.Upload)
	adminMedia.PATCH("/:id", h.Media.Update)
	adminMedia.DELETE("/:id", h.Media.Delete)

	adminAnalytics := admin.Group("/analytics")
	adminAnalytics.Use(mw.RequireRole("viewer"))
	adminAnalytics.GET("/overview", h.Analytics.AdminOverview)
	adminAnalytics.GET("/posts/:id", h.Analytics.AdminPostStats)
	adminAnalytics.GET("/realtime", h.Analytics.AdminRealtime)

	adminSubs := admin.Group("/subscribers")
	adminSubs.Use(mw.RequireRole("admin"))
	adminSubs.GET("/stats", h.Subscribers.AdminStats)
	adminSubs.POST("/notify", h.Subscribers.AdminNotify)
}
