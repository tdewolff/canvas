// +build gofuzz
package fuzz

import "github.com/tdewolff/canvas/font"

func Fuzz(data []byte) int {
	_, _ = font.ParseWOFF2(data)
	return 1
}
