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

// NewExpand returns a Gio renderer that fills the constraints either horizontally or vertically, whichever is met first.
func NewExpand(gtx layout.Context, width, height float64) *Gio {
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
	m = canvas.Identity.Scale(r.xScale, r.yScale).Mul(m)
	m = m.Translate(0.0, float64(img.Bounds().Max.Y))
	op.Affine(f32.NewAffine2D(
		float32(m[0][0]), -float32(m[0][1]), float32(m[0][2]),
		-float32(m[1][0]), float32(m[1][1]), float32(r.yScale*r.height-m[1][2]),
	)).Add(r.ops)
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
