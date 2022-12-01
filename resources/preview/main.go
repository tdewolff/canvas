// +build harfbuzz

package main

import (
	"fmt"
	"image/color"
	"os"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
	"github.com/tdewolff/canvas/text"
)

var fontLatin *canvas.FontFamily
var fontArabic *canvas.FontFamily
var fontDevanagari *canvas.FontFamily

func main() {
	fontLatin = canvas.NewFontFamily("DejaVu Serif")
	if err := fontLatin.LoadLocalFont("DejaVuSerif", canvas.FontRegular); err != nil {
		panic(err)
	}

	fontArabic = canvas.NewFontFamily("DejaVu Sans")
	if err := fontArabic.LoadLocalFont("DejaVuSans", canvas.FontRegular); err != nil {
		panic(err)
	}

	fontDevanagari = canvas.NewFontFamily("Devanagari")
	if err := fontDevanagari.LoadLocalFont("NotoSerifDevanagari", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(200, 100)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))
	draw(ctx)

	////////////////

	c.WriteFile("preview.png", renderers.PNG(canvas.DPMM(3.2)))
}

func drawText(c *canvas.Context, x, y float64, face *canvas.FontFace, rich *canvas.RichText) {
	metrics := face.Metrics()
	width, height := 90.0, 32.0

	text := rich.ToText(width, height, canvas.Justify, canvas.Top, 0.0, 0.0)

	c.SetFillColor(color.RGBA{192, 0, 64, 255})
	c.DrawPath(x, y, text.Bounds().ToPath())
	c.SetFillColor(color.RGBA{51, 51, 51, 51})
	c.DrawPath(x, y, canvas.Rectangle(width, -metrics.LineHeight))
	c.SetFillColor(color.RGBA{0, 0, 0, 51})
	c.DrawPath(x, y+metrics.CapHeight-metrics.Ascent, canvas.Rectangle(width, -metrics.CapHeight-metrics.Descent))
	c.DrawPath(x, y+metrics.XHeight-metrics.Ascent, canvas.Rectangle(width, -metrics.XHeight))

	c.SetFillColor(canvas.Black)
	c.DrawPath(x, y, canvas.Rectangle(width, -height).Stroke(0.2, canvas.RoundCap, canvas.RoundJoin))
	c.DrawText(x, y, text)
}

func draw(c *canvas.Context) {
	// Draw a comprehensive text box
	pt := 14.0
	face := fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	rt := canvas.NewRichText(face)
	rt.Add(face, "Lorem dolor ipsum ")
	rt.Add(fontLatin.Face(pt, canvas.White, canvas.FontBold, canvas.FontNormal), "confiscator")
	rt.Add(face, " cur\u200babitur ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontItalic, canvas.FontNormal), "mattis")
	rt.Add(face, " dui ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontBold|canvas.FontItalic, canvas.FontNormal), "tellus")
	rt.Add(face, " vel. Proin ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontUnderline), "sodales")
	rt.Add(face, " eros vel ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontSineUnderline), "nibh")
	rt.Add(face, " fringilla pellen\u200btesque eu ")

	c2 := canvas.New(6.144, 6.144)
	ctx2 := canvas.NewContext(c2)
	ctx2.SetView(canvas.Identity.Translate(0.0, 6.144).Scale(0.05, -0.05))
	// face
	ctx2.SetFillColor(canvas.Hex("#fbd433"))
	ctx2.DrawPath(0.0, 0.0, canvas.MustParseSVG("M45.54,2.11A61.42,61.42,0,1,1,2.11,77.34,61.42,61.42,0,0,1,45.54,2.11Z"))
	// eyes
	ctx2.SetFillColor(canvas.Hex("#141518"))
	ctx2.DrawPath(0.0, 0.0, canvas.MustParseSVG("M45.78,32.27c4.3,0,7.79,5,7.79,11.27s-3.49,11.27-7.79,11.27S38,49.77,38,43.54s3.48-11.27,7.78-11.27Z"))
	ctx2.DrawPath(0.0, 0.0, canvas.MustParseSVG("M77.1,32.27c4.3,0,7.78,5,7.78,11.27S81.4,54.81,77.1,54.81s-7.79-5-7.79-11.27S72.8,32.27,77.1,32.27Z"))
	// mouth
	ctx2.DrawPath(0.0, 0.0, canvas.MustParseSVG("M28.8,70.82a39.65,39.65,0,0,0,8.83,8.41,42.72,42.72,0,0,0,25,7.53,40.44,40.44,0,0,0,24.12-8.12,35.75,35.75,0,0,0,7.49-7.87.22.22,0,0,1,.31,0L97,73.14a.21.21,0,0,1,0,.29A45.87,45.87,0,0,1,82.89,88.58,37.67,37.67,0,0,1,62.83,95a39,39,0,0,1-20.68-5.55A50.52,50.52,0,0,1,25.9,73.57a.23.23,0,0,1,0-.28l2.52-2.5a.22.22,0,0,1,.32,0l0,0Z"))
	rt.AddCanvas(c2, canvas.FontMiddle)
	rt.Add(face, " cillum. ")

	face = fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	face.Language = "ru"
	face.Script = text.Cyrillic
	rt.Add(face, "дёжжэнтиюнт холст ")

	face = fontArabic.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	face.Language = "ar"
	face.Script = text.Arabic
	face.Direction = text.RightToLeft
	rt.Add(face, "تسجّل يتكلّم ")

	face = fontDevanagari.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	face.Language = "hi"
	face.Script = text.Devanagari
	rt.Add(face, "हालाँकि प्र ")

	drawText(c, 5, 95, face, rt)

	// Draw the word Stroke being stroked
	face = fontLatin.Face(80.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	p, _, _ := face.ToPath("Stroke")
	c.DrawPath(100, 5, p.Stroke(0.75, canvas.RoundCap, canvas.RoundJoin))

	// Draw a LaTeX formula
	latex, err := canvas.ParseLaTeX(`$y = \sin\left(\frac{x}{180}\pi\right)$`, 12.0)
	if err != nil {
		panic(err)
	}
	latex = latex.Transform(canvas.Identity.Rotate(-30))
	c.SetFillColor(canvas.Black)
	c.DrawPath(140, 85, latex)

	// Draw an elliptic arc being dashed
	ellipse, err := canvas.ParseSVG(fmt.Sprintf("A10 30 30 1 0 30 0z"))
	if err != nil {
		panic(err)
	}
	c.SetFillColor(canvas.Whitesmoke)
	c.DrawPath(110, 40, ellipse)

	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.75)
	c.SetStrokeCapper(canvas.RoundCap)
	c.SetStrokeJoiner(canvas.RoundJoin)
	c.SetDashes(0.0, 2.0, 4.0, 2.0, 2.0, 4.0, 2.0)
	//ellipse = ellipse.Dash(0.0, 2.0, 4.0, 2.0).Stroke(0.5, canvas.RoundCap, canvas.RoundJoin)
	c.DrawPath(110, 40, ellipse)
	c.SetStrokeColor(canvas.Transparent)
	c.SetDashes(0.0)

	// Draw a raster image
	lenna, err := os.Open("../lenna.png")
	if err != nil {
		panic(err)
	}
	img, err := canvas.NewPNGImage(lenna)
	if err != nil {
		panic(err)
	}
	c.Rotate(5)
	c.DrawImage(50.0, 0.0, img, 15)
	c.SetView(canvas.Identity)

	// Draw an closed set of points being smoothed
	polyline := &canvas.Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(30.0, 0.0)
	polyline.Add(30.0, 15.0)
	polyline.Add(0.0, 30.0)
	polyline.Add(0.0, 0.0)
	c.SetFillColor(canvas.Seagreen)
	c.FillColor.R = byte(float64(c.FillColor.R) * 0.25)
	c.FillColor.G = byte(float64(c.FillColor.G) * 0.25)
	c.FillColor.B = byte(float64(c.FillColor.B) * 0.25)
	c.FillColor.A = byte(float64(c.FillColor.A) * 0.25)
	c.SetStrokeColor(canvas.Seagreen)
	c.DrawPath(155, 35, polyline.Smoothen())

	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.5)
	c.DrawPath(155, 35, polyline.ToPath())
	c.SetStrokeWidth(0.75)
	for _, coord := range polyline.Coords() {
		c.DrawPath(155, 35, canvas.Circle(2.0).Translate(coord.X, coord.Y))
	}

	// Draw a open set of points being smoothed
	polyline = &canvas.Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(20.0, 10.0)
	polyline.Add(40.0, 30.0)
	polyline.Add(60.0, 40.0)
	polyline.Add(80.0, 20.0)
	c.SetStrokeColor(canvas.Dodgerblue)
	c.DrawPath(10, 15, polyline.Smoothen())
	c.SetStrokeColor(canvas.Black)
	for _, coord := range polyline.Coords() {
		c.DrawPath(10, 15, canvas.Circle(2.0).Translate(coord.X, coord.Y))
	}
}
