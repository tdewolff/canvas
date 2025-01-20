package canvas

import (
	"strings"
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
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[0].y, 0.0)
	test.T(t, len(text.lines[1].spans), 1)
	test.Float(t, text.lines[1].spans[0].X, 0.0)
	test.Float(t, text.lines[1].y, face.Metrics().LineHeight)

	text = NewTextLine(face, "test\nline", Center)
	test.Float(t, text.lines[0].spans[0].X, -0.5*text.lines[0].spans[0].Width)
	test.Float(t, text.lines[1].spans[0].X, -0.5*text.lines[1].spans[0].Width)

	text = NewTextLine(face, "test\nline", Right)
	test.Float(t, text.lines[0].spans[0].X, -text.lines[0].spans[0].Width)
	test.Float(t, text.lines[1].spans[0].X, -text.lines[1].spans[0].Width)
}

func TestRichTextPositions(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	pt := ptPerMm * float64(family.fonts[FontRegular].Head.UnitsPerEm)
	face := family.Face(pt, Black, FontRegular, FontNormal) // line height is 13.96875

	rt := NewRichText(face)
	rt.WriteString("ee. ee eeee") // e is 1212 wide, dot and space are 651 wide

	// test halign
	text := rt.ToText(6500.0, 5000.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 2)
	test.Float(t, text.lines[0].y, 1901)
	test.Float(t, text.lines[1].y, 4285)
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[0].spans[0].Width, 6150)
	test.Float(t, text.lines[1].spans[0].X, 0.0)
	test.Float(t, text.lines[1].spans[0].Width, 4848)

	text = rt.ToText(6500.0, 5000.0, Right, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].X, 6500-6150)
	test.Float(t, text.lines[1].spans[0].X, 6500-4848)

	text = rt.ToText(6500.0, 5000.0, Center, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].X, (6500-6150)/2)
	test.Float(t, text.lines[1].spans[0].X, (6500-4848)/2)

	text = rt.ToText(6500.0, 5000.0, Justify, Top, 0.0, 0.0)
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[1].spans[0].X, 0.0)

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
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[1].spans[0].X, 0.0)
	test.Float(t, text.lines[2].spans[0].X, 0.0)

	// test special cases
	text = rt.ToText(6500.0, 2000.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 0)

	text = rt.ToText(0.0, 5000.0, Left, Top, 0.0, 0.0)
	test.T(t, len(text.lines), 1)
	test.T(t, len(text.lines[0].spans), 1)
	test.Float(t, text.lines[0].spans[0].X, 0.0)

	//rt = NewRichText(face)
	//text = rt.ToText(55.0, 50.0, Left, Top, 0.0, 0.0)
	//test.T(t, len(text.lines), 0)

	//rt = NewRichText(face)
	//rt.WriteString("mm ")
	//rt.WriteString(" mm ")
	//rt.WriteString(" \n ")
	//rt.WriteString("mmmm")
	//rt.WriteString(" mmmm ")
	//text = rt.ToText(75.0, 30.0, Justify, Top, 0.0, 0.0)
	//test.T(t, len(text.lines), 2)
	//test.Float(t, text.lines[0].spans[0].dx, 0.0)
	//test.Float(t, text.lines[0].spans[0].width, 75.0)
	//test.Float(t, text.lines[0].spans[0].GlyphSpacing, (75.0-22.75-3.8125-MaxWordSpacing*face.Metrics().XHeight-22.75)/4)
	//test.Float(t, text.lines[1].spans[0].dx, 0.0)
	//test.Float(t, text.lines[1].spans[0].width, 45.5) // cannot stretch in any reasonable way

	//rt = NewRichText(face)
	//rt.WriteString("mm. ")
	//text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0) // wrap at space
	//test.T(t, len(text.lines), 1)

	//rt = NewRichText(face)
	//rt.WriteString("mm\u200bmm \r\nmm")
	//text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0) // wrap at word break
	//test.T(t, len(text.lines), 3)
	//test.T(t, text.lines[0].spans[0].Text, "mm-")

	//rt = NewRichText(face)
	//rt.WriteString("\u200bmm")
	//text = rt.ToText(20.0, 50.0, Left, Top, 0.0, 0.0) // wrap at space
	//test.T(t, len(text.lines), 1)

	rt = NewRichText(face)
	rt.WriteString("\uFFFC")
	rt.ToText(10.0, 10.0, Left, Top, 0.0, 0.0)
}

func TestRichText(t *testing.T) {
	familyLatin := NewFontFamily("dejavu-serif")
	if err := familyLatin.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	faceLatin := familyLatin.Face(12.0, Black, FontRegular, FontNormal) // line height is 13.96875

	familyCJK := NewFontFamily("unifont")
	if err := familyCJK.LoadFontFile("resources/unifont-13.0.05.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	faceCJK := familyCJK.Face(12.0, Black, FontRegular, FontNormal) // line height is 13.96875

	var tests = []struct {
		face  *FontFace
		align TextAlign
		in    string
		out   string
	}{
		{faceLatin, Left, " a", " a"},
		{
			faceLatin, Left,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			"Lorem ipsum dolor sit amet, consectetur\nadipiscing elit, sed do eiusmod tempor\nincididunt ut labore et dolore magna aliqua.\nUt enim ad minim veniam, quis nostrud\nexercitation ullamco laboris nisi ut aliquip ex\nea commodo consequat.",
		},
		{
			faceLatin, Left,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do\neiusmod tempor incididunt ut labore et dolore magna aliqua.\nUt enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			"Lorem ipsum dolor sit amet, consectetur\nadipiscing elit, sed do\neiusmod tempor incididunt ut labore et dolore\nmagna aliqua.\nUt enim ad minim veniam, quis nostrud\nexercitation ullamco laboris nisi ut aliquip ex\nea commodo consequat.",
		},
		{
			faceLatin, Center,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			"Lorem ipsum dolor sit amet, consectetur\nadipiscing elit, sed do eiusmod tempor\nincididunt ut labore et dolore magna aliqua.\nUt enim ad minim veniam, quis nostrud\nexercitation ullamco laboris nisi ut aliquip ex\nea commodo consequat.",
		},
		{
			faceCJK, Left,
			"执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出出执行送出执行送出执行送出执行送出出执行送出执行送出执行送出执行送出出执行送出执行送出执行送出执行送出",
			"执行送出执行送出执行送出执行送出执行送出执行送\n出执行送出执行送出执行送出执行送出执行送出执行\n送出执行送出出执行送出执行送出执行送出执行送出\n出执行送出执行送出执行送出执行送出出执行送出执\n行送出执行送出执行送出",
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			rt := NewRichText(tt.face)
			rt.WriteString(tt.in)
			text := rt.ToText(100.0, 100.0, tt.align, Top, 0.0, 0.0)
			var lines []string
			for _, line := range text.lines {
				var spans []string
				for _, span := range line.spans {
					spans = append(spans, span.Text)
				}
				lines = append(lines, strings.Join(spans, " "))
			}
			test.T(t, strings.Join(lines, "\n"), tt.out)
		})
	}
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
	rt.WriteString("test")
	rt.WriteFace(face12, "test")
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
	test.Float(t, bounds.X0, 0.0)
	test.Float(t, bounds.Y0, -(1901+483)*1.5)
	test.Float(t, bounds.W(), face8.TextWidth("test")+face12.TextWidth("test"))
	test.Float(t, bounds.H(), (1901+483)*1.5)

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
