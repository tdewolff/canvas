package gio

import (
	"image"
	"image/color"
	"math"

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

func (r *Gio) renderPath(path *canvas.Path, col color.RGBA) {
	path = path.ReplaceArcs()

	p := clip.Path{}
	p.Begin(r.ops)
	for _, seg := range path.Segments() {
		switch seg.Cmd {
		case canvas.MoveToCmd:
			p.MoveTo(r.point(seg.End))
		case canvas.LineToCmd:
			p.LineTo(r.point(seg.End))
		case canvas.QuadToCmd:
			p.QuadTo(r.point(seg.CP1()), r.point(seg.End))
		case canvas.CubeToCmd:
			p.CubeTo(r.point(seg.CP1()), r.point(seg.CP2()), r.point(seg.End))
		case canvas.ArcToCmd:
			// TODO: ArcTo
			p.LineTo(r.point(seg.End))
		case canvas.CloseCmd:
			p.Close()
		}
	}

	shape := clip.Outline{p.End()}
	paint.FillShape(r.ops, RGBAToNRGBA(col), shape.Op())
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

// from https://git.sr.ht/~eliasnaur/gio/tree/main/item/internal/f32color/rgba.go#L97
func RGBAToNRGBA(col color.RGBA) color.NRGBA {
	if col.A == 0xFF {
		return color.NRGBA(col)
	} else if col.A == 0 {
		return color.NRGBA{}
	}

	linear := RGBA{
		R: sRGBToLinear(float32(col.R) / 0xff),
		G: sRGBToLinear(float32(col.G) / 0xff),
		B: sRGBToLinear(float32(col.B) / 0xff),
		A: float32(col.A) / 0xff,
	}
	return linear.SRGB()
}

// RGBA is a 32 bit floating point linear premultiplied color space.
type RGBA struct {
	R, G, B, A float32
}

// SRGBA converts from linear to sRGB color space.
func (col RGBA) SRGB() color.NRGBA {
	if col.A == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(linearTosRGB(col.R/col.A)*255 + .5),
		G: uint8(linearTosRGB(col.G/col.A)*255 + .5),
		B: uint8(linearTosRGB(col.B/col.A)*255 + .5),
		A: uint8(col.A*255 + .5),
	}
}

// linearTosRGB transforms color value from linear to sRGB.
func linearTosRGB(c float32) float32 {
	// Formula from EXT_sRGB.
	switch {
	case c <= 0:
		return 0
	case 0 < c && c < 0.0031308:
		return 12.92 * c
	case 0.0031308 <= c && c < 1:
		return 1.055*float32(math.Pow(float64(c), 0.41666)) - 0.055
	}

	return 1
}

// sRGBToLinear transforms color value from sRGB to linear.
func sRGBToLinear(c float32) float32 {
	// Formula from EXT_sRGB.
	if c <= 0.04045 {
		return c / 12.92
	} else {
		return float32(math.Pow(float64((c+0.055)/1.055), 2.4))
	}
}
