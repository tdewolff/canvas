package canvas

import (
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseTTF(t *testing.T) {
	b, err := ioutil.ReadFile("test/DejaVuSerif.ttf")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)
	test.That(t, font.sfnt.UnitsPerEm() == 2048)

	bounds, italicAngle, ascent, descent, capHeight, widths := font.pdfInfo()
	test.T(t, bounds, Rect{-769.53125, -1109.375, 2875, 1456.0546875})
	test.Float(t, italicAngle, 0)
	test.Float(t, ascent, 928.22265625)
	test.Float(t, descent, 235.83984375)
	test.Float(t, capHeight, -729.00390625)
	test.T(t, len(widths), 3528)

	indices := font.toIndices("test")
	test.T(t, len(indices), 4)
}

func TestParseOTF(t *testing.T) {
	b, err := ioutil.ReadFile("test/EBGaramond12-Regular.otf")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)
	test.That(t, font.sfnt.UnitsPerEm() == 1000)
}

func TestParseWOFF(t *testing.T) {
	b, err := ioutil.ReadFile("test/DejaVuSerif.woff")
	test.Error(t, err)

	font, err := parseFont("dejavu-serif", b)
	test.Error(t, err)
	test.That(t, font.sfnt.UnitsPerEm() == 2048)
}

func TestSubstitutes(t *testing.T) {
	b, err := ioutil.ReadFile("test/DejaVuSerif.ttf")
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
