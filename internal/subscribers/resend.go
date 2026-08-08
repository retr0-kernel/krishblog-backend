package subscribers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var resendAPIURL = "https://api.resend.com/emails"

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text,omitempty"`
	HTML    string   `json:"html,omitempty"`
}

type resendErrorResponse struct {
	Message string `json:"message"`
}

func (s *Service) sendViaResend(to, subject, textBody, htmlBody string) error {
	payload := resendEmailRequest{
		From:    s.cfg.From,
		To:      []string{to},
		Subject: subject,
		Text:    textBody,
		HTML:    htmlBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr resendErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("resend api %d: %s", resp.StatusCode, apiErr.Message)
		}
		return fmt.Errorf("resend api %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
