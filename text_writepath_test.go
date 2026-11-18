package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestRichTextWritePathDepth(t *testing.T) {
	// A path that extends below its baseline (y < 0), like a formula with a
	// subscript, must keep that part, and with Baseline alignment it must
	// hang below the text baseline rather than sit on it.
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	face := family.Face(12.0, Black, FontRegular, FontNormal)

	p := Rectangle(4.0, 3.0).Translate(0.0, -1.0) // y from -1 to 2
	rt := NewRichText(face)
	rt.WriteFace(face, "a")
	rt.WritePath(p, Black, Baseline)
	txt := rt.ToText(0.0, 0.0, Left, Top, nil)

	found := false
	txt.WalkSpans(func(_, _ float64, span TextSpan) {
		for _, obj := range span.Objects {
			found = true
			test.Float(t, obj.Height, 3.0, "object keeps the part below the baseline")
			ascent, descent := obj.Heights(face)
			test.Float(t, ascent, 2.0, "ascent")
			test.Float(t, descent, 1.0, "descent below the baseline")
		}
	})
	test.T(t, found, true, "path object")
}
