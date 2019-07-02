package canvas

import (
	"testing"
)

func TestParseWOFF(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("example/DejaVuSerif.woff", FontRegular)
}
