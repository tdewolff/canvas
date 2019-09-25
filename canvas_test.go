package canvas

import (
	"bytes"
	"image"
	"io/ioutil"
	"testing"
)

func TestCanvas(t *testing.T) {
	path := MustParseSVG("M10 0L20 0Q25 10 30 0C30 10 40 10 40 0A5 5 0 0 0 50 0z")

	dejaVuSerif := NewFontFamily("dejavu-serif")
	dejaVuSerif.LoadFontFile("test/DejaVuSerif.ttf", FontRegular)
	face := dejaVuSerif.Face(10.0, Green, FontItalic|FontBold, FontNormal)
	text := NewTextLine(face, "Text", Left)

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, Black)
	img.Set(1, 0, Red)
	img.Set(0, 1, Green)
	img.Set(1, 1, Black)

	c := New(100, 100)
	c.SetView(Identity.Rotate(90).Scale(2.0, 1.0))
	c.SetFillColor(Red)
	c.SetStrokeColor(Gray)
	c.SetStrokeWidth(1.0)
	c.SetStrokeCapper(RoundCapper)
	c.SetStrokeJoiner(RoundJoiner)
	c.SetDashes(1.0, 2.0, 3.0, 4.0)

	c.DrawPath(10.0, 10.0, path)                // 50x7.5 => -7.5x100
	c.DrawText(30.0, 30.0, text)                // contained between the other two
	c.DrawImage(50.0, 50.0, img, Lossless, 0.1) // 20x20 => -20x40

	c.Fit(6.0)
	//test.Float(t, c.W, 50.0-2.5+12.0) // img upper bound - path lower bound + margin
	//test.Float(t, c.H, 110-30+12.0)   // path bounds + margin

	buf := &bytes.Buffer{}
	c.WriteSVG(buf)
	ioutil.WriteFile("test/canvas.svg", buf.Bytes(), 0644)
	//s := regexp.MustCompile(`base64,.+'`).ReplaceAllString(buf.String(), "base64,'") // remove embedded font
	//test.String(t, s, `<svg version="1.1" width="59.5" height="92" viewBox="0 0 59.5 92" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><defs><style>`+"\n"+`@font-face{font-family:'dejavu-serif';src:url('data:font/truetype;base64,');}`+"\n"+`</style></defs><path d="M13.5 86V66Q3.5 56 13.5 46C3.5 46 3.5 26 13.5 26A5 10 0 0 1 13.5 6z" style="fill:#f00;stroke:#808080;stroke-linecap:round;stroke-linejoin:round;stroke-dasharray:2 3 4;stroke-dashoffset:1"/><text transform="translate(33.5,86) rotate(-90) scale(2,1)" style="font: italic 700 3.5277778px dejavu-serif;fill:#008000"><tspan x="0" y="0">Text</tspan></text><image transform="translate(33.5,66) rotate(-90) scale(20,10)" width="2" height="2" xlink:href="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAGElEQVR4nGJiYGD4z8DAwNAAIgABAAD//wygAYJr2xzBAAAAAElFTkSuQmCC"/></svg>`)

	buf.Reset()
	pdfCompress = false
	c.WritePDF(buf)
	ioutil.WriteFile("test/canvas.pdf", buf.Bytes(), 0644)
	//s = regexp.MustCompile(`stream\nx(.|\n)+\nendstream\n`).ReplaceAllString(buf.String(), `stream\n\nendstream\n`) // remove embedded font
	//test.String(t, s, ``)

	buf.Reset()
	c.WriteEPS(buf)
	ioutil.WriteFile("test/canvas.eps", buf.Bytes(), 0644)
}
