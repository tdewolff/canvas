package main

import (
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func main() {
	c := canvas.New(1000, 1000)
	ctx := canvas.NewContext(c)

	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(ctx.Width(), ctx.Height()))
	ctx.Fill()

	// Create a triangle path from an SVG path and draw it to the canvas
	triangle, err := canvas.ParseSVG("L600 00L300 600z")
	if err != nil {
		panic(err)
	}

	g := canvas.NewRadialGradient(500, 500, 100, 200, 500, 500)
	g.AddColorStop(0, canvas.Mediumaquamarine)
	g.AddColorStop(1, canvas.Red)

	ctx.Style.GradientInfo = &g
	ctx.DrawPath(200, 200, triangle)
	ctx.Style.GradientInfo = nil

	fontDejaVu := canvas.NewFontFamily("dejavu")
	if err := fontDejaVu.LoadFontFile("../../resources/DejaVuSerif.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}

	f := canvas.NewLinearGradient(0, 0, 300, 200)
	f.AddColorStop(0, canvas.Red)
	f.AddColorStop(1, canvas.Blue)

	face := fontDejaVu.Face(300.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	face.GradientInfo = &f

	ctx.DrawText(10, 30, canvas.NewTextLine(face, "Lorem ipsum", canvas.Left))

	renderers.Write("out.png", c)
}
