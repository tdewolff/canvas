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
	ops  *op.Ops
	size image.Point
}

// New returns a Gio renderer.
func New(gtx layout.Context, size image.Point) *Gio {
	return &Gio{
		ops:  gtx.Ops,
		size: size,
	}
}

// Dimensions returns the dimensions for Gio.
func (r *Gio) Dimensions() layout.Dimensions {
	return layout.Dimensions{Size: r.size}
}

// Size returns the size of the canvas in millimeters.
func (r *Gio) Size() (float64, float64) {
	return float64(r.size.X), float64(r.size.Y)
}

func (r *Gio) point(p canvas.Point) f32.Point {
	return f32.Point{float32(p.X), float32(float64(r.size.Y) - p.Y)}
}

// TODO: color blending is bad
func nrgba(col color.Color) color.NRGBA {
	r, g, b, a := col.RGBA()
	if a == 0 {
		return color.NRGBA{}
	}
	r = (r * 0xffff) / a
	g = (g * 0xffff) / a
	b = (b * 0xffff) / a
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func (r *Gio) renderPath(path *canvas.Path, col color.RGBA) {
	path = path.ReplaceArcs()

	p := clip.Path{}
	p.Begin(r.ops)
	path.Iterate(func(_, end canvas.Point) {
		p.MoveTo(r.point(end))
	}, func(_, end canvas.Point) {
		p.LineTo(r.point(end))
	}, func(_, cp, end canvas.Point) {
		p.QuadTo(r.point(cp), r.point(end))
	}, func(_, cp1, cp2, end canvas.Point) {
		p.CubeTo(r.point(cp1), r.point(cp2), r.point(end))
	}, func(_ canvas.Point, rx, ry, phi float64, large, sweep bool, end canvas.Point) {
		// TODO: ArcTo
		p.LineTo(r.point(end))
	}, func(_, _ canvas.Point) {
		p.Close()
	})

	shape := clip.Outline{p.End()}
	paint.FillShape(r.ops, nrgba(col), shape.Op())
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *Gio) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	if style.HasFill() {
		r.renderPath(path.Transform(m), style.FillColor)
	}

	if style.HasStroke() {
		if style.IsDashed() {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)
		r.renderPath(path.Transform(m), style.StrokeColor)
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *Gio) RenderText(text *canvas.Text, m canvas.Matrix) {
	text.RenderAsPath(r, m, canvas.DefaultResolution)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *Gio) RenderImage(img image.Image, m canvas.Matrix) {
	defer op.Save(r.ops).Load()

	paint.NewImageOp(img).Add(r.ops)
	//m = canvas.Identity.Translate(0.0, float64(r.size.Y)).Mul(m)
	m = m.Translate(0.0, float64(img.Bounds().Max.Y))
	op.Affine(f32.NewAffine2D(float32(m[0][0]), -float32(m[0][1]), float32(m[0][2]),
		-float32(m[1][0]), float32(m[1][1]), float32(float64(r.size.Y)-m[1][2]))).Add(r.ops)
	paint.PaintOp{}.Add(r.ops)
}
