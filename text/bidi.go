// +build !fribidi js

package text

var usesFriBidi = false

// Bidi maps the string from its logical order to the visual order to correctly display mixed LTR/RTL text. It returns a mapping of rune positions.
func Bidi(text string) (string, []int) {
	// linear map
	mapV2L := make([]int, len([]rune(text)))
	for pos := range mapV2L {
		mapV2L[pos] = pos
	}
	return text, mapV2L
}
