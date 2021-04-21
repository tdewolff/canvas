package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestFontFamily(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}

	face := family.Face(12.0*ptPerMm, Black, FontRegular, FontNormal)
	test.Float(t, face.FauxBold, 0.0)
	test.T(t, face.Boldness(), 400)

	face = family.Face(12.0*ptPerMm, Black, FontBold|FontItalic, FontNormal)
	test.Float(t, face.FauxBold, 0.02)
	test.Float(t, face.FauxItalic, 0.3)
	test.T(t, face.Boldness(), 700)

	//face = family.Face(12.0*ptPerMm, Black, FontBold|FontItalic, FontSubscript)
	//test.T(t, face.YOffset, int32(0))
	//test.Float(t, face.FauxBold, 0.48*0.583)
	//test.Float(t, face.FauxItalic, 0.3)
	//test.T(t, face.Boldness(), 1000)
}

func TestFontFace(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	pt := ptPerMm * float64(family.fonts[FontRegular].Head.UnitsPerEm)
	face := family.Face(pt, Black, FontRegular, FontNormal)

	metrics := face.Metrics()
	test.Float(t, face.Size, 2048)
	test.Float(t, metrics.LineHeight, 2384)
	test.Float(t, metrics.Ascent, 1901)
	test.Float(t, metrics.Descent, 483)
	test.Float(t, metrics.XHeight, 1063)   // height of x
	test.Float(t, metrics.CapHeight, 1493) // height of H

	test.Float(t, face.TextWidth("T"), 1366)
	test.Float(t, face.TextWidth("AV"), face.TextWidth("A")+face.TextWidth("V")-102)

	//Epsilon = 1e-3
	//p, width, err := face.ToPath("AO")
	//test.Error(t, err)
	//test.T(t, p, MustParseSVG("M2.4062 3.1719L5.6094 3.1719L4.0156 7.3281L2.4062 3.1719zM-0.078125 0L-0.078125 0.625L0.70312 0.625L3.8125 8.75L4.7969 8.75L7.9219 0.625L8.7812 0.625L8.7812 0L5.6094 0L5.6094 0.625L6.5781 0.625L5.8438 2.5469L2.1562 2.5469L1.4375 0.625L2.3906 0.625L2.3906 0L-0.078125 0zM13.594 0.45312Q15.031 0.45312 15.766 1.4375Q16.5 2.4375 16.5 4.3594Q16.5 6.3125 15.766 7.2969Q15.031 8.2812 13.594 8.2812Q12.156 8.2812 11.422 7.2969Q10.688 6.3125 10.688 4.3594Q10.688 2.4375 11.422 1.4375Q12.156 0.45312 13.594 0.45312zM13.594 -0.17188Q12.703 -0.17188 11.953 0.125Q11.203 0.42188 10.641 0.98438Q9.9844 1.6406 9.6562 2.4688Q9.3438 3.3125 9.3438 4.3594Q9.3438 5.4219 9.6562 6.2656Q9.9844 7.0938 10.641 7.75Q11.219 8.3281 11.953 8.6094Q12.688 8.9062 13.594 8.9062Q15.5 8.9062 16.672 7.6562Q17.844 6.4062 17.844 4.3594Q17.844 3.3125 17.516 2.4688Q17.203 1.6406 16.547 0.98438Q15.969 0.40625 15.234 0.125Q14.484 -0.17188 13.594 -0.17188z"))
	//test.Float(t, width, 18.515625)
}

func TestFontDecoration(t *testing.T) {
	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	pt := ptPerMm * float64(family.fonts[FontRegular].Head.UnitsPerEm)

	// ascent = 1901
	// underlineDistance = 130, underlineThickness = 90
	// yStrikoutSize = 102, yStrikeoutPosition = 530
	// note that we increase distance by half the thickness to match the implementation of Firefox
	face := family.Face(pt, Black, FontRegular, FontNormal, FontUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 -265L10 -265L10 -175L0 -175z"))

	face = family.Face(pt, Black, FontRegular, FontNormal, FontOverline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 1811L10 1811L10 1901L0 1901z"))

	face = family.Face(pt, Black, FontRegular, FontNormal, FontStrikethrough)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 530L10 530L10 632L0 632z"))

	face = family.Face(pt, Black, FontRegular, FontNormal, FontDoubleUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG("M0 -265L10 -265L10 -175L0 -175zM0 -400L10 -400L10 -310L0 -310z"))

	face = family.Face(pt, Black, FontRegular, FontNormal, FontDottedUnderline)
	test.T(t, face.Decorate(89.0), MustParseSVG(""))
	test.T(t, face.Decorate(90.0), MustParseSVG("M90 -220A45 45 0 0 1 0 -220A45 45 0 0 1 90 -220z"))
	test.T(t, face.Decorate(269.0), MustParseSVG("M179.5 -220A45 45 0 0 1 89.5 -220A45 45 0 0 1 179.5 -220z"))
	test.T(t, face.Decorate(270.0), MustParseSVG("M90 -220A45 45 0 0 1 0 -220A45 45 0 0 1 90 -220zM270 -220A45 45 0 0 1 180 -220A45 45 0 0 1 270 -220z"))

	face = family.Face(pt, Black, FontRegular, FontNormal, FontDashedUnderline)
	test.T(t, face.Decorate(809.0), MustParseSVG("M0 -265L809 -265L809 -175L0 -175z"))
	test.T(t, face.Decorate(810.0), MustParseSVG("M0 -265L270 -265L270 -175L0 -175zM540 -265L810 -265L810 -175L540 -175z"))

	face = family.Face(pt, Black, FontRegular, FontNormal, FontWavyUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG(""))
	test.T(t, face.Decorate(1000.0), MustParseSVG("M63.629999999999995 -265L189.72532469255538 -265L480.6386580258887 -572.2L664.8180086407781 -572.2L969.0441492937913 -250.94188048466083L903.6958507062087 -189.05811951533917L626.0953246925554 -482.20000000000005L519.3613419741113 -482.2000000000001L228.448008640778 -175L63.629999999999995 -175z"))

	origTolerance := Tolerance
	Tolerance = 10.0
	face = family.Face(pt, Black, FontRegular, FontNormal, FontSineUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG(""))
	test.T(t, face.Decorate(1000.0), MustParseSVG("M90 -265L112.73688056867151 -281.1023416347538L275.5649926129448 -528.3142895062493L363.3333333333333 -572.2L451.1016740537216 -528.3142895062495L613.9297860979952 -281.1023416347538L636.6666666666666 -265L659.403547235338 -281.1023416347537L822.2316592796112 -528.3142895062493L910 -572.2A45 45 0 0 1 910 -482.20000000000005L887.2631194313283 -466.09765836524616L724.4350073870551 -218.8857104937506L636.6666666666666 -175L548.8983259462782 -218.88571049375085L386.0702139020045 -466.09765836524633L363.3333333333333 -482.20000000000005L340.59645276466176 -466.0976583652462L177.7683407203886 -218.88571049375076L90 -175A45 45 0 0 1 90 -265z"))
	Tolerance = origTolerance

	face = family.Face(pt, Black, FontRegular, FontNormal, FontSawtoothUnderline)
	test.T(t, face.Decorate(10.0), MustParseSVG(""))
	test.T(t, face.Decorate(1000.0), MustParseSVG("M37.72575810060956 -256.7963347579329L500 -582.2326551087655L962.2742418993904 -256.7963347579329L910.4657581006096 -183.20366524206705L499.9999999999999 -472.1673448912347L89.53424189939042 -183.20366524206705z"))
}
