package main

import (
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

var fontFamily *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("times")
	if err := fontFamily.LoadSystemFont("Liberation Serif, serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(160, 80)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))
	draw(ctx)
	c.WriteFile("paths.png", renderers.PNG(canvas.DPMM(5.0)))
}

func drawPos(c *canvas.Context, x, y float64) {
	c.SetFillColor(canvas.Lightblue)
	c.SetStrokeColor(canvas.Transparent)
	c.DrawPath(x-1.5, y-1.5, canvas.Rectangle(3.0, 3.0))
}

func drawControl(c *canvas.Context, x, y float64) {
	c.SetFillColor(canvas.Lightblue)
	c.SetStrokeColor(canvas.Transparent)
	c.DrawPath(x-1.5, y-1.5, canvas.Circle(1.2))
}

func drawPath(c *canvas.Context, x, y float64, path string, moveto bool) {
	p, err := canvas.ParseSVGPath(path)
	if err != nil {
		panic(err)
	}
	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.5)
	if moveto {
		c.SetDashes(0.0, 2.0)
	}
	c.DrawPath(x, y, p)
	if moveto {
		c.SetDashes(0.0)
	}
}

func drawText(c *canvas.Context, x, y float64, text string) {
	face := fontFamily.Face(18.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	c.SetFillColor(canvas.Black)
	c.DrawText(x, y, canvas.NewTextLine(face, text, canvas.Center))
}

func draw(c *canvas.Context) {
	x, y := 20.0, 60.0
	drawText(c, x, y+10, "MoveTo")
	drawPath(c, x, y, "M-15,-5L15,5", true)
	drawPos(c, x-15, y-5)
	drawPos(c, x+15, y+5)
	x += 60

	drawText(c, x, y+10, "LineTo")
	drawPath(c, x, y, "M-15,-5L15,5", false)
	drawPos(c, x-15, y-5)
	drawPos(c, x+15, y+5)
	x += 60

	drawText(c, x, y+10, "QuadTo")
	drawPath(c, x, y, "M-15,-5Q-10,5 15,5", false)
	drawPos(c, x-15, y-5)
	drawControl(c, x-10, y+5)
	drawPos(c, x+15, y+5)
	x = 20
	y -= 40.0

	drawText(c, x, y+10, "CubeTo")
	drawPath(c, x, y, "M-15,-5C-10,5 20,-10 15,5", false)
	drawPos(c, x-15, y-5)
	drawControl(c, x-10, y+5)
	drawControl(c, x+20, y-10)
	drawPos(c, x+15, y+5)
	x += 60

	drawText(c, x, y+10, "ArcTo")
	drawPath(c, x, y, "M-15,-5A10,10 0 1 1 15,5", false)
	drawPos(c, x-15, y-5)
	drawPos(c, x+15, y+5)
	x += 60
}
