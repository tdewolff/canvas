package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

var dejaVuSerif *canvas.FontFamily

func main() {
	dejaVuSerif = canvas.NewFontFamily("dejavu-serif")
	dejaVuSerif.Use(canvas.CommonLigatures)
	if err := dejaVuSerif.LoadFontFile("DejaVuSerif.woff", canvas.FontRegular); err != nil {
		panic(err)
	}

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

func drawText(c *canvas.Canvas, x, y float64, face canvas.FontFace, rich *canvas.RichText) {
	metrics := face.Metrics()
	width, height := 80.0, 25.0

	text := rich.ToText(width, height, canvas.Justify, canvas.Top, 0.0, 0.0)

	c.SetFillColor(canvas.Orangered)
	fmt.Println("-")
	c.DrawPath(x, y, text.Bounds().ToPath())
	fmt.Println("=")
	c.SetFillColor(color.RGBA{0, 0, 0, 50})
	c.DrawPath(x, y, canvas.Rectangle(0, 0, width, -metrics.LineHeight))
	c.DrawPath(x, y, canvas.Rectangle(0, metrics.CapHeight-metrics.Ascent, width, -metrics.CapHeight-metrics.Descent))
	c.DrawPath(x, y, canvas.Rectangle(0, metrics.XHeight-metrics.Ascent, width, -metrics.XHeight))

	c.SetFillColor(canvas.Black)
	fmt.Println("-")
	c.DrawPath(x, y, canvas.Rectangle(0.0, 0.0, width, -height).Stroke(0.2, canvas.RoundCapper, canvas.RoundJoiner))
	fmt.Println("a")
	c.DrawText(x, y, text)
	fmt.Println("b")
}

func Draw(c *canvas.Canvas) {
	// Draw a comprehensive text box
	face := dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black)
	rich := canvas.NewRichText(face)
	rich.Add(face, "\"Lorem ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Teal), "ipsum ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontSubscript, canvas.Black), "1")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontSuperscript, canvas.Black), "2")
	rich.Add(face, " dolor\", confis\u200bcatur. ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontBold, canvas.FontNormal, canvas.Black), "faux bold")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontItalic, canvas.FontNormal, canvas.Black), "faux italic")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontUnderline), "underline")
	rich.Add(face, " ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontDoubleUnderline), "double underline")
	rich.Add(face, " ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontSineUnderline), "sine")
	rich.Add(face, " ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontSawtoothUnderline), "sawtooth")
	rich.Add(face, " ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontDottedUnderline), "dotted")
	rich.Add(face, " ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontDashedUnderline), "dashed")
	rich.Add(face, " ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontRegular, canvas.FontNormal, canvas.Black, canvas.FontOverline), "overline ")
	rich.Add(dejaVuSerif.Face(12.0, canvas.FontItalic, canvas.FontNormal, canvas.Black, canvas.FontStrikethrough, canvas.FontSineUnderline, canvas.FontOverline), "combi ")
	rich.Add(face, ".")
	drawText(c, 10, 70, face, rich)

	// Draw the word Stroke being stroked
	face = dejaVuSerif.Face(80.0, canvas.FontRegular, canvas.FontNormal, canvas.Black)
	p, _ := face.ToPath("Stroke")
	c.DrawPath(5, 10, p.Stroke(0.75, canvas.RoundCapper, canvas.RoundJoiner))

	// Draw a LaTeX formula
	latex, err := canvas.ParseLaTeX(`$y = \sin\left(\frac{x}{180}\pi\right)$`)
	if err != nil {
		panic(err)
	}
	latex = latex.Transform(canvas.Identity.Rotate(-30))
	c.SetFillColor(canvas.Black)
	c.DrawPath(140, 65, latex)

	// Draw an elliptic arc being dashed
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

	// Draw an open set of points being smoothed
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

	// Draw a closed set of points being smoothed
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

	// Draw a raster image
	lenna, err := os.Open("lenna.png")
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(lenna)
	if err != nil {
		panic(err)
	}
	c.DrawImage(105.0, 15.0, img, canvas.Lossy, 650.24)
}
