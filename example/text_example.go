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

	c := canvas.New(265, 90)
	Draw(c)

	pngFile, err := os.Create("text_example.png")
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

func drawText(c *canvas.Canvas, x, y float64, halign, valign canvas.TextAlign, indent float64) {
	face := dejaVuSerif.Face(6.0)
	phrase := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Morbi egestas, augue eget blandit laoreet, dolor lorem interdum ante, quis consectetur lorem massa vitae nulla. Sed cursus tellus id venenatis suscipit. Nunc volutpat imperdiet ipsum vel varius. Pellentesque mattis viverra odio, ullamcorper iaculis massa tristique imperdiet. Aliquam posuere nisl tortor, in scelerisque elit eleifend sed."

	text := canvas.NewTextBox(face, canvas.Black, phrase, 60.0, 35.0, halign, valign, indent, 0.0)
	rect := text.Bounds()
	rect.Y = 0.0
	rect.H = -35.0
	c.SetFillColor(canvas.Whitesmoke)
	c.DrawPath(x, y, rect.ToPath())
	c.SetFillColor(canvas.Black)
	c.DrawText(x, y, text)
}

func Draw(c *canvas.Canvas) {
	face := dejaVuSerif.Face(14.0)
	c.SetFillColor(canvas.Black)
	c.DrawText(132.5, 90.0, canvas.NewTextBox(face, canvas.Black, "Different horizontal and vertical alignments with indent", 0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0))

	drawText(c, 5.0, 80.0, canvas.Left, canvas.Top, 10.0)
	drawText(c, 70.0, 80.0, canvas.Center, canvas.Top, 10.0)
	drawText(c, 135.0, 80.0, canvas.Right, canvas.Top, 10.0)
	drawText(c, 200.0, 80.0, canvas.Justify, canvas.Top, 10.0)
	drawText(c, 5.0, 40.0, canvas.Left, canvas.Top, 10.0)
	drawText(c, 70.0, 40.0, canvas.Left, canvas.Center, 10.0)
	drawText(c, 135.0, 40.0, canvas.Left, canvas.Bottom, 10.0)
	drawText(c, 200.0, 40.0, canvas.Left, canvas.Justify, 10.0)
}
