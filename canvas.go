package canvas

import (
	"image"
	"image/color"
	"io"
	"os"
)

const mmPerPt = 0.3527777777777778
const ptPerMm = 2.8346456692913384
const mmPerInch = 25.4
const inchPerMm = 1 / 25.4

// ImageEncoding defines whether the embedded image shall be embedded as Lossless (typically PNG) or Lossy (typically JPG).
type ImageEncoding int

// see ImageEncoding
const (
	Lossless ImageEncoding = iota
	Lossy
)

// DPMM (Dots-per-Millimetter) for the resolution of raster images. Higher DPMM will result in bigger images.
type DPMM float64

// DPI is a shortcut for Dots-per-Inch for the resolution of raster images.
const DPI = DPMM(1 / 25.4)

////////////////////////////////////////////////////////////////

// Style is the path style that defines how to draw the path. When FillColor is transparent it will not fill the path. If StrokeColor is transparent or StrokeWidth is zero, it will not stroke the path. If Dashes is an empty array, it will not draw dashes but instead a solid stroke line. FillRule determines how to fill the path when paths overlap and have certain directions (clockwise, counter clockwise).
type Style struct {
	FillColor    color.RGBA
	StrokeColor  color.RGBA
	StrokeWidth  float64
	StrokeCapper Capper
	StrokeJoiner Joiner
	DashOffset   float64
	Dashes       []float64
	FillRule
}

// DefaultStyle is the default style for paths. It fills the path with a black color.
var DefaultStyle = Style{
	FillColor:    Black,
	StrokeColor:  Transparent,
	StrokeWidth:  1.0,
	StrokeCapper: ButtCap,
	StrokeJoiner: MiterJoin,
	DashOffset:   0.0,
	Dashes:       []float64{},
	FillRule:     NonZero,
}

// Renderer is an interface that renderers implement. It defines the size of the target (in mm) and functions to render paths, text objects and raster images.
type Renderer interface {
	Size() (float64, float64)
	RenderPath(path *Path, style Style, m Matrix)
	RenderText(text *Text, m Matrix)
	RenderImage(img image.Image, m Matrix)
}

////////////////////////////////////////////////////////////////

// Context maintains the state for the current path, path style, and view transformation matrix.
type Context struct {
	Renderer

	path *Path
	Style
	styleStack []Style
	view       Matrix
	viewStack  []Matrix
}

// NewContext returns a new Context which is a wrapper around a Renderer. Context maintains state for the current path, path style, and view transformation matrix.
func NewContext(r Renderer) *Context {
	return &Context{r, &Path{}, DefaultStyle, nil, Identity, nil}
}

// Width returns the width of the canvas.
func (c *Context) Width() float64 {
	w, _ := c.Size()
	return w
}

// Height returns the height of the canvas.
func (c *Context) Height() float64 {
	_, h := c.Size()
	return h
}

// Push saves the current draw state, so that it can be popped later on.
func (c *Context) Push() {
	c.viewStack = append(c.viewStack, c.view)
	c.styleStack = append(c.styleStack, c.Style)
}

// Pop restores the last pushed draw state and uses that as the current draw state. If there are no states on the stack, this will do nothing.
func (c *Context) Pop() {
	if len(c.viewStack) == 0 {
		return
	}
	c.view = c.viewStack[len(c.viewStack)-1]
	c.Style = c.styleStack[len(c.styleStack)-1]
	c.viewStack = c.viewStack[:len(c.viewStack)-1]
	c.styleStack = c.styleStack[:len(c.styleStack)-1]
}

// View returns the current affine transformation matrix.
func (c *Context) View() Matrix {
	return c.view
}

// SetView sets the current affine transformation matrix through which all operations will be transformed.
func (c *Context) SetView(view Matrix) {
	c.view = view
}

// ResetView resets the current affine transformation matrix to the Identity matrix, ie. no transformations.
func (c *Context) ResetView() {
	c.view = Identity
}

// ComposeView post-multiplies the current affine transformation matrix by the given one.
func (c *Context) ComposeView(view Matrix) {
	c.view = c.view.Mul(view)
}

// Translate moves the view.
func (c *Context) Translate(x, y float64) {
	c.view = c.view.Mul(Identity.Translate(x, y))
}

// ReflectX inverts the X axis of the view.
func (c *Context) ReflectX() {
	c.view = c.view.Mul(Identity.ReflectX())
}

// ReflectXAbout inverts the X axis of the view.
func (c *Context) ReflectXAbout(x float64) {
	c.view = c.view.Mul(Identity.ReflectXAbout(x))
}

// ReflectY inverts the Y axis of the view.
func (c *Context) ReflectY() {
	c.view = c.view.Mul(Identity.ReflectY())
}

// ReflectYAbout inverts the Y axis of the view.
func (c *Context) ReflectYAbout(y float64) {
	c.view = c.view.Mul(Identity.ReflectYAbout(y))
}

// Rotate rotates the view with rot in degrees.
func (c *Context) Rotate(rot float64) {
	c.view = c.view.Mul(Identity.Rotate(rot))
}

// RotateAbout rotates the view around (x,y) with rot in degrees.
func (c *Context) RotateAbout(rot, x, y float64) {
	c.view = c.view.Mul(Identity.RotateAbout(rot, x, y))
}

// Scale scales the view.
func (c *Context) Scale(sx, sy float64) {
	c.view = c.view.Mul(Identity.Scale(sx, sy))
}

// ScaleAbout scales the view around (x,y).
func (c *Context) ScaleAbout(sx, sy, x, y float64) {
	c.view = c.view.Mul(Identity.ScaleAbout(sx, sy, x, y))
}

// Shear shear stretches the view.
func (c *Context) Shear(sx, sy float64) {
	c.view = c.view.Mul(Identity.Shear(sx, sy))
}

// ShearAbout shear stretches the view around (x,y).
func (c *Context) ShearAbout(sx, sy, x, y float64) {
	c.view = c.view.Mul(Identity.ShearAbout(sx, sy, x, y))
}

// SetFillColor sets the color to be used for filling operations.
func (c *Context) SetFillColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.Style.FillColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// SetStrokeColor sets the color to be used for stroking operations.
func (c *Context) SetStrokeColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.Style.StrokeColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// SetStrokeWidth sets the width in mm for stroking operations.
func (c *Context) SetStrokeWidth(width float64) {
	c.Style.StrokeWidth = width
}

// SetStrokeCapper sets the line cap function to be used for stroke endpoints.
func (c *Context) SetStrokeCapper(capper Capper) {
	c.Style.StrokeCapper = capper
}

// SetStrokeJoiner sets the line join function to be used for stroke midpoints.
func (c *Context) SetStrokeJoiner(joiner Joiner) {
	c.Style.StrokeJoiner = joiner
}

// SetDashes sets the dash pattern to be used for stroking operations. The dash offset denotes the offset into the dash array in mm from where to start. Negative values are allowed.
func (c *Context) SetDashes(offset float64, dashes ...float64) {
	c.Style.DashOffset = offset
	c.Style.Dashes = dashes
}

// SetFillRule sets the fill rule to be used for filling paths.
func (c *Context) SetFillRule(rule FillRule) {
	c.Style.FillRule = rule
}

// ResetStyle resets the draw state to its default (colors, stroke widths, dashes, ...).
func (c *Context) ResetStyle() {
	c.Style = DefaultStyle
}

// Pos returns the current position of the path, which is the end point of the last command.
func (c *Context) Pos() (float64, float64) {
	return c.path.Pos().X, c.path.Pos().Y
}

// MoveTo moves the path to x,y without connecting the path. It starts a new independent subpath. Multiple subpaths can be
// useful when negating parts of a previous path by overlapping it with a path in the opposite direction. The behaviour for
// overlapping paths depend on the FillRule.
func (c *Context) MoveTo(x, y float64) {
	c.path.MoveTo(x, y)
}

// LineTo adds a linear path to x,y.
func (c *Context) LineTo(x, y float64) {
	c.path.LineTo(x, y)
}

// QuadTo adds a quadratic Bézier path with control point cpx,cpy and end point x,y.
func (c *Context) QuadTo(cpx, cpy, x, y float64) {
	c.path.QuadTo(cpx, cpy, x, y)
}

// CubeTo adds a cubic Bézier path with control points cpx1,cpy1 and cpx2,cpy2 and end point x,y.
func (c *Context) CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float64) {
	c.path.CubeTo(cpx1, cpy1, cpx2, cpy2, x, y)
}

// ArcTo adds an arc with radii rx and ry, with rot the counter clockwise rotation with respect to the coordinate system in degrees,
// large and sweep booleans (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs),
// and x,y the end position of the pen. The start position of the pen was given by a previous command end point.
// When sweep is true it means following the arc in a CCW direction in the Cartesian coordinate system, ie. that is CW in the upper-left coordinate system as is the case in SVGs.
func (c *Context) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) {
	c.path.ArcTo(rx, ry, rot, large, sweep, x, y)
}

// Arc adds an elliptical arc with radii rx and ry, with rot the counter clockwise rotation in degrees, and theta0 and theta1
// the angles in degrees of the ellipse (before rot is applies) between which the arc will run. If theta0 < theta1, the arc will
// run in a CCW direction. If the difference between theta0 and theta1 is bigger than 360 degrees, one full circle will be drawn
// and the remaining part of diff % 360 (eg. a difference of 810 degrees will draw one full circle and an arc over 90 degrees).
func (c *Context) Arc(rx, ry, rot, theta0, theta1 float64) {
	c.path.Arc(rx, ry, rot, theta0, theta1)
}

// Close closes the current path.
func (c *Context) Close() {
	c.path.Close()
}

// Fill fills the current path and resets it.
func (c *Context) Fill() {
	style := c.Style
	style.StrokeColor = Transparent
	c.RenderPath(c.path, style, c.view)
	c.path = &Path{}
}

// Stroke strokes the current path and resets it.
func (c *Context) Stroke() {
	style := c.Style
	style.FillColor = Transparent
	c.RenderPath(c.path, style, c.view)
	c.path = &Path{}
}

// FillStroke fills and then strokes the current path and resets it.
func (c *Context) FillStroke() {
	c.RenderPath(c.path, c.Style, c.view)
	c.path = &Path{}
}

// DrawPath draws a path at position (x,y) using the current draw state.
func (c *Context) DrawPath(x, y float64, paths ...*Path) {
	if c.Style.FillColor.A == 0 && (c.Style.StrokeColor.A == 0 || c.Style.StrokeWidth == 0.0) {
		return
	}

	m := c.view.Translate(x, y)
	for _, path := range paths {
		var dashes []float64
		path, dashes = path.checkDash(c.Style.DashOffset, c.Style.Dashes)
		if path.Empty() {
			continue
		}
		style := c.Style
		style.Dashes = dashes
		c.RenderPath(path, style, m)
	}
}

// DrawText draws text at position (x,y) using the current draw state. In particular, it only uses the current affine transformation matrix.
func (c *Context) DrawText(x, y float64, texts ...*Text) {
	m := c.view.Translate(x, y)
	for _, text := range texts {
		if text.Empty() {
			continue
		}
		c.RenderText(text, m)
	}
}

// DrawImage draws an image at position (x,y), using an image encoding (Lossy or Lossless) and DPM (dots-per-millimeter). A higher DPM will draw a smaller image.
func (c *Context) DrawImage(x, y float64, img image.Image, dpm float64) {
	if img.Bounds().Size().Eq(image.Point{}) {
		return
	}

	m := c.view.Translate(x, y).Scale(1.0/dpm, 1.0/dpm)
	c.RenderImage(img, m)
}

////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

type layer struct {
	// path, text OR img is set
	path *Path
	text *Text
	img  image.Image

	m     Matrix
	style Style // only for path
}

// Canvas stores all drawing operations as layers that can be re-rendered to other renderers.
type Canvas struct {
	layers []layer
	W, H   float64
}

// New returns a new Canvas that records all drawing operations into layers. The canvas can then be rendered to any other renderer.
func New(width, height float64) *Canvas {
	return &Canvas{
		layers: []layer{},
		W:      width,
		H:      height,
	}
}

// Size returns the size of the canvas in mm.
func (c *Canvas) Size() (float64, float64) {
	return c.W, c.H
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (c *Canvas) RenderPath(path *Path, style Style, m Matrix) {
	path = path.Copy()
	c.layers = append(c.layers, layer{path: path, m: m, style: style})
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (c *Canvas) RenderText(text *Text, m Matrix) {
	c.layers = append(c.layers, layer{text: text, m: m})
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (c *Canvas) RenderImage(img image.Image, m Matrix) {
	c.layers = append(c.layers, layer{img: img, m: m})
}

// Empty return true if the canvas is empty.
func (c *Canvas) Empty() bool {
	return len(c.layers) == 0
}

// Reset empties the canvas.
func (c *Canvas) Reset() {
	c.layers = c.layers[:0]
}

// Fit shrinks the canvas size so all elements fit. The elements are translated towards the origin when any left/bottom margins exist and the canvas size is decreased if any margins exist. It will maintain a given margin.
func (c *Canvas) Fit(margin float64) {
	if len(c.layers) == 0 {
		c.W = 2 * margin
		c.H = 2 * margin
		return
	}

	rect := Rect{}
	// TODO: slow when we have many paths (see Graph example)
	for i, l := range c.layers {
		bounds := Rect{}
		if l.path != nil {
			bounds = l.path.Bounds()
			if l.style.StrokeColor.A != 0 && 0.0 < l.style.StrokeWidth {
				bounds.X -= l.style.StrokeWidth / 2.0
				bounds.Y -= l.style.StrokeWidth / 2.0
				bounds.W += l.style.StrokeWidth
				bounds.H += l.style.StrokeWidth
			}
		} else if l.text != nil {
			bounds = l.text.Bounds()
		} else if l.img != nil {
			size := l.img.Bounds().Size()
			bounds = Rect{0.0, 0.0, float64(size.X), float64(size.Y)}
		}
		bounds = bounds.Transform(l.m)
		if i == 0 {
			rect = bounds
		} else {
			rect = rect.Add(bounds)
		}
	}
	for i := range c.layers {
		c.layers[i].m = Identity.Translate(-rect.X+margin, -rect.Y+margin).Mul(c.layers[i].m)
	}
	c.W = rect.W + 2*margin
	c.H = rect.H + 2*margin
}

// Render renders the accumulated canvas drawing operations to another renderer.
func (c *Canvas) Render(r Renderer) {
	view := Identity
	if viewer, ok := r.(interface{ View() Matrix }); ok {
		view = viewer.View()
	}
	for _, l := range c.layers {
		m := view.Mul(l.m)
		if l.path != nil {
			r.RenderPath(l.path, l.style, m)
		} else if l.text != nil {
			r.RenderText(l.text, m)
		} else if l.img != nil {
			r.RenderImage(l.img, m)
		}
	}
}

// Writer can write a canvas to a writer
type Writer func(w io.Writer, c *Canvas) error

// WriteFile writes the canvas to a file named by filename using the given Writer (for the encoding).
func (c *Canvas) WriteFile(filename string, w Writer) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	if err = w(f, c); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
