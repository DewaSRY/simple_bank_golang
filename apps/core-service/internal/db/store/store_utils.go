package store

import (
	"fmt"
	"strings"
	"time"
)

// generateAccountNumber derives an account's human-facing number from its own
// DB id, which is immutable and never reused, so the result is guaranteed
// globally unique (a plain per-user or per-day counter is not).
func generateAccountNumber(id int64) string {
	date := strings.ToUpper(time.Now().Format("02-Jan-2006"))
	return fmt.Sprintf("ACT/%s/%03d", date, id)
}
