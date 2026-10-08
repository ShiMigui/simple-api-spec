// Package response writes the HTTP response to the correct stream.
package response

import (
	"io"
	"net/http"
)

// WriteBody copies the response body to w byte-for-byte, without adding status
// lines, logs or automatic formatting. The caller must not use resp.Body after
// this call. It is the caller's responsibility to report non-2xx status.
func WriteBody(w io.Writer, resp *http.Response) (int64, error) {
	defer resp.Body.Close()
	return io.Copy(w, resp.Body)
}
