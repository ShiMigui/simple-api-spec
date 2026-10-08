package request

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRedirectLimit(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Redirect(w, r, "/again", http.StatusFound)
	}))
	defer srv.Close()

	req := Request{Method: http.MethodGet, URL: srv.URL, Timeout: 5 * time.Second}
	_, err := req.Do(context.Background())
	if err == nil {
		t.Fatal("expected error after too many redirects")
	}
	if got := atomic.LoadInt32(&hits); got < 1 || got > maxRedirects {
		t.Fatalf("server hits = %d, want between 1 and %d", got, maxRedirects)
	}
}

func TestDoStripsAuthorizationOnHostChange(t *testing.T) {
	var leaked string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	// Use a different hostname for the redirect target so the client sees a
	// cross-host redirect and must drop credentials.
	targetURL := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetURL, http.StatusFound)
	}))
	defer redirector.Close()

	req := Request{
		Method:  http.MethodGet,
		URL:     redirector.URL,
		Headers: http.Header{"Authorization": {"Bearer secret"}},
		Timeout: 5 * time.Second,
	}
	resp, err := req.Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if leaked != "" {
		t.Fatalf("Authorization leaked across hosts: %q", leaked)
	}
}
