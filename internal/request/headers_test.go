package request

import (
	"testing"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
	"github.com/ShiMigui/simple-api-spec/internal/config"
)

func TestParseHeader(t *testing.T) {
	name, value, err := ParseHeader("X-Debug:  true ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "X-Debug" || value != "true" {
		t.Fatalf("got %q: %q", name, value)
	}
}

func TestParseHeaderInvalid(t *testing.T) {
	cases := []string{
		"NoColon",
		": missing name",
		"Bad Name: value",
		"X-Test: bad\r\nInjected: yes",
		"X-Test: bad\nInjected: yes",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			_, _, err := ParseHeader(raw)
			if err == nil {
				t.Fatalf("expected error for %q", raw)
			}
			if apperr.KindOf(err) != apperr.Usage {
				t.Fatalf("KindOf = %v, want Usage", apperr.KindOf(err))
			}
		})
	}
}

func TestBuildHeadersRepeatedValues(t *testing.T) {
	h, err := BuildHeaders([]string{"X-Debug: a", "X-Debug: b"}, config.Settings{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := h.Values("X-Debug"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("X-Debug = %v", got)
	}
}

func TestBuildHeadersPrecedence(t *testing.T) {
	settings := config.Settings{
		ContentType: "application/json",
		Accept:      "application/json",
		BearerToken: "env-token",
	}

	// Automatic values are applied when nothing explicit is given.
	h, err := BuildHeaders(nil, settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Get("Content-Type") != "application/json" || h.Get("Accept") != "application/json" {
		t.Fatalf("defaults not applied: %v", h)
	}
	if h.Get("Authorization") != "Bearer env-token" {
		t.Fatalf("Authorization = %q", h.Get("Authorization"))
	}

	// Explicit -H headers win over automatic values.
	h, err = BuildHeaders([]string{
		"Content-Type: text/plain",
		"Accept: text/csv",
		"Authorization: Basic abc",
	}, settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Get("Content-Type") != "text/plain" {
		t.Fatalf("Content-Type = %q", h.Get("Content-Type"))
	}
	if h.Get("Accept") != "text/csv" {
		t.Fatalf("Accept = %q", h.Get("Accept"))
	}
	if h.Get("Authorization") != "Basic abc" {
		t.Fatalf("Authorization = %q", h.Get("Authorization"))
	}
}

func TestBuildHeadersNoAutomaticType(t *testing.T) {
	h, err := BuildHeaders(nil, config.Settings{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Get("Content-Type") != "" {
		t.Fatalf("Content-Type = %q, want empty (must not be inferred)", h.Get("Content-Type"))
	}
}
