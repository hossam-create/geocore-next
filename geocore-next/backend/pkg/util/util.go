package util

import "os"

// Getenv returns the value of the environment variable key, or fallback if not set.
func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Getenv2 returns the value of key1, or key2, or fallback — whichever is first non-empty.
func Getenv2(key1, key2, fallback string) string {
	if v := os.Getenv(key1); v != "" {
		return v
	}
	if v := os.Getenv(key2); v != "" {
		return v
	}
	return fallback
}

// DefaultStr returns s if non-empty, otherwise def.
func DefaultStr(s, def string) string {
	if s != "" {
		return s
	}
	return def
}
