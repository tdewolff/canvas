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

	//p := canvas.ParseSVGPath("m 24.352,74.57 c -0.286667,-0.254667 -0.690333,-0.382 -1.211,-0.382 -3.43517,2.486883 -2.766525,-0.580521 0.062,-0.594 0.864667,0 1.523667,0.216 1.977,0.648 -2.385124,9.355731 1.183897,2.189771 -0.828,0.328 z")
	p := canvas.ParseSVGPath("C20 -20 0 -20 20 0z")
	c.DrawPath(20, 50, p)
	p = p.Stroke(3, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(20, 50, p)

	// face, _ := c.SetFont("DejaVuSerif", 12)
	// c.DrawText(50, 55, "Test")
	// fmt.Println(face.LineHeight())

	face, _ := c.SetFont("DejaVuSerif", 12)

	pText := face.ToPath("a")
	c.DrawPath(20, 80, pText)
	pText = pText.Stroke(0.2, canvas.RoundCapper, canvas.RoundJoiner, 0.01)
	c.SetColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(20, 80, pText)
	// c.DrawText(40, 55+face.LineHeight(), "Test")
	// fmt.Println(face.LineHeight())
}
