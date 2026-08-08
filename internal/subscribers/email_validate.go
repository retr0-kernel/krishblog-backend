package subscribers

import (
	"fmt"
	"net/mail"
	"strings"
)

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", fmt.Errorf("email is required")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("invalid email address")
	}

	normalized := strings.ToLower(addr.Address)
	if !strings.Contains(normalized, "@") || !strings.Contains(normalized, ".") {
		return "", fmt.Errorf("invalid email address")
	}

	return normalized, nil
}
