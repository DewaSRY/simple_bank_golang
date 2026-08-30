package api

import (
	"fmt"
	"strings"
	"time"
)

// GenerateAccountNumber generates an account number with the format:
// ACT/DD-MMM-YYYY/NNN
//
// Example:
// GenerateAccountNumber(1) -> ACT/30-AUG-2026/001
func GenerateAccountNumber(sequence int) string {
	date := strings.ToUpper(time.Now().Format("02-Jan-2006"))

	return fmt.Sprintf(
		"ACT/%s/%03d",
		date,
		sequence,
	)
}
