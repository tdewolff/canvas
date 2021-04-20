// +build gofuzz

package fuzz

import "github.com/tdewolff/canvas"

// Fuzz is a fuzz test.
func Fuzz(data []byte) int {
	_, _ = canvas.ParseSVG(string(data))
	return 1
}
