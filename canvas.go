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
	StrokeCapper: ButtCap,
	StrokeJoiner: MiterJoin,
	DashOffset:   0.0,
	Dashes:       []float64{},
	FillRule:     NonZero,
}

type Renderer interface {
	Size() (float64, float64)
	RenderPath(path *Path, style Style, m Matrix)
	RenderText(text *Text, m Matrix)
	RenderImage(img image.Image, m Matrix)
}

////////////////////////////////////////////////////////////////

type Context struct {
	Renderer

	path       *Path
	view       Matrix
	style      Style
	viewStack  []Matrix
	styleStack []Style
}

// NewContext returns a new Context.
func NewContext(r Renderer) *Context {
	return &Context{r, &Path{}, Identity, DefaultStyle, nil, nil}
}

func (c *Context) Width() float64 {
	w, _ := c.Size()
	return w
}

func (c *Context) Height() float64 {
	_, h := c.Size()
	return h
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
func (c *Context) ReflectXAbout(x float64) {
	c.view = c.view.Mul(Identity.ReflectXAbout(x))
}

// ReflectX inverts the Y axis of the view.
func (c *Context) ReflectY() {
	c.view = c.view.Mul(Identity.ReflectY())
}

// ReflectX inverts the Y axis of the view.
func (c *Context) ReflectYAbout(y float64) {
	c.view = c.view.Mul(Identity.ReflectYAbout(y))
}

// Rotate rotates the view with rot in degrees.
func (c *Context) Rotate(rot float64) {
	c.view = c.view.Mul(Identity.Rotate(rot))
}

// RotateAt rotates the view around (x,y) with rot in degrees.
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
	c.RenderPath(c.path, style, c.view)
	c.path = &Path{}
}

func (c *Context) Stroke() {
	style := c.style
	style.FillColor = Transparent
	c.RenderPath(c.path, style, c.view)
	c.path = &Path{}
}

func (c *Context) FillStroke() {
	c.RenderPath(c.path, c.style, c.view)
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
		c.RenderPath(path, c.style, m)
	}
}

func (c *Context) DrawText(x, y float64, texts ...*Text) {
	m := c.view.Translate(x, y)
	for _, text := range texts {
		if text.Empty() {
			continue
		}
		c.RenderText(text, m)
	}
}

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

type Canvas struct {
	layers []layer
	W, H   float64
}

func New(width, height float64) *Canvas {
	return &Canvas{
		layers: []layer{},
		W:      width,
		H:      height,
	}
}

func (c *Canvas) Size() (float64, float64) {
	return c.W, c.H
}

func (c *Canvas) RenderPath(path *Path, style Style, m Matrix) {
	path = path.Copy()
	c.layers = append(c.layers, layer{path: path, m: m, style: style})
}

func (c *Canvas) RenderText(text *Text, m Matrix) {
	c.layers = append(c.layers, layer{text: text, m: m})
}

func (c *Canvas) RenderImage(img image.Image, m Matrix) {
	c.layers = append(c.layers, layer{img: img, m: m})
}

func (c *Canvas) Empty() bool {
	return len(c.layers) == 0
}

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

// SaveSVG writes the stored drawing operations in Context in the SVG file format.
func (c *Canvas) SaveSVG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	svg := SVG(f, c.W, c.H)
	c.Render(svg)
	return svg.Close()
}

// SavePDF writes the stored drawing operations in Context in the PDF file format.
func (c *Canvas) SavePDF(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	pdf := PDF(f, c.W, c.H)
	c.Render(pdf)
	return pdf.Close()
}

// SaveEPS writes the stored drawing operations in Context in the EPS file format.
// Be aware that EPS does not support transparency of colors.
func (c *Canvas) SaveEPS(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	eps := EPS(f, c.W, c.H)
	c.Render(eps)
	return f.Close()
}

// WriteImage writes the stored drawing operations in Context as a rasterized image with given DPM (dots-per-millimeter). Higher DPM will result in bigger images.
func (c *Canvas) WriteImage(dpm float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.W*dpm+0.5), int(c.H*dpm+0.5)))
	draw.Draw(img, img.Bounds(), image.NewUniform(White), image.Point{}, draw.Src)

	ras := Rasterizer(img, dpm)
	c.Render(ras)
	return img
}

func (c *Canvas) SavePNG(filename string, dpm float64) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	img := c.WriteImage(dpm)
	// TODO: optimization: cache img until canvas changes
	if err = png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (c *Canvas) SaveJPG(filename string, dpm float64, opts *jpeg.Options) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	img := c.WriteImage(dpm)
	if err = jpeg.Encode(f, img, opts); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (c *Canvas) SaveGIF(filename string, dpm float64, opts *gif.Options) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	img := c.WriteImage(dpm)
	if err = gif.Encode(f, img, opts); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
