package font

import (
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseTTF(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.ttf")
	test.Error(t, err)

	sfnt, err := ParseFont(b)
	test.Error(t, err)
	test.T(t, sfnt.Head.UnitsPerEm, uint16(2048))
}

func TestParseOTF(t *testing.T) {
	b, err := ioutil.ReadFile("EBGaramond12-Regular.otf")
	test.Error(t, err)

	sfnt, err := ParseFont(b)
	test.Error(t, err)
	test.T(t, sfnt.Head.UnitsPerEm, uint16(1000))
}

func TestParseWOFF(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.woff")
	test.Error(t, err)

	sfnt, err := ParseFont(b)
	test.Error(t, err)
	test.T(t, sfnt.Head.UnitsPerEm, uint16(2048))
}

func TestParseWOFF2(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.woff2")
	test.Error(t, err)

	sfnt, err := ParseFont(b)
	test.Error(t, err)
	test.T(t, sfnt.Head.UnitsPerEm, uint16(2048))
}

func TestParseEOT(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.eot")
	test.Error(t, err)

	sfnt, err := ParseFont(b)
	test.Error(t, err)
	test.T(t, sfnt.Head.UnitsPerEm, uint16(2048))
}
