package main

import (
	_ "fmt"
	"image/color"
	_ "image/png"
	"os"

	_ "github.com/jung-kurt/gofpdf"
	"github.com/tdewolff/canvas"
)

func main() {
	fonts := canvas.NewFonts()
	fonts.Add("DejaVuSerif", canvas.Regular, "DejaVuSerif.ttf")

	svgFile, err := os.Create("example.svg")
	if err != nil {
		panic(err)
	}
	defer svgFile.Close()

	pngFile, err := os.Create("example.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	svg := canvas.NewSVG(svgFile, fonts)
	Draw(svg)
	svg.Close()

	// img := canvas.NewImage(72.0, fonts)
	// Draw(img)
	// _ = png.Encode(pngFile, img.Image())

	// pdfFile := gofpdf.New("P", "mm", "A4", ".")
	// pdfFile.AddFont("DejaVuSerif", "", "DejaVuSerif.json")
	// pdf := canvas.NewPDF(pdfFile, fonts)
	// Draw(pdf)
	// _ = pdfFile.OutputFileAndClose("example.pdf")
}

func drawStrokedPath(c canvas.C, x, y float64, path string) {
	c.SetColor(canvas.Black)
	p := canvas.ParseSVGPath(path)
	c.DrawPath(x, y, p)

	c.SetColor(color.RGBA{255, 0, 0, 127})
	p = p.Stroke(2, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.DrawPath(x, y, p)
}

func Draw(c canvas.C) {
	c.Open(100, 150)

	drawStrokedPath(c, 5, 20, "C0 -20 20 -20 20 0z")
	drawStrokedPath(c, 30, 20, "C10 -20 10 -20 20 0z")
	drawStrokedPath(c, 55, 20, "C20 -20 0 -20 20 0z")
	drawStrokedPath(c, 5, 50, "C0 0 0 -20 20 0z")
	drawStrokedPath(c, 30, 50, "C0 -20 0 0 20 0z")
	drawStrokedPath(c, 55, 50, "C0 -20 0 0 0 0z")
	drawStrokedPath(c, 80, 50, "C0 0 0 -20 0 0z")
	drawStrokedPath(c, 80, 50, "C0 0 0 0 0 0z")

	face, _ := c.SetFont("DejaVuSerif", 40)
	pText := face.ToPath("a")
	c.DrawPath(20, 80, pText)

	//pText = pText.Stroke(0.2, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(20, 80, pText)
}
