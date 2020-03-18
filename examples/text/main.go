package main

import (
	"image/color"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/rasterizer"
)

var fontFamily *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("times")
	fontFamily.Use(canvas.CommonLigatures)
	if err := fontFamily.LoadLocalFont("NimbusRoman-Regular", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(265, 90)
	ctx := canvas.NewContext(c)
	draw(ctx)
	c.WriteFile("out.png", rasterizer.PNGWriter(5.0))
}

func drawText(c *canvas.Context, x, y float64, halign, valign canvas.TextAlign, indent float64) {
	face := fontFamily.Face(6.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	phrase := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Morbi egestas, augue eget blandit laoreet, dolor lorem interdum ante, quis consectetur lorem massa vitae nulla. Sed cursus tellus id venenatis suscipit. Nunc volutpat imperdiet ipsum vel varius. Pellentesque mattis viverra odio, ullamcorper iaculis massa tristique imperdiet. Aliquam posuere nisl tortor, in scelerisque elit eleifend sed. Suspendisse in risus aliquam leo vestibulum gravida. Sed ipsum massa, fringilla at pellentesque vitae, dictum nec libero. Morbi lorem ante, facilisis a justo vel, mollis fringilla massa. Mauris aliquet imperdiet magna, ac tempor sem fringilla sed."

	text := canvas.NewTextBox(face, phrase, 60.0, 35.0, halign, valign, indent, 0.0)
	rect := text.Bounds()
	rect.Y = 0.0
	rect.H = -35.0
	c.SetFillColor(canvas.Whitesmoke)
	c.DrawPath(x, y, rect.ToPath())
	c.SetFillColor(canvas.Black)
	c.DrawText(x, y, text)
}

func draw(c *canvas.Context) {
	face := fontFamily.Face(14.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	c.SetFillColor(canvas.Black)
	c.DrawText(132.5, 88.0, canvas.NewTextBox(face, "Different horizontal and vertical alignments with indent", 0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0))

	drawText(c, 5.0, 80.0, canvas.Left, canvas.Top, 10.0)
	drawText(c, 70.0, 80.0, canvas.Center, canvas.Top, 10.0)
	drawText(c, 135.0, 80.0, canvas.Right, canvas.Top, 10.0)
	drawText(c, 200.0, 80.0, canvas.Justify, canvas.Top, 10.0)
	drawText(c, 5.0, 40.0, canvas.Left, canvas.Top, 10.0)
	drawText(c, 70.0, 40.0, canvas.Left, canvas.Center, 10.0)
	drawText(c, 135.0, 40.0, canvas.Left, canvas.Bottom, 10.0)
	drawText(c, 200.0, 40.0, canvas.Left, canvas.Justify, 10.0)
}
