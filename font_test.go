package canvas

import (
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

// TODO: move to font directory
func TestParseTTF(t *testing.T) {
	b, err := ioutil.ReadFile("font/DejaVuSerif.ttf")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)
	test.That(t, font.sfnt.UnitsPerEm() == 2048)

	units := font.UnitsPerEm()

	test.T(t, font.Bounds(units), Rect{-1576, -2272, 5888, 2982})
	test.Float(t, font.ItalicAngle(), 0)

	metrics := font.Metrics(units)
	test.Float(t, metrics.Ascent*1000/units, 928.22265625)
	test.Float(t, metrics.Descent*1000/units, 235.83984375)
	test.Float(t, metrics.CapHeight*1000/units, -729.00390625)
	test.T(t, len(font.Widths(units)), 3528)

	indices := font.IndicesOf("test")
	test.T(t, len(indices), 4)
}

func TestParseOTF(t *testing.T) {
	b, err := ioutil.ReadFile("font/EBGaramond12-Regular.otf")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)
	test.That(t, font.sfnt.UnitsPerEm() == 1000)
}

func TestParseWOFF(t *testing.T) {
	b, err := ioutil.ReadFile("font/DejaVuSerif.woff")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)
	test.That(t, font.sfnt.UnitsPerEm() == 2048)
}

func TestSubstitutes(t *testing.T) {
	b, err := ioutil.ReadFile("font/DejaVuSerif.ttf")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)

	font.Use(CommonLigatures)

	test.String(t, font.substituteLigatures("fi fl ffi ffl"), "ﬁ ﬂ ﬃ ﬄ")
	s, inSingleQuote, inDoubleQuote := font.substituteTypography(`... . . . --- -- (c) (r) (tm) 1/2 1/4 3/4 +/- '' ""`, false, false)
	test.String(t, s, "… … — – © ® ™ ½ ¼ ¾ ± ‘’ “”")
	test.That(t, !inSingleQuote)
	test.That(t, !inDoubleQuote)
}
