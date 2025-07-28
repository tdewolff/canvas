package main

import (
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func main() {
	c := canvas.New(100.0, 100.0)
	ctx := canvas.NewContext(c)

	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0.0, 0.0, canvas.Rectangle(100.0, 100.0))

	ctx.SetFillColor(canvas.Steelblue)
	ctx.DrawPath(0.0, 0.0, canvas.Rectangle(10.0, 100.0))
	ctx.DrawPath(15.0, 0.0, canvas.Rectangle(10.0, 100.0))
	ctx.DrawPath(75.0, 0.0, canvas.Rectangle(10.0, 100.0))
	ctx.DrawPath(90.0, 0.0, canvas.Rectangle(10.0, 100.0))

	ctx.SetFillColor(canvas.Lightskyblue)
	ctx.DrawPath(25.0, 0.0, canvas.Rectangle(50.0, 100.0))

	p := &canvas.Path{}
	p.ArcTo(25.0, 25.0, 0.0, false, true, 25.0, 25.0)
	p.ArcTo(25.0, 25.0, 0.0, false, true, 0.0, 0.0)
	p.Close()

	ctx.SetFillColor(canvas.Lightcoral)
	for _, rot := range []float64{0.0, 90.0, 180.0, 270.0} {
		ctx.DrawPath(50.0, 25.0, p.Transform(canvas.Identity.Rotate(rot)))
		ctx.DrawPath(50.0, 75.0, p.Transform(canvas.Identity.Rotate(rot)))
	}

	renderers.Write("ceramics.png", c, canvas.DPMM(20.0))
}
