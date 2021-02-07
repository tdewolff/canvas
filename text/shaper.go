// +build !harfbuzz

package text

import (
	"github.com/tdewolff/canvas/font"
)

type Shaper struct {
	sfnt *font.SFNT
}

func NewShaper(b []byte, index int) (Shaper, error) {
	sfnt, err := font.ParseSFNT(b, index)
	if err != nil {
		return Shaper{}, err
	}
	return Shaper{
		sfnt: sfnt,
	}, nil
}

func NewShaperSFNT(sfnt *font.SFNT) (Shaper, error) {
	return Shaper{
		sfnt: sfnt,
	}, nil
}

func (s Shaper) Destroy() {
}

func (s Shaper) Shape(text string, ppem uint16, direction Direction, script Script, language string, features string, variations string) []Glyph {
	rs := []rune(text)
	glyphs := make([]Glyph, len(rs))
	var prevIndex uint16
	for i, r := range rs {
		index := s.sfnt.GlyphIndex(r)
		glyphs[i].ID = index
		glyphs[i].Cluster = uint32(i)
		glyphs[i].XAdvance = int32(s.sfnt.GlyphAdvance(index))
		if 0 < i {
			glyphs[i-1].XAdvance += int32(s.sfnt.Kerning(prevIndex, index))
		}
		prevIndex = index
	}
	return glyphs
}

type Direction int

const (
	DirectionInvalid Direction = iota
	LeftToRight
	RightToLeft
	TopToBottom
	BottomToTop
)

type Script uint32

const (
	ScriptInvalid Script = 0
)
