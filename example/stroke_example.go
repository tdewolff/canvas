package main

import (
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

var dejaVuSerif canvas.Font

func main() {
	var err error
	dejaVuSerif, err = canvas.LoadFontFile("DejaVuSerif", canvas.Regular, "DejaVuSerif.woff")
	if err != nil {
		panic(err)
	}

	c := canvas.New(200, 80)
	Draw(c)

	pngFile, err := os.Create("stroke_example.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := c.WriteImage(144.0)
	err = png.Encode(pngFile, img)
	if err != nil {
		panic(err)
	}
}

func drawStrokedPath(c *canvas.C, x, y float64, path string, cr canvas.Capper, jr canvas.Joiner) {
	p, err := canvas.ParseSVG(path)
	if err != nil {
		panic(err)
	}
	c.SetColor(canvas.Black)
	c.DrawPath(x, y, 0.0, p.Stroke(10.0, cr, jr))
	c.SetColor(canvas.Red)
	c.DrawPath(x, y, 0.0, p.Stroke(0.3, canvas.ButtCapper, canvas.BevelJoiner))
}

func drawText(c *canvas.C, x, y float64, text string) {
	face := dejaVuSerif.Face(18.0)
	c.SetColor(canvas.Black)
	c.DrawText(x, y, 0.0, canvas.NewTextBox(face, text, 0.0, 0.0, canvas.Center, canvas.Top, 0.0))
}

func Draw(c *canvas.C) {
	pathCapper := "M-20 0L0 0"
	pathJoiner := "M-20 -10A25 25 0 0 0 0 0A20 20 0 0 1 -5 -15"

	drawText(c, 20.0, 70.0, "ButtCapper")
	drawStrokedPath(c, 30.0, 60.0, pathCapper, canvas.ButtCapper, canvas.RoundJoiner)

	drawText(c, 70.0, 70.0, "SquareCapper")
	drawStrokedPath(c, 80.0, 60.0, pathCapper, canvas.SquareCapper, canvas.RoundJoiner)

	drawText(c, 120.0, 70.0, "RoundCapper")
	drawStrokedPath(c, 130.0, 60.0, pathCapper, canvas.RoundCapper, canvas.RoundJoiner)

	drawText(c, 23.0, 37.0, "RoundJoiner")
	drawStrokedPath(c, 30.0, 25.0, pathJoiner, canvas.ButtCapper, canvas.RoundJoiner)

	drawText(c, 73.0, 37.0, "BevelJoiner")
	drawStrokedPath(c, 80.0, 25.0, pathJoiner, canvas.ButtCapper, canvas.BevelJoiner)

	drawText(c, 123.0, 37.0, "MiterJoiner")
	drawStrokedPath(c, 130.0, 25.0, pathJoiner, canvas.ButtCapper, canvas.MiterJoiner)

	drawText(c, 173.0, 37.0, "ArcsJoiner")
	drawStrokedPath(c, 180.0, 25.0, pathJoiner, canvas.ButtCapper, canvas.ArcsJoiner)
}
