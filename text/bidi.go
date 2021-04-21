// +build !fribidi js

package text

var FriBidi = false

func Bidi(text string) (string, []int) {
	// linear map
	mapV2L := make([]int, len([]rune(text)))
	for pos := range mapV2L {
		mapV2L[pos] = pos
	}
	return text, mapV2L
}
