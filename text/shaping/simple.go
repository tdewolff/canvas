// +build !harfbuzz

package shaping

import (
	"fmt"

	"github.com/tdewolff/canvas/font"
)

type Font struct {
	sfnt *font.SFNT
}

func NewFont(b []byte, index uint) (Font, error) {
	sfnt, err := font.ParseSFNT(b)
	if err != nil {
		return Font{}, err
	}
	return Font{
		sfnt: sfnt,
	}, nil
}

func (f Font) Destroy() {
}

func (f Font) Shape(text string, size float64, direction Direction, script Script) []Glyph {
	rs := []rune(text)
	glyphs := make([]Glyph, len(rs))
	var prevIndex uint16
	for i, r := range rs {
		index := f.sfnt.GlyphIndex(r)
		fmt.Printf("%X %s => %d\n", r, string(r), index)
		glyphs[i].ID = index
		glyphs[i].XAdvance = int32(f.sfnt.GlyphAdvance(index))
		if 0 < i {
			glyphs[i-1].XAdvance += int32(f.sfnt.Kerning(prevIndex, index))
		}
		prevIndex = index
	}
	return glyphs
}

type Direction int

const (
	LeftToRight Direction = 0
)

type Script uint32

const (
	Latin Script = 0
)
