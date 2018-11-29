package main

import (
	_ "fmt"
	"image/png"
	"image/color"
	"os"

	"github.com/jung-kurt/gofpdf"
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

	img := canvas.NewImage(72.0, fonts)
	Draw(img)
	_ = png.Encode(pngFile, img.Image())

	pdfFile := gofpdf.New("P", "mm", "A4", ".")
	pdfFile.AddFont("DejaVuSerif", "", "DejaVuSerif.json")
	pdf := canvas.NewPDF(pdfFile, fonts)
	Draw(pdf)
	_ = pdfFile.OutputFileAndClose("example.pdf")
}

func Draw(c canvas.C) {
	c.Open(120, 110)

	p := canvas.ParseSVGPath("M60,20V50A20,20 0 0 0 80,70L70,60")
	p = p.Stroke(5.0, canvas.RoundCapper, canvas.RoundJoiner)
	c.DrawPath(p)

	p = canvas.ParseSVGPath("M100 100A50 50 0 0 1 114.64 64.645")
	p = p.Stroke(5.0, canvas.RoundCapper, canvas.RoundJoiner)
	c.SetColor(color.RGBA{255, 127, 0, 0})
	c.DrawPath(p)

	// face, _ := c.SetFont("DejaVuSerif", 12)
	// c.DrawText(50, 55, "Test")
	// fmt.Println(face.LineHeight())

	// face, _ = c.SetFont("DejaVuSerif", 18)
	// c.DrawText(40, 55, "Test")
	// c.DrawText(40, 55+face.LineHeight(), "Test")
	// fmt.Println(face.LineHeight())
}
