// +build gofuzz
package fuzz

import "github.com/tdewolff/canvas"

func Fuzz(data []byte) int {
	ff := canvas.NewFontFamily("")
	_ = ff.LoadFont(data, canvas.FontRegular)
	return 1
}
