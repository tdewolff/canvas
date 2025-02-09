package gio

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/tdewolff/canvas"
)

type Gio struct {
	ops            *op.Ops
	width, height  float64
	xScale, yScale float64
	dimensions     layout.Dimensions
}

// New returns a Gio renderer of fixed size.
func New(gtx layout.Context, width, height float64) *Gio {
	dimensions := layout.Dimensions{Size: image.Point{int(width + 0.5), int(height + 0.5)}}
	return &Gio{
		ops:        gtx.Ops,
		width:      width,
		height:     height,
		xScale:     1.0,
		yScale:     1.0,
		dimensions: dimensions,
	}
}

// NewContain returns a Gio renderer that fills the constraints either horizontally or vertically, whichever is met first.
func NewContain(gtx layout.Context, width, height float64) *Gio {
	xScale := float64(gtx.Constraints.Max.X-gtx.Constraints.Min.X) / width
	yScale := float64(gtx.Constraints.Max.Y-gtx.Constraints.Min.Y) / height
	if yScale < xScale {
		xScale = yScale
	} else {
		yScale = xScale
	}

	dimensions := layout.Dimensions{Size: image.Point{int(width*xScale + 0.5), int(height*yScale + 0.5)}}
	return &Gio{
		ops:        gtx.Ops,
		width:      width,
		height:     height,
		xScale:     xScale,
		yScale:     yScale,
		dimensions: dimensions,
	}
}

// NewStretch returns a Gio renderer that stretches the view to fit the constraints.
func NewStretch(gtx layout.Context, width, height float64) *Gio {
	xScale := float64(gtx.Constraints.Max.X-gtx.Constraints.Min.X) / width
	yScale := float64(gtx.Constraints.Max.Y-gtx.Constraints.Min.Y) / height

	dimensions := layout.Dimensions{Size: image.Point{int(width*xScale + 0.5), int(height*yScale + 0.5)}}
	return &Gio{
		ops:        gtx.Ops,
		width:      width,
		height:     height,
		xScale:     xScale,
		yScale:     yScale,
		dimensions: dimensions,
	}
}

// Dimensions returns the dimensions for Gio.
func (r *Gio) Dimensions() layout.Dimensions {
	return r.dimensions
}

// Size returns the size of the canvas in millimeters.
func (r *Gio) Size() (float64, float64) {
	return r.width, r.height
}

func (r *Gio) point(p canvas.Point) f32.Point {
	return f32.Point{float32(r.xScale * p.X), float32(r.yScale * (r.height - p.Y))}
}

func (r *Gio) renderPath(path *canvas.Path, fill canvas.Paint) {
	path = path.ReplaceArcs()

	p := clip.Path{}
	p.Begin(r.ops)
	for scanner := path.Scanner(); scanner.Scan(); {
		switch scanner.Cmd() {
		case canvas.MoveToCmd:
			p.MoveTo(r.point(scanner.End()))
		case canvas.LineToCmd:
			p.LineTo(r.point(scanner.End()))
		case canvas.QuadToCmd:
			p.QuadTo(r.point(scanner.CP1()), r.point(scanner.End()))
		case canvas.CubeToCmd:
			p.CubeTo(r.point(scanner.CP1()), r.point(scanner.CP2()), r.point(scanner.End()))
		case canvas.ArcToCmd:
			// TODO: ArcTo
			p.LineTo(r.point(scanner.End()))
		case canvas.CloseCmd:
			p.Close()
		}
	}

	shape := clip.Outline{p.End()}
	defer shape.Op().Push(r.ops).Pop()

	if fill.IsColor() {
		paint.Fill(r.ops, toNRGBA(fill.Color))
	} else if fill.IsGradient() {
		if g, ok := fill.Gradient.(*canvas.LinearGradient); ok && len(g.Stops) == 2 {
			linearGradient := paint.LinearGradientOp{}
			linearGradient.Stop1 = r.point(g.Start)
			linearGradient.Stop2 = r.point(g.End)
			linearGradient.Color1 = toNRGBA(g.Stops[0].Color)
			linearGradient.Color2 = toNRGBA(g.Stops[1].Color)
			linearGradient.Add(r.ops)
			paint.PaintOp{}.Add(r.ops)
		}
	}
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *Gio) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	if style.HasFill() {
		r.renderPath(path.Copy().Transform(m), style.Fill)
	}

	if style.HasStroke() {
		if style.IsDashed() {
			dashOffset, dashes := canvas.ScaleDash(style.StrokeWidth, style.DashOffset, style.Dashes)
			path = path.Dash(dashOffset, dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner, canvas.Tolerance)
		r.renderPath(path.Transform(m), style.Stroke)
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *Gio) RenderText(text *canvas.Text, m canvas.Matrix) {
	text.RenderAsPath(r, m, 0.0)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *Gio) RenderImage(img image.Image, m canvas.Matrix) {
	paint.NewImageOp(img).Add(r.ops)
	m = canvas.Identity.Scale(r.xScale, r.yScale).Mul(m)
	m = m.Translate(0.0, float64(img.Bounds().Max.Y))
	trans := op.Affine(f32.NewAffine2D(
		float32(m[0][0]), -float32(m[0][1]), float32(m[0][2]),
		-float32(m[1][0]), float32(m[1][1]), float32(r.yScale*r.height-m[1][2]),
	)).Push(r.ops)
	paint.PaintOp{}.Add(r.ops)
	trans.Pop()
}

func toNRGBA(col color.Color) color.NRGBA {
	r, g, b, a := col.RGBA()
	if a == 0 {
		return color.NRGBA{}
	}
	r = (r * 0xffff) / a
	g = (g * 0xffff) / a
	b = (b * 0xffff) / a
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}
