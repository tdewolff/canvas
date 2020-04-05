// +build js

package htmlcanvas

import (
	"image"
	"math"
	"syscall/js"

	"github.com/tdewolff/canvas"
)

type htmlCanvas struct {
	ctx           js.Value
	width, height float64
	dpm           float64
	style         canvas.Style
}

func New(c js.Value, width, height, dpm float64) *htmlCanvas {
	c.Set("width", width*dpm)
	c.Set("height", height*dpm)

	ctx := c.Call("getContext", "2d")
	ctx.Call("clearRect", 0, 0, width*dpm, height*dpm)
	ctx.Set("imageSmoothingEnabled", true)
	ctx.Set("imageSmoothingQuality", "high")
	return &htmlCanvas{
		ctx:    ctx,
		width:  width * dpm,
		height: height * dpm,
		dpm:    dpm,
		style:  canvas.DefaultStyle,
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

	r.ctx.Call("beginPath")
	path.Iterate(func(start, end canvas.Point) {
		r.ctx.Call("moveTo", end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start, end canvas.Point) {
		r.ctx.Call("lineTo", end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start, cp, end canvas.Point) {
		r.ctx.Call("quadraticCurveTo", cp.X*r.dpm, r.height-cp.Y*r.dpm, end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start, cp1, cp2, end canvas.Point) {
		r.ctx.Call("bezierCurveTo", cp1.X*r.dpm, r.height-cp1.Y*r.dpm, cp2.X*r.dpm, r.height-cp2.Y*r.dpm, end.X*r.dpm, r.height-end.Y*r.dpm)
	}, func(start canvas.Point, rx, ry, rot float64, large, sweep bool, end canvas.Point) {
		panic("arcs should have been replaced")
	}, func(start, end canvas.Point) {
		r.ctx.Call("closePath")
	})

	if style.FillColor.A != 0 {
		if style.FillColor != r.style.FillColor {
			r.ctx.Set("fillStyle", canvas.CSSColor(style.FillColor).String())
		}
		r.ctx.Call("fill")
	}
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		if style.StrokeCapper != r.style.StrokeCapper {
			if _, ok := style.StrokeCapper.(canvas.RoundCapper); ok {
				r.ctx.Set("lineCap", "round")
			} else if _, ok := style.StrokeCapper.(canvas.SquareCapper); ok {
				r.ctx.Set("lineCap", "square")
			} else if _, ok := style.StrokeCapper.(canvas.ButtCapper); ok {
				r.ctx.Set("lineCap", "butt")
			} else {
				panic("HTML Canvas: line cap not support")
			}
		}

		if style.StrokeJoiner != r.style.StrokeJoiner {
			if _, ok := style.StrokeJoiner.(canvas.BevelJoiner); ok {
				r.ctx.Set("lineJoin", "bevel")
			} else if _, ok := style.StrokeJoiner.(canvas.RoundJoiner); ok {
				r.ctx.Set("lineJoin", "round")
			} else if miter, ok := style.StrokeJoiner.(canvas.MiterJoiner); ok && !math.IsNaN(miter.Limit) && miter.GapJoiner == canvas.BevelJoin {
				r.ctx.Set("lineJoin", "miter")
				r.ctx.Set("miterLimit", miter.Limit)
			} else {
				panic("HTML Canvas: line join not support")
			}
		}

		dashesEqual := len(style.Dashes) == len(r.style.Dashes)
		if dashesEqual {
			for i, dash := range style.Dashes {
				if dash != r.style.Dashes[i] {
					dashesEqual = false
					break
				}
			}
		}

		if !dashesEqual {
			dashes := []interface{}{}
			for _, dash := range style.Dashes {
				dashes = append(dashes, dash*r.dpm)
			}
			jsDashes := js.Global().Get("Array").New(dashes...)
			r.ctx.Call("setLineDash", jsDashes)
		}

		if style.DashOffset != r.style.DashOffset {
			r.ctx.Set("lineDashOffset", style.DashOffset*r.dpm)
		}

		if style.StrokeWidth != r.style.StrokeWidth {
			r.ctx.Set("lineWidth", style.StrokeWidth*r.dpm)
		}
		if style.StrokeColor != r.style.StrokeColor {
			r.ctx.Set("strokeStyle", canvas.CSSColor(style.StrokeColor).String())
		}
		r.ctx.Call("stroke")
	}
	r.style = style
}

func (r *htmlCanvas) RenderText(text *canvas.Text, m canvas.Matrix) {
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := canvas.DefaultStyle
		style.FillColor = colors[i]
		r.RenderPath(path, style, m)
	}
}

func jsAwait(v js.Value) (result js.Value, ok bool) {
	// COPIED FROM https://go-review.googlesource.com/c/go/+/150917/
	if v.Type() != js.TypeObject || v.Get("then").Type() != js.TypeFunction {
		return v, true
	}

	done := make(chan struct{})

	onResolve := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		ok = true
		close(done)
		return nil
	})
	defer onResolve.Release()

	onReject := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		ok = false
		close(done)
		return nil
	})
	defer onReject.Release()

	v.Call("then", onResolve, onReject)
	<-done
	return
}

func (r *htmlCanvas) RenderImage(img image.Image, m canvas.Matrix) {
	size := img.Bounds().Size()
	buf := make([]byte, 4*size.X*size.Y)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 4
			r, g, b, a := img.At(x, y).RGBA()
			alpha := float64(a>>8) / 256.0
			buf[i+0] = byte(float64(r>>8) / alpha)
			buf[i+1] = byte(float64(g>>8) / alpha)
			buf[i+2] = byte(float64(b>>8) / alpha)
			buf[i+3] = byte(a >> 8)
		}
	}
	jsBuf := js.Global().Get("Uint8Array").New(len(buf))
	js.CopyBytesToJS(jsBuf, buf)
	jsBufClamped := js.Global().Get("Uint8ClampedArray").New(jsBuf)
	imageData := js.Global().Get("ImageData").New(jsBufClamped, size.X, size.Y)
	imageBitmapPromise := js.Global().Call("createImageBitmap", imageData)
	imageBitmap, ok := jsAwait(imageBitmapPromise)
	if !ok {
		panic("error while waiting for createImageBitmap promise")
	}

	origin := m.Dot(canvas.Point{0, float64(img.Bounds().Size().Y)}).Mul(r.dpm)
	m = m.Scale(r.dpm, r.dpm)
	r.ctx.Call("setTransform", m[0][0], m[0][1], m[1][0], m[1][1], origin.X, r.height-origin.Y)
	r.ctx.Call("drawImage", imageBitmap, 0, 0)
	r.ctx.Call("setTransform", 1.0, 0.0, 0.0, 1.0, 0.0, 0.0)
}
