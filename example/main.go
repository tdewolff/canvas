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

	p := canvas.ParseSVGPath("M50 50Q50 30 80 30L80 50z")
	//p = p.Stroke(1.0, canvas.RoundCapper, canvas.RoundJoiner)
	c.DrawPath(0, 0, p)

	p = p.Stroke()
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(0, 0, p)

	// face, _ := c.SetFont("DejaVuSerif", 12)
	// c.DrawText(50, 55, "Test")
	// fmt.Println(face.LineHeight())

	// face, _ := c.SetFont("DejaVuSerif", 12)
	// c.DrawText(40, 55, "Testestest")

	// pText := face.ToPath("Testestest")
	// pText.Translate(40, 58)
	// c.DrawPath(pText)
	// c.DrawText(40, 55+face.LineHeight(), "Test")
	// fmt.Println(face.LineHeight())
}
