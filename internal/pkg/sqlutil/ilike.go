package sqlutil

import "strings"

// ILIKEContains escapa %, _ y \ para búsquedas ILIKE ... ESCAPE '\'.
func ILIKEContains(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return "%" + s + "%"
}
