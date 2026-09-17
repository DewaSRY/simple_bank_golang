package logger

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	config "github.com/DewaSRY/core-service/internal/config"
)

func TestNew_PrettyJSONIndentsRecordsAcrossWithAttrs(t *testing.T) {
	var buf strings.Builder
	handler := newPrettyHandler(&buf, nil)
	log := slog.New(handler).With("request_id", "req-1")
	log.Info("request completed", "status", 200)

	out := buf.String()
	require.Contains(t, out, "\n  \"", "pretty output should be indented across multiple lines")
	require.Contains(t, out, `"request_id": "req-1"`)
	require.Contains(t, out, `"status": 200`)
}

func TestNew_CompactJSONByDefault(t *testing.T) {
	log := New(config.Config{LogFormat: "json"})
	require.NotNil(t, log)
}
