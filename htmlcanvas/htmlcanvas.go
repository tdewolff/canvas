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

	r.context.Call("beginPath")
	d := path.Data()
	for i := 0; i < len(d); {
		cmd := d[i]
		switch cmd {
		case canvas.MoveToCmd:
			x, y := d[i+1], d[i+2]
			r.context.Call("moveTo", x*r.dpm, r.height-y*r.dpm)
		case canvas.LineToCmd:
			x, y := d[i+1], d[i+2]
			r.context.Call("lineTo", x*r.dpm, r.height-y*r.dpm)
		case canvas.QuadToCmd:
			cpx, cpy := d[i+1], d[i+2]
			x, y := d[i+3], d[i+4]
			r.context.Call("quadraticCurveTo", cpx*r.dpm, r.height-cpy*r.dpm, x*r.dpm, r.height-y*r.dpm)
		case canvas.CubeToCmd:
			cpx1, cpy1 := d[i+1], d[i+2]
			cpx2, cpy2 := d[i+3], d[i+4]
			x, y := d[i+5], d[i+6]
			r.context.Call("cubicCurveTo", cpx1*r.dpm, r.height-cpy1*r.dpm, cpx2*r.dpm, r.height-cpy2*r.dpm, x*r.dpm, r.height-y*r.dpm)
		case canvas.ArcToCmd:
			//rx, ry, phi := path.d[i+1], path.d[i+2], path.d[i+3]
			//large, sweep := fromArcFlags(path.d[i+4])
			//x, y := path.d[i+5], path.d[i+6]
			// TODO
		case canvas.CloseCmd:
			r.context.Call("closePath")
		}
		i += canvas.CmdLen(cmd)
	}
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
