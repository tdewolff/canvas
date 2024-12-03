package main

import (
	"image/color"
	"math"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

var fontFamily *canvas.FontFamily
var strokeWidth = 10.0

func main() {
	fontFamily = canvas.NewFontFamily("latin")
	if err := fontFamily.LoadSystemFont("Liberation Serif, serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(200, 120)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))
	draw(ctx)
	c.WriteFile("stroke.png", renderers.PNG(canvas.DPMM(5.0)))
}

func drawStrokedPath(c *canvas.Context, x, y float64, path string, cr canvas.Capper, jr canvas.Joiner) {
	p, err := canvas.ParseSVGPath(path)
	if err != nil {
		panic(err)
	}

	outerStroke := p.Stroke(strokeWidth, cr, jr, 0.01)
	c.SetFillColor(canvas.Lightblue)
	c.DrawPath(x, y, outerStroke)
	c.SetFillColor(color.RGBA{155, 194, 207, 255})
	c.DrawPath(x, y, outerStroke.Stroke(0.3, canvas.ButtCap, canvas.RoundJoin, 0.01))
	c.SetFillColor(canvas.Black)
	c.DrawPath(x, y, p.Stroke(0.5, canvas.ButtCap, canvas.BevelJoin, 0.01))
}

func drawText(c *canvas.Context, x, y float64, text string) {
	face := fontFamily.Face(18.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	c.SetFillColor(canvas.Black)
	c.DrawText(x, y, canvas.NewTextLine(face, text, canvas.Center))
}

func draw(c *canvas.Context) {
	pathCapper := "M-20 0L0 0"
	pathJoiner := "M-20 -10A25 25 0 0 0 0 0A20 20 0 0 1 -5 -15"

	drawText(c, 20.0, 109.0, "ButtCap")
	drawStrokedPath(c, 30.0, 100.0, pathCapper, canvas.ButtCap, canvas.RoundJoin)

	drawText(c, 70.0, 109.0, "SquareCap")
	drawStrokedPath(c, 80.0, 100.0, pathCapper, canvas.SquareCap, canvas.RoundJoin)

	drawText(c, 120.0, 109.0, "RoundCap")
	drawStrokedPath(c, 130.0, 100.0, pathCapper, canvas.RoundCap, canvas.RoundJoin)

	drawText(c, 23.0, 77.0, "RoundJoin")
	drawStrokedPath(c, 30.0, 67.0, pathJoiner, canvas.ButtCap, canvas.RoundJoin)

	drawText(c, 73.0, 77.0, "BevelJoin")
	drawStrokedPath(c, 80.0, 67.0, pathJoiner, canvas.ButtCap, canvas.BevelJoin)

	drawText(c, 123.0, 77.0, "MiterJoin")
	drawStrokedPath(c, 130.0, 67.0, pathJoiner, canvas.ButtCap, canvas.MiterJoiner{canvas.BevelJoin, math.NaN()})

	drawText(c, 173.0, 77.0, "ArcsJoin")
	drawStrokedPath(c, 180.0, 67.0, pathJoiner, canvas.ButtCap, canvas.ArcsJoiner{canvas.BevelJoin, math.NaN()})

	drawText(c, 123.0, 35.0, "MiterClipJoin")
	drawStrokedPath(c, 130.0, 25.0, pathJoiner, canvas.ButtCap, canvas.MiterJoiner{nil, 1.5})

	drawText(c, 173.0, 35.0, "ArcsClipJoin")
	drawStrokedPath(c, 180.0, 25.0, pathJoiner, canvas.ButtCap, canvas.ArcsJoiner{nil, 1.5})

	strokeWidth = 15.0
	drawText(c, 25.0, 35.0, "Tight corners")
	drawStrokedPath(c, 5.0, 12.0, "C30 0 30 10 25 10", canvas.ButtCap, canvas.BevelJoin)
}
