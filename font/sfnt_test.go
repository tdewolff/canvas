package font

import (
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestSFNTDejaVuSerifTTF(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseSFNT(b)
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

	sfnt, err := ParseSFNT(b)
	test.Error(t, err)

	subset, glyphIDs := sfnt.Subset([]uint16{0, 3, 6, 36, 55, 131}) // .notdef, space, #, A, T, Á
	_, err = ParseSFNT(subset)
	test.Error(t, err)

	test.T(t, len(glyphIDs), 7) // Á is a composite glyph containing two simple glyphs: 36 and 3452
	test.T(t, glyphIDs[6], uint16(3452))

	ioutil.WriteFile("out.otf", subset, 0644)
}
