// +build js

package htmlcanvas

import (
	"image"
	"math"
	"syscall/js"

	"github.com/tdewolff/canvas"
)

// HTMLCanvas is an HTMLCanvas renderer.
type HTMLCanvas struct {
	ctx           js.Value
	width, height float64
	dpm           float64
	style         canvas.Style
}

// New returns an HTMLCanvas renderer.
func New(c js.Value, width, height, dpm float64) *HTMLCanvas {
	c.Set("width", width*dpm)
	c.Set("height", height*dpm)

	ctx := c.Call("getContext", "2d")
	ctx.Call("clearRect", 0, 0, width*dpm, height*dpm)
	ctx.Set("imageSmoothingEnabled", true)
	ctx.Set("imageSmoothingQuality", "high")
	style := canvas.DefaultStyle
	style.StrokeWidth = 0
	return &HTMLCanvas{
		ctx:    ctx,
		width:  width * dpm,
		height: height * dpm,
		dpm:    dpm,
		style:  style,
	}
}

// Size returns the size of the canvas in millimeters.
func (r *HTMLCanvas) Size() (float64, float64) {
	return r.width / r.dpm, r.height / r.dpm
}

func (r *HTMLCanvas) writePath(path *canvas.Path) {
	r.ctx.Call("beginPath")
	for scanner := path.Scanner(); scanner.Scan(); {
		end := scanner.End()
		switch scanner.Cmd() {
		case canvas.MoveToCmd:
			r.ctx.Call("moveTo", end.X*r.dpm, r.height-end.Y*r.dpm)
		case canvas.LineToCmd:
			r.ctx.Call("lineTo", end.X*r.dpm, r.height-end.Y*r.dpm)
		case canvas.QuadToCmd:
			cp := scanner.CP1()
			r.ctx.Call("quadraticCurveTo", cp.X*r.dpm, r.height-cp.Y*r.dpm, end.X*r.dpm, r.height-end.Y*r.dpm)
		case canvas.CubeToCmd:
			cp1, cp2 := scanner.CP1(), scanner.CP2()
			r.ctx.Call("bezierCurveTo", cp1.X*r.dpm, r.height-cp1.Y*r.dpm, cp2.X*r.dpm, r.height-cp2.Y*r.dpm, end.X*r.dpm, r.height-end.Y*r.dpm)
		case canvas.ArcToCmd:
			panic("arcs should have been replaced")
		case canvas.CloseCmd:
			r.ctx.Call("closePath")
		}
	}
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *HTMLCanvas) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	if path.Empty() {
		return
	}

	strokeUnsupported := false
	if m.IsSimilarity() {
		scale := math.Sqrt(math.Abs(m.Det()))
		style.StrokeWidth *= scale
		style.DashOffset *= scale
		dashes := make([]float64, len(style.Dashes))
		for i := range style.Dashes {
			dashes[i] = style.Dashes[i] * scale
		}
		style.Dashes = dashes
	} else {
		strokeUnsupported = true
	}

	if style.HasFill() || style.HasStroke() && !strokeUnsupported {
		r.writePath(path.Transform(m).ReplaceArcs())
	}

	if style.HasFill() {
		if style.FillColor != r.style.FillColor {
			r.ctx.Set("fillStyle", canvas.CSSColor(style.FillColor).String())
			r.style.FillColor = style.FillColor
		}
		r.ctx.Call("fill")
	}
	if style.HasStroke() && !strokeUnsupported {
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
			r.style.StrokeCapper = style.StrokeCapper
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
			r.style.StrokeJoiner = style.StrokeJoiner
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
			r.style.Dashes = style.Dashes
		}

		if style.DashOffset != r.style.DashOffset {
			r.ctx.Set("lineDashOffset", style.DashOffset*r.dpm)
			r.style.DashOffset = style.DashOffset
		}

		if style.StrokeWidth != r.style.StrokeWidth {
			r.ctx.Set("lineWidth", style.StrokeWidth*r.dpm)
			r.style.StrokeWidth = style.StrokeWidth
		}
		if style.StrokeColor != r.style.StrokeColor {
			r.ctx.Set("strokeStyle", canvas.CSSColor(style.StrokeColor).String())
			r.style.StrokeColor = style.StrokeColor
		}
		r.ctx.Call("stroke")
	} else if style.HasStroke() {
		// stroke settings unsupported by HTML Canvas, draw stroke explicitly
		if style.IsDashed() {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)
		r.writePath(path.Transform(m).ReplaceArcs())
		if style.StrokeColor != r.style.FillColor {
			r.ctx.Set("fillStyle", canvas.CSSColor(style.StrokeColor).String())
			r.style.FillColor = style.StrokeColor
		}
		r.ctx.Call("fill")
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *HTMLCanvas) RenderText(text *canvas.Text, m canvas.Matrix) {
	text.RenderAsPath(r, m, canvas.DefaultResolution)
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

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *HTMLCanvas) RenderImage(img image.Image, m canvas.Matrix) {
	size := img.Bounds().Size()
	sp := img.Bounds().Min // starting point
	buf := make([]byte, 4*size.X*size.Y)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 4
			r, g, b, a := img.At(sp.X+x, sp.Y+y).RGBA()
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
