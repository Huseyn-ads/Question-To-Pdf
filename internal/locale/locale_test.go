package locale

import (
	"context"
	"testing"
)

func TestFromAcceptLanguage(t *testing.T) {
	for header, want := range map[string]string{
		"":                        Russian,
		"az":                      Azerbaijani,
		"az-Latn-AZ,ru;q=0.8":     Azerbaijani,
		"en-US,en;q=0.9,az;q=0.8": Azerbaijani,
		"ru-RU,ru;q=0.9":          Russian,
		"en-US":                   Russian,
		"tr, ru":                  Russian,
	} {
		if got := FromAcceptLanguage(header); got != want {
			t.Errorf("FromAcceptLanguage(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestContextDefaultsToRussian(t *testing.T) {
	if FromContext(context.Background()) != Russian {
		t.Fatal("missing language should be Russian")
	}
	if FromContext(WithLanguage(context.Background(), "az")) != Azerbaijani {
		t.Fatal("language was not kept in context")
	}
	if FromContext(WithLanguage(context.Background(), "<script>")) != Russian {
		t.Fatal("unsupported language leaked into context")
	}
}
