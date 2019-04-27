package main

import (
	"fmt"
	"image/color"
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

	c := canvas.New()
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
	//pngFile, err := os.Create("example.png")
	//if err != nil {
	//	panic(err)
	//}
	//defer pngFile.Close()

	//img := c.WriteImage(144.0)
	//err = png.Encode(pngFile, img)
	//if err != nil {
	//	panic(err)
	//}

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

func drawStrokedPath(c *canvas.C, x, y float64, path string) {
	c.SetColor(canvas.Black)
	p, err := canvas.ParseSVG(path)
	if err != nil {
		panic(err)
	}
	c.DrawPath(x, y, p)

	c.SetColor(color.RGBA{255, 0, 0, 127})
	p = p.Stroke(2, canvas.RoundCapper, canvas.RoundJoiner)
	c.DrawPath(x, y, p)
}

func drawText(c *canvas.C, x, y float64, size float64, text string) {
	face := dejaVuSerif.Face(size)

	metrics := face.Metrics()
	w, h := face.Bounds(text)

	c.SetColor(color.RGBA{0, 0, 0, 20})
	c.DrawPath(x, y, canvas.Rectangle(0, 0, w, h))
	c.SetColor(color.RGBA{0, 0, 0, 100})
	c.DrawPath(x, y, canvas.Rectangle(0, metrics.CapHeight, -2.5, -metrics.Height))
	c.DrawPath(x, y, canvas.Rectangle(0, 0, -2.5, metrics.XHeight))

	c.SetColor(canvas.Black)
	c.SetFont(face)
	c.DrawText(x, y, 0.0, text)
}

func Draw(c *canvas.C) {
	c.Open(200, 80)

	drawText(c, 10, 60, 12.0, "Aap noot mies")

	face := dejaVuSerif.Face(30)
	c.SetFont(face)
	p := face.ToPath("Stroke")
	c.DrawPath(5, 20, p.Stroke(1, canvas.RoundCapper, canvas.RoundJoiner))

	latex, err := canvas.ParseLaTeX(`$y = \sin\left(\frac{x}{180}\pi\right)$`)
	if err != nil {
		panic(err)
	}
	latex.Rotate(-30, 0, 0)
	c.SetColor(canvas.Black)
	c.DrawPath(140, 60, latex)

	ellipse, err := canvas.ParseSVG(fmt.Sprintf("A10 20 30 1 0 20 0z"))
	if err != nil {
		panic(err)
	}
	c.SetColor(canvas.LightGrey)
	c.DrawPath(130, 20, ellipse)
	ellipse = ellipse.Dash(2.0, 4.0, 2.0).Stroke(0.5, canvas.RoundCapper, canvas.RoundJoiner)
	c.SetColor(canvas.Black)
	c.DrawPath(130, 20, ellipse)
}
