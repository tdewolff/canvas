// +build !harfbuzz js

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
	glyphs := make([]Glyph, len([]rune(text)))
	i := 0
	var prevIndex uint16
	for cluster, r := range text {
		index := s.sfnt.GlyphIndex(r)
		glyphs[i].Text = string(r)
		glyphs[i].ID = index
		glyphs[i].Cluster = uint32(cluster)
		glyphs[i].XAdvance = int32(s.sfnt.GlyphAdvance(index))
		if 0 < i {
			glyphs[i-1].XAdvance += int32(s.sfnt.Kerning(prevIndex, index))
		}
		prevIndex = index
		i++
	}
	return glyphs
}

func ScriptItemizer(text string) []string {
	return []string{text}
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
