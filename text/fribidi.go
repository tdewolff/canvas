// +build !fribidi js

package text

import "github.com/benoitkugler/textlayout/fribidi"

// Bidi maps the string from its logical order to the visual order to correctly display mixed LTR/RTL text. It returns a mapping of rune positions.
func Bidi(text string) (string, []int) {
	str := []rune(text)
	pbase := fribidi.CharType(fribidi.ON)
	vis, _ := fribidi.LogicalToVisual(fribidi.DefaultFlags, str, &pbase)
	text = string(vis.Str)

	mapV2L := make([]int, len(vis.VisualToLogical))
	for i, pos := range vis.VisualToLogical {
		mapV2L[i] = int(pos)
	}
	return text, mapV2L
}
