package request

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
)

// maxRedirects caps how many redirects the client follows. Redirects are never
// followed without limit.
const maxRedirects = 10

// Request describes one HTTP request, independent of method and invocation
// format.
type Request struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
	Timeout time.Duration
}

// Do builds and executes the request and returns the raw response. The caller
// owns the response body.
//
// The standard library strips sensitive headers (Authorization, Cookie, ...)
// when a redirect points at a different host, so credentials are not leaked
// across hosts. This function only adds a hard redirect limit on top.
func (r Request) Do(ctx context.Context) (*http.Response, error) {
	var body io.Reader
	if r.Body != nil {
		body = bytes.NewReader(r.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, r.Method, r.URL, body)
	if err != nil {
		return nil, apperr.Usagef("could not build request: %v", err)
	}
	for name, values := range r.Headers {
		for _, value := range values {
			httpReq.Header.Add(name, value)
		}
	}

	client := &http.Client{
		Timeout: r.Timeout,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, apperr.Transportf("%v", err)
	}
	return resp, nil
}
