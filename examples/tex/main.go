package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

var fontFamily *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("DejaVu Serif")
	fontFamily.Use(canvas.CommonLigatures)
	if err := fontFamily.LoadLocalFont("DejaVuSerif", canvas.FontRegular); err != nil {
		panic(err)
	}

	f, err := os.Create("out.tex")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c := canvas.NewTeX(f, 20, 10)
	defer c.Close()

	ctx := canvas.NewContext(c)
	ctx.SetView(canvas.Identity.Translate(10, 10).Scale(0.5, 0.5))
	draw(ctx)
}

func drawText(c *canvas.Context, x, y float64, face canvas.FontFace, rich *canvas.RichText) {
	metrics := face.Metrics()
	width, height := 90.0, 35.0

	text := rich.ToText(width, height, canvas.Justify, canvas.Top, 0.0, 0.0)

	c.SetFillColor(color.RGBA{192, 0, 64, 255})
	c.DrawPath(x, y, text.Bounds().ToPath())
	c.SetFillColor(color.RGBA{50, 50, 50, 50})
	c.DrawPath(x, y, canvas.Rectangle(width, -metrics.LineHeight))
	c.SetFillColor(color.RGBA{0, 0, 0, 50})
	c.DrawPath(x, y+metrics.CapHeight-metrics.Ascent, canvas.Rectangle(width, -metrics.CapHeight-metrics.Descent))
	c.DrawPath(x, y+metrics.XHeight-metrics.Ascent, canvas.Rectangle(width, -metrics.XHeight))

	c.SetFillColor(canvas.Black)
	c.DrawPath(x, y, canvas.Rectangle(width, -height).Stroke(0.2, canvas.RoundCap, canvas.RoundJoin))
	c.DrawText(x, y, text)
}

func draw(c *canvas.Context) {
	// Draw a comprehensive text box
	fontSize := 14.0
	face := fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	rich := canvas.NewRichText()
	rich.Add(face, "\"Lorem ")
	rich.Add(face, " dolor ")
	rich.Add(face, "ipsum")
	rich.Add(face, "\". Confiscator")
	rich.Add(face, " Curabitur mattis dui tellus vel.")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontBold, canvas.FontNormal), " faux bold")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontItalic, canvas.FontNormal), "faux italic")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontUnderline), "underline")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.White, canvas.FontRegular, canvas.FontNormal, canvas.FontDoubleUnderline), "double underline")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontSineUnderline), "sine")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontSawtoothUnderline), "sawtooth")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontDottedUnderline), "dotted")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontDashedUnderline), "dashed")
	rich.Add(face, " ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontOverline), "overline ")
	rich.Add(fontFamily.Face(fontSize, canvas.Black, canvas.FontItalic, canvas.FontNormal, canvas.FontStrikethrough, canvas.FontSineUnderline, canvas.FontOverline), "combi")
	rich.Add(face, ".")
	drawText(c, 5, 95, face, rich)

	// Draw the word Stroke being stroked
	face = fontFamily.Face(80.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	p, _ := face.ToPath("Stroke")
	c.DrawPath(100, 5, p.Stroke(0.75, canvas.RoundCap, canvas.RoundJoin))

	// Draw a LaTeX formula
	latex, err := canvas.ParseLaTeX(`$y = \sin\left(\frac{x}{180}\pi\right)$`)
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
	img, err := png.Decode(lenna)
	if err != nil {
		panic(err)
	}
	c.Push()
	c.Rotate(5)
	c.DrawImage(50.0, 0.0, img, 15)
	c.Pop()

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
