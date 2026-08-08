package subscribers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net/smtp"
	"strings"
)

var ErrNotFound = errors.New("subscriber not found")
var ErrAlreadyConfirmed = errors.New("subscriber already confirmed")
var ErrTokenExpired = errors.New("confirmation link expired")

// EmailConfig holds email delivery settings.
// Prefer ResendAPIKey on Railway and other hosts that block outbound SMTP.
type EmailConfig struct {
	ResendAPIKey string
	Host         string
	Port         string
	Username     string
	Password     string
	From         string
	SiteURL      string
	SiteName     string
}

// Service handles subscriber business logic and email dispatch.
type Service struct {
	repo *Repository
	cfg  EmailConfig
	log  *slog.Logger
}

func NewService(repo *Repository, cfg EmailConfig, log *slog.Logger) *Service {
	return &Service{repo: repo, cfg: cfg, log: log}
}

// Confirm marks subscriber as confirmed.
func (s *Service) Confirm(ctx context.Context, token string) (*Subscriber, error) {
	return s.repo.Confirm(ctx, token)
}

// Unsubscribe removes a subscriber.
func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	return s.repo.Unsubscribe(ctx, token)
}

// NotifyResult summarizes a new-post email batch.
type NotifyResult struct {
	TotalConfirmed int `json:"total_confirmed"`
	SentCount      int `json:"sent_count"`
	FailedCount    int `json:"failed_count"`
}

// AdminStatsResponse is returned by GET /admin/subscribers/stats.
type AdminStatsResponse struct {
	Total               int                `json:"total"`
	Confirmed           int                `json:"confirmed"`
	Pending             int                `json:"pending"`
	RecentNotifications []PostNotification `json:"recent_notifications"`
}

// NotifyNewPost sends a new-post email to all confirmed subscribers.
func (s *Service) NotifyNewPost(ctx context.Context, postID, postTitle, postSlug, postSummary string) (*NotifyResult, error) {
	subs, err := s.repo.ListConfirmed(ctx)
	if err != nil {
		return nil, fmt.Errorf("list confirmed: %w", err)
	}

	result := &NotifyResult{TotalConfirmed: len(subs)}
	postURL := fmt.Sprintf("%s/post/%s", s.cfg.SiteURL, postSlug)

	for _, sub := range subs {
		unsubURL := fmt.Sprintf("%s/unsubscribe?token=%s", s.cfg.SiteURL, sub.Token)
		if err := s.sendNewPostEmail(sub, postTitle, postURL, postSummary, unsubURL); err != nil {
			result.FailedCount++
			s.log.Error("failed to send new post notification",
				slog.String("email", sub.Email),
				slog.String("post", postTitle),
				slog.String("error", err.Error()),
			)
			continue
		}
		result.SentCount++
	}

	if postID != "" {
		if err := s.repo.RecordNotification(ctx, postID, postSlug, postTitle, result.TotalConfirmed, result.SentCount, result.FailedCount); err != nil {
			s.log.Error("failed to record notification stats", slog.String("error", err.Error()))
		}
	}

	return result, nil
}

// Count returns subscriber counts.
func (s *Service) Count(ctx context.Context) (total int, confirmed int, pending int, err error) {
	return s.repo.Count(ctx)
}

// AdminStats returns subscriber totals and recent notification batches.
func (s *Service) AdminStats(ctx context.Context) (*AdminStatsResponse, error) {
	total, confirmed, pending, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	notifications, err := s.repo.ListRecentNotifications(ctx, 20)
	if err != nil {
		return nil, err
	}
	return &AdminStatsResponse{
		Total:               total,
		Confirmed:           confirmed,
		Pending:             pending,
		RecentNotifications: notifications,
	}, nil
}

// ─── email helpers ────────────────────────────────────────────────────────────

func (s *Service) sendConfirmation(sub *Subscriber) error {
	confirmURL := fmt.Sprintf("%s/confirm-subscription?token=%s", s.cfg.SiteURL, sub.Token)
	subject := fmt.Sprintf("Confirm your subscription to %s", s.cfg.SiteName)
	text, htmlBody := confirmationBodies(s.cfg.SiteName, confirmURL, sub.Name)
	return s.sendEmail(sub.Email, subject, text, htmlBody)
}

func (s *Service) sendNewPostEmail(sub Subscriber, title, postURL, summary, unsubURL string) error {
	subject := fmt.Sprintf("New post: %s", title)
	text, htmlBody := newPostBodies(s.cfg.SiteName, title, summary, postURL, unsubURL, sub.Name)
	return s.sendEmail(sub.Email, subject, text, htmlBody)
}

func (s *Service) sendEmail(to, subject, textBody, htmlBody string) error {
	if s.cfg.ResendAPIKey != "" {
		if err := s.sendViaResend(to, subject, textBody, htmlBody); err != nil {
			s.log.Error("resend send failed",
				slog.String("to", to),
				slog.String("from", s.cfg.From),
				slog.String("error", err.Error()),
			)
			return err
		}
		return nil
	}

	if s.cfg.Host == "" {
		// No email transport configured — just log (dev mode)
		s.log.Info("email (dev mode - not sent)",
			slog.String("to", to),
			slog.String("subject", subject),
		)
		return nil
	}
	fromHeader := NormalizeEmailFrom(s.cfg.From)
	envelope := envelopeFrom(fromHeader)

	var msg strings.Builder
	boundary := "krishblog-" + mustToken(8)
	msg.WriteString("From: " + fromHeader + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
	msg.WriteString("\r\n")

	writePart := func(contentType, body string) {
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		msg.WriteString("\r\n")
		var qp strings.Builder
		w := quotedprintable.NewWriter(&qp)
		_, _ = w.Write([]byte(body))
		_ = w.Close()
		msg.WriteString(qp.String())
		msg.WriteString("\r\n")
	}

	writePart("text/plain", textBody)
	if htmlBody != "" {
		writePart("text/html", htmlBody)
	}
	msg.WriteString("--" + boundary + "--\r\n")

	addr := s.cfg.Host + ":" + s.cfg.Port
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, envelope, []string{to}, []byte(msg.String())); err != nil {
		s.log.Error("smtp send failed",
			slog.String("to", to),
			slog.String("envelope", envelope),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}

func mustToken(n int) string {
	t, err := generateToken()
	if err != nil {
		return "fallback"
	}
	if len(t) > n*2 {
		return t[:n*2]
	}
	return t
}

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
