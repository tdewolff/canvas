package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestTextLine(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontUnderline)

	text := NewTextLine(face, "test\nline", Left)
	test.T(t, len(text.fonts), 1)
	test.T(t, len(text.lines), 2)
	test.T(t, len(text.lines[0].spans), 1)
	test.Float(t, text.lines[0].spans[0].x, 0.0)
	test.Float(t, text.lines[0].y, 0.0)
	test.T(t, len(text.lines[1].spans), 1)
	test.Float(t, text.lines[1].spans[0].x, 0.0)
	test.Float(t, text.lines[1].y, face.Metrics().LineHeight)

	text = NewTextLine(face, "test\nline", Center)
	test.Float(t, text.lines[0].spans[0].x, -0.5*text.lines[0].spans[0].Width)
	test.Float(t, text.lines[1].spans[0].x, -0.5*text.lines[1].spans[0].Width)

	text = NewTextLine(face, "test\nline", Right)
	test.Float(t, text.lines[0].spans[0].x, -text.lines[0].spans[0].Width)
	test.Float(t, text.lines[1].spans[0].x, -text.lines[1].spans[0].Width)
}

func TestRichText(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	pt := ptPerMm * float64(family.fonts[FontRegular].Head.UnitsPerEm)
	face := family.Face(pt, Black, FontRegular, FontNormal) // line height is 13.96875

	rt := NewRichText(face)
	rt.Add(face, "ee. ee eeee") // e is 1212 wide, dot and space are 651 wide

	// test halign
	text := rt.ToText(6500.0, 5000.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 2)
	test.Float(t, text.lines[0].y, 1901)
	test.Float(t, text.lines[1].y, 4285)
	test.Float(t, text.lines[0].spans[0].x, 0.0)
	test.Float(t, text.lines[0].spans[0].Width, 3075)
	test.Float(t, text.lines[0].spans[1].x, 3726)
	test.Float(t, text.lines[0].spans[1].Width, 2424)
	test.Float(t, text.lines[1].spans[0].x, 0.0)
	test.Float(t, text.lines[1].spans[0].Width, 4848)

	text = rt.ToText(6500.0, 5000.0, Right, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].x, 6500-6150)
	test.Float(t, text.lines[0].spans[1].x, 6500-2424)
	test.Float(t, text.lines[1].spans[0].x, 6500-4848)

	text = rt.ToText(6500.0, 5000.0, Center, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].x, (6500-6150)/2)
	test.Float(t, text.lines[0].spans[1].x, (6500-6150)/2+3726)
	test.Float(t, text.lines[1].spans[0].x, (6500-4848)/2)

	text = rt.ToText(6500.0, 5000.0, Justify, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].x, 0.0)
	test.Float(t, text.lines[0].spans[1].x, 6500-2424)
	test.Float(t, text.lines[1].spans[0].x, 0.0)

	// test valign
	text = rt.ToText(6500.0, 5000.0, Left, Bottom, 0.0, 0.0)
	test.Float(t, text.lines[0].y, 5000-2867)
	test.Float(t, text.lines[1].y, 5000-483)

	text = rt.ToText(6500.0, 5000.0, Left, Center, 0.0, 0.0)
	test.Float(t, text.lines[0].y, (1901+(5000-1901-483*2))/2)
	test.Float(t, text.lines[1].y, (1901*2+483+(5000-483))/2)

	text = rt.ToText(6500.0, 5000.0, Left, Justify, 0.0, 0.0)
	test.Float(t, text.lines[0].y, 1901)
	test.Float(t, text.lines[1].y, 5000-483)

	// test wrapping
	text = rt.ToText(6000.0, 7500.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 3)
	test.Float(t, text.lines[0].spans[0].x, 0.0)
	test.Float(t, text.lines[1].spans[0].x, 0.0)
	test.Float(t, text.lines[2].spans[0].x, 0.0)

	// test special cases
	text = rt.ToText(6500.0, 2000.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 0)

	text = rt.ToText(0.0, 5000.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 1)
	test.T(t, len(text.lines[0].spans), 3)
	test.Float(t, text.lines[0].spans[0].x, 0.0)
	test.Float(t, text.lines[0].spans[1].x, 3726)
	test.Float(t, text.lines[0].spans[2].x, 6801)

	//rt = NewRichText()
	//text = rt.ToText(55.0, 50.0, Left, Top, 0.0, 0.0)
	//test.T(t, len(text.lines), 0)

	//rt = NewRichText()
	//rt.Add(face, "mm ")
	//rt.Add(face, " mm ")
	//rt.Add(face, " \n ")
	//rt.Add(face, "mmmm")
	//rt.Add(face, " mmmm ")
	//text = rt.ToText(75.0, 30.0, Justify, Top, 0.0, 0.0)
	//test.T(t, len(text.lines), 2)
	//test.Float(t, text.lines[0].spans[0].dx, 0.0)
	//test.Float(t, text.lines[0].spans[0].width, 75.0)
	//test.Float(t, text.lines[0].spans[0].GlyphSpacing, (75.0-22.75-3.8125-MaxWordSpacing*face.Metrics().XHeight-22.75)/4)
	//test.Float(t, text.lines[1].spans[0].dx, 0.0)
	//test.Float(t, text.lines[1].spans[0].width, 45.5) // cannot stretch in any reasonable way

	//rt = NewRichText()
	//rt.Add(face, "mm. ")
	//text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0) // wrap at space
	//test.T(t, len(text.lines), 1)

	//rt = NewRichText()
	//rt.Add(face, "mm\u200bmm \r\nmm")
	//text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0) // wrap at word break
	//test.T(t, len(text.lines), 3)
	//test.T(t, text.lines[0].spans[0].Text, "mm-")

	//rt = NewRichText()
	//rt.Add(face, "\u200bmm")
	//text = rt.ToText(20.0, 50.0, Left, Top, 0.0, 0.0) // wrap at space
	//test.T(t, len(text.lines), 1)
}

func TestRichText2(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	pt := ptPerMm * float64(family.fonts[FontRegular].Head.UnitsPerEm)
	face := family.Face(pt, Black, FontRegular, FontNormal) // line height is 13.96875

	rt := NewRichText(face)
	rt.Add(face, " a")
	rt.ToText(100.0, 100.0, Left, Top, 0.0, 0.0)
}

func TestTextBounds(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	pt := ptPerMm * float64(family.fonts[FontRegular].Head.UnitsPerEm)
	face8 := family.Face(pt, Black, FontRegular, FontNormal, FontUnderline)
	face12 := family.Face(1.5*pt, Black, FontRegular, FontNormal, FontUnderline)

	rt := NewRichText(face8)
	rt.Add(face8, "test")
	rt.Add(face12, "test")
	text := rt.ToText(4096.0, 4096.0, Left, Top, 0.0, 0.0)

	top, ascent, descent, bottom := text.lines[0].Heights(0.0)
	test.Float(t, top, 1901*1.5)
	test.Float(t, ascent, 1901*1.5)
	test.Float(t, descent, 483*1.5)
	test.Float(t, bottom, 483*1.5)

	ascent, descent = text.Heights()
	test.Float(t, ascent, 0.0)
	test.Float(t, descent, face12.Metrics().LineHeight)

	bounds := text.Bounds()
	test.Float(t, bounds.X, 0.0)
	test.Float(t, bounds.Y, -(1901+483)*1.5)
	test.Float(t, bounds.W, face8.TextWidth("test")+face12.TextWidth("test"))
	test.Float(t, bounds.H, (1901+483)*1.5)

	//bounds = text.OutlineBounds()
	//test.Float(t, bounds.X, 0.0)
	//test.Float(t, bounds.Y, -13.390625)
	//test.Float(t, bounds.W, face8.TextWidth("test")+face12.TextWidth("test"))
	//test.Float(t, bounds.H, 10.40625)
}

func TestTextBox(t *testing.T) {
	c := New(100, 100)
	ctx := NewContext(c)
	font, err := LoadFontFile("resources/DejaVuSerif.ttf", FontRegular)
	if err != nil {
		t.Fatal(err)
	}
	face := font.Face(12, Black)
	ctx.DrawText(0, 0, NewTextBox(face, "\ntext", 100, 100, Left, Top, 0, 0))
	ctx.DrawText(0, 0, NewTextBox(face, "text\n\ntext2", 100, 100, Left, Top, 0, 0))
}
