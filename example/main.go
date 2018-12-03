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

func Draw(c canvas.C) {
	c.Open(100, 150)

	p := canvas.ParseSVGPath("C0 1 0 0 20 0L0 -20z")
	//p := canvas.ParseSVGPath("C20 -20 0 -20 20 0z")
	c.DrawPath(20, 50, p)
	p = p.FlattenBeziers(0.1)
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(20, 50, p)

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
