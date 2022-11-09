package text

import (
	"fmt"

	"github.com/tdewolff/canvas/font"
)

// Glyph is a shaped glyph for the given font and font size. It specified the glyph ID, the cluster ID, its X and Y advance and offset in font units, and its representation as text.
type Glyph struct {
	SFNT *font.SFNT
	Size float64
	Script
	Vertical bool // is false for Latin/Mongolian/etc in a vertical layout

	ID       uint16
	Cluster  uint32
	XAdvance int32
	YAdvance int32
	XOffset  int32
	YOffset  int32
	Text     string
}

func (g Glyph) String() string {
	return fmt.Sprintf("%s GID=%v Cluster=%v Adv=(%v,%v) Off=(%v,%v)", g.Text, g.ID, g.Cluster, g.XAdvance, g.YAdvance, g.XOffset, g.YOffset)
}

func (g Glyph) Rotation() Rotation {
	rot := NoRotation
	if !g.Vertical {
		rot = ScriptRotation(g.Script)
		if rot == NoRotation {
			rot = CW
		}
	}
	return rot
}

// TODO: implement Liang's (soft) hyphenation algorithm?

// IsParagraphSeparator returns true for paragraph separator runes.
func IsParagraphSeparator(r rune) bool {
	// line feed, vertical tab, form feed, carriage return, next line, line separator, paragraph separator
	return 0x0A <= r && r <= 0x0D || r == 0x85 || r == '\u2008' || r == '\u2009'
}

func IsSpacelessScript(script Script) bool {
	// missing: S'gaw Karen
	return script == Han || script == Hangul || script == Katakana || script == Khmer || script == Lao || script == PhagsPa || script == Brahmi || script == TaiTham || script == NewTaiLue || script == TaiLe || script == TaiViet || script == Thai || script == Tibetan || script == Myanmar
}

func IsVerticalScript(script Script) bool {
	return script == Bopomofo || script == EgyptianHieroglyphs || script == Hiragana || script == Katakana || script == Han || script == Hangul || script == MeroiticCursive || script == MeroiticHieroglyphs || script == Mongolian || script == Ogham || script == OldTurkic || script == PhagsPa || script == Yi
}

type Rotation float64

const (
	NoRotation Rotation = 0.0
	CW         Rotation = -90.0
	CCW        Rotation = 90.0
)

func ScriptRotation(script Script) Rotation {
	if script == Mongolian || script == PhagsPa {
		return CW
	} else if script == Ogham || script == OldTurkic {
		return CCW
	}
	return NoRotation
}
