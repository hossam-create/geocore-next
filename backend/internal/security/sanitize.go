package security

import (
	"html"
	"regexp"
	"strings"
)

// SanitizeText strips all HTML tags and decodes entities.
// Use for plain-text fields like names, titles, descriptions.
func SanitizeText(input string) string {
	// Remove all HTML tags
	stripped := stripTags(stripScriptBlocks(input))
	// Decode HTML entities
	decoded := html.UnescapeString(stripped)
	// Trim whitespace
	return strings.TrimSpace(decoded)
}

// SanitizeHTML removes script/style blocks and dangerous inline handlers while
// keeping text content safe for storage and later rendering.
func SanitizeHTML(input string) string {
	cleaned := stripScriptBlocks(input)
	cleaned = inlineEventPattern.ReplaceAllString(cleaned, "")
	cleaned = javascriptURLPattern.ReplaceAllString(cleaned, "")
	cleaned = stripTags(cleaned)
	cleaned = html.UnescapeString(cleaned)
	return strings.TrimSpace(cleaned)
}

// SanitizeSearchQuery removes dangerous characters from search queries.
func SanitizeSearchQuery(input string) string {
	// Remove SQL-like injection patterns
	cleaned := sqlPattern.ReplaceAllString(input, "")
	// Remove HTML
	cleaned = stripTags(cleaned)
	return strings.TrimSpace(cleaned)
}

var (
	tagPattern           = regexp.MustCompile(`<[^>]*>`)
	scriptStylePattern   = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	inlineEventPattern   = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*("[^"]*"|'[^']*')`)
	javascriptURLPattern = regexp.MustCompile(`(?i)javascript:`)
	sqlPattern           = regexp.MustCompile(`(?i)(--|;|'|"|\\|DROP\s|ALTER\s|DELETE\s|INSERT\s|UPDATE\s|UNION\s|SELECT\s)`)
)

func stripScriptBlocks(s string) string {
	return scriptStylePattern.ReplaceAllString(s, "")
}

func stripTags(s string) string {
	return tagPattern.ReplaceAllString(s, "")
}
