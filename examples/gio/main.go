package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/gio"
)

func main() {
	go func() {
		w := new(app.Window)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window) error {
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				c := gio.NewContain(gtx, 200.0, 100.0)
				ctx := canvas.NewContext(c)
				if err := canvas.DrawPreview(ctx); err != nil {
					panic(err)
				}
				return c.Dimensions()
			})
			e.Frame(gtx.Ops)
		}
	}
}
