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

	p = p.Stroke(2, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(x, y, p)
}

func Draw(c canvas.C) {
	c.Open(100, 150)

	p := canvas.ParseSVGPath("m 39.516,67.031 -0.202511,2.081751 C 37.457,71.443128 35.427421,71.038866 35.922,66.984 v -1.422 c 0,-1.572667 -0.479333,-2.791333 -1.438,-3.656 -2.158,-1.196 -2.223439,-5.660617 2.774,-1.094 1.505333,1.427333 2.521228,3.513106 2.258,6.219 z")
	p = p.FlattenBeziers(0.01)
	c.DrawPath(30, 10, p)

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
