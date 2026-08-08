package subscribers

import (
	"fmt"
	"html"
	"strings"
)

func confirmationBodies(siteName, confirmURL, name string) (text, htmlOut string) {
	greeting := "Hello"
	if name != "" {
		greeting = "Hello " + name
	}

	text = fmt.Sprintf(`%s,

Thanks for subscribing to %s.

Confirm your email address:
%s

If you didn't subscribe, you can ignore this email.

— %s`, greeting, siteName, confirmURL, siteName)

	htmlOut = emailLayout(siteName, emailContent{
		Title:    "Confirm your subscription",
		Greeting: greeting,
		Body:     fmt.Sprintf("Thanks for subscribing to <strong>%s</strong>. Tap the button below to confirm your email address.", html.EscapeString(siteName)),
		CTALabel: "Confirm subscription",
		CTAURL:   confirmURL,
		Footer:   "If you didn't subscribe, you can safely ignore this email. This link expires in 7 days.",
	})
	return text, htmlOut
}

func newPostBodies(siteName, title, summary, postURL, unsubURL, name string) (text, htmlOut string) {
	greeting := "Hello"
	if name != "" {
		greeting = "Hello " + name
	}

	text = fmt.Sprintf(`%s,

A new post is live on %s:

%s

%s

Read it: %s

—
You're receiving this because you subscribed to %s.
Unsubscribe: %s`, greeting, siteName, title, summary, postURL, siteName, unsubURL)

	summaryBlock := ""
	if strings.TrimSpace(summary) != "" {
		summaryBlock = fmt.Sprintf(`<p style="margin:0 0 24px;font-size:16px;line-height:1.6;color:#4b5563;">%s</p>`, html.EscapeString(summary))
	}

	htmlOut = emailLayout(siteName, emailContent{
		Title:    "New post",
		Greeting: greeting,
		Body: fmt.Sprintf(
			`A new post is live on <strong>%s</strong>.<br><br><span style="font-size:20px;font-weight:600;color:#111827;">%s</span>`,
			html.EscapeString(siteName),
			html.EscapeString(title),
		) + summaryBlock,
		CTALabel: "Read article",
		CTAURL:   postURL,
		Footer:   fmt.Sprintf(`You're receiving this because you subscribed to %s. <a href="%s" style="color:#6b7280;">Unsubscribe</a>`, html.EscapeString(siteName), html.EscapeString(unsubURL)),
	})
	return text, htmlOut
}

type emailContent struct {
	Title    string
	Greeting string
	Body     string
	CTALabel string
	CTAURL   string
	Footer   string
}

func emailLayout(siteName string, c emailContent) string {
	siteName = html.EscapeString(siteName)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:Georgia,'Times New Roman',serif;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f4f4f5;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border:1px solid #e5e7eb;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:28px 32px 12px;text-align:center;border-bottom:1px solid #f3f4f6;">
              <div style="font-size:13px;letter-spacing:0.12em;text-transform:uppercase;color:#6b7280;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">%s</div>
              <h1 style="margin:12px 0 0;font-size:28px;line-height:1.25;color:#111827;font-weight:700;">%s</h1>
            </td>
          </tr>
          <tr>
            <td style="padding:28px 32px 8px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
              <p style="margin:0 0 16px;font-size:16px;line-height:1.6;color:#111827;">%s,</p>
              <div style="margin:0 0 28px;font-size:16px;line-height:1.7;color:#374151;">%s</div>
              <table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 auto 28px;">
                <tr>
                  <td style="border-radius:8px;background:#111827;">
                    <a href="%s" style="display:inline-block;padding:14px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">%s</a>
                  </td>
                </tr>
              </table>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#9ca3af;word-break:break-all;">
                Or copy this link:<br>
                <a href="%s" style="color:#6b7280;">%s</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:16px 32px 28px;border-top:1px solid #f3f4f6;font-size:12px;line-height:1.6;color:#9ca3af;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
              %s
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		c.Title,
		siteName,
		html.EscapeString(c.Title),
		html.EscapeString(c.Greeting),
		c.Body,
		html.EscapeString(c.CTAURL),
		html.EscapeString(c.CTALabel),
		html.EscapeString(c.CTAURL),
		html.EscapeString(c.CTAURL),
		c.Footer,
	)
}
