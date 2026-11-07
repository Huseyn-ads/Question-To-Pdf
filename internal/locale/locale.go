// Package locale carries the interface language of a request (Russian or
// Azerbaijani) to code that writes user-facing text, such as emails.
package locale

import (
	"context"
	"strings"
)

const (
	Russian     = "ru"
	Azerbaijani = "az"
	Default     = Russian
)

type contextKey struct{}

// FromAcceptLanguage returns the first supported language in an
// Accept-Language header. The frontend sends the language chosen in the
// interface; browsers send their own list. Anything else falls back to Russian.
func FromAcceptLanguage(header string) string {
	for _, item := range strings.Split(header, ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(item), ";")
		primary, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
		switch primary {
		case Azerbaijani:
			return Azerbaijani
		case Russian:
			return Russian
		}
	}
	return Default
}

func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, contextKey{}, Normalize(lang))
}

func FromContext(ctx context.Context) string {
	if lang, ok := ctx.Value(contextKey{}).(string); ok {
		return lang
	}
	return Default
}

// Normalize keeps only the supported language codes.
func Normalize(lang string) string {
	if lang == Azerbaijani {
		return Azerbaijani
	}
	return Russian
}
