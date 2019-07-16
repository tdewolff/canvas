package main

import (
	"image/color"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

var font *canvas.FontFamily

func main() {
	font = canvas.NewFontFamily("font")
	font.Use(canvas.CommonLigatures)
	//if err := font.LoadLocalFont("Monoton", canvas.FontRegular); err != nil {
	//if err := font.LoadLocalFont("Rye", canvas.FontRegular); err != nil {
	if err := font.LoadLocalFont("Dynalight", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(65, 27)
	draw(c)

	////////////////

	pngFile, err := os.Create("title.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := c.WriteImage(5.0)
	err = png.Encode(pngFile, img)
	if err != nil {
		panic(err)
	}
}

func draw(c *canvas.Canvas) {
	x := 2.0
	face := font.Face(80.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)

	p, adv := face.ToPath("C")
	c.SetFillColor(color.RGBA{128, 0, 64, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv = face.ToPath("a")
	c.SetFillColor(color.RGBA{192, 0, 64, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv = face.ToPath("n")
	c.SetFillColor(color.RGBA{224, 64, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv = face.ToPath("v")
	c.SetFillColor(color.RGBA{224, 96, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv = face.ToPath("a")
	c.SetFillColor(color.RGBA{224, 128, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv = face.ToPath("s")
	c.SetFillColor(color.RGBA{224, 160, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	c.SetFillColor(color.RGBA{224, 224, 224, 255})
	c.DrawPath(2, 2, canvas.Rect{0, 0, x - 2.0, 1.0}.ToPath())
}
