package subscribers

import "testing"

func TestNormalizeEmailFrom(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Krish Blog krish22092003@gmail.com", "Krish Blog <krish22092003@gmail.com>"},
		{"Krish Blog <onboarding@resend.dev>", "Krish Blog <onboarding@resend.dev>"},
		{"onboarding@resend.dev", "onboarding@resend.dev"},
	}
	for _, tc := range tests {
		if got := NormalizeEmailFrom(tc.in); got != tc.want {
			t.Fatalf("NormalizeEmailFrom(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnvelopeFrom(t *testing.T) {
	if got := envelopeFrom("Krish Blog <onboarding@resend.dev>"); got != "onboarding@resend.dev" {
		t.Fatalf("envelopeFrom = %q", got)
	}
}
