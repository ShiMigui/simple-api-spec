// Package cli parses command-line arguments into a method-agnostic Options
// value. Both invocation formats (sas METHOD URL and sas-METHOD URL) converge
// on the same Parse entry point, so the HTTP core is never duplicated.
package cli

import (
	"net/http"
	"strings"

	"github.com/ShiMigui/simple-api-spec/internal/apperr"
)

// Options is the result of parsing the command line. It is free of HTTP and
// environment concerns so the same struct serves every method and format.
type Options struct {
	Method         string
	URL            string
	Body           string
	HasBody        bool
	Headers        []string
	ContentType    string
	HasContentType bool
	ShowHelp       bool
	ShowVersion    bool
}

// supportedMethods are the HTTP methods handled by the shared executor.
var supportedMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// MethodFromProgram reports the HTTP method encoded in a sas-METHOD executable
// name (for example sas-get). The boolean is false when the name does not
// follow that convention.
func MethodFromProgram(prog string) (string, bool) {
	name := prog
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if strings.HasSuffix(strings.ToLower(name), ".exe") {
		name = name[:len(name)-len(".exe")]
	}
	if len(name) < len("sas-") || !strings.EqualFold(name[:len("sas-")], "sas-") {
		return "", false
	}
	method := strings.ToUpper(name[len("sas-"):])
	if !supportedMethods[method] {
		return "", false
	}
	return method, true
}

// Parse interprets args (excluding the program name) into Options.
//
// When defaultMethod is empty the first argument must be the HTTP method
// (format "sas METHOD URL"). Otherwise defaultMethod is used and every argument
// is treated as URL or option (format "sas-METHOD URL").
func Parse(defaultMethod string, args []string) (Options, error) {
	var opts Options

	method := defaultMethod
	rest := args
	if method == "" {
		if len(rest) == 0 {
			return opts, apperr.Usagef("missing HTTP method and URL")
		}
		switch rest[0] {
		case "-h", "--help":
			opts.ShowHelp = true
			return opts, nil
		case "--version":
			opts.ShowVersion = true
			return opts, nil
		}
		m, ok := normalizeMethod(rest[0])
		if !ok {
			return opts, apperr.Usagef("unsupported HTTP method %q", rest[0])
		}
		method = m
		rest = rest[1:]
	}
	opts.Method = method

	urlSet := false
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		name, inline, hasInline := splitFlag(arg)

		switch name {
		case "-h", "--help":
			opts.ShowHelp = true
		case "--version":
			opts.ShowVersion = true
		case "-b", "--body":
			v, err := flagValue(name, inline, hasInline, rest, &i)
			if err != nil {
				return opts, err
			}
			opts.Body = v
			opts.HasBody = true
		case "-H", "--header":
			v, err := flagValue(name, inline, hasInline, rest, &i)
			if err != nil {
				return opts, err
			}
			opts.Headers = append(opts.Headers, v)
		case "-t", "--content-type":
			v, err := flagValue(name, inline, hasInline, rest, &i)
			if err != nil {
				return opts, err
			}
			opts.ContentType = v
			opts.HasContentType = true
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return opts, apperr.Usagef("unknown option %q", arg)
			}
			if urlSet {
				return opts, apperr.Usagef("unexpected argument %q", arg)
			}
			opts.URL = arg
			urlSet = true
		}
	}

	if opts.ShowHelp || opts.ShowVersion {
		return opts, nil
	}
	if opts.URL == "" {
		return opts, apperr.Usagef("missing URL")
	}
	if err := checkContentTypeConflict(opts); err != nil {
		return opts, err
	}
	return opts, nil
}

func normalizeMethod(raw string) (string, bool) {
	m := strings.ToUpper(strings.TrimSpace(raw))
	if supportedMethods[m] {
		return m, true
	}
	return "", false
}

// splitFlag separates "--name=value" and "-x=value" forms. Long and short flags
// without an inline value are returned unchanged.
func splitFlag(arg string) (name, value string, hasValue bool) {
	if len(arg) > 1 && strings.HasPrefix(arg, "-") {
		if i := strings.IndexByte(arg, '='); i >= 0 {
			return arg[:i], arg[i+1:], true
		}
	}
	return arg, "", false
}

func flagValue(name, inline string, hasInline bool, args []string, i *int) (string, error) {
	if hasInline {
		return inline, nil
	}
	if *i+1 >= len(args) {
		return "", apperr.Usagef("option %s requires a value", name)
	}
	*i++
	return args[*i], nil
}

// checkContentTypeConflict rejects the ambiguous combination of -t and an
// explicit -H 'Content-Type: ...' before anything is sent.
func checkContentTypeConflict(opts Options) error {
	if !opts.HasContentType {
		return nil
	}
	for _, raw := range opts.Headers {
		if i := strings.IndexByte(raw, ':'); i > 0 {
			if strings.EqualFold(strings.TrimSpace(raw[:i]), "Content-Type") {
				return apperr.Usagef("ambiguous Content-Type: use either -t/--content-type or -H 'Content-Type: ...', not both")
			}
		}
	}
	return nil
}
