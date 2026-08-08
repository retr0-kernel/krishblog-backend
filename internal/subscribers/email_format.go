package subscribers

import (
	"fmt"
	"strings"
)

// NormalizeEmailFrom ensures RFC 5322 From header format.
func NormalizeEmailFrom(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Krish Blog <onboarding@resend.dev>"
	}
	if strings.Contains(raw, "<") && strings.Contains(raw, ">") {
		return raw
	}
	parts := strings.Fields(raw)
	if len(parts) >= 2 {
		email := parts[len(parts)-1]
		if strings.Contains(email, "@") {
			name := strings.Join(parts[:len(parts)-1], " ")
			return fmt.Sprintf("%s <%s>", name, email)
		}
	}
	if strings.Contains(raw, "@") {
		return raw
	}
	return raw
}

// envelopeFrom extracts the bare email for SMTP MAIL FROM (required by net/smtp).
func envelopeFrom(formatted string) string {
	formatted = strings.TrimSpace(formatted)
	if i := strings.LastIndex(formatted, "<"); i >= 0 {
		if j := strings.LastIndex(formatted, ">"); j > i {
			return strings.TrimSpace(formatted[i+1 : j])
		}
	}
	return formatted
}
