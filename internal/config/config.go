package config

import "time"

// Overrides carries CLI-provided values that take precedence over the
// environment. The Has* flags distinguish "not provided" from an explicit
// empty value.
type Overrides struct {
	ContentType    string
	HasContentType bool
}

// Settings is the fully resolved configuration used by the request layer.
type Settings struct {
	BaseURL     string
	BearerToken string
	ContentType string
	Accept      string
	Timeout     time.Duration
	Verbose     bool
}

// Resolve applies the single project-wide precedence policy:
//
//	CLI option > environment variable > internal default
//
// Defaults are already materialised by LoadFromEnv, so this function only needs
// to let explicit CLI overrides win over the environment.
func Resolve(env Config, ov Overrides) Settings {
	s := Settings{
		BaseURL:     env.BaseURL,
		BearerToken: env.BearerToken,
		ContentType: env.ContentType,
		Accept:      env.Accept,
		Timeout:     env.Timeout,
		Verbose:     env.Verbose,
	}
	if ov.HasContentType {
		s.ContentType = ov.ContentType
	}
	return s
}
