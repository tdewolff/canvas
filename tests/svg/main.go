// +build gofuzz
package fuzz

import "github.com/tdewolff/canvas"

func Fuzz(data []byte) int {
	_, _ = canvas.ParseSVG(string(data))
	return 1
}
