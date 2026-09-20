package logger

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer secret-token")
	h.Set("Content-Type", "application/json")
	h.Add("Set-Cookie", "a=1")
	h.Add("Set-Cookie", "b=2")

	got := RedactHeaders(h)

	require.Equal(t, RedactedPlaceholder, got["authorization"])
	require.Equal(t, RedactedPlaceholder, got["set-cookie"])
	require.Equal(t, "application/json", got["content-type"])
}

func TestRedactJSONValue_NestedFields(t *testing.T) {
	input := map[string]any{
		"email": "john@example.com",
		"credentials": map[string]any{
			"password": "hunter2",
			"nested": []any{
				map[string]any{"access_token": "abc123", "keep": "me"},
			},
		},
	}

	got := RedactJSONValue(input).(map[string]any)

	require.Equal(t, "john@example.com", got["email"])
	creds := got["credentials"].(map[string]any)
	require.Equal(t, RedactedPlaceholder, creds["password"])
	nested := creds["nested"].([]any)[0].(map[string]any)
	require.Equal(t, RedactedPlaceholder, nested["access_token"])
	require.Equal(t, "me", nested["keep"])
}
