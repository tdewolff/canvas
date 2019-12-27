package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
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

////////////////////////////////////////////////////////////////

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

var DefaultStyle = Style{
	FillColor:    Black,
	StrokeColor:  Transparent,
	StrokeWidth:  1.0,
	StrokeCapper: ButtCapper,
	StrokeJoiner: MiterJoiner,
	DashOffset:   0.0,
	Dashes:       []float64{},
	FillRule:     NonZero,
}

////////////////////////////////////////////////////////////////

type Canvas interface {
	Renderer

	Push()
	Pop()
	View() Matrix
	SetView(view Matrix)
	ResetView()
	ComposeView(view Matrix)
	Translate(x, y float64)
	ReflectX()
	ReflectXAt(x float64)
	ReflectY()
	ReflectYAt(y float64)
	Rotate(rot float64)
	RotateAt(rot, x, y float64)
	Scale(x, y float64)
	Shear(x, y float64)
	SetFillColor(col color.Color)
	SetStrokeColor(col color.Color)
	SetStrokeWidth(width float64)
	SetStrokeCapper(capper Capper)
	SetStrokeJoiner(joiner Joiner)
	SetDashes(offset float64, dashes ...float64)
	SetFillRule(rule FillRule)
	Style() Style
	SetStyle(style Style)
	ResetStyle()
	Pos() (float64, float64)
	Path() *Path
	MoveTo(x, y float64)
	LineTo(x, y float64)
	QuadTo(cpx, cpy, x, y float64)
	CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float64)
	ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64)
	Arc(rx, ry, rot, theta0, theta1 float64)
	ClosePath()
	Fill()
	Stroke()
	FillStroke()
	DrawPath(x, y float64, paths ...*Path)
	DrawText(x, y float64, texts ...*Text)
	DrawImage(x, y float64, img image.Image, dpm float64)
}

type Renderer interface {
	renderPath(path *Path, style Style, m Matrix)
	renderText(text *Text, m Matrix)
	renderImage(img image.Image, m Matrix)
}

////////////////////////////////////////////////////////////////

type Context struct {
	r    Renderer
	W, H float64

	path       *Path
	view       Matrix
	style      Style
	viewStack  []Matrix
	styleStack []Style
}

// newContext returns a new Context.
func newContext(r Renderer, width, height float64) *Context {
	return &Context{r, width, height, &Path{}, Identity, DefaultStyle, nil, nil}
}

// Push saves the current draw state, so that it can be popped later on.
func (c *Context) Push() {
	c.viewStack = append(c.viewStack, c.view)
	c.styleStack = append(c.styleStack, c.style)
}

// Pop restores the last pushed draw state and uses that as the current draw state. If there are no states on the stack, this will do nothing.
func (c *Context) Pop() {
	if len(c.viewStack) == 0 {
		return
	}
	c.view = c.viewStack[len(c.viewStack)-1]
	c.style = c.styleStack[len(c.styleStack)-1]
	c.viewStack = c.viewStack[:len(c.viewStack)-1]
	c.styleStack = c.styleStack[:len(c.styleStack)-1]
}

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

// ReflectXAt inverts the X axis of the view.
func (c *Context) ReflectXAt(x float64) {
	c.view = c.view.Mul(Identity.ReflectXAt(x))
}

// ReflectX inverts the Y axis of the view.
func (c *Context) ReflectY() {
	c.view = c.view.Mul(Identity.ReflectY())
}

// ReflectX inverts the Y axis of the view.
func (c *Context) ReflectYAt(y float64) {
	c.view = c.view.Mul(Identity.ReflectYAt(y))
}

// Rotate rotates the view with rot in degrees.
func (c *Context) Rotate(rot float64) {
	c.view = c.view.Mul(Identity.Rotate(rot))
}

// RotateAt rotates the view around (x,y) with rot in degrees.
func (c *Context) RotateAt(rot, x, y float64) {
	c.view = c.view.Mul(Identity.RotateAt(rot, x, y))
}

// Scale scales the view.
func (c *Context) Scale(x, y float64) {
	c.view = c.view.Mul(Identity.Scale(x, y))
}

// Shear shear stretches the view.
func (c *Context) Shear(x, y float64) {
	c.view = c.view.Mul(Identity.Shear(x, y))
}

// SetFillColor sets the color to be used for filling operations.
func (c *Context) SetFillColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.style.FillColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// SetStrokeColor sets the color to be used for stroking operations.
func (c *Context) SetStrokeColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.style.StrokeColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// SetStrokeWidth sets the width in mm for stroking operations.
func (c *Context) SetStrokeWidth(width float64) {
	c.style.StrokeWidth = width
}

// SetStrokeCapper sets the line cap function to be used for stroke endpoints.
func (c *Context) SetStrokeCapper(capper Capper) {
	c.style.StrokeCapper = capper
}

// SetStrokeJoiner sets the line join function to be used for stroke midpoints.
func (c *Context) SetStrokeJoiner(joiner Joiner) {
	c.style.StrokeJoiner = joiner
}

// SetDashes sets the dash pattern to be used for stroking operations. The dash offset denotes the offset into the dash array in mm from where to start. Negative values are allowed.
func (c *Context) SetDashes(offset float64, dashes ...float64) {
	c.style.DashOffset = offset
	c.style.Dashes = dashes
}

// SetFillRule sets the fill rule to be used for filling paths.
func (c *Context) SetFillRule(rule FillRule) {
	c.style.FillRule = rule
}

func (c *Context) Style() Style {
	return c.style
}

func (c *Context) SetStyle(style Style) {
	c.style = style
}

// ResetStyle resets the draw state to its default (colors, stroke widths, dashes, ...).
func (c *Context) ResetStyle() {
	c.style = DefaultStyle
}

func (c *Context) Pos() (float64, float64) {
	return c.path.Pos().X, c.path.Pos().Y
}

func (c *Context) Path() *Path {
	return c.path
}

func (c *Context) MoveTo(x, y float64) {
	c.path.MoveTo(x, y)
}

func (c *Context) LineTo(x, y float64) {
	c.path.LineTo(x, y)
}

func (c *Context) QuadTo(cpx, cpy, x, y float64) {
	c.path.QuadTo(cpx, cpy, x, y)
}

func (c *Context) CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float64) {
	c.path.CubeTo(cpx1, cpy1, cpx2, cpy2, x, y)
}

func (c *Context) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) {
	c.path.ArcTo(rx, ry, rot, large, sweep, x, y)
}

func (c *Context) Arc(rx, ry, rot, theta0, theta1 float64) {
	c.path.Arc(rx, ry, rot, theta0, theta1)
}

func (c *Context) ClosePath() {
	c.path.Close()
}

func (c *Context) Fill() {
	style := c.style
	style.StrokeColor = Transparent
	c.r.renderPath(c.path, style, c.view)
	c.path = &Path{}
}

func (c *Context) Stroke() {
	style := c.style
	style.FillColor = Transparent
	c.r.renderPath(c.path, style, c.view)
	c.path = &Path{}
}

func (c *Context) FillStroke() {
	c.r.renderPath(c.path, c.style, c.view)
	c.path = &Path{}
}

func (c *Context) DrawPath(x, y float64, paths ...*Path) {
	if c.style.FillColor.A == 0 && (c.style.StrokeColor.A == 0 || c.style.StrokeWidth == 0.0) {
		return
	}

	m := c.view.Translate(x, y)
	for _, path := range paths {
		var dashes []float64
		path, dashes = path.checkDash(c.style.DashOffset, c.style.Dashes)
		if path.Empty() {
			continue
		}
		style := c.style
		style.Dashes = dashes
		c.r.renderPath(path, c.style, m)
	}
}

func (c *Context) DrawText(x, y float64, texts ...*Text) {
	m := c.view.Translate(x, y)
	for _, text := range texts {
		if text.Empty() {
			continue
		}
		c.r.renderText(text, m)
	}
}

func (c *Context) DrawImage(x, y float64, img image.Image, dpm float64) {
	if img.Bounds().Size().Eq(image.Point{}) {
		return
	}

	m := c.view.Translate(x, y).Scale(1.0/dpm, 1.0/dpm)
	c.r.renderImage(img, m)
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

type canvas struct {
	*Context
	layers []layer
}

func New(width, height float64) *canvas {
	r := &canvas{
		layers: []layer{},
	}
	r.Context = newContext(r, width, height)
	return r
}

func (r *canvas) renderPath(path *Path, style Style, m Matrix) {
	path = path.Copy()
	r.layers = append(r.layers, layer{path: path, m: m, style: style})
}

func (r *canvas) renderText(text *Text, m Matrix) {
	r.layers = append(r.layers, layer{text: text, m: m})
}

func (r *canvas) renderImage(img image.Image, m Matrix) {
	r.layers = append(r.layers, layer{img: img, m: m})
}

func (r *canvas) Empty() bool {
	return len(r.layers) == 0
}

func (r *canvas) Reset() {
	r.layers = r.layers[:0]
}

// Fit shrinks the canvas size so all elements fit. The elements are translated towards the origin when any left/bottom margins exist and the canvas size is decreased if any margins exist. It will maintain a given margin.
func (r *canvas) Fit(margin float64) {
	// TODO: slow when we have many paths (see Graph example)
	if len(r.layers) == 0 {
		r.W = 2 * margin
		r.H = 2 * margin
		return
	}

	rect := Rect{}
	for i, l := range r.layers {
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
	for i := range r.layers {
		r.layers[i].m = Identity.Translate(-rect.X+margin, -rect.Y+margin).Mul(r.layers[i].m)
	}
	r.W = rect.W + 2*margin
	r.H = rect.H + 2*margin
}

func (r *canvas) Render(c Canvas) {
	for _, l := range r.layers {
		m := c.View().Mul(l.m)
		if l.path != nil {
			c.renderPath(l.path, l.style, m)
		} else if l.text != nil {
			c.renderText(l.text, m)
		} else if l.img != nil {
			c.renderImage(l.img, m)
		}
	}
}

// SaveSVG writes the stored drawing operations in Canvas in the SVG file format.
func (r *canvas) SaveSVG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	svg := SVG(f, r.W, r.H)
	r.Render(svg)
	return svg.Close()
}

// SavePDF writes the stored drawing operations in Canvas in the PDF file format.
func (r *canvas) SavePDF(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	pdf := PDF(f, r.W, r.H)
	r.Render(pdf)
	return pdf.Close()
}

// SaveEPS writes the stored drawing operations in Canvas in the EPS file format.
// Be aware that EPS does not support transparency of colors.
func (r *canvas) SaveEPS(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	eps := EPS(f, r.W, r.H)
	r.Render(eps)
	return f.Close()
}

// WriteImage writes the stored drawing operations in Canvas as a rasterized image with given DPM (dots-per-millimeter). Higher DPM will result in bigger images.
func (r *canvas) WriteImage(dpm float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(r.W*dpm+0.5), int(r.H*dpm+0.5)))
	draw.Draw(img, img.Bounds(), image.NewUniform(White), image.Point{}, draw.Src)

	ras := Rasterizer(img, dpm)
	r.Render(ras)
	return img
}

func (r *canvas) SavePNG(filename string, dpm float64) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	img := r.WriteImage(dpm)
	// TODO: optimization: cache img until canvas changes
	if err = png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (r *canvas) SaveJPG(filename string, dpm float64, opts *jpeg.Options) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	img := r.WriteImage(dpm)
	if err = jpeg.Encode(f, img, opts); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (r *canvas) SaveGIF(filename string, dpm float64, opts *gif.Options) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	img := r.WriteImage(dpm)
	if err = gif.Encode(f, img, opts); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
