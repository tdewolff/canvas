package font

import (
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestSFNTDejaVuSerifTTF(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)

	test.T(t, sfnt.Head.UnitsPerEm, uint16(2048))
	test.T(t, sfnt.Hhea.Ascender, int16(1901))
	test.T(t, sfnt.Hhea.Descender, int16(-483))
	test.T(t, sfnt.OS2.SCapHeight, int16(1493)) // height of H glyph
	test.T(t, sfnt.Head.XMin, int16(-1576))
	test.T(t, sfnt.Head.YMin, int16(-710))
	test.T(t, sfnt.Head.XMax, int16(4312))
	test.T(t, sfnt.Head.YMax, int16(2272))

	id := sfnt.GlyphIndex(' ')
	contour, err := sfnt.GlyphContour(id)
	test.Error(t, err)
	test.T(t, contour.GlyphID, id)
	test.T(t, len(contour.XCoordinates), 0)
}

func TestSFNTSubset(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b, 0)
	test.Error(t, err)

	subset, glyphIDs := sfnt.Subset([]uint16{0, 3, 6, 36, 37, 38, 55, 131}) // .notdef, space, #, A, B, C, T, Á
	sfntSubset, err := ParseSFNT(subset, 0)
	test.Error(t, err)

	test.T(t, len(glyphIDs), 9) // Á is a composite glyph containing two simple glyphs: 36 and 3452
	test.T(t, glyphIDs[8], uint16(3452))

	test.T(t, sfntSubset.GlyphIndex('A'), uint16(3))
	test.T(t, sfntSubset.GlyphIndex('B'), uint16(4))
	test.T(t, sfntSubset.GlyphIndex('C'), uint16(5))

	//ioutil.WriteFile("out.otf", subset, 0644)
}
