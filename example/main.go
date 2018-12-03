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

	p := canvas.ParseSVGPath("C0 -20 20 -20 20 0z")
	//p := canvas.ParseSVGPath("C20 -20 0 -20 20 0z")
	c.DrawPath(20, 50, p)
	p = p.Stroke(3, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(20, 50, p)

	// face, _ := c.SetFont("DejaVuSerif", 12)
	// c.DrawText(50, 55, "Test")
	// fmt.Println(face.LineHeight())

	c.SetColor(color.RGBA{0, 0, 0, 255})
	face, _ := c.SetFont("DejaVuSerif", 12)
	// c.DrawText(40, 55, "Testestest")

	pText := face.ToPath("Taco")
	//pText = pText.Stroke(3, canvas.RoundCapper, canvas.RoundJoiner, 0.1)
	c.DrawPath(20, 80, pText)
	// c.DrawText(40, 55+face.LineHeight(), "Test")
	// fmt.Println(face.LineHeight())
}
