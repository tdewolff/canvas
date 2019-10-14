package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestFontFamily(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("font/DejaVuSerif.ttf", FontRegular)

	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal)
	test.Float(t, face.fauxBold, 0.0)
	test.T(t, face.boldness(), 400)

	face = family.Face(12.0*ptPerMm, Black, FontBold|FontItalic, FontNormal)
	test.Float(t, face.fauxBold, 0.24)
	test.Float(t, face.fauxItalic, 0.3)
	test.T(t, face.boldness(), 700)

	face = family.Face(12.0*ptPerMm, Black, FontBold|FontItalic, FontSubscript)
	test.Float(t, face.voffset, -12.0*0.33)
	test.Float(t, face.fauxBold, 0.48*0.583)
	test.Float(t, face.fauxItalic, 0.3)
	test.T(t, face.boldness(), 1000)
}

func TestFontFace(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("font/DejaVuSerif.ttf", FontRegular)
	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal)

	metrics := face.Metrics()
	test.Float(t, metrics.Size, 12.0)
	test.Float(t, metrics.LineHeight, 13.96875)
	test.Float(t, metrics.Ascent, 11.140625)
	test.Float(t, metrics.Descent, 2.828125)
	test.Float(t, metrics.XHeight, 6.234375)
	test.Float(t, metrics.CapHeight, 8.75)

	test.Float(t, face.Kerning('M', 'M'), 0.0)
	test.Float(t, face.Kerning('A', 'V'), -0.59375)
	test.Float(t, face.TextWidth("T"), 8.0)
	test.Float(t, face.TextWidth("AV"), face.TextWidth("A")+face.TextWidth("V")+face.Kerning('A', 'V'))

	Epsilon = 1e-3
	p, width := face.ToPath("AO")
	test.T(t, p, MustParseSVG("M2.4062 3.1719L5.6094 3.1719L4.0156 7.3281L2.4062 3.1719zM-0.078125 0L-0.078125 0.625L0.70312 0.625L3.8125 8.75L4.7969 8.75L7.9219 0.625L8.7812 0.625L8.7812 0L5.6094 0L5.6094 0.625L6.5781 0.625L5.8438 2.5469L2.1562 2.5469L1.4375 0.625L2.3906 0.625L2.3906 0L-0.078125 0zM13.594 0.45312Q15.031 0.45312 15.766 1.4375Q16.5 2.4375 16.5 4.3594Q16.5 6.3125 15.766 7.2969Q15.031 8.2812 13.594 8.2812Q12.156 8.2812 11.422 7.2969Q10.688 6.3125 10.688 4.3594Q10.688 2.4375 11.422 1.4375Q12.156 0.45312 13.594 0.45312zM13.594 -0.17188Q12.703 -0.17188 11.953 0.125Q11.203 0.42188 10.641 0.98438Q9.9844 1.6406 9.6562 2.4688Q9.3438 3.3125 9.3438 4.3594Q9.3438 5.4219 9.6562 6.2656Q9.9844 7.0938 10.641 7.75Q11.219 8.3281 11.953 8.6094Q12.688 8.9062 13.594 8.9062Q15.5 8.9062 16.672 7.6562Q17.844 6.4062 17.844 4.3594Q17.844 3.3125 17.516 2.4688Q17.203 1.6406 16.547 0.98438Q15.969 0.40625 15.234 0.125Q14.484 -0.17188 13.594 -0.17188z"))
	test.Float(t, width, 18.515625)
}

func TestFontDecoration(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	family.LoadFontFile("font/DejaVuSerif.ttf", FontRegular)

	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 -2.25L10 -2.25L10 -1.35L0 -1.35L0 -2.25z"))

	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontOverline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 7.5844L10 7.5844L10 8.4844L0 8.4844L0 7.5844z"))

	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontStrikethrough)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 2.6672L10 2.6672L10 3.5672L0 3.5672L0 2.6672z"))

	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontDoubleUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 -1.8L10 -1.8L10 -0.9L0 -0.9L0 -1.8zM0 -3.6L10 -3.6L10 -2.7L0 -2.7L0 -3.6z"))

	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontDottedUnderline)
	test.T(t, face.Decorate(4.0), MustParseSVG("M1.44 -1.8A0.72 0.72 0 0 1 0 -1.8A0.72 0.72 0 0 1 1.44 -1.8zM2.72 -1.8A0.72 0.72 0 0 1 1.28 -1.8A0.72 0.72 0 0 1 2.72 -1.8zM4 -1.8A0.72 0.72 0 0 1 2.56 -1.8A0.72 0.72 0 0 1 4 -1.8z"))

	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontDashedUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 -2.25L10 -2.25L10 -1.35L0 -1.35L0 -2.25z"))

	Tolerance = 1e-1
	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontSineUnderline)
	test.T(t, face.Decorate(4.0), MustParseSVG("M0.9 -2.25L0.71841 -2.2046L1.3867 -3.987L1.6333 -4.05L1.88 -3.987L2.5483 -2.2046L2.3667 -2.25L2.1851 -2.2046L2.8534 -3.987L3.1 -4.05A0.45 0.45 0 0 1 3.1 -3.15L3.2816 -3.1954L2.6133 -1.413L2.3667 -1.35L2.12 -1.413L1.4517 -3.1954L1.6333 -3.15L1.8149 -3.1954L1.1466 -1.413L0.9 -1.35A0.45 0.45 0 0 1 0.9 -2.25z"))

	face = family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal, FontSawtoothUnderline)
	test.T(t, face.Decorate(4.0), MustParseSVG("M.2056 -1.931L.7511 -3.731L1.182 -5.152L1.612 -3.731L1.727 -3.352L1.842 -3.731L2.273 -5.152L2.703 -3.731L2.818 -3.352L2.933 -3.731L3.794 -3.469L3.249 -1.669L2.818 -.2484L2.388 -1.669L2.273 -2.048L2.158 -1.669L1.727 -.2484L1.297 -1.669L1.182 -2.048L1.067 -1.669z"))
}
