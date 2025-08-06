package helper

import (
	"strings"
)

// ContainsCaseInsensitive is the same as strings.Contains but comparing strings in a case-insensitive manner.
func ContainsCaseInsensitive(str string, pattern string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(pattern))
}

// TrimLeftOf trims everything left of the provided pattern in string str.
func TrimLeftOf(str string, pattern string) string {
	if idx := strings.Index(str, pattern); idx != -1 {
		return str[idx:]
	}
	return str
}
