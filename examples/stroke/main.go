package main

import (
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/tdewolff/canvas"
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
	draw(c)
	c.Fit(1.0)

	pngFile, err := os.Create("out.png")
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

func drawStrokedPath(c *canvas.Canvas, x, y float64, path string, cr canvas.Capper, jr canvas.Joiner) {
	p, err := canvas.ParseSVG(path)
	if err != nil {
		panic(err)
	}

	outerStroke := p.Stroke(strokeWidth, cr, jr)
	c.SetFillColor(canvas.Lightblue)
	c.DrawPath(x, y, outerStroke)
	c.SetFillColor(color.RGBA{155, 194, 207, 255})
	c.DrawPath(x, y, outerStroke.Stroke(0.3, canvas.ButtCapper, canvas.RoundJoiner))
	c.SetFillColor(canvas.Black)
	c.DrawPath(x, y, p.Stroke(0.5, canvas.ButtCapper, canvas.BevelJoiner))
}

func drawText(c *canvas.Canvas, x, y float64, text string) {
	face := fontFamily.Face(18.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	c.SetFillColor(canvas.Black)
	c.DrawText(x, y, canvas.NewTextBox(face, text, 0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0))
}

func draw(c *canvas.Canvas) {
	pathCapper := "M-20 0L0 0"
	pathJoiner := "M-20 -10A25 25 0 0 0 0 0A20 20 0 0 1 -5 -15"

	drawText(c, 20.0, 115.0, "ButtCapper")
	drawStrokedPath(c, 30.0, 100.0, pathCapper, canvas.ButtCapper, canvas.RoundJoiner)

	drawText(c, 70.0, 115.0, "SquareCapper")
	drawStrokedPath(c, 80.0, 100.0, pathCapper, canvas.SquareCapper, canvas.RoundJoiner)

	drawText(c, 120.0, 115.0, "RoundCapper")
	drawStrokedPath(c, 130.0, 100.0, pathCapper, canvas.RoundCapper, canvas.RoundJoiner)

	drawText(c, 23.0, 82.0, "RoundJoiner")
	drawStrokedPath(c, 30.0, 67.0, pathJoiner, canvas.ButtCapper, canvas.RoundJoiner)

	drawText(c, 73.0, 82.0, "BevelJoiner")
	drawStrokedPath(c, 80.0, 67.0, pathJoiner, canvas.ButtCapper, canvas.BevelJoiner)

	drawText(c, 123.0, 82.0, "MiterJoiner")
	drawStrokedPath(c, 130.0, 67.0, pathJoiner, canvas.ButtCapper, canvas.MiterClipJoiner(canvas.BevelJoiner, math.NaN()))

	drawText(c, 173.0, 82.0, "ArcsJoiner")
	drawStrokedPath(c, 180.0, 67.0, pathJoiner, canvas.ButtCapper, canvas.ArcsClipJoiner(canvas.BevelJoiner, math.NaN()))

	strokeWidth = 15.0
	drawText(c, 25.0, 40.0, "Tight corners")
	drawStrokedPath(c, 5.0, 12.0, "C30 0 30 10 25 10", canvas.ButtCapper, canvas.BevelJoiner)
}
