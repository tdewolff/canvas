// +build !fribidi

package text

var FriBidi = false

func Bidi(text string) string {
	return text
}
