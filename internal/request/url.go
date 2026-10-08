package request

import (
	"net/url"
	"strings"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
	"github.com/ShiMigui/simple-api-spec/internal/config"
)

// ResolveURL turns an endpoint into a final URL.
//
// Absolute http/https URLs are used as-is. Relative endpoints are appended to
// baseURL, preserving the base path (for example base "https://host/api/v1"
// plus "/users" yields "https://host/api/v1/users"). Query strings are kept and
// fragments are dropped, since fragments are never sent in HTTP requests.
func ResolveURL(endpoint, baseURL string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", apperr.Usagef("empty URL")
	}

	ref, err := url.Parse(endpoint)
	if err != nil {
		return "", apperr.Usagef("invalid URL %q: %v", endpoint, err)
	}

	if ref.IsAbs() || ref.Host != "" {
		if !isHTTPScheme(ref.Scheme) {
			return "", apperr.Usagef("unsupported URL scheme %q (use http or https)", ref.Scheme)
		}
		if ref.Host == "" {
			return "", apperr.Usagef("invalid URL %q: missing host", endpoint)
		}
		ref.Fragment = ""
		return ref.String(), nil
	}

	if baseURL == "" {
		return "", apperr.Configf("relative endpoint %q requires %s to be set", endpoint, config.EnvBaseURL)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", apperr.Configf("invalid %s %q: %v", config.EnvBaseURL, baseURL, err)
	}
	if !isHTTPScheme(base.Scheme) {
		return "", apperr.Configf("%s %q: unsupported scheme (use http or https)", config.EnvBaseURL, baseURL)
	}
	if base.Host == "" {
		return "", apperr.Configf("%s %q: missing host", config.EnvBaseURL, baseURL)
	}

	// Append the endpoint path to the base path rather than replacing it, as
	// traditional URL resolution would do for a leading "/".
	basePath := strings.TrimRight(base.EscapedPath(), "/")
	endPath := ref.EscapedPath()
	if endPath != "" && !strings.HasPrefix(endPath, "/") {
		endPath = "/" + endPath
	}
	fullPath := basePath + endPath
	if fullPath == "" {
		fullPath = "/"
	}

	query := mergeQuery(base.RawQuery, ref.RawQuery)

	var b strings.Builder
	b.WriteString(strings.ToLower(base.Scheme))
	b.WriteString("://")
	if base.User != nil {
		b.WriteString(base.User.String())
		b.WriteByte('@')
	}
	b.WriteString(base.Host)
	b.WriteString(fullPath)
	if query != "" {
		b.WriteByte('?')
		b.WriteString(query)
	}

	final, err := url.Parse(b.String())
	if err != nil {
		return "", apperr.Usagef("could not resolve %q against %q: %v", endpoint, baseURL, err)
	}
	final.Fragment = ""
	return final.String(), nil
}

func isHTTPScheme(scheme string) bool {
	return strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https")
}

func mergeQuery(baseQuery, endpointQuery string) string {
	switch {
	case endpointQuery == "":
		return baseQuery
	case baseQuery == "":
		return endpointQuery
	default:
		return baseQuery + "&" + endpointQuery
	}
}
