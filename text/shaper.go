// +build !harfbuzz js

package text

import (
	"github.com/tdewolff/canvas/font"
)

// Shaper is a text shaper formatting a string in properly positioned glyphs.
type Shaper struct {
	sfnt *font.SFNT
}

// NewShaper returns a new text shaper.
func NewShaper(b []byte, index int) (Shaper, error) {
	sfnt, err := font.ParseSFNT(b, index)
	if err != nil {
		return Shaper{}, err
	}
	return Shaper{
		sfnt: sfnt,
	}, nil
}

// NewShaperSFNT returns a new text shaper using a SFNT structure.
func NewShaperSFNT(sfnt *font.SFNT) (Shaper, error) {
	return Shaper{
		sfnt: sfnt,
	}, nil
}

// Destroy destroys the allocated C memory.
func (s Shaper) Destroy() {
}

// Shape shapes the string for a given direction, script, and language.
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

// ScriptItemizer divides the string in parts for each different script.
func ScriptItemizer(text string) []string {
	return []string{text}
}

// Direction is the text direction.
type Direction int

// see Direction
const (
	DirectionInvalid Direction = iota
	LeftToRight
	RightToLeft
	TopToBottom
	BottomToTop
)

// Script is the script.
type Script uint32

// see Script
const (
	ScriptInvalid Script = 0
)
