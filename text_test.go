package canvas

import (
	"fmt"
	"testing"

	"github.com/tdewolff/canvas/text"
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
	text := rt.ToText(6500.0, 5000.0, Left, Top, nil)
	test.T(t, len(text.lines), 2)
	test.Float(t, text.lines[0].y, 1901)
	test.Float(t, text.lines[1].y, 4285)
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[0].spans[0].Width, 6150)
	test.Float(t, text.lines[1].spans[0].X, 0.0)
	test.Float(t, text.lines[1].spans[0].Width, 4848)

	text = rt.ToText(6500.0, 5000.0, Right, Top, nil)
	test.Float(t, text.lines[0].spans[0].X, 6500-6150)
	test.Float(t, text.lines[1].spans[0].X, 6500-4848)

	text = rt.ToText(6500.0, 5000.0, Center, Top, nil)
	test.Float(t, text.lines[0].spans[0].X, (6500-6150)/2)
	test.Float(t, text.lines[1].spans[0].X, (6500-4848)/2)

	text = rt.ToText(6500.0, 5000.0, Justify, Top, nil)
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[1].spans[0].X, 0.0)

	// test valign
	text = rt.ToText(6500.0, 5000.0, Left, Bottom, nil)
	test.Float(t, text.lines[0].y, 5000-2867)
	test.Float(t, text.lines[1].y, 5000-483)

	text = rt.ToText(6500.0, 5000.0, Left, Center, nil)
	test.Float(t, text.lines[0].y, (1901+(5000-1901-483*2))/2)
	test.Float(t, text.lines[1].y, (1901*2+483+(5000-483))/2)

	text = rt.ToText(6500.0, 5000.0, Left, Justify, nil)
	test.Float(t, text.lines[0].y, 1901)
	test.Float(t, text.lines[1].y, 5000-483)

	// test wrapping
	text = rt.ToText(6000.0, 7500.0, Left, Top, nil)
	test.T(t, len(text.lines), 3)
	test.Float(t, text.lines[0].spans[0].X, 0.0)
	test.Float(t, text.lines[1].spans[0].X, 0.0)
	test.Float(t, text.lines[2].spans[0].X, 0.0)

	// test special cases
	text = rt.ToText(6500.0, 2000.0, Left, Top, nil)
	test.T(t, len(text.lines), 0)

	text = rt.ToText(0.0, 5000.0, Left, Top, nil)
	test.T(t, len(text.lines), 1)
	test.T(t, len(text.lines[0].spans), 1)
	test.Float(t, text.lines[0].spans[0].X, 0.0)

	//rt = NewRichText(face)
	//text = rt.ToText(55.0, 50.0, Left, Top, 0.0, 0.0, KnuthLinebreaker{})
	//test.T(t, len(text.lines), 0)

	//rt = NewRichText(face)
	//rt.WriteString("mm ")
	//rt.WriteString(" mm ")
	//rt.WriteString(" \n ")
	//rt.WriteString("mmmm")
	//rt.WriteString(" mmmm ")
	//text = rt.ToText(75.0, 30.0, Justify, Top, 0.0, 0.0, KnuthLinebreaker{})
	//test.T(t, len(text.lines), 2)
	//test.Float(t, text.lines[0].spans[0].dx, 0.0)
	//test.Float(t, text.lines[0].spans[0].width, 75.0)
	//test.Float(t, text.lines[0].spans[0].GlyphSpacing, (75.0-22.75-3.8125-MaxWordSpacing*face.Metrics().XHeight-22.75)/4)
	//test.Float(t, text.lines[1].spans[0].dx, 0.0)
	//test.Float(t, text.lines[1].spans[0].width, 45.5) // cannot stretch in any reasonable way

	//rt = NewRichText(face)
	//rt.WriteString("mm. ")
	//text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0, KnuthLinebreaker{}) // wrap at space
	//test.T(t, len(text.lines), 1)

	//rt = NewRichText(face)
	//rt.WriteString("mm\u200bmm \r\nmm")
	//text = rt.ToText(30.0, 50.0, Left, Top, 0.0, 0.0, KnuthLinebreaker{}) // wrap at word break
	//test.T(t, len(text.lines), 3)
	//test.T(t, text.lines[0].spans[0].Text, "mm-")

	//rt = NewRichText(face)
	//rt.WriteString("\u200bmm")
	//text = rt.ToText(20.0, 50.0, Left, Top, 0.0, 0.0, KnuthLinebreaker{}) // wrap at space
	//test.T(t, len(text.lines), 1)

	rt = NewRichText(face)
	rt.WriteString("\uFFFC")
	rt.ToText(10.0, 10.0, Left, Top, nil)
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
		str   string
		spans [][]string
	}{
		{faceLatin, Left, " a", [][]string{{" a"}}},
		{
			faceLatin, Left,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			[][]string{{"Lorem ipsum dolor sit amet, consectetur "}, {"adipiscing elit, sed do eiusmod tempor "}, {"incididunt ut labore et dolore magna aliqua. "}, {"Ut enim ad minim veniam, quis nostrud "}, {"exercitation ullamco laboris nisi ut aliquip ex "}, {"ea commodo consequat."}},
		},
		{
			faceLatin, Left,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do\neiusmod tempor incididunt ut labore et dolore magna aliqua.\nUt enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			[][]string{{"Lorem ipsum dolor sit amet, consectetur "}, {"adipiscing elit, sed do\n"}, {"eiusmod tempor incididunt ut labore et dolore "}, {"magna aliqua.\n"}, {"Ut enim ad minim veniam, quis nostrud "}, {"exercitation ullamco laboris nisi ut aliquip ex "}, {"ea commodo consequat."}},
		},
		{
			faceCJK, Left,
			"执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出执行送出",
			[][]string{{"执行送出执行送出执行送出执行送出执行送出执行送"}, {"出执行送出执行送出执行送出执行送出执行送出执行"}, {"送出执行送出执行送出执行送出执行送出执行送出执"}, {"行送出执行送出执行送出执行送出执行送出执行送出"}, {"执行送出执行送出"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			rt := NewRichText(tt.face)
			rt.WriteString(tt.str)
			text := rt.ToText(100.0, 100.0, tt.align, Top, nil)
			var lines [][]string
			for _, line := range text.lines {
				var spans []string
				for _, span := range line.spans {
					spans = append(spans, span.Text)
				}
				lines = append(lines, spans)
			}
			test.T(t, lines, tt.spans)
		})
	}
}

func TestTextLinebreaking(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	face := family.Face(40.0, Black, FontRegular, FontNormal)

	indent := 1.0
	width := 352.0
	str := " L’POUR VOTRE SANTÉ, MANGEZ AU MOINS 5 FRUITS ET LEGUMES PAR JOUR. WWW.MANGERBOUGER.FR\nOffre personnelle réservée aux porteurs de la carte de fidélité. \n ©2023, The Coca-Cola Company. Coca-Cola, la Bouteille Contour et Savoure l'instant sont des marques déposées de The Coca-Cola Company. Coca-Cola Europacific Partners France SAS Issy-les-Moulineaux RCS 343 688 016 Nanterre. "

	var tests = []struct {
		lb        text.Linebreaker
		align     text.Align
		result    string
		overflows bool
	}{
		{GreedyLinebreaker{}, text.Left, " L’POUR VOTRE SANTÉ, MANGEZ AU MOINS 5 \nFRUITS ET LEGUMES PAR JOUR. \nWWW.MANGERBOUGER.FR\nOffre personnelle réservée aux porteurs de la \ncarte de fidélité. \n ©2023, The Coca-Cola Company. Coca-Cola, la \nBouteille Contour et Savoure l'instant sont des \nmarques déposées de The Coca-Cola Company. \nCoca-Cola Europacific Partners France SAS Issy-\nles-Moulineaux RCS 343 688 016 Nanterre. ", false},
		{GreedyLinebreaker{}, text.Justified, " L’POUR VOTRE SANTÉ, MANGEZ AU MOINS 5 \nFRUITS ET LEGUMES PAR JOUR. \nWWW.MANGERBOUGER.FR\nOffre personnelle réservée aux porteurs de la \ncarte de fidélité. \n ©2023, The Coca-Cola Company. Coca-Cola, la \nBouteille Contour et Savoure l'instant sont des \nmarques déposées de The Coca-Cola Company. \nCoca-Cola Europacific Partners France SAS Issy-\nles-Moulineaux RCS 343 688 016 Nanterre. ", false},
		{KnuthLinebreaker{}, text.Left, " L’POUR VOTRE SANTÉ, MANGEZ AU MOINS \n5 FRUITS ET LEGUMES PAR JOUR. \nWWW.MANGERBOUGER.FR\nOffre personnelle réservée aux porteurs de la \ncarte de fidélité. \n ©2023, The Coca-Cola Company. Coca-Cola, la \nBouteille Contour et Savoure l'instant sont des \nmarques déposées de The Coca-Cola Company. \nCoca-Cola Europacific Partners France SAS Issy-\nles-Moulineaux RCS 343 688 016 Nanterre. ", false},
		{KnuthLinebreaker{}, text.Justified, " L’POUR VOTRE SANTÉ, MANGEZ AU MOINS \n5 FRUITS ET LEGUMES PAR JOUR. \nWWW.MANGERBOUGER.FR\nOffre personnelle réservée aux porteurs de la \ncarte de fidélité. \n ©2023, The Coca-Cola Company. Coca-Cola, la \nBouteille Contour et Savoure l'instant sont des \nmarques déposées de The Coca-Cola Company. \nCoca-Cola Europacific Partners France SAS Issy-\nles-Moulineaux RCS 343 688 016 Nanterre. ", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.lb, tt.align), func(t *testing.T) {
			ppem := face.PPEM(DefaultResolution)
			glyphs := face.Font.shaper.Shape(str, ppem, text.LeftToRight, text.Latin, "fr", "", "")
			for i := range glyphs {
				glyphs[i].SFNT = face.Font.SFNT
				glyphs[i].Size = face.Size
				glyphs[i].Script = text.Latin
			}
			glyphs = append(glyphs, text.Glyph{Cluster: uint32(len(str))})

			items := text.GlyphsToItems(glyphs[:len(glyphs)-1], text.Options{
				Indent: indent,
				Align:  tt.align,
			})
			breaks := tt.lb.Linebreak(items, width)

			result := ""
			i, g0, g := 0, 0, 0 // item index, glyph index
			lines := []string{}
			wyz := [][3]float64{}
			WYZ := [][3]float64{}
			for j, b := range breaks {
				for i < b.Position {
					g += items[i].Size
					i++
				}

				c0, c := int(glyphs[g0].Cluster), int(glyphs[g].Cluster)
				W, Y, Z := face.TextWidth(str[c0:c]), 0.0, 0.0
				if j == 0 {
					W += indent
				}
				n := 0
				for k := g0; k < g; k++ {
					if text.IsSpace(glyphs[k].Text) {
						w, y, z := text.SpaceGlue(glyphs, k)
						W += w - glyphs[k].Advance()
						Y += y
						Z += z
						n++
					}
				}
				if text.IsNewline(glyphs[g].Text) || g+1 == len(glyphs) {
					Y += text.Infinity
				} else if tt.align == text.Left {
					Y += 5.0 * text.SpaceRaggedStretch
				}
				wyz = append(wyz, [3]float64{b.Width, b.Stretch, b.Shrink})
				WYZ = append(WYZ, [3]float64{W, Y, Z})
				lines = append(lines, str[c0:c])

				g += items[i].Size
				i++
				if i < len(items) && items[i].Type == text.GlueType {
					g += items[i].Size
					i++
				}
				c = int(glyphs[g].Cluster)

				if 0 < len(result) && result[len(result)-1] != '\n' {
					result += "\n"
				}
				result += str[c0:c]
				g0 = g
			}
			test.String(t, result, tt.result)
			//test.T(t, !ok, tt.overflows)
			for j := range WYZ {
				t.Run(fmt.Sprint(tt.lb, tt.align, j), func(t *testing.T) {
					test.Floats(t, wyz[j][:], WYZ[j][:], lines[j])
				})
			}
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
	text := rt.ToText(4096.0, 4096.0, Left, Top, nil)

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
	ctx.DrawText(0, 0, NewTextBox(face, "\ntext", 100, 100, Left, Top, nil))
	ctx.DrawText(0, 0, NewTextBox(face, "text\n\ntext2", 100, 100, Left, Top, nil))
}
