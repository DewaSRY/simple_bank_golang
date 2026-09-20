package logger

import (
	"net/http"
	"net/url"
	"strings"
)

// RedactedPlaceholder replaces the value of anything matched by
// SensitiveHeaderKeys or SensitiveFieldKeys.
const RedactedPlaceholder = "[REDACTED]"

// SensitiveHeaderKeys are header names (case-insensitive) whose values
// RedactHeaders replaces with RedactedPlaceholder. Centralized here so
// every call site that logs headers (access log, future debugging tools)
// shares one denylist instead of each reimplementing it.
var SensitiveHeaderKeys = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"proxy-authorization": true,
	"x-api-key":           true,
	"api-key":             true,
	"x-auth-token":        true,
}

// SensitiveFieldKeys are JSON object keys (case-insensitive) whose values
// RedactJSONValue replaces with RedactedPlaceholder, at any nesting depth.
var SensitiveFieldKeys = map[string]bool{
	"password":              true,
	"password_confirmation": true,
	"confirm_password":      true,
	"old_password":          true,
	"new_password":          true,
	"token":                 true,
	"access_token":          true,
	"refresh_token":         true,
	"id_token":              true,
	"jwt":                   true,
	"secret":                true,
	"client_secret":         true,
	"api_key":               true,
	"apikey":                true,
	"authorization":         true,
	"card_number":           true,
	"cvv":                   true,
}

// RedactHeaders returns a lowercase-keyed copy of h suitable for logging:
// single-value headers flatten to a string, multi-value ones stay a slice,
// and anything in SensitiveHeaderKeys becomes RedactedPlaceholder regardless
// of how many values it had.
func RedactHeaders(h http.Header) map[string]any {
	out := make(map[string]any, len(h))
	for key, values := range h {
		lower := strings.ToLower(key)
		if SensitiveHeaderKeys[lower] {
			out[lower] = RedactedPlaceholder
			continue
		}
		if len(values) == 1 {
			out[lower] = values[0]
			continue
		}
		out[lower] = values
	}
	return out
}

// RedactQuery returns a lowercase-keyed copy of q suitable for logging,
// replacing the value of any parameter in SensitiveFieldKeys with
// RedactedPlaceholder — a token/reset-code/API key accepted as a query
// param must not land in logs unredacted, the same guarantee RedactHeaders
// and RedactJSONValue already give headers and JSON bodies.
func RedactQuery(q url.Values) map[string]any {
	out := make(map[string]any, len(q))
	for key, values := range q {
		lower := strings.ToLower(key)
		if SensitiveFieldKeys[lower] {
			out[lower] = RedactedPlaceholder
			continue
		}
		if len(values) == 1 {
			out[lower] = values[0]
			continue
		}
		out[lower] = values
	}
	return out
}

// RedactJSONValue walks a value produced by json.Unmarshal(data, &v) with
// v of type any — so maps decode as map[string]any and arrays as []any —
// and replaces every map value whose key matches SensitiveFieldKeys with
// RedactedPlaceholder, however deeply nested. It mutates and returns the
// same structure, so the only copy made is the one json.Unmarshal already
// made when decoding the original body.
func RedactJSONValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, vv := range val {
			if SensitiveFieldKeys[strings.ToLower(k)] {
				val[k] = RedactedPlaceholder
				continue
			}
			val[k] = RedactJSONValue(vv)
		}
		return val
	case []any:
		for i, vv := range val {
			val[i] = RedactJSONValue(vv)
		}
		return val
	default:
		return v
	}
}
