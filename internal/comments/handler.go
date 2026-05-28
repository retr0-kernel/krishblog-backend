package comments

import (
	"net/http"

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

// ListPublic handles GET /public/posts/:postId/comments
func (h *Handler) ListPublic(c echo.Context) error {
	postID := c.Param("postId")
	comments, err := h.svc.ListPublic(c.Request().Context(), postID)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, comments)
}

// Create handles POST /public/posts/:postId/comments
func (h *Handler) Create(c echo.Context) error {
	postID := c.Param("postId")
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	comment, err := h.svc.Create(c.Request().Context(), postID, req)
	if err != nil {
		c.Logger().Error("failed to create comment: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    comment,
		"message": "Comment submitted and pending approval",
	})
}

// AdminList handles GET /admin/comments
func (h *Handler) AdminList(c echo.Context) error {
	var approved *bool
	if q := c.QueryParam("approved"); q == "true" {
		v := true
		approved = &v
	} else if q == "false" {
		v := false
		approved = &v
	}
	comments, err := h.svc.ListAll(c.Request().Context(), approved)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, comments)
}

// AdminApprove handles PATCH /admin/comments/:id/approve
func (h *Handler) AdminApprove(c echo.Context) error {
	var req ApproveRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	comment, err := h.svc.SetApproval(c.Request().Context(), c.Param("id"), req.IsApproved)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, comment)
}

// AdminReply handles POST /admin/comments/:id/reply
func (h *Handler) AdminReply(c echo.Context) error {
	var req ReplyRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	parentID := c.Param("id")
	comments, err := h.svc.ListAll(c.Request().Context(), nil)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	var postID string
	for _, cm := range comments {
		if cm.ID == parentID {
			postID = cm.PostID
			break
		}
	}
	if postID == "" {
		return response.NotFound(c, "comment")
	}
	reply, err := h.svc.Reply(c.Request().Context(), postID, parentID, req.Content)
	if err != nil {
		c.Logger().Error("failed to create reply: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.Created(c, reply)
}

// AdminDelete handles DELETE /admin/comments/:id
func (h *Handler) AdminDelete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.NoContent(c)
}
