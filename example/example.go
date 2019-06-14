package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

var dejaVuSerif canvas.Font

func main() {
	var err error
	dejaVuSerif, err = canvas.LoadFontFile("DejaVuSerif", canvas.Regular, "DejaVuSerif.woff")
	if err != nil {
		panic(err)
	}
	dejaVuSerif.Use(canvas.CommonLigatures)

	c := canvas.New(200, 80)
	Draw(c)

	////////////////

	svgFile, err := os.Create("example.svg")
	if err != nil {
		panic(err)
	}
	defer svgFile.Close()
	c.WriteSVG(svgFile)

	////////////////

	// SLOW
	pngFile, err := os.Create("example.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := c.WriteImage(144.0)
	err = png.Encode(pngFile, img)
	if err != nil {
		panic(err)
	}

	////////////////

	pdfFile, err := os.Create("example.pdf")
	if err != nil {
		panic(err)
	}
	defer pdfFile.Close()

	err = c.WritePDF(pdfFile)
	if err != nil {
		panic(err)
	}

	////////////////

	epsFile, err := os.Create("example.eps")
	if err != nil {
		panic(err)
	}
	defer epsFile.Close()
	c.WriteEPS(epsFile)
}

func drawText(c *canvas.C, x, y float64, face canvas.FontFace, rich *canvas.RichText) {
	metrics := face.Metrics()
	width, height := 80.0, 25.0
	text := rich.ToText(width, height, canvas.Justify, canvas.Top, 0.0)

	c.SetFillColor(canvas.Orangered)
	c.DrawPath(x, y, text.Bounds().ToPath())
	c.SetFillColor(color.RGBA{0, 0, 0, 50})
	c.DrawPath(x, y, canvas.Rectangle(0, 0, width, -metrics.LineHeight))
	c.DrawPath(x, y, canvas.Rectangle(0, metrics.CapHeight-metrics.Ascent, width, -metrics.CapHeight-metrics.Descent))
	c.DrawPath(x, y, canvas.Rectangle(0, metrics.XHeight-metrics.Ascent, width, -metrics.XHeight))

	c.SetFillColor(canvas.Black)
	c.DrawPath(x, y, canvas.Rectangle(0.0, 0.0, width, -height).Stroke(0.2, canvas.RoundCapper, canvas.RoundJoiner))
	c.DrawText(x, y, text)
}

func Draw(c *canvas.C) {
	face := dejaVuSerif.Face(12.0)
	rich := canvas.NewRichText()
	rich.Add(face, canvas.Black, "\"Lorem ")
	rich.Add(face, canvas.Teal, "ipsum ")
	rich.Add(face.Faux(canvas.Subscript), canvas.Black, "1")
	rich.Add(face.Faux(canvas.Inferior), canvas.Black, "2")
	rich.Add(face.Faux(canvas.Superior), canvas.Black, "3")
	rich.Add(face.Faux(canvas.Superscript), canvas.Black, "4")
	rich.Add(face, canvas.Black, " dolor\", confis\u200bcatur. ")
	rich.Add(face.Faux(canvas.Bold), canvas.Black, "faux bold ")
	rich.Add(face.Faux(canvas.Italic), canvas.Black, "faux\titalic ")
	rich.Add(face.Decoration(canvas.Underline), canvas.Black, "underline")
	rich.Add(face, canvas.Black, " ")
	rich.Add(face.Decoration(canvas.DoubleUnderline), canvas.Black, "double underline")
	rich.Add(face, canvas.Black, " ")
	rich.Add(face.Decoration(canvas.SineUnderline), canvas.Black, "sine")
	rich.Add(face, canvas.Black, " ")
	rich.Add(face.Decoration(canvas.SawtoothUnderline), canvas.Black, "sawtooth")
	rich.Add(face, canvas.Black, " ")
	rich.Add(face.Decoration(canvas.DottedUnderline), canvas.Black, "dotted")
	rich.Add(face, canvas.Black, " ")
	rich.Add(face.Decoration(canvas.DashedUnderline), canvas.Black, "dashed")
	rich.Add(face, canvas.Black, " ")
	rich.Add(face.Decoration(canvas.Overline), canvas.Black, "overline ")
	rich.Add(face.Faux(canvas.Italic).Decoration(canvas.Strikethrough, canvas.SineUnderline, canvas.Overline), canvas.Black, "combi")
	rich.Add(face, canvas.Black, ".")
	drawText(c, 10, 70, face, rich)

	face = dejaVuSerif.Face(80.0)
	p := canvas.NewTextLine(face, canvas.Black, "Stroke").ToPath()
	c.DrawPath(5, 10, p.Stroke(0.75, canvas.RoundCapper, canvas.RoundJoiner))

	latex, err := canvas.ParseLaTeX(`$y = \sin\left(\frac{x}{180}\pi\right)$`)
	if err != nil {
		panic(err)
	}
	latex = latex.Transform(canvas.Identity.Rotate(-30))
	c.SetFillColor(canvas.Black)
	c.DrawPath(140, 65, latex)

	ellipse, err := canvas.ParseSVG(fmt.Sprintf("A10 20 30 1 0 20 0z"))
	if err != nil {
		panic(err)
	}
	c.SetFillColor(canvas.Whitesmoke)
	c.DrawPath(110, 40, ellipse)

	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.5)
	c.SetStrokeCapper(canvas.RoundCapper)
	c.SetStrokeJoiner(canvas.RoundJoiner)
	c.SetDashes(0.0, 2.0, 4.0, 2.0, 2.0, 4.0, 2.0)
	//ellipse = ellipse.Dash(0.0, 2.0, 4.0, 2.0).Stroke(0.5, canvas.RoundCapper, canvas.RoundJoiner)
	c.DrawPath(110, 40, ellipse)
	c.SetStrokeColor(canvas.Transparent)

	polyline := &canvas.Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(20.0, 0.0)
	polyline.Add(20.0, 10.0)
	polyline.Add(0.0, 20.0)
	polyline.Add(0.0, 0.0)
	c.SetFillColor(canvas.Seagreen)
	c.DrawPath(170, 10, polyline.Smoothen())
	c.SetFillColor(canvas.Black)
	c.DrawPath(170, 10, polyline.ToPath().Stroke(0.25, canvas.RoundCapper, canvas.RoundJoiner))

	polyline = &canvas.Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(10.0, 5.0)
	polyline.Add(20.0, 15.0)
	polyline.Add(30.0, 20.0)
	polyline.Add(40.0, 10.0)
	c.SetFillColor(canvas.Seagreen)
	c.DrawPath(120, 5, polyline.Smoothen().Stroke(0.5, canvas.RoundCapper, canvas.RoundJoiner))
	c.SetFillColor(canvas.Black)
	for _, coord := range polyline.Coords() {
		c.DrawPath(120, 5, canvas.Circle(1.0).Translate(coord.X, coord.Y).Stroke(0.5, canvas.RoundCapper, canvas.RoundJoiner))
	}
}
