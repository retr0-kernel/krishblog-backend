package subscribers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

var ErrNotFound = errors.New("subscriber not found")
var ErrAlreadyConfirmed = errors.New("subscriber already confirmed")

// EmailConfig holds SMTP settings.
type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	SiteURL  string
	SiteName string
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

// NotifyNewPost sends a new-post email to all confirmed subscribers.
func (s *Service) NotifyNewPost(ctx context.Context, postTitle, postSlug, postSummary string) error {
	subs, err := s.repo.ListConfirmed(ctx)
	if err != nil {
		return fmt.Errorf("list confirmed: %w", err)
	}

	postURL := fmt.Sprintf("%s/post/%s", s.cfg.SiteURL, postSlug)

	for _, sub := range subs {
		unsubURL := fmt.Sprintf("%s/unsubscribe?token=%s", s.cfg.SiteURL, sub.Token)
		if err := s.sendNewPostEmail(sub, postTitle, postURL, postSummary, unsubURL); err != nil {
			// Log but continue — don't fail the whole batch for one bad address
			s.log.Error("failed to send new post notification",
				slog.String("email", sub.Email),
				slog.String("post", postTitle),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

// Count returns subscriber counts.
func (s *Service) Count(ctx context.Context) (total int, confirmed int, err error) {
	return s.repo.Count(ctx)
}

// ─── email helpers ────────────────────────────────────────────────────────────

func (s *Service) sendConfirmation(sub *Subscriber) error {
	confirmURL := fmt.Sprintf("%s/confirm-subscription?token=%s", s.cfg.SiteURL, sub.Token)
	subject := fmt.Sprintf("Confirm your subscription to %s", s.cfg.SiteName)
	body := fmt.Sprintf(`Hello%s,

Thanks for subscribing to %s!

Please confirm your email address by clicking the link below:

%s

If you didn't subscribe, you can safely ignore this email.

— %s`, nameGreeting(sub.Name), s.cfg.SiteName, confirmURL, s.cfg.SiteName)

	return s.sendEmail(sub.Email, subject, body)
}

func (s *Service) sendNewPostEmail(sub Subscriber, title, postURL, summary, unsubURL string) error {
	subject := fmt.Sprintf("New post: %s", title)
	body := fmt.Sprintf(`Hello%s,

A new post has been published on %s:

%s

%s

Read it here: %s

─────────────────────────────
You're receiving this because you subscribed to %s.
To unsubscribe: %s`,
		nameGreeting(sub.Name),
		s.cfg.SiteName,
		title,
		summary,
		postURL,
		s.cfg.SiteName,
		unsubURL,
	)
	return s.sendEmail(sub.Email, subject, body)
}

func (s *Service) sendEmail(to, subject, body string) error {
	if s.cfg.Host == "" {
		// No SMTP configured — just log (dev mode)
		s.log.Info("email (dev mode - not sent)",
			slog.String("to", to),
			slog.String("subject", subject),
		)
		return nil
	}
	fromHeader := NormalizeEmailFrom(s.cfg.From)
	envelope := envelopeFrom(fromHeader)
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	addr := s.cfg.Host + ":" + s.cfg.Port
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, envelope, []string{to}, []byte(msg)); err != nil {
		s.log.Error("smtp send failed",
			slog.String("to", to),
			slog.String("envelope", envelope),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func nameGreeting(name string) string {
	if name == "" {
		return ""
	}
	return " " + name
}
