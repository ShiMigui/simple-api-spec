package config

import (
	"testing"
	"time"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
)

func lookupFrom(m map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func TestLoadFromEnvDefaults(t *testing.T) {
	cfg, err := LoadFromEnv(lookupFrom(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "" || cfg.BearerToken != "" || cfg.ContentType != "" || cfg.Accept != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}
	if cfg.Verbose {
		t.Fatal("Verbose = true, want false")
	}
}

func TestLoadFromEnvReadsValues(t *testing.T) {
	cfg, err := LoadFromEnv(lookupFrom(map[string]string{
		EnvBaseURL:     " http://localhost:8080/api ",
		EnvBearerToken: "secret",
		EnvContentType: "application/json",
		EnvAccept:      "application/json",
		EnvTimeout:     "5s",
		EnvVerbose:     "true",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "http://localhost:8080/api" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.BearerToken != "secret" {
		t.Fatalf("BearerToken = %q", cfg.BearerToken)
	}
	if cfg.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %v", cfg.Timeout)
	}
	if !cfg.Verbose {
		t.Fatal("Verbose = false, want true")
	}
}

func TestLoadFromEnvInvalidTimeout(t *testing.T) {
	for _, raw := range []string{"nope", "0s", "-1s"} {
		_, err := LoadFromEnv(lookupFrom(map[string]string{EnvTimeout: raw}))
		if err == nil {
			t.Fatalf("expected error for %q", raw)
		}
		if apperr.KindOf(err) != apperr.Config {
			t.Fatalf("KindOf(%v) = %v, want Config", err, apperr.KindOf(err))
		}
	}
}

func TestLoadFromEnvInvalidVerbose(t *testing.T) {
	_, err := LoadFromEnv(lookupFrom(map[string]string{EnvVerbose: "maybe"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.KindOf(err) != apperr.Config {
		t.Fatalf("KindOf = %v, want Config", apperr.KindOf(err))
	}
}

func TestResolvePrecedence(t *testing.T) {
	env := Config{ContentType: "text/plain", Accept: "application/json", Timeout: time.Second}

	// No CLI override: environment wins over the (already materialised) default.
	got := Resolve(env, Overrides{})
	if got.ContentType != "text/plain" {
		t.Fatalf("ContentType = %q, want env value", got.ContentType)
	}

	// CLI override wins over the environment.
	got = Resolve(env, Overrides{ContentType: "application/json", HasContentType: true})
	if got.ContentType != "application/json" {
		t.Fatalf("ContentType = %q, want CLI value", got.ContentType)
	}
}
