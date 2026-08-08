package media

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"krishblog/internal/database"
	mw "krishblog/internal/middleware"
	"krishblog/pkg/pagination"
	"krishblog/pkg/response"
)

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type UpdateRequest struct {
	AltText string `json:"alt_text" validate:"omitempty,max=300"`
	Caption string `json:"caption"  validate:"omitempty,max=500"`
}

type MediaResponse struct {
	ID           string                 `json:"id"`
	UploaderID   string                 `json:"uploader_id"`
	Filename     string                 `json:"filename"`
	OriginalName string                 `json:"original_name"`
	MimeType     string                 `json:"mime_type"`
	MediaType    string                 `json:"media_type"`
	SizeBytes    int64                  `json:"size_bytes"`
	Width        *int                   `json:"width,omitempty"`
	Height       *int                   `json:"height,omitempty"`
	PublicURL    string                 `json:"public_url"`
	ThumbnailURL string                 `json:"thumbnail_url,omitempty"`
	AltText      string                 `json:"alt_text,omitempty"`
	Caption      string                 `json:"caption,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ─── Service ──────────────────────────────────────────────────────────────────

type Service struct {
	db    *database.Postgres
	redis *database.Redis
}

func NewService(db *database.Postgres, redis *database.Redis) *Service {
	return &Service{db: db, redis: redis}
}

func (s *Service) List(_ context.Context, _ string, _ pagination.Params) ([]MediaResponse, int64, error) {
	return []MediaResponse{}, 0, nil
}

func (s *Service) Upload(_ context.Context, _, _, _ string, _ int64, _ []byte) (*MediaResponse, error) {
	return nil, errors.New("images are hosted in the krishblog-images GitHub repo — use a raw.githubusercontent.com URL")
}

func (s *Service) Update(_ context.Context, _ string, _ UpdateRequest) (*MediaResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Service) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

// ─── Handler ──────────────────────────────────────────────────────────────────

const maxUploadSize = 50 << 20 // 50 MB

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c echo.Context) error {
	p := pagination.Parse(c)
	assets, total, err := h.svc.List(c.Request().Context(), c.QueryParam("type"), p)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OKWithMeta(c, assets, pagination.NewMeta(p, total))
}

func (h *Handler) Upload(c echo.Context) error {
	claims := mw.GetClaims(c)
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxUploadSize)

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return response.BadRequest(c, "MISSING_FILE", "file field is required", nil)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return response.BadRequest(c, "READ_ERROR", "could not read uploaded file", nil)
	}

	asset, err := h.svc.Upload(
		c.Request().Context(),
		claims.UserID,
		header.Filename,
		header.Header.Get("Content-Type"),
		header.Size,
		data,
	)
	if err != nil {
		return response.BadRequest(c, "USE_GITHUB_IMAGES", "Add images to github.com/retr0-kernel/krishblog-images and paste the raw URL in your post", nil)
	}
	return response.Created(c, asset)
}

func (h *Handler) Update(c echo.Context) error {
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "malformed request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	asset, err := h.svc.Update(c.Request().Context(), c.Param("id"), req)
	if err != nil {
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.OK(c, asset)
}

func (h *Handler) Delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		c.Logger().Error("failed to delete media: ", err)
		return response.InternalServerError(c, mw.GetRequestID(c))
	}
	return response.NoContent(c)
}
