// Package config is the single place in the program that reads and resolves
// environment variables. No other package calls os.Getenv directly.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
)

// LookupEnv reads an environment variable, mirroring os.LookupEnv. It is a
// function type so tests can inject their own environment without touching
// global process state.
type LookupEnv func(key string) (value string, ok bool)

// Default values applied when neither the CLI nor the environment provides one.
const DefaultTimeout = 30 * time.Second

// Supported environment variable names. Keeping them as constants avoids
// inconsistent spellings such as BAERER_TOKEN.
const (
	EnvBaseURL     = "API_BASE_URL"
	EnvBearerToken = "API_BEARER_TOKEN"
	EnvContentType = "API_CONTENT_TYPE"
	EnvAccept      = "API_ACCEPT"
	EnvTimeout     = "API_TIMEOUT"
	EnvVerbose     = "API_VERBOSE"
)

// Config holds the values read from the environment, with defaults applied.
type Config struct {
	BaseURL     string
	BearerToken string
	ContentType string
	Accept      string
	Timeout     time.Duration
	Verbose     bool
}

// LoadFromEnv reads and validates the supported environment variables.
// A nil lookup defaults to os.LookupEnv. The environment is never read at
// import time (no init), so callers control exactly when it happens.
func LoadFromEnv(lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}

	cfg := Config{
		BaseURL:     strings.TrimSpace(lookupValue(lookup, EnvBaseURL)),
		BearerToken: strings.TrimSpace(lookupValue(lookup, EnvBearerToken)),
		ContentType: strings.TrimSpace(lookupValue(lookup, EnvContentType)),
		Accept:      strings.TrimSpace(lookupValue(lookup, EnvAccept)),
		Timeout:     DefaultTimeout,
	}

	if raw := strings.TrimSpace(lookupValue(lookup, EnvTimeout)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, apperr.Configf("invalid %s %q: %v", EnvTimeout, raw, err)
		}
		if d <= 0 {
			return Config{}, apperr.Configf("invalid %s %q: must be greater than zero", EnvTimeout, raw)
		}
		cfg.Timeout = d
	}

	if raw := strings.TrimSpace(lookupValue(lookup, EnvVerbose)); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, apperr.Configf("invalid %s %q: expected a boolean", EnvVerbose, raw)
		}
		cfg.Verbose = v
	}

	return cfg, nil
}

func lookupValue(lookup LookupEnv, key string) string {
	v, _ := lookup(key)
	return v
}
