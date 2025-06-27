package commands

import "strings"

// titleCase provides a simple title case function to replace deprecated strings.Title
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
