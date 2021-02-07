// +build !fribidi

package text

var FriBidi = false

func Bidi(text string) string {
	return text
}

func BidiEmbeddings(text string) []string {
	return []string{text}
}
