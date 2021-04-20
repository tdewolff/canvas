// +build gofuzz

package fuzz

import "github.com/tdewolff/canvas"

// Fuzz is a fuzz test.
func Fuzz(data []byte) int {
	ff := canvas.NewFontFamily("")
	_ = ff.LoadFont(data, canvas.FontRegular)
	return 1
}
