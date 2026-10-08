package request

// BuildBody returns the request body bytes.
//
// In the MVP only a literal body is supported. The content is treated as raw
// bytes and is never parsed or reformatted. A body is attached only when it was
// explicitly requested with -b, which covers GET and DELETE.
func BuildBody(body string, hasBody bool) []byte {
	if !hasBody {
		return nil
	}
	return []byte(body)
}
