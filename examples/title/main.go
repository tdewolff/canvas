package main

import (
	"image/color"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/rasterizer"
)

var font *canvas.FontFamily

func main() {
	font = canvas.NewFontFamily("font")
	font.Use(canvas.CommonLigatures)
	if err := font.LoadLocalFont("Dynalight", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(65, 27)
	ctx := canvas.NewContext(c)
	draw(ctx)
	c.WriteFile("out.png", rasterizer.PNGWriter(5.0))
}

func draw(c *canvas.Context) {
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
