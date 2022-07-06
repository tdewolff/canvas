package main

import (
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func main() {
	// Create new canvas of dimension 100x100 mm
	c := canvas.New(1000, 1000)

	// Create a canvas context used to keep drawing state
	ctx := canvas.NewContext(c)

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

	face := fontDejaVu.Face(100.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)

	ctx.SetFillColor(canvas.Mediumaquamarine)
	ctx.Style.GradientInfo = &g
	ctx.DrawText(10, 10, canvas.NewTextLine(face, "Lorem ipsum", canvas.Left))

	// Rasterize the canvas and write to a PNG file with 3.2 dots-per-mm (320x320 px)
	renderers.Write("out.png", c)
}
