package subscribers

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"  SwatiPrashil@gmail.com  ", "swatiprashil@gmail.com", false},
		{"you@example.com", "you@example.com", false},
		{"prashil@gmail.comitaws", "prashil@gmail.comitaws", false}, // syntactically valid; Resend will reject delivery
		{"not-an-email", "", true},
		{"", "", true},
	}

	for _, tc := range tests {
		got, err := normalizeEmail(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeEmail(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeEmail(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("normalizeEmail(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
