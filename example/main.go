package main

import (
	"fmt"
	_ "fmt"
	"image/color"
	"image/png"
	_ "image/png"
	"os"

	_ "github.com/jung-kurt/gofpdf"
	"github.com/tdewolff/canvas"
)

func main() {
	svgFile, err := os.Create("example.svg")
	if err != nil {
		panic(err)
	}
	defer svgFile.Close()

	svg := canvas.NewSVG(svgFile)
	svg.AddFontFile("DejaVuSerif", canvas.Regular, "Cantarell-Regular.otf")
	//svg.AddFontFile("DejaVuSerif", canvas.Regular, "DejaVuSerif.ttf")
	Draw(svg)
	svg.Close()

	pngFile, err := os.Create("example.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := canvas.NewImage(72.0)
	img.AddFontFile("DejaVuSerif", canvas.Regular, "DejaVuSerif.ttf")
	img.AddFontFile("DejaVuSerif", canvas.Regular, "Cantarell-Regular.otf")
	Draw(img)
	_ = png.Encode(pngFile, img.Image())

	// pdfFile := gofpdf.New("P", "mm", "A4", ".")
	// pdfFile.AddFont("DejaVuSerif", "", "DejaVuSerif.json")
	// pdf := canvas.NewPDF(pdfFile, fonts)
	// Draw(pdf)
	// _ = pdfFile.OutputFileAndClose("example.pdf")
}

func drawStrokedPath(c canvas.C, x, y float64, path string) {
	c.SetColor(canvas.Black)
	p, err := canvas.ParseSVGPath(path)
	if err != nil {
		panic(err)
	}
	c.DrawPath(x, y, p)

	c.SetColor(color.RGBA{255, 0, 0, 127})
	p = p.Stroke(2, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.DrawPath(x, y, p)
}

func drawText(c canvas.C, x, y float64, size float64, text string) {
	font, err := c.Font("DejaVuSerif")
	if err != nil {
		panic(err.Error())
	}
	face := font.Face(size)

	metrics := face.Metrics()
	w, h := face.BBox(text)
	fmt.Println(metrics, w, h)

	c.SetColor(canvas.Red)
	c.DrawPath(x, y, canvas.Rectangle(0, 0, w, h))
	c.SetColor(canvas.Lime)
	c.DrawPath(x, y, canvas.Rectangle(0, 0, -5.0, -12.0))
	c.SetColor(canvas.Blue)
	c.DrawPath(x, y, canvas.Rectangle(0, 0, -2.5, metrics.CapHeight))
	c.SetColor(canvas.Yellow)
	c.DrawPath(x, y, canvas.Rectangle(-2.5, 0, -2.5, metrics.XHeight))

	c.SetColor(canvas.Black)
	c.SetFont(face)
	c.DrawText(x, y, text)

	p := face.ToPath(text)
	c.DrawPath(x, y+size, p)
}

func Draw(c canvas.C) {
	c.Open(400, 150)

	//drawStrokedPath(c, 5, 20, "C0 -20 20 -20 20 0z")
	//drawStrokedPath(c, 30, 20, "C10 -20 10 -20 20 0z")
	//drawStrokedPath(c, 55, 20, "C20 -20 0 -20 20 0z")
	//drawStrokedPath(c, 5, 50, "C0 0 0 -20 20 0z")
	//drawStrokedPath(c, 30, 50, "C0 -20 0 0 20 0z")
	//drawStrokedPath(c, 55, 50, "C0 -20 0 0 0 0z")
	//drawStrokedPath(c, 80, 50, "C0 0 0 -20 0 0z")
	//drawStrokedPath(c, 80, 50, "C0 0 0 0 0 0z")

	drawText(c, 10, 40, 12.0, "10")

	latex, err := canvas.ParseLaTeX(`$y = \left(\frac{5}{x}\right)$`)
	if err != nil {
		panic(err)
	}
	c.DrawPath(0, 0, latex)
}
