// +build fribidi,!js

package text

//#cgo CPPFLAGS: -I/usr/include/fribidi
//#cgo LDFLAGS: -L/usr/lib -lfribidi
/*
#include <fribidi.h>
*/
import "C"
import (
	"unsafe"
)

// Bidi maps the string from its logical order to the visual order to correctly display mixed LTR/RTL text. It returns a mapping of rune positions.
func Bidi(text string) (string, []int) {
	str := []rune(text)
	pbaseDir := C.FriBidiParType(C.FRIBIDI_PAR_ON) // neutral direction
	visualStr := make([]rune, len(str))
	positionsL2V := make([]C.FriBidiStrIndex, len(str))
	positionsV2L := make([]C.FriBidiStrIndex, len(str))
	embeddingLevels := make([]C.FriBidiLevel, len(str))
	C.fribidi_log2vis(
		// input
		(*C.FriBidiChar)(unsafe.Pointer(&str[0])),
		C.FriBidiStrIndex(len(str)),
		&pbaseDir,

		// output
		(*C.FriBidiChar)(unsafe.Pointer(&visualStr[0])),
		&positionsL2V[0],
		&positionsV2L[0],
		&embeddingLevels[0],
	)
	text = string(visualStr)

	mapV2L := make([]int, len(positionsV2L))
	for i, pos := range positionsV2L {
		mapV2L[i] = int(pos)
	}
	return text, mapV2L
}
