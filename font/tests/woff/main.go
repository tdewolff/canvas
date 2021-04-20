// +build gofuzz
package fuzz

import "github.com/tdewolff/canvas/font"

// Fuzz is a fuzz test.
func Fuzz(data []byte) int {
	_, _ = font.ParseWOFF(data)
	return 1
}
