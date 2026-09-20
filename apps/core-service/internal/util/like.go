package util

import "strings"

// likeEscaper escapes the backslash itself first, then the two SQL LIKE
// wildcards, so a literal "%"/"_" in user input can't widen a search into an
// unintended pattern match. Pair with an SQL `LIKE ... ESCAPE '\'` clause.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLikePattern escapes s for safe interpolation into a LIKE pattern.
func EscapeLikePattern(s string) string {
	return likeEscaper.Replace(s)
}
