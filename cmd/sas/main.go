// Command sas is a small HTTP client for manual API development and testing.
//
// It supports two equivalent invocation formats backed by one shared core:
//
//	sas METHOD URL [options]
//	sas-METHOD URL [options]
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
	"github.com/ShiMigui/simple-api-spec/internal/cli"
	"github.com/ShiMigui/simple-api-spec/internal/config"
	"github.com/ShiMigui/simple-api-spec/internal/request"
	"github.com/ShiMigui/simple-api-spec/internal/response"
)

// version is reported by --version.
const version = "0.1.0"

// Exit codes, as documented in the README.
const (
	exitOK    = 0 // 2xx response
	exitError = 1 // transport or configuration error
	exitUsage = 2 // invalid arguments or options
	exitHTTP  = 3 // non-2xx HTTP response
)

func main() {
	os.Exit(run(os.Args[1:], os.Args[0], os.Stdout, os.Stderr, os.LookupEnv))
}

// run is the testable entry point. It returns the process exit code.
func run(args []string, prog string, stdout, stderr io.Writer, lookup config.LookupEnv) int {
	defaultMethod, _ := cli.MethodFromProgram(prog)

	opts, err := cli.Parse(defaultMethod, args)
	if err != nil {
		fmt.Fprintf(stderr, "sas: %v\n", err)
		fmt.Fprintln(stderr, "Try 'sas --help' for usage.")
		return exitUsage
	}
	if opts.ShowHelp {
		printUsage(stdout)
		return exitOK
	}
	if opts.ShowVersion {
		fmt.Fprintf(stdout, "sas %s\n", version)
		return exitOK
	}

	envCfg, err := config.LoadFromEnv(lookup)
	if err != nil {
		return fail(stderr, err)
	}
	settings := config.Resolve(envCfg, config.Overrides{
		ContentType:    opts.ContentType,
		HasContentType: opts.HasContentType,
	})

	resolvedURL, err := request.ResolveURL(opts.URL, settings.BaseURL)
	if err != nil {
		return fail(stderr, err)
	}
	headers, err := request.BuildHeaders(opts.Headers, settings)
	if err != nil {
		return fail(stderr, err)
	}

	req := request.Request{
		Method:  opts.Method,
		URL:     resolvedURL,
		Headers: headers,
		Body:    request.BuildBody(opts.Body, opts.HasBody),
		Timeout: settings.Timeout,
	}

	resp, err := req.Do(context.Background())
	if err != nil {
		return fail(stderr, err)
	}

	if _, err := response.WriteBody(stdout, resp); err != nil {
		fmt.Fprintf(stderr, "sas: could not read response body: %v\n", err)
		return exitError
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return exitOK
	}
	// The body was already written to stdout; the status goes to stderr so it
	// never contaminates redirected output.
	fmt.Fprintf(stderr, "sas: HTTP %s\n", resp.Status)
	return exitHTTP
}

// fail prints an operational error to stderr and maps it to an exit code.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "sas: %v\n", err)
	if apperr.KindOf(err) == apperr.Usage {
		return exitUsage
	}
	return exitError
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `sas %s - thin HTTP client for manual API testing

Usage:
  sas METHOD URL [options]
  sas-METHOD URL [options]

Methods:
  GET, POST, PUT, PATCH, DELETE

Options:
  -b, --body VALUE            literal request body
  -H, --header "Name: value"  extra header; may be repeated
  -t, --content-type TYPE     Content-Type of the body
  -h, --help                  show this help
      --version               show the version

Environment:
  API_BASE_URL       base URL for relative endpoints
  API_BEARER_TOKEN   sent as "Authorization: Bearer <token>"
  API_CONTENT_TYPE   default Content-Type
  API_ACCEPT         default Accept
  API_TIMEOUT        request timeout (default 30s)
  API_VERBOSE        enable verbose output

Precedence: CLI option > environment variable > default.

Exit codes:
  0  successful 2xx response
  1  transport or configuration error
  2  invalid arguments or options
  3  non-2xx HTTP response

Examples:
  API_BASE_URL=http://localhost:8080/api sas get /health
  sas post /users -t application/json -b '{"name":"Ana"}'
  sas get /users -H 'Accept: application/json'
`, version)
}
