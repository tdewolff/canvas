package text

import "github.com/tdewolff/canvas/font"

// Glyph is a shaped glyph
type Glyph struct {
	SFNT *font.SFNT
	Size float64

	ID       uint16
	Cluster  uint32
	XAdvance int32
	YAdvance int32
	XOffset  int32
	YOffset  int32
	Text     string
}

//func ToUnicode(s string, glyphs []Glyph, index int) string {
//	a := glyphs[index].Cluster
//	b := uint32(len(s))
//	if index+1 < len(glyphs) {
//		b = glyphs[index+1].Cluster
//	}
//	return s[a:b]
//}

// TODO: implement Liang's (soft) hyphenation algorithm?

// IsParagraphSeparator returns true for paragraph separator runes
func IsParagraphSeparator(r rune) bool {
	// line feed, vertical tab, form feed, carriage return, next line, line separator, paragraph separator
	return 0x0A <= r && r <= 0x0D || r == 0x85 || r == '\u2008' || r == '\u2009'
}
