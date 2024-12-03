//go:build js

package main

//go:generate go-bindata -o files.go /usr/share/fonts/TTF/DejaVuSerif.ttf /usr/share/fonts/TTF/DejaVuSans.ttf /usr/share/fonts/noto/NotoSerifDevanagari-Regular.ttf ../../resources/lenna.png

import (
	"syscall/js"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers/htmlcanvas"
)

func main() {
	latin := MustAsset("usr/share/fonts/TTF/DejaVuSerif.ttf")
	arabic := MustAsset("usr/share/fonts/TTF/DejaVuSans.ttf")
	devanagari := MustAsset("usr/share/fonts/noto/NotoSerifDevanagari-Regular.ttf")
	lenna := MustAsset("../../resources/lenna.png")

	cvs := js.Global().Get("document").Call("getElementById", "canvas")
	c := htmlcanvas.New(cvs, 200, 100, 5.0)
	ctx := canvas.NewContext(c)
	if err := canvas.DrawPreviewWithAssets(ctx, latin, arabic, devanagari, lenna); err != nil {
		panic(err)
	}

	alive := make(chan bool)
	<-alive
}
