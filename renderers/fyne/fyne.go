package fyne

import (
	"fyne.io/fyne/v2"
	fyneCanvas "fyne.io/fyne/v2/canvas"
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers/rasterizer"
)

type Fyne struct {
	*canvas.Canvas
	resolution canvas.Resolution
}

// New returns a Fyne renderer.
func New(width, height float64, resolution canvas.Resolution) *Fyne {
	return &Fyne{
		Canvas:     canvas.New(width, height),
		resolution: resolution,
	}
}

func (r *Fyne) Content() fyne.CanvasObject {
	ras := rasterizer.New(r.W, r.H, r.resolution, canvas.LinearColorSpace{})
	r.RenderTo(ras)
	ras.Close()
	return fyneCanvas.NewImageFromImage(ras)
}
