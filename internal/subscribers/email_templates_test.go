package subscribers

import (
	"strings"
	"testing"
)

func TestConfirmationBodies(t *testing.T) {
	text, htmlOut := confirmationBodies("Krish Blog", "https://example.com/confirm?token=abc", "Krish")
	if !strings.Contains(text, "Krish Blog") || !strings.Contains(text, "https://example.com/confirm?token=abc") {
		t.Fatalf("unexpected text body: %q", text)
	}
	if !strings.Contains(htmlOut, "Confirm subscription") || !strings.Contains(htmlOut, "https://example.com/confirm?token=abc") {
		t.Fatalf("unexpected html body")
	}
}

func TestNewPostBodies(t *testing.T) {
	_, htmlOut := newPostBodies("Krish Blog", "My Post", "Summary here", "https://example.com/post/x", "https://example.com/unsub", "")
	if !strings.Contains(htmlOut, "My Post") || !strings.Contains(htmlOut, "Read article") {
		t.Fatalf("unexpected new post html")
	}
}
