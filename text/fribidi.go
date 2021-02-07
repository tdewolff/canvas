// +build fribidi

package text

//#cgo CPPFLAGS: -I/usr/include/fribidi
//#cgo LDFLAGS: -L/usr/lib -lfribidi
/*
#include <fribidi.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

var FriBidi = true

func Bidi(text string) string {
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
	return string(visualStr)
}

func BidiEmbeddings(text string) []string {
	str := []rune(text)
	bidiTypes := make([]C.FriBidiCharType, len(str))
	C.fribidi_get_bidi_types(
		// input
		(*C.FriBidiChar)(unsafe.Pointer(&str[0])),
		C.FriBidiStrIndex(len(str)),

		// output
		&bidiTypes[0],
	)

	bracketTypes := make([]C.FriBidiBracketType, len(str))
	C.fribidi_get_bracket_types(
		// input
		(*C.FriBidiChar)(unsafe.Pointer(&str[0])),
		C.FriBidiStrIndex(len(str)),
		&bidiTypes[0],

		// output
		&bracketTypes[0],
	)

	//pbaseDir := C.FriBidiParType(C.FRIBIDI_PAR_ON) // neutral direction
	pbaseDir := C.fribidi_get_par_direction(
		&bidiTypes[0],
		C.FriBidiStrIndex(len(str)),
	)

	embeddingLevels := make([]C.FriBidiLevel, len(str))
	C.fribidi_get_par_embedding_levels_ex(
		// input
		&bidiTypes[0],
		&bracketTypes[0],
		C.FriBidiStrIndex(len(str)),
		&pbaseDir,

		// output
		&embeddingLevels[0],
	)

	fmt.Println(text)
	fmt.Println(embeddingLevels)

	ss := []string{}
	if len(embeddingLevels) == 0 {
		ss = append(ss, text)
		return ss
	}
	i := 0
	curLevel := embeddingLevels[0]
	for j, level := range embeddingLevels {
		if level != curLevel {
			ss = append(ss, string(str[i:j]))
			curLevel = level
			i = j
		}
	}
	ss = append(ss, string(str[i:]))
	return ss
}
