// +build gofuzz
package fuzz

import "github.com/tdewolff/canvas/font"

func Fuzz(data []byte) int {
	_, _ = font.ParseWOFF(data)
	return 1
}
