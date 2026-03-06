package analytics

import (
	"net/http"

	"github.com/labstack/echo/v4"

	mw "krishblog/internal/middleware"
	"krishblog/pkg/response"
)

// Handler handles analytics HTTP routes.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RecordEvent handles POST /v1/analytics/event
// This is the public endpoint called by the frontend JS tracker.
// It must respond in <10ms — all work is async via Redis queue.
func (h *Handler) RecordEvent(c echo.Context) error {
	var req EventRequest
	if err := c.Bind(&req); err != nil {
		// Return 204 even on bad input — we never want to break the page
		return c.NoContent(http.StatusNoContent)
	}
	if err := c.Validate(&req); err != nil {
		return c.NoContent(http.StatusNoContent)
	}

	// Enqueue is non-blocking (Redis RPUSH)
	_ = h.svc.Enqueue(
		c.Request().Context(),
		req,
		c.RealIP(),
		c.Request().UserAgent(),
		c.Request().Header.Get("CF-IPCountry"), // Cloudflare injects this
	)

	return c.NoContent(http.StatusNoContent)
}

// AdminOverview handles GET /v1/admin/analytics/overview
func (h *Handler) AdminOverview(c echo.Context) error {
	days := parseDays(c.QueryParam("days"))
	overview, err := h.svc.Overview(c.Request().Context(), days)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, overview)
}

// AdminPostStats handles GET /v1/admin/analytics/posts/:id
func (h *Handler) AdminPostStats(c echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return response.BadRequest(c, "MISSING_ID", "post id is required", nil)
	}

	days := parseDays(c.QueryParam("days"))

	stats, err := h.svc.PostStats(c.Request().Context(), postID, days)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, stats)
}

// AdminRealtime handles GET /v1/admin/analytics/realtime
// Returns queue depth and recent event count as a cheap real-time indicator.
func (h *Handler) AdminRealtime(c echo.Context) error {
	ctx := c.Request().Context()

	queueDepth, _ := h.svc.redis.Client.LLen(ctx, redisQueueKey).Result()
	recentCount, _ := h.svc.redis.Client.LLen(ctx, redisQueueKey).Result()

	return response.OK(c, map[string]interface{}{
		"queue_depth":   queueDepth,
		"recent_events": recentCount,
	})
}

// SessionStart handles POST /v1/analytics/session/start
func (h *Handler) SessionStart(c echo.Context) error {
	var req EventRequest
	req.Type = EventSessionStart

	if err := c.Bind(&req); err != nil {
		return c.NoContent(http.StatusNoContent)
	}
	req.Type = EventSessionStart // enforce type regardless of body

	_ = h.svc.Enqueue(
		c.Request().Context(), req,
		c.RealIP(), c.Request().UserAgent(),
		c.Request().Header.Get("CF-IPCountry"),
	)
	return c.NoContent(http.StatusNoContent)
}

// SessionEnd handles POST /v1/analytics/session/end
func (h *Handler) SessionEnd(c echo.Context) error {
	var req EventRequest
	if err := c.Bind(&req); err != nil {
		return c.NoContent(http.StatusNoContent)
	}
	req.Type = EventSessionEnd

	_ = h.svc.Enqueue(
		c.Request().Context(), req,
		c.RealIP(), c.Request().UserAgent(),
		c.Request().Header.Get("CF-IPCountry"),
	)
	return c.NoContent(http.StatusNoContent)
}
