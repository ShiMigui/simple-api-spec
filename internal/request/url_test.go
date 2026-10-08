package request

import (
	"testing"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
)

func TestResolveURL(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		base     string
		want     string
	}{
		{"absolute", "https://example.com/api/users", "", "https://example.com/api/users"},
		{"absolute wins over base", "http://other.test/x", "http://base.test/api", "http://other.test/x"},
		{"relative with base", "/users", "http://localhost:8080/api", "http://localhost:8080/api/users"},
		{"base with trailing slash", "/users", "http://localhost:8080/api/", "http://localhost:8080/api/users"},
		{"base prefix preserved", "/users", "https://host/api/v1", "https://host/api/v1/users"},
		{"endpoint without slash", "users", "https://host/api/v1", "https://host/api/v1/users"},
		{"base without path", "/health", "http://host", "http://host/health"},
		{"query preserved", "/users?page=2&limit=10", "http://host/api", "http://host/api/users?page=2&limit=10"},
		{"base query merged", "/users", "http://host/api?key=1", "http://host/api/users?key=1"},
		{"fragment dropped", "/users#section", "http://host/api", "http://host/api/users"},
		{"absolute fragment dropped", "http://host/a#frag", "", "http://host/a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveURL(c.endpoint, c.base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("ResolveURL(%q, %q) = %q, want %q", c.endpoint, c.base, got, c.want)
			}
		})
	}
}

func TestResolveURLErrors(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		base     string
		wantKind apperr.Kind
	}{
		{"relative without base", "/users", "", apperr.Config},
		{"unsupported absolute scheme", "ftp://host/x", "", apperr.Usage},
		{"unsupported base scheme", "/x", "ftp://host", apperr.Config},
		{"empty endpoint", "", "http://host", apperr.Usage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ResolveURL(c.endpoint, c.base)
			if err == nil {
				t.Fatal("expected error")
			}
			if apperr.KindOf(err) != c.wantKind {
				t.Fatalf("KindOf = %v, want %v (%v)", apperr.KindOf(err), c.wantKind, err)
			}
		})
	}
}
