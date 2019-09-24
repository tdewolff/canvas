package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestTextLine(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("./test/DejaVuSerif.ttf", FontRegular)
	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontUnderline)

	text := NewTextLine(face, "test\nline", Left)
	test.T(t, len(text.fonts), 1)
	test.T(t, len(text.lines), 2)
	test.T(t, len(text.lines[0].spans), 1)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.T(t, len(text.lines[0].decos), 1)
	test.Float(t, text.lines[0].y, 0.0)
	test.T(t, len(text.lines[1].spans), 1)
	test.Float(t, text.lines[1].spans[0].dx, 0.0)
	test.T(t, len(text.lines[1].decos), 1)
	test.Float(t, text.lines[1].y, -face.Metrics().LineHeight)

	text = NewTextLine(face, "test\nline", Center)
	test.Float(t, text.lines[0].spans[0].dx, -0.5*text.lines[0].spans[0].width)
	test.Float(t, text.lines[1].spans[0].dx, -0.5*text.lines[1].spans[0].width)

	text = NewTextLine(face, "test\nline", Right)
	test.Float(t, text.lines[0].spans[0].dx, -text.lines[0].spans[0].width)
	test.Float(t, text.lines[1].spans[0].dx, -text.lines[1].spans[0].width)
}

func TestRichText(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("./test/DejaVuSerif.ttf", FontRegular)
	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal) // line height is 13.96875

	rt := NewRichText()
	rt.Add(face, "mm. mm mmmm") // mm is 22.75 wide, mmmm is 45.5 wide, dot and space are 3.8125 wide

	// test halign
	text := rt.ToText(55.0, 50.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 2)
	test.Float(t, text.lines[0].y, -11.140625)
	test.Float(t, text.lines[1].y, -25.109375)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.Float(t, text.lines[0].spans[0].width, 22.75+7.625)
	test.Float(t, text.lines[0].spans[1].dx, 30.375)
	test.Float(t, text.lines[0].spans[1].width, 22.75)
	test.Float(t, text.lines[1].spans[0].dx, 0.0)
	test.Float(t, text.lines[1].spans[0].width, 45.5)

	text = rt.ToText(55.0, 50.0, Right, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].dx, 55.0-53.125)
	test.Float(t, text.lines[0].spans[1].dx, 55.0-22.75)
	test.Float(t, text.lines[1].spans[0].dx, 55.0-45.5)

	text = rt.ToText(55.0, 50.0, Center, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].dx, (55.0-53.125)/2.0)
	test.Float(t, text.lines[0].spans[1].dx, (55.0-53.125)/2.0+22.75+7.625)
	test.Float(t, text.lines[1].spans[0].dx, (55.0-45.5)/2.0)

	text = rt.ToText(55.0, 50.0, Justify, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.Float(t, text.lines[0].spans[0].width, 32.25) // space is stretched
	test.Float(t, text.lines[0].spans[1].dx, 55.0-22.75)
	test.Float(t, text.lines[0].spans[1].width, 22.75)
	test.Float(t, text.lines[1].spans[0].dx, 0.0)
	test.Float(t, text.lines[1].spans[0].width, 45.5) // last row does not justify

	// test valign
	text = rt.ToText(55.0, 50.0, Left, Bottom, 0.0, 0.0)
	test.Float(t, text.lines[0].y, -33.203125)
	test.Float(t, text.lines[1].y, -47.171875)

	text = rt.ToText(55.0, 50.0, Left, Center, 0.0, 0.0)
	test.Float(t, text.lines[0].y, -22.171875)
	test.Float(t, text.lines[1].y, -36.140625)

	text = rt.ToText(55.0, 50.0, Left, Justify, 0.0, 0.0)
	test.Float(t, text.lines[0].y, -11.140625)
	test.Float(t, text.lines[1].y, -47.171875)

	// test wrapping
	text = rt.ToText(50.0, 50.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 3)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.Float(t, text.lines[1].spans[0].dx, 0.0)
	test.Float(t, text.lines[2].spans[0].dx, 0.0)

	text = rt.ToText(27.0, 50.0, Left, Top, 0.0, 0.0) // wrap in space
	test.T(t, len(text.lines), 3)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.Float(t, text.lines[0].spans[0].width, 26.5625) // space removed
	test.Float(t, text.lines[1].spans[0].dx, 0.0)
	test.Float(t, text.lines[1].spans[0].width, 22.75)
	test.Float(t, text.lines[2].spans[0].dx, 0.0)
	test.Float(t, text.lines[2].spans[0].width, 45.5)

	// test special cases
	text = rt.ToText(55.0, 10.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 0)

	text = rt.ToText(0.0, 50.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 1)
	test.T(t, len(text.lines[0].spans), 2)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.Float(t, text.lines[0].spans[1].dx, 30.375)

	rt = NewRichText()
	text = rt.ToText(55.0, 50.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 0)

	rt = NewRichText()
	rt.Add(face, "mm ")
	rt.Add(face, " mm ")
	rt.Add(face, " \n ")
	rt.Add(face, "mmmm")
	rt.Add(face, " mmmm ")
	text = rt.ToText(75.0, 30.0, Justify, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 2)
	test.Float(t, text.lines[0].spans[0].dx, 0.0)
	test.Float(t, text.lines[0].spans[0].width, 75.0)
	test.Float(t, text.lines[0].spans[0].glyphSpacing, (75.0-22.75-3.8125-MaxWordSpacing*face.Metrics().XHeight-22.75)/4)
	test.Float(t, text.lines[1].spans[0].dx, 0.0)
	test.Float(t, text.lines[1].spans[0].width, 45.5) // cannot stretch in any reasonable way

	rt = NewRichText()
	rt.Add(face, "mm. ")
	text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0) // wrap at space
	test.T(t, len(text.lines), 1)

	rt = NewRichText()
	rt.Add(face, "mm\u200bmm \r\nmm")
	text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0) // wrap at word break
	test.T(t, len(text.lines), 3)
	test.T(t, text.lines[0].spans[0].text, "mm-")

	rt = NewRichText()
	rt.Add(face, "\u200bmm")
	text = rt.ToText(20.0, 50.0, Left, Top, 0.0, 0.0) // wrap at space
	test.T(t, len(text.lines), 1)
}

func TestTextLineHeights(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("./test/DejaVuSerif.ttf", FontRegular)
	face8 := family.Face(8.0*ptPerMm, Black, FontRegular, FontNormal, FontUnderline)
	face12 := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontUnderline)

	rt := NewRichText()
	rt.Add(face8, "test")
	rt.Add(face12, "test")
	text := rt.ToText(100.0, 100.0, Left, Top, 0.0, 0.0)

	top, ascent, descent, bottom := text.lines[0].Heights()
	test.Float(t, top, 11.140625)
	test.Float(t, ascent, 11.140625)
	test.Float(t, descent, 2.828125)
	test.Float(t, bottom, 2.828125)
}
