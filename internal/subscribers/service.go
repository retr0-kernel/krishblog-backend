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

var (
	ErrNotFound         = errors.New("subscriber not found")
	ErrAlreadyConfirmed = errors.New("subscriber already confirmed")
)

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	SiteURL  string
	SiteName string
}

type Service struct {
	repo *Repository
	cfg  EmailConfig
}

func NewService(repo *Repository, cfg EmailConfig) *Service {
	return &Service{repo: repo, cfg: cfg}
}

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
	return s.sendConfirmation(sub)
}

func (s *Service) Confirm(ctx context.Context, token string) (*Subscriber, error) {
	return s.repo.Confirm(ctx, token)
}

func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	return s.repo.Unsubscribe(ctx, token)
}

func (s *Service) NotifyNewPost(ctx context.Context, postTitle, postSlug, postSummary string) error {
	subs, err := s.repo.ListConfirmed(ctx)
	if err != nil {
		return fmt.Errorf("list confirmed: %w", err)
	}
	postURL := fmt.Sprintf("%s/post/%s", s.cfg.SiteURL, postSlug)
	for _, sub := range subs {
		unsubURL := fmt.Sprintf("%s/unsubscribe?token=%s", s.cfg.SiteURL, sub.Token)
		if err := s.sendNewPostEmail(sub, postTitle, postURL, postSummary, unsubURL); err != nil {
			fmt.Printf("[subscribers] failed to email %s: %v\n", sub.Email, err)
		}
	}
	return nil
}

func (s *Service) Count(ctx context.Context) (total int, confirmed int, err error) {
	return s.repo.Count(ctx)
}

func (s *Service) sendConfirmation(sub *Subscriber) error {
	confirmURL := fmt.Sprintf("%s/confirm-subscription?token=%s", s.cfg.SiteURL, sub.Token)
	subject := fmt.Sprintf("Confirm your subscription to %s", s.cfg.SiteName)
	body := fmt.Sprintf(
		"Hello%s,\n\nThanks for subscribing to %s!\n\nConfirm here:\n\n%s\n\nIf you didn't subscribe, ignore this.\n\n— %s",
		nameGreeting(sub.Name), s.cfg.SiteName, confirmURL, s.cfg.SiteName,
	)
	return s.sendEmail(sub.Email, subject, body)
}

func (s *Service) sendNewPostEmail(sub Subscriber, title, postURL, summary, unsubURL string) error {
	subject := fmt.Sprintf("New post: %s", title)
	body := fmt.Sprintf(
		"Hello%s,\n\nNew post on %s:\n\n%s\n\n%s\n\nRead: %s\n\n---\nUnsubscribe: %s",
		nameGreeting(sub.Name), s.cfg.SiteName,
		title, summary, postURL, unsubURL,
	)
	return s.sendEmail(sub.Email, subject, body)
}

func (s *Service) sendEmail(to, subject, body string) error {
	if s.cfg.Host == "" {
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
