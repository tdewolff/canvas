package canvas

import (
	"image"
	"image/color"
	"io"
	"os"
	"reflect"
	"sort"
)

// const mmPerPx = 25.4 / 96.0
// const pxPerMm = 96.0 / 25.4
const mmPerPt = 25.4 / 72.0
const ptPerMm = 72.0 / 25.4
const mmPerInch = 25.4
const inchPerMm = 1.0 / 25.4

// Resolution is used for rasterizing. Higher resolutions will result in larger images.
type Resolution float64

// DPMM (dots-per-millimeter) for the resolution of rasterization.
func DPMM(dpmm float64) Resolution {
	return Resolution(dpmm)
}

// DPI (dots-per-inch) for the resolution of rasterization.
func DPI(dpi float64) Resolution {
	return Resolution(dpi * inchPerMm)
}

// DPMM returns the resolution in dots-per-millimeter.
func (res Resolution) DPMM() float64 {
	return float64(res)
}

// DPI returns the resolution in dots-per-inch.
func (res Resolution) DPI() float64 {
	return float64(res) * mmPerInch
}

// DefaultResolution is the default resolution used for font PPEMs and is set to 96 DPI.
const DefaultResolution = Resolution(96.0 * inchPerMm)

// Size defines a size (width and height).
type Size struct {
	W, H float64
}

// Predefined paper sizes.
var (
	A0        = Size{841.0, 1189.0}
	A1        = Size{594.0, 841.0}
	A2        = Size{420.0, 594.0}
	A3        = Size{297.0, 420.0}
	A4        = Size{210.0, 297.0}
	A5        = Size{148.0, 210.0}
	A6        = Size{105.0, 148.0}
	A7        = Size{74.0, 105.0}
	A8        = Size{52.0, 74.0}
	B0        = Size{1000.0, 1414.0}
	B1        = Size{707.0, 1000.0}
	B2        = Size{500.0, 707.0}
	B3        = Size{353.0, 500.0}
	B4        = Size{250.0, 353.0}
	B5        = Size{176.0, 250.0}
	B6        = Size{125.0, 176.0}
	B7        = Size{88.0, 125.0}
	B8        = Size{62.0, 88.0}
	B9        = Size{44.0, 62.0}
	B10       = Size{31.0, 44.0}
	C2        = Size{648.0, 458.0}
	C3        = Size{458.0, 324.0}
	C4        = Size{324.0, 229.0}
	C5        = Size{229.0, 162.0}
	C6        = Size{162.0, 114.0}
	D0        = Size{1090.0, 771.0}
	SRA0      = Size{1280.0, 900.0}
	SRA1      = Size{900.0, 640.0}
	SRA2      = Size{640.0, 450.0}
	SRA3      = Size{450.0, 320.0}
	SRA4      = Size{320.0, 225.0}
	RA0       = Size{1220.0, 860.0}
	RA1       = Size{860.0, 610.0}
	RA2       = Size{610.0, 430.0}
	Letter    = Size{215.9, 279.4}
	Legal     = Size{215.9, 355.6}
	Ledger    = Size{279.4, 431.8}
	Tabloid   = Size{431.8, 279.4}
	Executive = Size{184.1, 266.7}
)

// ImageFit specifies how an image should fit a rectangle. ImageFill completely fills a rectangle by stretching the image. ImageContain and ImageCover both keep the aspect ratio of an image, where ImageContain scales the image such that it is complete contained in the rectangle (but possibly not completely covered), while ImageCover scales the image such that is completely covers the rectangle (but possibly extends beyond the boundaries of the rectangle).
type ImageFit int

// See ImageFit.
const (
	ImageFill ImageFit = iota
	ImageContain
	ImageCover
)

////////////////////////////////////////////////////////////////

// Paint is the type of paint used to fill or stroke a path. It can be either a color or a pattern. Default is transparent (no paint).
type Paint struct {
	Color color.RGBA
	Gradient
	Pattern
}

// Equal returns true if Paints are equal.
func (paint Paint) Equal(other Paint) bool {
	if paint.IsColor() && other.IsColor() && paint.Color == other.Color {
		return true
	} else if paint.IsGradient() && other.IsGradient() && reflect.DeepEqual(paint.Gradient, other.Gradient) {
		return true
	} else if paint.IsPattern() && other.IsPattern() && reflect.DeepEqual(paint.Pattern, other.Pattern) {
		return true
	}
	return false
}

// Has returns true if paint has a color or pattern.
func (paint Paint) Has() bool {
	return paint.Color.A != 0 || paint.Gradient != nil || paint.Pattern != nil
}

// IsColor returns true when paint is a uniform color.
func (paint Paint) IsColor() bool {
	return paint.Color.A != 0 && paint.Gradient == nil && paint.Pattern == nil
}

// IsGradient returns true when paint is a gradient.
func (paint Paint) IsGradient() bool {
	return paint.Gradient != nil && paint.Pattern == nil
}

// IsPattern returns true when paint is a pattern.
func (paint Paint) IsPattern() bool {
	return paint.Pattern != nil
}

// Dash patterns
var (
	Solid              = []float64{}
	Dotted             = []float64{1.0, 2.0}
	DenselyDotted      = []float64{1.0, 1.0}
	SparselyDotted     = []float64{1.0, 4.0}
	Dashed             = []float64{3.0, 3.0}
	DenselyDashed      = []float64{3.0, 1.0}
	SparselyDashed     = []float64{3.0, 6.0}
	Dashdotted         = []float64{3.0, 2.0, 1.0, 2.0}
	DenselyDashdotted  = []float64{3.0, 1.0, 1.0, 1.0}
	SparselyDashdotted = []float64{3.0, 4.0, 1.0, 4.0}
)

func ScaleDash(scale float64, offset float64, d []float64) (float64, []float64) {
	d2 := make([]float64, len(d))
	for i := range d {
		d2[i] = d[i] * scale
	}
	return offset * scale, d2
}

// Style is the path style that defines how to draw the path. When Fill is not set it will not fill the path. If StrokeColor is transparent or StrokeWidth is zero, it will not stroke the path. If Dashes is an empty array, it will not draw dashes but instead a solid stroke line. FillRule determines how to fill the path when paths overlap and have certain directions (clockwise, counter clockwise).
type Style struct {
	Fill         Paint
	Stroke       Paint
	StrokeWidth  float64
	StrokeCapper Capper
	StrokeJoiner Joiner
	DashOffset   float64
	Dashes       []float64
	FillRule     // TODO: test for all renderers
}

// HasFill returns true if the style has a fill
func (style Style) HasFill() bool {
	return style.Fill.Has()
}

// HasStroke returns true if the style has a stroke
func (style Style) HasStroke() bool {
	return style.Stroke.Has() && 0.0 < style.StrokeWidth
}

// IsDashed returns true if the style has dashes
func (style Style) IsDashed() bool {
	return 0 < len(style.Dashes)
}

// DefaultStyle is the default style for paths. It fills the path with a black color and has no stroke.
var DefaultStyle = Style{
	Fill:         Paint{Color: Black},
	Stroke:       Paint{},
	StrokeWidth:  1.0,
	StrokeCapper: ButtCap,
	StrokeJoiner: MiterJoin,
	DashOffset:   0.0,
	Dashes:       []float64{},
	FillRule:     NonZero,
}

// Renderer is an interface that renderers implement. It defines the size of the target (in mm) and functions to render paths, text objects and images.
type Renderer interface {
	Size() (float64, float64)
	RenderPath(path *Path, style Style, m Matrix)
	RenderText(text *Text, m Matrix)
	RenderImage(img image.Image, m Matrix)
}

////////////////////////////////////////////////////////////////

// CoordSystem is the coordinate system, which can be either of the four cartesian quadrants. Most useful are the I'th and IV'th quadrants. CartesianI is the default quadrant with the zero-point in the bottom-left (the default for mathematics). The CartesianII has its zero-point in the bottom-right, CartesianIII in the top-right, and CartesianIV in the top-left (often used as default for printing devices). See https://en.wikipedia.org/wiki/Cartesian_coordinate_system#Quadrants_and_octants for an explanation.
type CoordSystem int

// See CoordSystem.
const (
	CartesianI CoordSystem = iota
	CartesianII
	CartesianIII
	CartesianIV
)

// ContextState defines the state of the context, including fill or stroke style, view and coordinate view.
type ContextState struct {
	Style
	view        Matrix
	coordView   Matrix
	coordSystem CoordSystem
}

// Context maintains the state for the current path, path style, and view transformation matrix.
type Context struct {
	Renderer

	path *Path
	ContextState
	stack []ContextState
}

// NewContext returns a new context which is a wrapper around a renderer. Contexts maintain the state of the current path, path style, and view transformation matrix.
func NewContext(r Renderer) *Context {
	return &Context{
		Renderer: r,
		path:     &Path{},
		ContextState: ContextState{
			Style:       DefaultStyle,
			view:        Identity,
			coordView:   Identity,
			coordSystem: CartesianI,
		},
		stack: nil,
	}
}

// Width returns the width of the canvas in millimeters.
func (c *Context) Width() float64 {
	w, _ := c.Renderer.Size()
	return w
}

// Height returns the height of the canvas in millimeters.
func (c *Context) Height() float64 {
	_, h := c.Renderer.Size()
	return h
}

// Size returns the width and height of the canvas in millimeters.
func (c *Context) Size() (float64, float64) {
	return c.Renderer.Size()
}

// Push saves the current draw state so that it can be popped later on.
func (c *Context) Push() {
	c.stack = append(c.stack, c.ContextState)
}

// Pop restores the last pushed draw state and uses that as the current draw state. If there are no states on the stack, this will do nothing.
func (c *Context) Pop() {
	if len(c.stack) == 0 {
		return
	}
	c.ContextState = c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *Context) CoordSystemView() Matrix {
	// a function since renderer's width/height may change
	switch c.coordSystem {
	case CartesianII:
		return Identity.ReflectXAbout(c.Width() / 2.0)
	case CartesianIII:
		return Identity.ReflectXAbout(c.Width() / 2.0).ReflectYAbout(c.Height() / 2.0)
	case CartesianIV:
		return Identity.ReflectYAbout(c.Height() / 2.0)
	}
	return Identity
}

// SetCoordSystem sets the Cartesian coordinate system.
func (c *Context) SetCoordSystem(coordSystem CoordSystem) {
	c.coordSystem = coordSystem
}

// CoordView returns the current affine transformation matrix for coordinates.
func (c *Context) CoordView() Matrix {
	return c.coordView
}

// SetCoordView sets the current affine transformation matrix for coordinates. Coordinate transformation are applied before View transformations. See `Matrix` for how transformations work.
func (c *Context) SetCoordView(coordView Matrix) {
	c.coordView = coordView
}

// SetCoordRect sets the current affine transformation matrix for coordinates. Coordinate transformation are applied before View transformations. It will transform coordinates from (0,0)--(width,height) to the target `rect`.
func (c *Context) SetCoordRect(rect Rect, width, height float64) {
	c.coordView = Identity.Translate(rect.X0, rect.Y0).Scale(rect.W()/width, rect.H()/height)
}

// View returns the current affine transformation matrix through which all operations will be transformed.
func (c *Context) View() Matrix {
	return c.view
}

// SetView sets the current affine transformation matrix through which all operations will be transformed. See `Matrix` for how transformations work.
func (c *Context) SetView(view Matrix) {
	c.view = view
}

// ResetView resets the current affine transformation matrix to the Identity matrix, ie. no transformations.
func (c *Context) ResetView() {
	c.view = Identity
}

// ComposeView post-multiplies the current affine transformation matrix by the given matrix. This means that any draw action will first be transformed by the new view matrix (parameter) and then by the current view matrix (ie. `Context.View()`). `Context.ComposeView(Identity.ReflectX())` is the same as `Context.ReflectX()`.
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

// ReflectXAbout inverts the X axis of the view about the given X coordinate.
func (c *Context) ReflectXAbout(x float64) {
	c.view = c.view.Mul(Identity.ReflectXAbout(x))
}

// ReflectY inverts the Y axis of the view.
func (c *Context) ReflectY() {
	c.view = c.view.Mul(Identity.ReflectY())
}

// ReflectYAbout inverts the Y axis of the view about the given Y coordinate.
func (c *Context) ReflectYAbout(y float64) {
	c.view = c.view.Mul(Identity.ReflectYAbout(y))
}

// Rotate rotates the view counter clockwise with rot in degrees.
func (c *Context) Rotate(rot float64) {
	c.view = c.view.Mul(Identity.Rotate(rot))
}

// RotateAbout rotates the view counter clockwise around (x,y) with rot in degrees.
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

// SetFill sets the color, gradient, or pattern to be used for filling operations. The default fill color is black.
func (c *Context) SetFill(ifill interface{}) {
	if paint, ok := ifill.(Paint); ok {
		c.Style.Fill = paint
	} else if pattern, ok := ifill.(Pattern); ok {
		c.Style.Fill = Paint{Pattern: pattern}
	} else if gradient, ok := ifill.(Gradient); ok {
		c.Style.Fill = Paint{Gradient: gradient}
	} else if col, ok := ifill.(color.Color); ok {
		c.Style.Fill = Paint{Color: rgbaColor(col)}
	} else {
		c.Style.Fill = Paint{}
	}
}

// SetFillColor sets the color to be used for filling operations. The default fill color is black.
func (c *Context) SetFillColor(col color.Color) {
	c.Style.Fill.Color = rgbaColor(col)
	c.Style.Fill.Gradient = nil
	c.Style.Fill.Pattern = nil
}

// SetFillGradient sets the gradient to be used for filling operations. The default fill color is black.
func (c *Context) SetFillGradient(gradient Gradient) {
	c.Style.Fill.Color = Transparent
	c.Style.Fill.Gradient = gradient
	c.Style.Fill.Pattern = nil
}

// SetFillPattern sets the pattern to be used for filling operations. The default fill color is black.
func (c *Context) SetFillPattern(pattern Pattern) {
	c.Style.Fill.Color = Transparent
	c.Style.Fill.Gradient = nil
	c.Style.Fill.Pattern = pattern
}

// SetStroke sets the color, gradient, or pattern to be used for stroke operations. The default stroke color is transparent.
func (c *Context) SetStroke(istroke interface{}) {
	if paint, ok := istroke.(Paint); ok {
		c.Style.Stroke = paint
	} else if pattern, ok := istroke.(Pattern); ok {
		c.Style.Stroke = Paint{Pattern: pattern}
	} else if gradient, ok := istroke.(Gradient); ok {
		c.Style.Stroke = Paint{Gradient: gradient}
	} else if col, ok := istroke.(color.Color); ok {
		c.Style.Stroke = Paint{Color: rgbaColor(col)}
	} else {
		c.Style.Stroke = Paint{}
	}
}

// SetStrokeColor sets the color to be used for stroking operations. The default stroke color is transparent.
func (c *Context) SetStrokeColor(col color.Color) {
	c.Style.Stroke.Color = rgbaColor(col)
	c.Style.Stroke.Gradient = nil
	c.Style.Stroke.Pattern = nil
}

// SetStrokeGradient sets the gradients to be used for stroking operations. The default stroke color is transparent.
func (c *Context) SetStrokeGradient(gradient Gradient) {
	c.Style.Stroke.Color = Transparent
	c.Style.Stroke.Gradient = gradient
	c.Style.Stroke.Pattern = nil
}

// SetStrokePattern sets the pattern to be used for stroking operations. The default stroke color is transparent.
func (c *Context) SetStrokePattern(pattern Pattern) {
	c.Style.Stroke.Color = Transparent
	c.Style.Stroke.Gradient = nil
	c.Style.Stroke.Pattern = pattern
}

// SetStrokeWidth sets the width in millimeters for stroking operations. The default stroke width is 1.0.
func (c *Context) SetStrokeWidth(width float64) {
	c.Style.StrokeWidth = width
}

// SetStrokeCapper sets the line cap function to be used for stroke end points. The default stroke capper is a butt cap.
func (c *Context) SetStrokeCapper(capper Capper) {
	c.Style.StrokeCapper = capper
}

// SetStrokeJoiner sets the line join function to be used for stroke mid points. The default stroke joiner is a miter join with a limit of 4.0 for a bevel join.
func (c *Context) SetStrokeJoiner(joiner Joiner) {
	c.Style.StrokeJoiner = joiner
}

// SetDashes sets the dash pattern to be used for stroking operations. The dash offset denotes the offset into the dash array in millimeters from where to start. Negative values are allowed. The default dashes is empty at zero offset.
func (c *Context) SetDashes(offset float64, dashes ...float64) {
	c.Style.DashOffset = offset
	c.Style.Dashes = dashes
}

// SetFillRule sets the fill rule to be used for filling paths. The default fill rule is NonZero. Note that support is limited.
func (c *Context) SetFillRule(rule FillRule) {
	c.Style.FillRule = rule
}

// ResetStyle resets the draw state to its default (colors, stroke widths, dashes, ...).
func (c *Context) ResetStyle() {
	c.Style = DefaultStyle
}

// SetZIndex sets the z-index. This will call the renderer's `SetZIndex` function only if it exists (in this case only for `Canvas`).
func (c *Context) SetZIndex(zindex int) {
	if zindexer, ok := c.Renderer.(interface{ SetZIndex(int) }); ok {
		zindexer.SetZIndex(zindex)
	}
}

// Pos returns the current position of the path, which is the end point of the last command.
func (c *Context) Pos() (float64, float64) {
	return c.path.Pos().X, c.path.Pos().Y
}

// MoveTo moves the path to (x,y) without connecting with the previous path. It starts a new independent subpath. Multiple subpaths can be useful when negating parts of a previous path by overlapping it with a path in the opposite direction. The behaviour of overlapping paths depends on the FillRule.
func (c *Context) MoveTo(x, y float64) {
	c.path.MoveTo(x, y)
}

// LineTo adds a linear path to (x,y).
func (c *Context) LineTo(x, y float64) {
	c.path.LineTo(x, y)
}

// QuadTo adds a quadratic Bézier path with control point (cpx,cpy) and end point (x,y).
func (c *Context) QuadTo(cpx, cpy, x, y float64) {
	c.path.QuadTo(cpx, cpy, x, y)
}

// CubeTo adds a cubic Bézier path with control points (cpx1,cpy1) and (cpx2,cpy2) and end point (x,y).
func (c *Context) CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float64) {
	c.path.CubeTo(cpx1, cpy1, cpx2, cpy2, x, y)
}

// ArcTo adds an arc with radii rx and ry, with rot the counter clockwise rotation with respect to the coordinate system in degrees, large and sweep booleans (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs), and (x,y) the end position of the pen. The start position of the pen was given by a previous command's end point.
func (c *Context) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) {
	c.path.ArcTo(rx, ry, rot, large, sweep, x, y)
}

// Arc adds an elliptical arc with radii rx and ry, with rot the counter clockwise rotation in degrees, and theta0 and theta1 the angles in degrees of the ellipse (before rot is applied) between which the arc will run. If theta0 < theta1, the arc will run in a CCW direction. If the difference between theta0 and theta1 is bigger than 360 degrees, one full circle will be drawn and the remaining part of diff % 360, e.g. a difference of 810 degrees will draw one full circle and an arc over 90 degrees.
func (c *Context) Arc(rx, ry, rot, theta0, theta1 float64) {
	c.path.Arc(rx, ry, rot, theta0, theta1)
}

// Close closes the current path.
func (c *Context) Close() {
	c.path.Close()
}

// Fill fills the current path and resets the path.
func (c *Context) Fill() {
	stroke := c.Style.Stroke
	c.Style.Stroke = Paint{}
	c.DrawPath(0.0, 0.0, c.path)
	c.Style.Stroke = stroke
	c.path = &Path{}
}

// Stroke strokes the current path and resets the path.
func (c *Context) Stroke() {
	fill := c.Style.Fill
	c.Style.Fill = Paint{}
	c.DrawPath(0.0, 0.0, c.path)
	c.Style.Fill = fill
	c.path = &Path{}
}

// FillStroke fills and then strokes the current path and resets the path.
func (c *Context) FillStroke() {
	c.DrawPath(0.0, 0.0, c.path)
	c.path = &Path{}
}

// FitImage fits an image to a rectangle using different fit strategies.
func (c *Context) FitImage(img image.Image, rect Rect, fit ImageFit) Rect {
	if img.Bounds().Size().Eq(image.Point{}) || rect.Empty() {
		return Rect{}
	}

	width := float64(img.Bounds().Max.X - img.Bounds().Min.X)
	height := float64(img.Bounds().Max.Y - img.Bounds().Min.Y)

	x, y := rect.X0, rect.Y0 // offset to draw image
	xres := width / rect.W()
	yres := height / rect.H()
	switch fit {
	case ImageContain:
		if xres < yres {
			// less wide, height restricted
			dx := (rect.W() - width/yres) / 2.0
			x += dx
			rect.X0 += dx
			rect.X1 -= dx
			xres = yres
		} else {
			// less high, width restricted
			dy := (rect.H() - height/xres) / 2.0
			y += dy
			rect.Y0 += dy
			rect.Y1 -= dy
			yres = xres
		}
	case ImageCover:
		var dx, dy int // offset to crop image
		if xres < yres {
			dy = int((height-rect.H()*xres)/2.0 + 0.5)
			yres = (height - float64(2*dy)) / rect.H()
		} else {
			dx = int((width-rect.W()*yres)/2.0 + 0.5)
			xres = (width - float64(2*dx)) / rect.W()
		}
		if subimg, ok := img.(interface {
			SubImage(image.Rectangle) image.Image
		}); ok {
			imgRect := img.Bounds()
			imgRect.Min.X += dx
			imgRect.Min.Y += dy
			imgRect.Max.X -= dx
			imgRect.Max.Y -= dy
			img = subimg.SubImage(imgRect)
		} else {
			panic("image doesn't have SubImage method")
		}
	default:
		// ImageFill
	}

	// get view
	coord := c.coordView.Dot(Point{x, y})
	m := c.CoordSystemView().Mul(c.view).Translate(coord.X, coord.Y)

	// set resolution
	m = m.Scale(1.0/xres, 1.0/yres)

	// set origin of image closest to the image's origin (ie. top-left for CartesianIV)
	if c.coordSystem == CartesianIII || c.coordSystem == CartesianIV {
		m = m.ReflectYAbout(float64(img.Bounds().Size().Y) / 2.0)
	}
	if c.coordSystem == CartesianII || c.coordSystem == CartesianIII {
		m = m.ReflectXAbout(float64(img.Bounds().Size().X) / 2.0)
	}
	c.RenderImage(img, m)
	return rect
}

// DrawPath draws a path at position (x,y) using the current draw state.
func (c *Context) DrawPath(x, y float64, paths ...*Path) {
	if !c.Style.HasFill() && !c.Style.HasStroke() {
		return
	}

	// TODO: apply coordinate view to fill/stroke gradients/patterns
	style := c.Style
	m := c.CoordSystemView()
	//if style.Fill.IsPattern() {
	//	style.Fill.Pattern = style.Fill.Pattern.SetView(m)
	//} else if style.Fill.IsGradient() {
	//	style.Fill.Gradient = style.Fill.Gradient.SetView(m)
	//}
	//if style.Stroke.IsPattern() {
	//	style.Stroke.Pattern = style.Stroke.Pattern.SetView(m)
	//} else if style.Stroke.IsGradient() {
	//	style.Stroke.Gradient = style.Stroke.Gradient.SetView(m)
	//}

	// get view
	coord := c.coordView.Dot(Point{x, y})
	m = m.Mul(c.view).Translate(coord.X, coord.Y)

	dashes := style.Dashes
	for _, path := range paths {
		var ok bool
		style.Dashes, ok = path.checkDash(c.Style.DashOffset, dashes)
		if !ok {
			style.Stroke = Paint{}
		}
		c.RenderPath(path, style, m)
	}
}

// DrawText draws text at position (x,y) using the current draw state.
func (c *Context) DrawText(x, y float64, text *Text) {
	if text.Empty() {
		return
	}

	// get view
	coord := c.coordView.Dot(Point{x, y})
	m := c.CoordSystemView().Mul(c.view).Translate(coord.X, coord.Y)

	// keep textbox origin at the top-left
	if c.coordSystem == CartesianIII || c.coordSystem == CartesianIV {
		m = m.ReflectY()
	}
	if c.coordSystem == CartesianII || c.coordSystem == CartesianIII {
		m = m.ReflectX()
	}
	c.RenderText(text, m)
}

// DrawImage draws an image at position (x,y) using the current draw state and the given resolution in pixels-per-millimeter. A higher resolution will draw a smaller image (ie. more image pixels per millimeter of document).
func (c *Context) DrawImage(x, y float64, img image.Image, resolution Resolution) {
	if img.Bounds().Size().Eq(image.Point{}) {
		return
	}

	// get view
	coord := c.coordView.Dot(Point{x, y})
	m := c.CoordSystemView().Mul(c.view).Translate(coord.X, coord.Y)

	// set resolution
	m = m.Scale(1.0/resolution.DPMM(), 1.0/resolution.DPMM())

	// set origin of image closest to the image's origin (ie. top-left for CartesianIV)
	if c.coordSystem == CartesianIII || c.coordSystem == CartesianIV {
		m = m.ReflectYAbout(float64(img.Bounds().Size().Y) / 2.0)
	}
	if c.coordSystem == CartesianII || c.coordSystem == CartesianIII {
		m = m.ReflectXAbout(float64(img.Bounds().Size().X) / 2.0)
	}
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
	layers map[int][]layer
	zindex int
	W, H   float64
}

// New returns a new canvas with width and height in millimeters, that records all drawing operations into layers. The canvas can then be rendered to any other renderer.
func New(width, height float64) *Canvas {
	return &Canvas{
		layers: map[int][]layer{},
		W:      width,
		H:      height,
	}
}

// NewFromSize returns a new canvas of given size in millimeters, that records all drawing operations into layers. The canvas can then be rendered to any other renderer.
func NewFromSize(size Size) *Canvas {
	return New(size.W, size.H)
}

// Size returns the size of the canvas in millimeters.
func (c *Canvas) Size() (float64, float64) {
	return c.W, c.H
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (c *Canvas) RenderPath(path *Path, style Style, m Matrix) {
	path = path.Copy()
	c.layers[c.zindex] = append(c.layers[c.zindex], layer{path: path, m: m, style: style})
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (c *Canvas) RenderText(text *Text, m Matrix) {
	c.layers[c.zindex] = append(c.layers[c.zindex], layer{text: text, m: m})
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (c *Canvas) RenderImage(img image.Image, m Matrix) {
	c.layers[c.zindex] = append(c.layers[c.zindex], layer{img: img, m: m})
}

// Empty return true if the canvas is empty.
func (c *Canvas) Empty() bool {
	return len(c.layers) == 0
}

// Reset empties the canvas.
func (c *Canvas) Reset() {
	c.layers = map[int][]layer{}
}

// SetZIndex sets the z-index.
func (c *Canvas) SetZIndex(zindex int) {
	c.zindex = zindex
}

// Transform transforms the canvas.
func (c *Canvas) Transform(m Matrix) {
	for _, layers := range c.layers {
		for i, l := range layers {
			layers[i].m = m.Mul(l.m)
		}
	}
}

// Clip sets the canvas to the given rectangle.
func (c *Canvas) Clip(rect Rect) {
	c.Transform(Identity.Translate(-rect.X0, -rect.Y0))
	c.W = rect.W()
	c.H = rect.H()
}

// Fit shrinks the canvas' size that so all elements fit with a given margin in millimeters.
func (c *Canvas) Fit(margin float64) {
	rect := Rect{}
	// TODO: slow when we have many paths (see Graph example)
	for _, layers := range c.layers {
		for _, l := range layers {
			bounds := Rect{}
			if l.path != nil {
				bounds = l.path.Bounds()
				if l.style.HasStroke() {
					hw := l.style.StrokeWidth / 2.0
					bounds.X0 -= hw
					bounds.Y0 -= hw
					bounds.X1 += hw
					bounds.Y1 += hw
				}
			} else if l.text != nil {
				bounds = l.text.Bounds()
			} else if l.img != nil {
				size := l.img.Bounds().Size()
				bounds = Rect{0.0, 0.0, float64(size.X), float64(size.Y)}
			}
			if !bounds.Empty() {
				bounds = bounds.Transform(l.m)
				if rect.Empty() {
					rect = bounds
				} else {
					rect = rect.Add(bounds)
				}
			}
		}
	}
	rect.X0 -= margin
	rect.Y0 -= margin
	rect.X1 += margin
	rect.Y1 += margin
	c.Clip(rect)
}

// RenderTo renders the accumulated canvas drawing operations to another renderer.
func (c *Canvas) RenderTo(r Renderer) {
	c.RenderViewTo(r, Identity)
}

// RenderViewTo transforms and renders the accumulated canvas drawing operations to another renderer.
func (c *Canvas) RenderViewTo(r Renderer, view Matrix) {
	zindices := []int{}
	for zindex := range c.layers {
		zindices = append(zindices, zindex)
	}
	sort.Ints(zindices)

	for _, zindex := range zindices {
		for _, l := range c.layers[zindex] {
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
}

// Writer can write a canvas to a writer.
type Writer func(w io.Writer, c *Canvas) error

// Write writes the canvas to an io.Writer using the given writer. See renderers/ for an overview of implementations of canvas.Writer.
func (c *Canvas) Write(w io.Writer, writer Writer) error {
	return writer(w, c)
}

// WriteFile writes the canvas to a file using the given writer. See renderers/ for an overview of implementations of canvas.Writer.
func (c *Canvas) WriteFile(filename string, writer Writer) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	if err = writer(f, c); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
