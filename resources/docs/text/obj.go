package main

import (
	"image/color"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

var fontFamily *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("times")
	if err := fontFamily.LoadLocalFont("Liberation Serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	p := &canvas.Path{}
	p.LineTo(2.0, 0.0)
	p.LineTo(1.0, 2.0)
	p.Close()

	lenna, err := os.Open("../../lenna.png")
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(lenna)
	if err != nil {
		panic(err)
	}

	c := canvas.New(80, 15)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	face := fontFamily.Face(14.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.DrawText(40.0, 9.0, canvas.NewTextLine(face, "Mixing text with paths and images", canvas.Center))

	face = fontFamily.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	rt := canvas.NewRichText(face)
	rt.WriteString("Where ")
	rt.AddPath(p, canvas.Green)
	rt.WriteString(" and ")
	rt.AddImage(img, canvas.DPMM(200.0))
	rt.WriteString(" refer to foo and bar respectively.")
	ctx.DrawText(40.0, 7.0, rt.ToText(00.0, 00.0, canvas.Center, canvas.Top, 0.0, 0.0))

	renderers.Write("obj.png", c, canvas.DPMM(5.0))
}
