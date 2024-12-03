package main

import (
	"os"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers/tex"
)

func main() {
	f, err := os.Create("out.tex")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c := tex.New(f, 20, 10)
	defer c.Close()

	ctx := canvas.NewContext(c)
	ctx.SetCoordView(canvas.Identity.Scale(0.5, 0.5))
	ctx.SetView(canvas.Identity.Scale(0.5, 0.5))
	if err := canvas.DrawPreview(ctx); err != nil {
		panic(err)
	}
}
