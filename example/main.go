package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

func main() {
	fonts := canvas.NewFonts()
	fonts.AddFont("DejaVuSerif", canvas.Regular, "/usr/share/fonts/TTF/Roboto-LightItalic.ttf")

	svgFile, err := os.Create("example.svg")
	if err != nil {
		panic(err)
	}
	svg := canvas.NewSVG(svgFile, fonts)
	Draw(svg)
	svg.Close()
	svgFile.Close()

	pngFile, err := os.Create("example.png")
	if err != nil {
		panic(err)
	}
	img := canvas.NewImage(96.0, fonts)
	Draw(img)
	_ = png.Encode(pngFile, img.Image())
}

func Draw(c canvas.C) {
	c.Open(100, 100)

	p := &canvas.Path{}
	p.Rect(0, 0, 50, 50)
	c.DrawPath(p)

	c.SetFont("DejaVuSerif", 12)
	c.DrawText(50, 55, "Test")
	c.SetFont("DejaVuSerif", 18)
	c.DrawText(40, 55, "Test")

	fmt.Println(c.LineHeight())
}
