package main

import (
	"image/color"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

var font *canvas.FontFamily

func main() {
	font = canvas.NewFontFamily("font")
	if err := font.LoadFontFile("../Dynalight-Regular.otf", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(65, 27)
	ctx := canvas.NewContext(c)
	draw(ctx)
	c.WriteFile("title.png", renderers.PNG(canvas.DPMM(5.0)))
}

func draw(c *canvas.Context) {
	x := 2.0
	face := font.Face(80.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)

	p, adv, _ := face.ToPath("C")
	c.SetFillColor(color.RGBA{128, 0, 64, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv, _ = face.ToPath("a")
	c.SetFillColor(color.RGBA{192, 0, 64, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv, _ = face.ToPath("n")
	c.SetFillColor(color.RGBA{224, 64, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv, _ = face.ToPath("v")
	c.SetFillColor(color.RGBA{224, 96, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv, _ = face.ToPath("a")
	c.SetFillColor(color.RGBA{224, 128, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	p, adv, _ = face.ToPath("s")
	c.SetFillColor(color.RGBA{224, 160, 0, 255})
	c.DrawPath(x, 4, p)
	x += adv

	//c.SetFillColor(color.RGBA{224, 224, 224, 255})
	//c.DrawPath(2, 2, canvas.Rect{0, 0, x - 2.0, 1.0}.ToPath())
	_ = x
}
