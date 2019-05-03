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
	text := canvas.NewTextBox(face, s, 100.0, 40.0, canvas.Justify, canvas.Top, 0.0)
	w, h := text.Bounds()

	c.SetColor(canvas.Gainsboro)
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, metrics.Ascent, w, -h))
	c.SetColor(color.RGBA{0, 0, 0, 50})
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, metrics.Ascent, w, -metrics.LineHeight))
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, metrics.CapHeight, w, -metrics.CapHeight-metrics.Descent))
	c.DrawPath(x, y, 0.0, canvas.Rectangle(0, metrics.XHeight, w, -metrics.XHeight))

	c.SetColor(canvas.Black)
	c.DrawText(x, y, 0.0, text)
}

func Draw(c *canvas.C) {
	drawText(c, 10, 60, 12.0, "Aap noot mies wim zus teun vuur")

	face := dejaVuSerif.Face(30)
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
}
