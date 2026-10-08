package request

import (
	"net/http"
	"strings"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
	"github.com/ShiMigui/simple-api-spec/internal/config"
)

// ParseHeader validates and splits a single "Name: value" header. CR/LF and
// other control characters are rejected to prevent header injection.
func ParseHeader(raw string) (name, value string, err error) {
	i := strings.IndexByte(raw, ':')
	if i <= 0 {
		return "", "", apperr.Usagef("invalid header %q: expected \"Name: value\"", raw)
	}
	name = strings.TrimSpace(raw[:i])
	value = strings.Trim(raw[i+1:], " \t")
	if !validHeaderName(name) {
		return "", "", apperr.Usagef("invalid header name %q", name)
	}
	if !validHeaderValue(value) {
		return "", "", apperr.Usagef("invalid header value for %q", name)
	}
	return name, value, nil
}

// BuildHeaders resolves the final header set with a single precedence policy:
// explicit -H headers win over automatic values, which come from resolved
// settings (CLI > environment > default). Repeated -H headers keep every value.
func BuildHeaders(extra []string, s config.Settings) (http.Header, error) {
	headers := make(http.Header)

	var explicitContentType, explicitAccept, explicitAuth bool
	for _, raw := range extra {
		name, value, err := ParseHeader(raw)
		if err != nil {
			return nil, err
		}
		switch {
		case strings.EqualFold(name, "Content-Type"):
			explicitContentType = true
		case strings.EqualFold(name, "Accept"):
			explicitAccept = true
		case strings.EqualFold(name, "Authorization"):
			explicitAuth = true
		}
		headers.Add(name, value)
	}

	if !explicitContentType && s.ContentType != "" {
		headers.Set("Content-Type", s.ContentType)
	}
	if !explicitAccept && s.Accept != "" {
		headers.Set("Accept", s.Accept)
	}
	if !explicitAuth && s.BearerToken != "" {
		headers.Set("Authorization", "Bearer "+s.BearerToken)
	}
	return headers, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !isTokenByte(name[i]) {
			return false
		}
	}
	return true
}

func isTokenByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}

func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '\t' {
			continue
		}
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}
