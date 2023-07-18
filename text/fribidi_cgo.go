//go:build fribidi && !js

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

// EmbeddingLevels returns the embedding levels for each rune of a mixed LTR/RTL string. A change in level means a change in direction.
func EmbeddingLevels(str []rune) []int {
	pbaseDir := C.FriBidiParType(C.FRIBIDI_PAR_ON) // neutral direction
	bidiTypes := make([]C.FriBidiCharType, len(str))
	bracketTypes := make([]C.FriBidiBracketType, len(str))
	embeddingLevels := make([]C.FriBidiLevel, len(str))

	C.fribidi_get_bidi_types(
		// input
		(*C.FriBidiChar)(unsafe.Pointer(&str[0])),
		C.FriBidiStrIndex(len(str)),

		// output
		&bidiTypes[0],
	)

	C.fribidi_get_bracket_types(
		// input
		(*C.FriBidiChar)(unsafe.Pointer(&str[0])),
		C.FriBidiStrIndex(len(str)),
		&bidiTypes[0],

		// output
		&bracketTypes[0],
	)

	_ = C.fribidi_get_par_embedding_levels_ex(
		// input
		&bidiTypes[0],
		&bracketTypes[0],
		C.FriBidiStrIndex(len(str)),
		&pbaseDir,

		// output
		&embeddingLevels[0],
	)

	levels := make([]int, len(str))
	for i, level := range embeddingLevels {
		levels[i] = int(level)
	}
	return levels
}
