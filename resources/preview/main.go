package main

import (
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

func main() {
	c := canvas.New(200, 100)
	ctx := canvas.NewContext(c)
	if err := canvas.DrawPreview(ctx); err != nil {
		panic(err)
	}
	c.WriteFile("preview.png", renderers.PNG(canvas.DPMM(3.2)))
}
