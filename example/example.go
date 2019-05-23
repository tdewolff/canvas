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

func drawText(c *canvas.C, x, y float64, size float64, s string) {
	face := dejaVuSerif.Face(size)
	metrics := face.Metrics()
	text := canvas.NewTextBox(face, s, 80.0, 30.0, canvas.Justify, canvas.Top, 0.0)
	rect := text.Bounds()

	c.SetColor(canvas.Gainsboro)
	c.DrawPath(x, y, 0.0, rect.ToPath())
	c.SetColor(color.RGBA{0, 0, 0, 50})
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, 0, rect.W, -metrics.LineHeight))
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, metrics.CapHeight-metrics.Ascent, rect.W, -metrics.CapHeight-metrics.Descent))
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, metrics.XHeight-metrics.Ascent, rect.W, -metrics.XHeight))

	c.SetColor(canvas.Black)
	c.DrawText(x, y, 0.0, text)
}

func Draw(c *canvas.C) {
	drawText(c, 10, 70, 28.0, "Aap noot mies \"fi ffi ffl\"")

	face := dejaVuSerif.Face(80.0)
	p := canvas.NewText(face, "Stroke").ToPath(0.0, 0.0)
	c.DrawPath(5, 10, 0.0, p.Stroke(1, canvas.RoundCapper, canvas.RoundJoiner))

	latex, err := canvas.ParseLaTeX(`$y = \sin\left(\frac{x}{180}\pi\right)$`)
	if err != nil {
		panic(err)
	}
	latex.Rotate(-30, 0, 0)
	c.SetColor(canvas.Black)
	c.DrawPath(140, 60, 0.0, latex)

	ellipse, err := canvas.ParseSVG(fmt.Sprintf("A10 20 30 1 0 20 0z"))
	if err != nil {
		panic(err)
	}
	c.SetColor(canvas.WhiteSmoke)
	c.DrawPath(130, 20, 0.0, ellipse)
	ellipse = ellipse.Dash(2.0, 4.0, 2.0).Stroke(0.5, canvas.RoundCapper, canvas.RoundJoiner)
	c.SetColor(canvas.Black)
	c.DrawPath(130, 20, 0.0, ellipse)

	p = &canvas.Path{}
	p.LineTo(10.0, 0.0)
	p.LineTo(10.0, 10.0)
	p.LineTo(0.0, 10.0)
	q := p.Smoothen()
	c.DrawPath(160, 10, 0.0, q)
	c.SetColor(canvas.Grey)
	c.DrawPath(160, 10, 0.0, p)
}
