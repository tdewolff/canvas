package main

import (
	"fmt"
	"image/png"
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
	err = pdfFile.OutputFileAndClose("example.pdf")
	fmt.Println(err)
}

func Draw(c canvas.C) {
	c.Open(100, 100)

	p := &canvas.Path{}
	p.Rect(30, 55, 10, -18)
	c.DrawPath(p)

	face, _ := c.SetFont("DejaVuSerif", 12)
	c.DrawText(50, 55, "Test")
	fmt.Println(face.LineHeight())

	face, _ = c.SetFont("DejaVuSerif", 18)
	c.DrawText(40, 55, "Test")
	c.DrawText(40, 55+face.LineHeight(), "Test")
	fmt.Println(face.LineHeight())
}
