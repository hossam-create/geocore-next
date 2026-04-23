package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

// TemplateEngine renders HTML email templates from the embedded filesystem.
// Thread-safe — the underlying *template.Template is parsed once at startup.
type TemplateEngine struct {
	tmpl *template.Template
}

// NewTemplateEngine parses all templates from the embedded FS.
// Panics if any template contains a syntax error (caught at startup, not runtime).
func NewTemplateEngine() *TemplateEngine {
	funcMap := template.FuncMap{
		"year":     func() int { return time.Now().Year() },
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}
	tmpl := template.Must(
		template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html"),
	)
	return &TemplateEngine{tmpl: tmpl}
}

// Render executes the named template and returns (html, text, error).
// name may be "otp", "otp.html", or "templates/otp.html" — all normalised.
// The text version is derived by stripping HTML tags from the rendered output.
func (e *TemplateEngine) Render(name string, data map[string]any) (string, string, error) {
	name = normaliseName(name)
	if data == nil {
		data = map[string]any{}
	}

	var buf bytes.Buffer
	if err := e.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", "", fmt.Errorf("email template %q: %w", name, err)
	}

	htmlContent := buf.String()
	textContent := htmlToText(htmlContent)
	return htmlContent, textContent, nil
}

// normaliseName accepts "otp", "otp.html", and strips any directory prefix.
func normaliseName(name string) string {
	// Strip directory prefix if caller passes full path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if !strings.HasSuffix(name, ".html") {
		name += ".html"
	}
	return name
}

// htmlToText strips HTML tags and collapses whitespace for the plain-text fallback.
func htmlToText(html string) string {
	var out strings.Builder
	inTag := false
	for _, ch := range html {
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
		case !inTag:
			out.WriteRune(ch)
		}
	}

	// Collapse runs of blank lines to at most one
	lines := strings.Split(out.String(), "\n")
	var clean []string
	blank := 0
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			if blank == 0 {
				clean = append(clean, "")
			}
			blank++
		} else {
			blank = 0
			clean = append(clean, t)
		}
	}
	return strings.TrimSpace(strings.Join(clean, "\n"))
}

// ─── Template data helpers ───────────────────────────────────────────────────
// Callers should use these constructors to ensure the correct keys are present.

// OTPData builds template data for the "otp" email.
func OTPData(name, otp string, expiresMin int, baseURL string) map[string]any {
	return map[string]any{
		"Name":       name,
		"OTP":        otp,
		"ExpiresMin": expiresMin,
		"BaseURL":    baseURL,
	}
}

// PasswordResetData builds template data for the "password_reset" email.
func PasswordResetData(name, resetURL string, expiresHours int) map[string]any {
	return map[string]any{
		"Name":         name,
		"ResetURL":     resetURL,
		"ExpiresHours": expiresHours,
	}
}

// TransactionReceiptData builds template data for the "transaction_receipt" email.
func TransactionReceiptData(name, orderID, itemTitle string, amount float64, currency, orderURL string) map[string]any {
	return map[string]any{
		"Name":      name,
		"OrderID":   orderID,
		"ItemTitle": itemTitle,
		"Amount":    fmt.Sprintf("%.2f", amount),
		"Currency":  currency,
		"OrderURL":  orderURL,
	}
}

// NotificationData builds template data for the generic "notification" email.
func NotificationData(name, title, body, ctaText, ctaURL string) map[string]any {
	return map[string]any{
		"Name":    name,
		"Title":   title,
		"Body":    body,
		"CTAText": ctaText,
		"CTAURL":  ctaURL,
	}
}

// WelcomeData builds template data for the "welcome" email.
func WelcomeData(name, baseURL string) map[string]any {
	return map[string]any{
		"Name":    name,
		"BaseURL": baseURL,
	}
}
