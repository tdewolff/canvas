package main

import (
	"image/color"
	"math"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/rasterizer"
)

var fontFamily *canvas.FontFamily
var strokeWidth = 10.0

func main() {
	fontFamily = canvas.NewFontFamily("times")
	fontFamily.Use(canvas.CommonLigatures)
	if err := fontFamily.LoadLocalFont("NimbusRoman-Regular", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(200, 120)
	ctx := canvas.NewContext(c)
	draw(ctx)
	c.Fit(1.0)
	c.WriteFile("out.png", rasterizer.PNGWriter(5.0))
}

func drawStrokedPath(c *canvas.Context, x, y float64, path string, cr canvas.Capper, jr canvas.Joiner) {
	p, err := canvas.ParseSVG(path)
	if err != nil {
		panic(err)
	}

	outerStroke := p.Stroke(strokeWidth, cr, jr)
	c.SetFillColor(canvas.Lightblue)
	c.DrawPath(x, y, outerStroke)
	c.SetFillColor(color.RGBA{155, 194, 207, 255})
	c.DrawPath(x, y, outerStroke.Stroke(0.3, canvas.ButtCap, canvas.RoundJoin))
	c.SetFillColor(canvas.Black)
	c.DrawPath(x, y, p.Stroke(0.5, canvas.ButtCap, canvas.BevelJoin))
}

func drawText(c *canvas.Context, x, y float64, text string) {
	face := fontFamily.Face(18.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	c.SetFillColor(canvas.Black)
	c.DrawText(x, y, canvas.NewTextBox(face, text, 0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0))
}

func draw(c *canvas.Context) {
	pathCapper := "M-20 0L0 0"
	pathJoiner := "M-20 -10A25 25 0 0 0 0 0A20 20 0 0 1 -5 -15"

	drawText(c, 20.0, 115.0, "ButtCap")
	drawStrokedPath(c, 30.0, 100.0, pathCapper, canvas.ButtCap, canvas.RoundJoin)

	drawText(c, 70.0, 115.0, "SquareCap")
	drawStrokedPath(c, 80.0, 100.0, pathCapper, canvas.SquareCap, canvas.RoundJoin)

	drawText(c, 120.0, 115.0, "RoundCap")
	drawStrokedPath(c, 130.0, 100.0, pathCapper, canvas.RoundCap, canvas.RoundJoin)

	drawText(c, 23.0, 82.0, "RoundJoin")
	drawStrokedPath(c, 30.0, 67.0, pathJoiner, canvas.ButtCap, canvas.RoundJoin)

	drawText(c, 73.0, 82.0, "BevelJoin")
	drawStrokedPath(c, 80.0, 67.0, pathJoiner, canvas.ButtCap, canvas.BevelJoin)

	drawText(c, 123.0, 82.0, "MiterJoin")
	drawStrokedPath(c, 130.0, 67.0, pathJoiner, canvas.ButtCap, canvas.MiterClipJoin(canvas.BevelJoin, math.NaN()))

	drawText(c, 173.0, 82.0, "ArcsJoin")
	drawStrokedPath(c, 180.0, 67.0, pathJoiner, canvas.ButtCap, canvas.ArcsClipJoin(canvas.BevelJoin, math.NaN()))

	strokeWidth = 15.0
	drawText(c, 25.0, 40.0, "Tight corners")
	drawStrokedPath(c, 5.0, 12.0, "C30 0 30 10 25 10", canvas.ButtCap, canvas.BevelJoin)
}
