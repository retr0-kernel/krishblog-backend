package subscribers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendViaResend(t *testing.T) {
	var gotAuth string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	defer srv.Close()

	orig := resendAPIURL
	resendAPIURL = srv.URL
	t.Cleanup(func() { resendAPIURL = orig })

	svc := &Service{
		cfg: EmailConfig{
			ResendAPIKey: "re_test_key",
			From:         "Krish Blog <onboarding@resend.dev>",
		},
	}

	if err := svc.sendViaResend("you@example.com", "Hello", "plain", "<p>html</p>"); err != nil {
		t.Fatalf("sendViaResend: %v", err)
	}

	if gotAuth != "Bearer re_test_key" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"to":["you@example.com"]`) {
		t.Fatalf("body missing to: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"html":"`) || !strings.Contains(gotBody, "html") {
		t.Fatalf("body missing html: %s", gotBody)
	}
}

func TestSendEmailPrefersResendAPI(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	defer srv.Close()

	orig := resendAPIURL
	resendAPIURL = srv.URL
	t.Cleanup(func() { resendAPIURL = orig })

	svc := &Service{
		cfg: EmailConfig{
			ResendAPIKey: "re_test_key",
			From:         "onboarding@resend.dev",
			Host:         "smtp.resend.com",
			Port:         "587",
		},
	}

	if err := svc.sendEmail("you@example.com", "Subject", "text", ""); err != nil {
		t.Fatalf("sendEmail: %v", err)
	}
	if !called {
		t.Fatal("expected resend api to be called instead of smtp")
	}
}
