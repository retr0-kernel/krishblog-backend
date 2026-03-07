package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	R2        R2Config
	CORS      CORSConfig
	RateLimit RateLimitConfig
	Admin     AdminConfig
	Email     EmailConfig
	Site      SiteConfig
	Google    GoogleConfig
}

type AppConfig struct {
	Env    string
	Port   string
	Secret string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	Secret             string
	ExpiryHours        time.Duration
	RefreshExpiryHours time.Duration
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	RPS   float64
	Burst int
}

type AdminConfig struct {
	Email    string
	Password string
}

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type SiteConfig struct {
	URL  string
	Name string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// IsAdminAllowed returns true if the email is allowed admin access.
// If no allowlist is configured, only the ADMIN_EMAIL is allowed.
func (c *Config) IsAdminAllowed(email string) bool {
	if c.Admin.Email == "" {
		return true // no restriction configured
	}
	return strings.EqualFold(email, c.Admin.Email)
}

func Load() (*Config, error) {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}

	cfg := &Config{}

	cfg.App.Env = getRequired("APP_ENV")
	cfg.App.Port = getDefault("APP_PORT", "8080")
	cfg.App.Secret = getDefault("APP_SECRET", "dev-secret-change-me")

	cfg.Database.URL = getRequired("DATABASE_URL")
	cfg.Redis.URL = getRequired("REDIS_URL")

	cfg.JWT.Secret = getRequired("JWT_SECRET")

	jwtH, err := strconv.Atoi(getDefault("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}
	cfg.JWT.ExpiryHours = time.Duration(jwtH) * time.Hour

	refreshH, err := strconv.Atoi(getDefault("JWT_REFRESH_EXPIRY_HOURS", "168"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY_HOURS: %w", err)
	}
	cfg.JWT.RefreshExpiryHours = time.Duration(refreshH) * time.Hour

	// R2 — optional, server boots without it
	cfg.R2.AccountID = getDefault("R2_ACCOUNT_ID", "")
	cfg.R2.AccessKeyID = getDefault("R2_ACCESS_KEY_ID", "")
	cfg.R2.SecretAccessKey = getDefault("R2_SECRET_ACCESS_KEY", "")
	cfg.R2.BucketName = getDefault("R2_BUCKET_NAME", "")
	cfg.R2.PublicURL = getDefault("R2_PUBLIC_URL", "")

	originsRaw := getDefault("ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.CORS.AllowedOrigins = strings.Split(originsRaw, ",")

	rps, err := strconv.ParseFloat(getDefault("RATE_LIMIT_RPS", "20"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_RPS: %w", err)
	}
	cfg.RateLimit.RPS = rps

	burst, err := strconv.Atoi(getDefault("RATE_LIMIT_BURST", "50"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_BURST: %w", err)
	}
	cfg.RateLimit.Burst = burst

	cfg.Admin.Email = getDefault("ADMIN_EMAIL", "")
	cfg.Admin.Password = getDefault("ADMIN_PASSWORD", "")

	cfg.Email.Host = getDefault("SMTP_HOST", "")
	cfg.Email.Port = getDefault("SMTP_PORT", "587")
	cfg.Email.Username = getDefault("SMTP_USERNAME", "")
	cfg.Email.Password = getDefault("SMTP_PASSWORD", "")
	cfg.Email.From = getDefault("SMTP_FROM", "noreply@example.com")

	cfg.Site.URL = getDefault("SITE_URL", "http://localhost:3000")
	cfg.Site.Name = getDefault("SITE_NAME", "Krish Blog")

	// Google OAuth — optional, only needed if using Google login
	cfg.Google.ClientID = getDefault("GOOGLE_CLIENT_ID", "")
	cfg.Google.ClientSecret = getDefault("GOOGLE_CLIENT_SECRET", "")
	cfg.Google.RedirectURL = getDefault("GOOGLE_REDIRECT_URL", "")

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env != "production"
}

func getRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("[config] required environment variable %q is not set", key))
	}
	return v
}

func getDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
