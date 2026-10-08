package cli

import (
	"testing"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
)

func TestMethodFromProgram(t *testing.T) {
	cases := []struct {
		prog string
		want string
		ok   bool
	}{
		{"/usr/local/bin/sas-get", "GET", true},
		{"sas-post", "POST", true},
		{`C:\bin\sas-delete.exe`, "DELETE", true},
		{"sas-PATCH", "PATCH", true},
		{"sas", "", false},
		{"sas-options", "", false},
		{"curl", "", false},
	}
	for _, c := range cases {
		got, ok := MethodFromProgram(c.prog)
		if got != c.want || ok != c.ok {
			t.Errorf("MethodFromProgram(%q) = (%q, %v), want (%q, %v)", c.prog, got, ok, c.want, c.ok)
		}
	}
}

func TestParseSubcommandFormat(t *testing.T) {
	opts, err := Parse("", []string{"post", "/users", "-t", "application/json", "-b", `{"name":"Ana"}`, "-H", "X-Debug: true", "-H", "X-Debug: false"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "POST" {
		t.Errorf("Method = %q", opts.Method)
	}
	if opts.URL != "/users" {
		t.Errorf("URL = %q", opts.URL)
	}
	if !opts.HasBody || opts.Body != `{"name":"Ana"}` {
		t.Errorf("Body = %q (has=%v)", opts.Body, opts.HasBody)
	}
	if !opts.HasContentType || opts.ContentType != "application/json" {
		t.Errorf("ContentType = %q", opts.ContentType)
	}
	if len(opts.Headers) != 2 {
		t.Errorf("Headers = %v", opts.Headers)
	}
}

func TestParseProgramMethodFormat(t *testing.T) {
	opts, err := Parse("get", []string{"/health"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "get" {
		t.Errorf("Method = %q", opts.Method)
	}
	if opts.URL != "/health" {
		t.Errorf("URL = %q", opts.URL)
	}
}

func TestParseFlagsAnywhereAndInline(t *testing.T) {
	opts, err := Parse("", []string{"put", "--content-type=text/plain", "/x", "--body=olá"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Method != "PUT" || opts.URL != "/x" {
		t.Fatalf("got method=%q url=%q", opts.Method, opts.URL)
	}
	if opts.ContentType != "text/plain" || opts.Body != "olá" {
		t.Fatalf("got contentType=%q body=%q", opts.ContentType, opts.Body)
	}
}

func TestParseEmptyBodyIsExplicit(t *testing.T) {
	opts, err := Parse("post", []string{"/x", "-b", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.HasBody || opts.Body != "" {
		t.Fatalf("expected explicit empty body, got hasBody=%v body=%q", opts.HasBody, opts.Body)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"empty", nil},
		{"unsupported method", []string{"frobnicate", "/x"}},
		{"missing url", []string{"get"}},
		{"unknown option", []string{"get", "/x", "--nope"}},
		{"extra positional", []string{"get", "/x", "extra"}},
		{"missing value", []string{"get", "/x", "-t"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse("", c.args)
			if err == nil {
				t.Fatalf("expected error for %v", c.args)
			}
			if apperr.KindOf(err) != apperr.Usage {
				t.Fatalf("KindOf = %v, want Usage", apperr.KindOf(err))
			}
		})
	}
}

func TestParseContentTypeConflict(t *testing.T) {
	_, err := Parse("post", []string{"/x", "-t", "application/json", "-H", "Content-Type: text/plain"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if apperr.KindOf(err) != apperr.Usage {
		t.Fatalf("KindOf = %v, want Usage", apperr.KindOf(err))
	}
}

func TestParseHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"get", "--help"}} {
		opts, err := Parse("", args)
		if err != nil {
			t.Fatalf("Parse(%v): %v", args, err)
		}
		if !opts.ShowHelp {
			t.Fatalf("Parse(%v): ShowHelp = false", args)
		}
	}

	opts, err := Parse("", []string{"--version"})
	if err != nil || !opts.ShowVersion {
		t.Fatalf("Parse(--version) = %+v, %v", opts, err)
	}
}
