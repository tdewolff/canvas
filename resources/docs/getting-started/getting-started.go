package main

import (
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/rasterizer"
)

func main() {
	// Create new canvas of dimension 100x100 mm
	c := canvas.New(100, 100)

	// Create a canvas context used to keep drawing state
	ctx := canvas.NewContext(c)

	// Create a triangle path from an SVG path and draw it to the canvas
	triangle, err := canvas.ParseSVG("L60 0L30 60z")
	if err != nil {
		panic(err)
	}
	ctx.DrawPath(20, 20, triangle)

	// Rasterize the canvas and write to a PNG file with 3.2 dots-per-mm (320x320 px)
	c.WriteFile("getting-started.png", rasterizer.PNGWriter(3.2))
}
