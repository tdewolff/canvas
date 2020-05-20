// +build gofuzz
package fuzz

import "github.com/tdewolff/canvas/font"

func Fuzz(data []byte) int {
	_, _ = font.ParseEOT(data)
	return 1
}
