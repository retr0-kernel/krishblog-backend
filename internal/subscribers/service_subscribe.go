package subscribers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

var ErrResent = errors.New("confirmation email resent")

// Subscribe creates or refreshes a pending subscriber and sends a confirmation email.
func (s *Service) Subscribe(ctx context.Context, email, name string) error {
	existing, existingErr := s.repo.GetByEmail(ctx, email)
	if existingErr == nil && existing.Confirmed {
		return ErrAlreadyConfirmed
	}

	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	wasPending := existingErr == nil && !existing.Confirmed

	sub, err := s.repo.Create(ctx, email, name, token)
	if err != nil {
		return err
	}

	if sub.Confirmed {
		return ErrAlreadyConfirmed
	}

	go func() {
		if err := s.sendConfirmation(sub); err != nil {
			s.log.Error("failed to send confirmation email",
				slog.String("email", sub.Email),
				slog.String("error", err.Error()),
			)
		}
	}()

	if wasPending {
		return ErrResent
	}
	return nil
}
