package htmlcanvas

import (
	"image"
	"syscall/js"

	"github.com/tdewolff/canvas"
)

type htmlCanvas struct {
	context       js.Value
	width, height float64
	dpm           float64
}

func New(canvas js.Value, width, height, dpm float64) *htmlCanvas {
	canvas.Set("width", width*dpm)
	canvas.Set("height", height*dpm)

	context := canvas.Call("getContext", "2d")
	context.Call("clearRect", 0, 0, width*dpm, height*dpm)
	return &htmlCanvas{
		context: context,
		width:   width * dpm,
		height:  height * dpm,
		dpm:     dpm,
	}
}

func (r *htmlCanvas) Size() (float64, float64) {
	return r.width / r.dpm, r.height / r.dpm
}

func (r *htmlCanvas) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	if path.Empty() {
		return
	}
	path = path.Transform(m)
	path = path.ReplaceArcs()

	r.context.Call("beginPath")
	path.Iterate(func(start, end canvas.Point) {
		r.context.Call("moveTo", end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start, end canvas.Point) {
		r.context.Call("lineTo", end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start, cp, end canvas.Point) {
		r.context.Call("quadraticCurveTo", cp.X*r.dpm, r.height-cp.Y*r.dpm, end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start, cp1, cp2, end canvas.Point) {
		r.context.Call("cubicCurveTo", cp1.X*r.dpm, r.height-cp1.Y*r.dpm, cp2.X*r.dpm, r.height-cp2.Y*r.dpm, end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start canvas.Point, rx, ry, rot float64, large, sweep bool, end canvas.Point) {
		panic("arcs should have been replaced")
	}, func(start, end canvas.Point) {
		r.context.Call("closePath")
	})
	r.context.Call("fill")
}

func (r *htmlCanvas) RenderText(text *canvas.Text, m canvas.Matrix) {
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := canvas.DefaultStyle
		style.FillColor = colors[i]
		r.RenderPath(path, style, m)
	}
}

func (r *htmlCanvas) RenderImage(img image.Image, m canvas.Matrix) {
	panic("images not supported in HTML Canvas")
}
