package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/Seanld/canvas"
	canvasFyne "github.com/Seanld/canvas/renderers/fyne"
)

func main() {
	c := canvasFyne.New(200.0, 100.0, canvas.DPMM(10.0))
	ctx := canvas.NewContext(c)
	if err := canvas.DrawPreview(ctx); err != nil {
		panic(err)
	}

	a := app.New()
	w := a.NewWindow("Canvas")
	w.Resize(fyne.Size{800, 400})
	w.SetContent(c.Content())
	w.ShowAndRun()
}
