package subscribers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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
}

func NewService(repo *Repository, cfg EmailConfig) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// Subscribe creates a subscriber and sends a confirmation email.
// The email is sent asynchronously so that SMTP issues never block or fail the HTTP response.
func (s *Service) Subscribe(ctx context.Context, email, name string) error {
	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	sub, err := s.repo.Create(ctx, email, name, token)
	if err != nil {
		return err
	}

	if sub.Confirmed {
		return ErrAlreadyConfirmed
	}

	// Fire-and-forget: SMTP errors are logged but never returned to the caller.
	go func() {
		if err := s.sendConfirmation(sub); err != nil {
			fmt.Printf("[subscribers] confirmation email to %s failed: %v\n", sub.Email, err)
		}
	}()

	return nil
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
			fmt.Printf("[subscribers] failed to email %s: %v\n", sub.Email, err)
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
		fmt.Printf("[email] To: %s | Subject: %s\n%s\n---\n", to, subject, body)
		return nil
	}
	msg := strings.Join([]string{
		"From: " + s.cfg.From,
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
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg))
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
