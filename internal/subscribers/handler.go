package subscribers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

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
	if err != nil {
		return response.InternalServerError(c, "subscribe failed")
	}
	return response.OK(c, map[string]string{"message": "Check your email to confirm your subscription."})
}

func (h *Handler) Confirm(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return response.BadRequest(c, "MISSING_TOKEN", "token is required", nil)
	}
	_, err := h.svc.Confirm(c.Request().Context(), token)
	if errors.Is(err, ErrNotFound) {
		return response.BadRequest(c, "INVALID_TOKEN", "token not found or already confirmed", nil)
	}
	if err != nil {
		return response.InternalServerError(c, "confirm failed")
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
		return response.InternalServerError(c, "unsubscribe failed")
	}
	return response.OK(c, map[string]string{"message": "You've been unsubscribed."})
}

func (h *Handler) AdminStats(c echo.Context) error {
	total, confirmed, err := h.svc.Count(c.Request().Context())
	if err != nil {
		return response.InternalServerError(c, "stats failed")
	}
	return c.JSON(http.StatusOK, map[string]int{"total": total, "confirmed": confirmed})
}

func (h *Handler) AdminNotify(c echo.Context) error {
	var req struct {
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
	if err := h.svc.NotifyNewPost(c.Request().Context(), req.PostTitle, req.PostSlug, req.PostSummary); err != nil {
		return response.InternalServerError(c, "notify failed")
	}
	return response.OK(c, map[string]string{"message": "Notifications sent."})
}
