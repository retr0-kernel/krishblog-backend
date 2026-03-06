package analytics

import "strings"

var botSignatures = []string{
	"bot", "crawl", "spider", "slurp", "fetch",
	"scan", "check", "monitor", "ping", "curl", "wget",
	"python", "java/", "go-http", "okhttp", "axios",
	"lighthouse", "pagespeed", "gtmetrix", "pingdom",
	"uptimerobot", "facebookexternalhit", "twitterbot",
	"linkedinbot", "whatsapp", "telegrambot", "discordbot",
	"applebot", "bingpreview", "googlebot", "yandexbot",
	"baiduspider", "duckduckbot", "sogou", "exabot",
	"semrushbot", "ahrefsbot", "mj12bot", "dotbot",
	"rogerbot", "screaming frog", "headlesschrome",
	"phantomjs", "selenium", "webdriver",
}

// IsBot returns true if the User-Agent looks like a bot, crawler, or automation.
func IsBot(ua string) bool {
	if len(ua) < 10 {
		return true
	}
	lower := strings.ToLower(ua)
	for _, sig := range botSignatures {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// IsValidSession returns true if the session ID meets minimum requirements.
func IsValidSession(sessionID string) bool {
	if len(sessionID) < 16 || len(sessionID) > 128 {
		return false
	}
	alphanum := 0
	for _, r := range sessionID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			alphanum++
		}
	}
	return alphanum >= 8
}
