package validator

import (
	"regexp"
	"unicode"
)

// PasswordStrength validates password complexity
// Requirements:
// - Minimum 10 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
// - At least one special character
func PasswordStrength(password string) bool {
	if len(password) < 10 {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// PasswordStrengthScore returns a score from 0-4 for password strength
func PasswordStrengthScore(password string) int {
	score := 0

	if len(password) >= 10 {
		score++
	}
	if len(password) >= 14 {
		score++
	}
	if regexp.MustCompile(`[A-Z]`).MatchString(password) {
		score++
	}
	if regexp.MustCompile(`[a-z]`).MatchString(password) {
		score++
	}
	if regexp.MustCompile(`[0-9]`).MatchString(password) {
		score++
	}
	if regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		score++
	}

	// Cap at 4
	if score > 4 {
		score = 4
	}

	return score
}

// PasswordStrengthLabel returns a human-readable strength label
func PasswordStrengthLabel(password string) string {
	score := PasswordStrengthScore(password)
	switch score {
	case 0, 1:
		return "weak"
	case 2:
		return "fair"
	case 3:
		return "good"
	case 4:
		return "strong"
	default:
		return "weak"
	}
}
