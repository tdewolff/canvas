package canvas

import (
	"image"
	"testing"

	"github.com/tdewolff/test"
)

func setEpsilon(epsilon float64) func() {
	origEpsilon := Epsilon
	Epsilon = epsilon
	return func() {
		Epsilon = origEpsilon
	}
}

func TestCanvas(t *testing.T) {
	path := MustParseSVGPath("M10 0L20 0Q25 10 30 0C30 10 40 10 40 0A5 5 0 0 0 50 0z")

	family := NewFontFamily("dejavu-serif")
	if err := family.LoadFontFile("resources/DejaVuSerif.ttf", FontRegular); err != nil {
		test.Error(t, err)
	}
	face := family.Face(10.0, Green, FontItalic|FontBold, FontNormal)
	text := NewTextLine(face, "Text", Left)

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, Black)
	img.Set(1, 0, Red)
	img.Set(0, 1, Green)
	img.Set(1, 1, Black)

	c := New(100, 100)
	ctx := NewContext(c)
	ctx.SetView(Identity.Rotate(90).Scale(2.0, 1.0))
	ctx.SetFillColor(Red)
	ctx.SetStrokeColor(Gray)
	ctx.SetStrokeWidth(1.0)
	ctx.SetStrokeCapper(RoundCap)
	ctx.SetStrokeJoiner(RoundJoin)
	ctx.SetDashes(-1.0, 2.0, 3.0, 4.0)

	ctx.DrawPath(10.0, 10.0, path)      // 50x7.5 => -7.5x100
	ctx.DrawText(30.0, 30.0, text)      // contained between the other two
	ctx.DrawImage(50.0, 50.0, img, 0.1) // 20x20 => -20x40

	c.Fit(6.0)
	test.Float(t, c.W, 72.5) // img left bound - path right bound + path stroke width/2 + margin
	test.Float(t, c.H, 113)  // img top bound - path bottom bound + path stroke width/2 + margin

	//s := regexp.MustCompile(`base64,.+'`).ReplaceAllString(buf.String(), "base64,'") // remove embedded font
	//test.String(t, s, `<svg version="1.1" width="60" height="93" viewBox="0 0 60 93" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><path d="M14 86.5V66.5Q4 56.5 14 46.5C4 46.5 4 26.5 14 26.5A5 10 0 0114 6.5z" style="fill:#f00;stroke:#808080;stroke-linecap:round;stroke-linejoin:round;stroke-dasharray:2 3 4;stroke-dashoffset:-1"/><style>`+"\n"+`@font-face{font-family:'dejavu-serif';src:url('data:font/truetype;base64,');}`+"\n"+`</style><text transform="translate(34,86.5) rotate(-90) scale(2,1)" style="font: italic 700 3.5277778px dejavu-serif;fill:#008000"><tspan x="0" y="0">Text</tspan></text><image transform="translate(34,66.5) rotate(-90) scale(20,10)" width="2" height="2" xlink:href="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAGElEQVR4nGJiYGD4z8DAwNAAIgABAAD//wygAYJr2xzBAAAAAElFTkSuQmCC"/></svg>`)

	//buf.Reset()
	//pdfCompress = false
	//c.WritePDF(buf)
	//os.WriteFile("test/canvas.pdf", buf.Bytes(), 0644)
	//s = regexp.MustCompile(`stream\nx(.|\n)+\nendstream\n`).ReplaceAllString(buf.String(), "stream\n\nendstream\n") // remove embedded font
	//test.String(t, s, "%PDF-1.7\n1 0 obj\n<< /Subtype /TrueType /Filter /FlateDecode /Length 215980 >> stream\n\nendstream\nendobj\n5 0 obj\n<< /Type /Page /Contents 4 0 R /Group << /Type /Group /CS /DeviceRGB /I true /S /Transparency >> /MediaBox [0 0 60 93] /Parent 5 0 R /Resources << /Font << /F0 2 0 R >> /XObject << /Im0 3 0 R >> >> >>\nendobj\n6 0 obj\n<< /Type /Pages /Count 1 /Kids [5 0 R] >>\nendobj\n7 0 obj\n<< /Type /Catalog /Pages 6 0 R >>\nendobj\nxref\n0 8\n0000000000 65535 f\n0000000009 00000 n\n0000216083 00000 n\n0000227285 00000 n\n0000227491 00000 n\n0000227888 00000 n\n0000228104 00000 n\n0000228161 00000 n\ntrailer\n<< /Root 7 0 R /Size 7 >>\nstarxref\n228210\n%%EOF")

	//buf.Reset()
	//c.WriteEPS(buf)
	//os.WriteFile("test/canvas.eps", buf.Bytes(), 0644)
	// TODO: test EPS when fully supported
}

func TestCanvasFit(t *testing.T) {
	c := New(100, 100)
	c.Fit(10)

	test.Float(t, c.W, 20)
	test.Float(t, c.H, 20)
}
