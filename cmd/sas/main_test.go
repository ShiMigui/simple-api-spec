package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type captured struct {
	mu      sync.Mutex
	method  string
	path    string
	query   string
	headers http.Header
	body    []byte
}

func (c *captured) set(r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.method = r.Method
	c.path = r.URL.Path
	c.query = r.URL.RawQuery
	c.headers = r.Header.Clone()
	c.body, _ = io.ReadAll(r.Body)
}

func (c *captured) get() captured {
	c.mu.Lock()
	defer c.mu.Unlock()
	return captured{method: c.method, path: c.path, query: c.query, headers: c.headers, body: c.body}
}

// newCapture returns an httptest server that records the last request and
// replies with the given status and body.
func newCapture(status int, responseBody string) (*httptest.Server, *captured) {
	cap := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.set(r)
		w.WriteHeader(status)
		io.WriteString(w, responseBody)
	}))
	return srv, cap
}

func runWith(args []string, prog string, env map[string]string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	code := run(args, prog, &stdout, &stderr, lookup)
	return code, stdout.String(), stderr.String()
}

func TestGetResolvesRelativeURLAndWritesBody(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, `{"users":[]}`)
	defer srv.Close()

	code, stdout, stderr := runWith([]string{"get", "/users", "-H", "Accept: application/json"}, "sas", map[string]string{
		"API_BASE_URL": srv.URL + "/api",
	})
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if stdout != `{"users":[]}` {
		t.Fatalf("stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	got := cap.get()
	if got.method != "GET" || got.path != "/api/users" {
		t.Fatalf("server saw %s %s", got.method, got.path)
	}
	if got.headers.Get("Accept") != "application/json" {
		t.Fatalf("Accept = %q", got.headers.Get("Accept"))
	}
}

func TestAllMethodsShareCore(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "ok")
	defer srv.Close()

	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"get", "/x"}, "GET"},
		{[]string{"post", "/x"}, "POST"},
		{[]string{"put", "/x"}, "PUT"},
		{[]string{"patch", "/x"}, "PATCH"},
		{[]string{"delete", "/x"}, "DELETE"},
	} {
		code, _, stderr := runWith(c.args, "sas", map[string]string{"API_BASE_URL": srv.URL})
		if code != 0 {
			t.Fatalf("%v: exit %d (%s)", c.args, code, stderr)
		}
		if got := cap.get().method; got != c.want {
			t.Fatalf("%v: method = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestPostBodyAndContentType(t *testing.T) {
	srv, cap := newCapture(http.StatusCreated, "")
	defer srv.Close()

	body := `{"name":"Ana"}`
	code, _, stderr := runWith([]string{"post", "/users", "-t", "application/json", "-b", body}, "sas", map[string]string{
		"API_BASE_URL": srv.URL,
	})
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	got := cap.get()
	if string(got.body) != body {
		t.Fatalf("body = %q, want %q", got.body, body)
	}
	if got.headers.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", got.headers.Get("Content-Type"))
	}
}

func TestGetSendsNoBodyUnlessRequested(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "")
	defer srv.Close()

	runWith([]string{"get", "/x"}, "sas", map[string]string{"API_BASE_URL": srv.URL})
	if len(cap.get().body) != 0 {
		t.Fatalf("GET sent body %q", cap.get().body)
	}

	runWith([]string{"get", "/x", "-b", "explicit"}, "sas", map[string]string{"API_BASE_URL": srv.URL})
	if string(cap.get().body) != "explicit" {
		t.Fatalf("GET body = %q, want explicit", cap.get().body)
	}
}

func TestContentTypePrecedence(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "")
	defer srv.Close()

	// Environment only.
	runWith([]string{"post", "/x"}, "sas", map[string]string{
		"API_BASE_URL":     srv.URL,
		"API_CONTENT_TYPE": "text/plain",
	})
	if got := cap.get().headers.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("env Content-Type = %q", got)
	}

	// CLI -t wins over environment.
	runWith([]string{"post", "/x", "-t", "application/json"}, "sas", map[string]string{
		"API_BASE_URL":     srv.URL,
		"API_CONTENT_TYPE": "text/plain",
	})
	if got := cap.get().headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("CLI Content-Type = %q", got)
	}

	// Explicit -H wins over the environment default.
	runWith([]string{"post", "/x", "-H", "Content-Type: application/xml"}, "sas", map[string]string{
		"API_BASE_URL":     srv.URL,
		"API_CONTENT_TYPE": "text/plain",
	})
	if got := cap.get().headers.Get("Content-Type"); got != "application/xml" {
		t.Fatalf("explicit Content-Type = %q", got)
	}
}

func TestBearerTokenSentAndNeverPrinted(t *testing.T) {
	const token = "super-secret-token"

	srv, cap := newCapture(http.StatusOK, "fine")
	defer srv.Close()

	code, stdout, stderr := runWith([]string{"get", "/x"}, "sas", map[string]string{
		"API_BASE_URL":     srv.URL,
		"API_BEARER_TOKEN": token,
	})
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if got := cap.get().headers.Get("Authorization"); got != "Bearer "+token {
		t.Fatalf("Authorization = %q", got)
	}
	if strings.Contains(stdout, token) || strings.Contains(stderr, token) {
		t.Fatalf("token leaked: stdout=%q stderr=%q", stdout, stderr)
	}

	// Also ensure the token is not printed on the error path.
	errSrv, _ := newCapture(http.StatusUnauthorized, "nope")
	defer errSrv.Close()
	_, stdout, stderr = runWith([]string{"get", "/x"}, "sas", map[string]string{
		"API_BASE_URL":     errSrv.URL,
		"API_BEARER_TOKEN": token,
	})
	if strings.Contains(stdout, token) || strings.Contains(stderr, token) {
		t.Fatalf("token leaked on error: stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestExplicitAuthorizationWins(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "")
	defer srv.Close()

	runWith([]string{"get", "/x", "-H", "Authorization: Basic abc"}, "sas", map[string]string{
		"API_BASE_URL":     srv.URL,
		"API_BEARER_TOKEN": "env-token",
	})
	if got := cap.get().headers.Get("Authorization"); got != "Basic abc" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestRepeatedHeadersAndInvalidHeader(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "")
	defer srv.Close()

	runWith([]string{"get", "/x", "-H", "X-Debug: one", "-H", "X-Debug: two"}, "sas", map[string]string{
		"API_BASE_URL": srv.URL,
	})
	if got := cap.get().headers.Values("X-Debug"); len(got) != 2 {
		t.Fatalf("X-Debug values = %v", got)
	}

	code, _, _ := runWith([]string{"get", "/x", "-H", "NotAHeader"}, "sas", map[string]string{
		"API_BASE_URL": srv.URL,
	})
	if code != exitUsage {
		t.Fatalf("invalid header exit = %d, want %d", code, exitUsage)
	}
}

func TestAbsoluteURLWithoutBase(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "absolute")
	defer srv.Close()

	code, stdout, stderr := runWith([]string{"get", srv.URL + "/direct"}, "sas", nil)
	if code != 0 {
		t.Fatalf("exit = %d (%s)", code, stderr)
	}
	if stdout != "absolute" {
		t.Fatalf("stdout = %q", stdout)
	}
	if got := cap.get().path; got != "/direct" {
		t.Fatalf("path = %q", got)
	}
}

func TestExitCodes(t *testing.T) {
	ok, _ := newCapture(http.StatusOK, "ok")
	defer ok.Close()
	notFound, _ := newCapture(http.StatusNotFound, "missing")
	defer notFound.Close()
	serverErr, _ := newCapture(http.StatusInternalServerError, "boom")
	defer serverErr.Close()

	if code, _, _ := runWith([]string{"get", "/x"}, "sas", map[string]string{"API_BASE_URL": ok.URL}); code != exitOK {
		t.Fatalf("2xx exit = %d, want %d", code, exitOK)
	}
	if code, _, _ := runWith([]string{"get", "/x"}, "sas", map[string]string{"API_BASE_URL": notFound.URL}); code != exitHTTP {
		t.Fatalf("4xx exit = %d, want %d", code, exitHTTP)
	}
	if code, _, _ := runWith([]string{"get", "/x"}, "sas", map[string]string{"API_BASE_URL": serverErr.URL}); code != exitHTTP {
		t.Fatalf("5xx exit = %d, want %d", code, exitHTTP)
	}
	if code, _, _ := runWith([]string{"bogus", "/x"}, "sas", nil); code != exitUsage {
		t.Fatalf("invalid args exit = %d, want %d", code, exitUsage)
	}
	if code, _, _ := runWith([]string{"get", "/x"}, "sas", nil); code != exitError {
		t.Fatalf("missing base URL exit = %d, want %d", code, exitError)
	}

	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()
	if code, _, _ := runWith([]string{"get", closedURL}, "sas", nil); code != exitError {
		t.Fatalf("transport error exit = %d, want %d", code, exitError)
	}
}

func TestErrorBodyPreservedAndStreamsSeparated(t *testing.T) {
	srv, _ := newCapture(http.StatusNotFound, `{"error":"missing"}`)
	defer srv.Close()

	code, stdout, stderr := runWith([]string{"get", "/x"}, "sas", map[string]string{"API_BASE_URL": srv.URL})
	if code != exitHTTP {
		t.Fatalf("exit = %d", code)
	}
	if stdout != `{"error":"missing"}` {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, "404") {
		t.Fatalf("stderr = %q, want status", stderr)
	}
	if strings.Contains(stdout, "404") || strings.Contains(stdout, "sas:") {
		t.Fatalf("status/log leaked into stdout: %q", stdout)
	}
}

func TestResponseBodiesPreserved(t *testing.T) {
	cases := []struct {
		name string
		body []byte
	}{
		{"empty", nil},
		{"json", []byte(`{"a":1}`)},
		{"text", []byte("hello\nworld")},
		{"binary", []byte{0x00, 0x01, 0xff, 0xfe}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write(c.body)
			}))
			defer srv.Close()

			var stdout, stderr bytes.Buffer
			code := run([]string{"get", "/x"}, "sas", &stdout, &stderr, func(k string) (string, bool) {
				if k == "API_BASE_URL" {
					return srv.URL, true
				}
				return "", false
			})
			if code != exitOK {
				t.Fatalf("exit = %d", code)
			}
			if !bytes.Equal(stdout.Bytes(), c.body) {
				t.Fatalf("body = %v, want %v", stdout.Bytes(), c.body)
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	code, _, _ := runWith([]string{"get", "/x"}, "sas", map[string]string{
		"API_BASE_URL": srv.URL,
		"API_TIMEOUT":  "10ms",
	})
	if code != exitError {
		t.Fatalf("timeout exit = %d, want %d", code, exitError)
	}
}

func TestSasMethodProgramFormat(t *testing.T) {
	srv, cap := newCapture(http.StatusOK, "ok")
	defer srv.Close()

	for _, prog := range []string{"sas-get", "/usr/local/bin/sas-post", `C:\bin\sas-patch.exe`} {
		code, _, stderr := runWith([]string{"/x"}, prog, map[string]string{"API_BASE_URL": srv.URL})
		if code != 0 {
			t.Fatalf("%s: exit %d (%s)", prog, code, stderr)
		}
	}
	if got := cap.get().method; got != "PATCH" {
		t.Fatalf("last method = %q", got)
	}
}

func TestHelpAndVersion(t *testing.T) {
	code, stdout, stderr := runWith([]string{"--help"}, "sas", nil)
	if code != exitOK {
		t.Fatalf("help exit = %d", code)
	}
	if !strings.Contains(stdout, "Usage:") || stderr != "" {
		t.Fatalf("help stdout=%q stderr=%q", stdout, stderr)
	}

	code, stdout, stderr = runWith([]string{"--version"}, "sas", nil)
	if code != exitOK || !strings.Contains(stdout, version) || stderr != "" {
		t.Fatalf("version exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestContentTypeConflictIsUsageError(t *testing.T) {
	code, _, _ := runWith([]string{"post", "/x", "-t", "application/json", "-H", "Content-Type: text/plain"}, "sas", nil)
	if code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}
}
