package posts

import (
	"github.com/labstack/echo/v4"

	mw "krishblog/internal/middleware"
	"krishblog/pkg/pagination"
	"krishblog/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /public/posts
func (h *Handler) List(c echo.Context) error {
	p := pagination.Parse(c)
	posts, total, err := h.svc.ListPublished(
		c.Request().Context(),
		c.QueryParam("section"),
		c.QueryParam("tag"),
		c.QueryParam("q"),
		p,
	)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OKWithMeta(c, posts, pagination.NewMeta(p, total))
}

// GetBySlug handles GET /public/posts/:slug
func (h *Handler) GetBySlug(c echo.Context) error {
	post, err := h.svc.GetBySlug(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return response.NotFound(c, "post")
	}
	return response.OK(c, post)
}

// AdminList handles GET /admin/posts
func (h *Handler) AdminList(c echo.Context) error {
	p := pagination.Parse(c)
	posts, total, err := h.svc.AdminList(
		c.Request().Context(),
		c.QueryParam("status"),
		c.QueryParam("section"),
		p,
	)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OKWithMeta(c, posts, pagination.NewMeta(p, total))
}

// Create handles POST /admin/posts
func (h *Handler) Create(c echo.Context) error {
	claims := mw.GetClaims(c)

	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	post, err := h.svc.Create(c.Request().Context(), claims.UserID, req)
	if err != nil {
		c.Logger().Error("failed to create post: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.Created(c, post)
}

// Update handles PUT /admin/posts/:id
func (h *Handler) Update(c echo.Context) error {
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	post, err := h.svc.Update(c.Request().Context(), c.Param("id"), req)
	if err != nil {
		c.Logger().Error("failed to update post: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, post)
}

// UpdateStatus handles PATCH /admin/posts/:id/status
func (h *Handler) UpdateStatus(c echo.Context) error {
	var req StatusRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	post, err := h.svc.UpdateStatus(c.Request().Context(), c.Param("id"), req.Status)
	if err != nil {
		c.Logger().Error("failed to update post status: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, post)
}

// Delete handles DELETE /admin/posts/:id
func (h *Handler) Delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		c.Logger().Error("failed to delete post: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.NoContent(c)
}

func (h *Handler) AdminGet(c echo.Context) error {
	post, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return response.NotFound(c, "post")
	}
	return response.OK(c, post)
}
