package main

import (
	"fmt"
	"syscall/js"

	"github.com/tdewolff/canvas"
)

var fontFamily *canvas.FontFamily

func main() {
	//fontFamily = canvas.NewFontFamily("DejaVu Serif")
	//fontFamily.Use(canvas.CommonLigatures)
	//if err := fontFamily.LoadLocalFont("DejaVuSerif", canvas.FontRegular); err != nil {
	//	panic(err)
	//}

	cvs := js.Global().Get("document").Call("getElementById", "canvas")
	c := canvas.HTMLCanvas(cvs, 200, 80, 3.0)

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
	//ellipse = ellipse.Dash(0.0, 2.0, 4.0, 2.0).Stroke(0.5, canvas.RoundCapper, canvas.RoundJoiner)
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
}
