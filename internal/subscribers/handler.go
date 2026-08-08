package subscribers

import (
	"errors"

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

type subscribeRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *Handler) Subscribe(c echo.Context) error {
	var req subscribeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "invalid request body", nil)
	}
	if req.Email == "" {
		return response.BadRequest(c, "MISSING_EMAIL", "email is required", nil)
	}
	err := h.svc.Subscribe(c.Request().Context(), req.Email, req.Name)
	if errors.Is(err, ErrAlreadyConfirmed) {
		return response.OK(c, map[string]string{"message": "You're already subscribed!"})
	}
	if errors.Is(err, ErrResent) {
		return response.OK(c, map[string]string{"message": "Confirmation email resent. Check your inbox."})
	}
	if err != nil {
		c.Logger().Error("subscribe failed: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, map[string]string{"message": "Check your email to confirm your subscription."})
}

func (h *Handler) Confirm(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return response.BadRequest(c, "MISSING_TOKEN", "token is required", nil)
	}
	_, err := h.svc.Confirm(c.Request().Context(), token)
	if errors.Is(err, ErrAlreadyConfirmed) {
		return response.OK(c, map[string]string{"message": "You're already subscribed!"})
	}
	if errors.Is(err, ErrTokenExpired) {
		return response.BadRequest(c, "TOKEN_EXPIRED", "this confirmation link has expired — please subscribe again", nil)
	}
	if errors.Is(err, ErrNotFound) {
		return response.BadRequest(c, "INVALID_TOKEN", "token not found or already confirmed", nil)
	}
	if err != nil {
		c.Logger().Error("confirm failed: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, map[string]string{"message": "Subscription confirmed!"})
}

func (h *Handler) Unsubscribe(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return response.BadRequest(c, "MISSING_TOKEN", "token is required", nil)
	}
	err := h.svc.Unsubscribe(c.Request().Context(), token)
	if errors.Is(err, ErrNotFound) {
		return response.BadRequest(c, "INVALID_TOKEN", "token not found", nil)
	}
	if err != nil {
		c.Logger().Error("unsubscribe failed: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, map[string]string{"message": "You've been unsubscribed."})
}

func (h *Handler) AdminStats(c echo.Context) error {
	stats, err := h.svc.AdminStats(c.Request().Context())
	if err != nil {
		c.Logger().Error("subscriber stats failed: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, stats)
}

func (h *Handler) AdminNotify(c echo.Context) error {
	var req struct {
		PostID      string `json:"post_id"`
		PostTitle   string `json:"post_title"`
		PostSlug    string `json:"post_slug"`
		PostSummary string `json:"post_summary"`
	}
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "invalid request body", nil)
	}
	if req.PostTitle == "" || req.PostSlug == "" {
		return response.BadRequest(c, "MISSING_FIELDS", "post_title and post_slug are required", nil)
	}
	result, err := h.svc.NotifyNewPost(c.Request().Context(), req.PostID, req.PostTitle, req.PostSlug, req.PostSummary)
	if err != nil {
		c.Logger().Error("failed to notify subscribers: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, map[string]interface{}{
		"message":          "Notifications sent.",
		"total_confirmed":  result.TotalConfirmed,
		"sent_count":       result.SentCount,
		"failed_count":     result.FailedCount,
	})
}
