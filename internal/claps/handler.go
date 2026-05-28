package claps

import (
	"github.com/labstack/echo/v4"

	mw "krishblog/internal/middleware"
	"krishblog/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Get handles GET /public/posts/:postId/claps?session_id=xxx
func (h *Handler) Get(c echo.Context) error {
	postID := c.Param("postId")
	sessionID := c.QueryParam("session_id")
	stats, err := h.svc.GetStats(c.Request().Context(), postID, sessionID)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, stats)
}

// Clap handles POST /public/posts/:postId/claps
func (h *Handler) Clap(c echo.Context) error {
	postID := c.Param("postId")
	var req ClapRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	result, err := h.svc.Clap(c.Request().Context(), postID, req.SessionID, req.Count)
	if err != nil {
		c.Logger().Error("failed to clap: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, result)
}
