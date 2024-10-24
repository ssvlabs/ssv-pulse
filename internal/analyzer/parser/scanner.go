package parser

import (
	"bufio"
	"os"
)

func NewScanner(file *os.File) *bufio.Scanner {
	scanner := bufio.NewScanner(file)
	const maxCapacity = 1024 * 1024
	scanner.Buffer(make([]byte, maxCapacity), maxCapacity)

	return scanner
}
