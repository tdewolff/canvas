package main

//go:generate go-bindata -o dejavuserif.go ../../font/DejaVuSerif.ttf

import (
	"fmt"
	"image/color"
	"syscall/js"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/htmlcanvas"
)

var fontFamily *canvas.FontFamily

func main() {
	dejaVuSerif := MustAsset("../../font/DejaVuSerif.ttf")

	fontFamily = canvas.NewFontFamily("DejaVu Serif")
	fontFamily.Use(canvas.CommonLigatures)
	if err := fontFamily.LoadFont(dejaVuSerif, canvas.FontRegular); err != nil {
		panic(err)
	}

	cvs := js.Global().Get("document").Call("getElementById", "canvas")
	c := htmlcanvas.New(cvs, 200, 80, 3.0)

	ctx := canvas.NewContext(c)
	draw(ctx)
}

func draw(c *canvas.Context) {
	// Draw a shape
	shape, err := canvas.ParseSVG(fmt.Sprintf("L10 0L10 10Q5 15 0 10z"))
	if err != nil {
		panic(err)
	}
	c.SetFillColor(canvas.Whitesmoke)
	c.DrawPath(110, 40, shape)

	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.5)
	c.SetStrokeCapper(canvas.RoundCapper)
	c.SetStrokeJoiner(canvas.RoundJoiner)
	c.SetDashes(0.0, 2.0, 4.0, 2.0, 2.0, 4.0, 2.0)
	c.DrawPath(110, 40, shape)
	c.SetStrokeColor(canvas.Transparent)

	// Draw a raster image
	//lenna, err := os.Open("../lenna.png")
	//if err != nil {
	//	panic(err)
	//}
	//img, err := png.Decode(lenna)
	//if err != nil {
	//	panic(err)
	//}
	//c.DrawImage(105.0, 15.0, img, 25.6)

	// Draw text
	face := fontFamily.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	phrase := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Morbi egestas, augue eget blandit laoreet, dolor lorem interdum ante, quis consectetur lorem massa vitae nulla. Sed cursus tellus id venenatis suscipit. Nunc volutpat imperdiet ipsum vel varius."

	text := canvas.NewTextBox(face, phrase, 60.0, 35.0, canvas.Justify, canvas.Top, 0.0, 0.0)
	rect := text.Bounds()
	rect.Y = 0.0
	rect.H = -35.0
	//c.SetFillColor(canvas.Whitesmoke)
	//c.DrawPath(10.0, 40.0, rect.ToPath())
	c.SetFillColor(canvas.Black)
	c.DrawText(10.0, 40.0, text)
}
