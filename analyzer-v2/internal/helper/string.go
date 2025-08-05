package helper

import (
	"strings"
)

// TrimLeftOf trims everything left of the provided pattern in string str.
func TrimLeftOf(str string, pattern string) string {
	if idx := strings.Index(str, pattern); idx != -1 {
		return str[idx:]
	}
	return str
}
