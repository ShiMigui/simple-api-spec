# Simple API (SAS)

Small command-line tool for making HTTP requests during development and manual
API testing. Written in Go, with no external dependencies, it works on Linux,
macOS, and Windows.

The goal is to reduce repetition of `curl` commands, allow reusing
configuration via environment variables, and keep an implementation that is
simple, readable, and cross-platform.

## Installation

```bash
go install github.com/ShiMigui/simple-api-spec/cmd/sas@latest
```

Or, from the source code:

```bash
go build -o sas ./cmd/sas
```

## Usage

Two equivalent formats, both on the same core:

```text
sas METHOD URL [options]
sas-METHOD URL [options]
```

Examples:

```bash
sas get /health
sas post /users -t application/json -b '{"name":"Ana"}'
sas patch /users/42 -t application/json -b '{"active":false}'
sas delete /users/42

# Same flow through the executable name (symlink or copy on Windows):
sas-post /users -t application/json -b '{"name":"Ana"}'
```

MVP methods: `GET`, `POST`, `PUT`, `PATCH`, and `DELETE`.

### Options

```text
-b, --body VALUE            literal request body
-H, --header "Name: value"  additional header; can be repeated
-t, --content-type TYPE     body Content-Type
-h, --help                  help
    --version               version
```

The body provided in `-b` is treated as literal bytes: it is never interpreted,
corrected, or reformatted as JSON.

## Environment variables

| Variable            | Usage                                                |
| ------------------- | ---------------------------------------------------- |
| `API_BASE_URL`      | Base URL for relative endpoints                      |
| `API_BEARER_TOKEN`  | sent as `Authorization: Bearer <token>`              |
| `API_CONTENT_TYPE`  | default `Content-Type`                               |
| `API_ACCEPT`        | default `Accept` value                               |
| `API_TIMEOUT`       | request timeout, for example `30s` (default 30s)     |
| `API_VERBOSE`       | enables verbose logs                                 |

All environment reading is isolated in `internal/config`; no other package
calls `os.Getenv`.

### Precedence

Resolved at a single point (`config.Resolve`):

```text
CLI option  >  environment variable  >  default value
```

Specific rules:

- `-t` takes precedence over `API_CONTENT_TYPE`.
- An explicit `-H` header takes precedence over automatic values
  (`API_ACCEPT`, `API_BEARER_TOKEN`, and the default `Content-Type`).
- Using `-t` together with `-H 'Content-Type: ...'` is rejected as ambiguous.

## URL resolution

- Absolute URLs `http://` and `https://` are used directly.
- Relative endpoints require `API_BASE_URL`; its absence is a clear error.
- The endpoint is **appended** to the base path (explicit behavior, unlike
  traditional resolution, which would replace the path).
- Leading/trailing slashes are normalized; the base path is preserved.
- Query strings are preserved; fragments are not sent.

Examples:

```text
base https://host/api/v1  +  /users          -> https://host/api/v1/users
base http://host:8080/api/ +  users          -> http://host:8080/api/users
absolute URL https://example.com/api/users   -> used as is
```

## Output and exit codes

- The response **body** goes to `stdout`, without automatic formatting.
- Diagnostics and errors go to `stderr`.

This way, redirection works without pollution:

```bash
sas get /users > users.json   # users.json contains only the body
```

Exit codes:

| Code   | Meaning                                      |
| ------ | -------------------------------------------- |
| `0`    | successful HTTP response (`2xx`)             |
| `1`    | transport or configuration error             |
| `2`    | invalid arguments/options                    |
| `3`    | unsuccessful HTTP response (`4xx`/`5xx`)     |

For unsuccessful responses, the body is still written to `stdout` and the
status is reported on `stderr`.

## Security

- Tokens and credentials are never printed in logs, errors, or verbose mode.
- Headers are validated (`Name: value`); CR/LF and invalid characters are
  rejected to prevent injection.
- TLS verification remains enabled.
- A request is redirected at most 10 times; credentials are not
  propagated to different hosts on redirects.

## Project structure

```text
cmd/sas/main.go          entry point; converts the result into an exit code
internal/apperr          error classification -> exit codes
internal/cli             argument parsing (a single Options for all formats)
internal/config          environment reading and resolution (env.go, config.go)
internal/request         URL resolution, headers, body, and HTTP client
internal/response        writing the body and diagnostics to the correct streams
```

All methods and invocation formats converge on the same HTTP flow.

## Tests

```bash
go test ./...
go test -race ./...
```

The tests use `net/http/httptest`, with no dependency on external APIs, and cover
URL resolution, configuration precedence, headers, body, timeout,
redirects, stream separation, and exit codes.
