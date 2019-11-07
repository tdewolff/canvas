package canvas

import (
	"image"
	"image/color"
	"io"

	"golang.org/x/image/draw"
)

const mmPerPt = 0.3527777777777778
const ptPerMm = 2.8346456692913384
const mmPerInch = 25.4
const inchPerMm = 1 / 25.4

// CoordinateSystem defines which coordinate system to use for positioning and paths.
type CoordinateSystem int

// see CoordinateSystem
const (
	CartesianQuadrant1 CoordinateSystem = iota
	CartesianQuadrant2
	CartesianQuadrant3
	CartesianQuadrant4
)

// ImageEncoding defines whether the embedded image shall be embedded as Lossless (typically PNG) or Lossy (typically JPG).
type ImageEncoding int

// see ImageEncoding
const (
	Lossless ImageEncoding = iota
	Lossy
)

////////////////////////////////////////////////////////////////

type drawState struct {
	m                      Matrix
	fillColor, strokeColor color.RGBA
	strokeWidth            float64
	strokeCapper           Capper
	strokeJoiner           Joiner
	dashOffset             float64
	dashes                 []float64
	fillRule               FillRuleType
}

var defaultDrawState = drawState{
	m:            Identity,
	fillColor:    Black,
	strokeColor:  Transparent,
	strokeWidth:  1.0,
	strokeCapper: ButtCapper,
	strokeJoiner: MiterJoiner,
	dashOffset:   0.0,
	dashes:       []float64{},
	fillRule:     NonZero,
}

type layer interface {
	Bounds() Rect
	WriteSVG(*svgWriter)
	WritePDF(*pdfPageWriter)
	WriteEPS(*epsWriter)
	DrawImage(draw.Image, float64)
	ToOpenGL(*OpenGL)
}

////////////////////////////////////////////////////////////////

type pathLayer struct {
	path        *Path
	drawState   // view matrix has already been applied
	dashesClose bool
}

func (l pathLayer) Bounds() Rect {
	bounds := l.path.Bounds()
	if l.strokeColor.A != 0 && 0.0 < l.strokeWidth {
		bounds.X -= l.strokeWidth / 2.0
		bounds.Y -= l.strokeWidth / 2.0
		bounds.W += l.strokeWidth
		bounds.H += l.strokeWidth
	}
	return bounds
}

////////////////////////////////////////////////////////////////

type textLayer struct {
	text *Text
	m    Matrix
}

func (l textLayer) Bounds() Rect {
	return l.text.Bounds().Transform(l.m)
}

////////////////////////////////////////////////////////////////

type imageLayer struct {
	img image.Image
	enc ImageEncoding
	m   Matrix
}

func (l imageLayer) Bounds() Rect {
	size := l.img.Bounds().Size()
	return Rect{0.0, 0.0, float64(size.X), float64(size.Y)}.Transform(l.m)
}

////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

type Canvas struct {
	width, height float64
	coordSystem   CoordinateSystem
	drawState
	stack []drawState

	layers []layer
}

// New returns a new Canvas of given width and height in mm.
func New(width, height float64) *Canvas {
	return &Canvas{width, height, CartesianQuadrant1, defaultDrawState, nil, nil}
}

func (c *Canvas) getCoordinateSystem() Matrix {
	switch c.coordSystem {
	case CartesianQuadrant2:
		return Identity.ReflectXAt(c.width)
	case CartesianQuadrant3:
		return Identity.Translate(c.width, c.height).Scale(-1.0, -1.0).Translate(-c.width, -c.height)
	case CartesianQuadrant4:
		return Identity.ReflectYAt(c.height)
	}
	return Identity
}

// SetCoordinateSystem sets the coordinate system to the given Cartesian system quadrant. The default is Cartesian quadrant one.
// The coordinate system affects the x,y coordinates given to the Draw* functions, as well as to the entire path for DrawPath. Text and images are not affected, only their positioning.
func (c *Canvas) SetCoordinateSystem(coordSystem CoordinateSystem) {
	c.coordSystem = coordSystem
}

// PushState saves the current draw state, so that it can be popped later on.
func (c *Canvas) PushState() {
	c.stack = append(c.stack, c.drawState)
}

// PopState pops the last pushed draw state and uses that as the current draw state. If there are no states on the stack, this will do nothing.
func (c *Canvas) PopState() {
	if len(c.stack) == 0 {
		return
	}
	c.drawState = c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
}

// SetView sets the current affine transformation matrix through which all operations will be transformed.
func (c *Canvas) SetView(m Matrix) {
	c.m = m
}

// ResetView resets the current affine transformation matrix to the Identity matrix, ie. no transformations.
func (c *Canvas) ResetView() {
	c.m = Identity
}

// ComposeView post-multiplies the current affine transformation matrix by the given one.
func (c *Canvas) ComposeView(m Matrix) {
	c.m = c.m.Mul(m)
}

// SetFillColor sets the color to be used for filling operations.
func (c *Canvas) SetFillColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.fillColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// SetStrokeColor sets the color to be used for stroking operations.
func (c *Canvas) SetStrokeColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.strokeColor = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// SetStrokeWidth sets the width in mm for stroking operations.
func (c *Canvas) SetStrokeWidth(width float64) {
	c.strokeWidth = width
}

// SetStrokeCapper sets the line cap function to be used for stroke endpoints.
func (c *Canvas) SetStrokeCapper(capper Capper) {
	c.strokeCapper = capper
}

// SetStrokeJoiner sets the line join function to be used for stroke midpoints.
func (c *Canvas) SetStrokeJoiner(joiner Joiner) {
	c.strokeJoiner = joiner
}

// SetDashes sets the dash pattern to be used for stroking operations. The dash offset denotes the offset into the dash array in mm from where to start. Negative values are allowed.
func (c *Canvas) SetDashes(dashOffset float64, dashes ...float64) {
	dashOffset, dashes = dashCanonical(dashOffset, dashes)
	c.dashOffset = dashOffset
	c.dashes = dashes
}

// Clear clears all previous draw actions to the canvas buffer.
func (c *Canvas) Clear() {
	c.layers = c.layers[:0]
}

// DrawPath draws a path at position (x,y) using the current draw state.
func (c *Canvas) DrawPath(x, y float64, path *Path) {
	if c.fillColor.A == 0 && (c.strokeColor.A == 0 || c.strokeWidth == 0.0) {
		return
	}
	if !path.Empty() {
		dashes := c.dashes
		dashesClose := false
		if 0.0 < c.strokeWidth && c.strokeColor.A != 0 && len(c.dashes) != 0 && path.Closed() {
			d := c.dashes
			if len(d) == 1 && d[0] == 0.0 {
				return
			} else if len(d)%2 == 1 {
				d = append(d, d...)
			}

			// will draw dashes
			length := path.Length()
			i, pos := dashStart(c.dashOffset, d)
			if length <= pos+d[i] {
				if i%2 == 0 { // first dash covers whole path
					dashes = []float64{}
				} else { // first space covers whole path
					return
				}
			} else if i%2 == 0 { // starts with dash
				for pos+d[i] < length {
					pos += d[i]
					i++
					if i == len(d) {
						i = 0
					}
				}
				if i%2 == 0 { // ends with dash
					dashesClose = true
				}
			}
		}

		m := c.getCoordinateSystem().Translate(x, y).Mul(c.m)
		path = path.Transform(m)

		l := pathLayer{path, c.drawState, dashesClose}
		l.dashes = dashes
		l.fillRule = FillRule
		c.layers = append(c.layers, l)
	}
}

// DrawText draws text at position (x,y) using the current draw state. In particular, it only uses the current affine transformation matrix.
func (c *Canvas) DrawText(x, y float64, text *Text) {
	if !text.Empty() {
		coord := c.getCoordinateSystem().Dot(Point{x, y})
		c.layers = append(c.layers, textLayer{text, Identity.Translate(coord.X, coord.Y).Mul(c.m)})
	}
}

// DrawImage draws an image at position (x,y), using an image encoding (Lossy or Lossless) and DPM (dots-per-millimeter). A higher DPM will draw a smaller image.
func (c *Canvas) DrawImage(x, y float64, img image.Image, enc ImageEncoding, dpm float64) {
	if img.Bounds().Size().Eq(image.Point{}) {
		return
	}
	coord := c.getCoordinateSystem().Dot(Point{x, y})
	m := Identity.Translate(coord.X, coord.Y).Mul(c.m).Scale(1/dpm, 1/dpm)
	c.layers = append(c.layers, imageLayer{img, enc, m})
}

// Fit shrinks the canvas size so all elements fit. The elements are translated towards the origin when any left/bottom margins exist and the canvas size is decreased if any margins exist. It will maintain a given margin.
func (c *Canvas) Fit(margin float64) {
	// TODO: slow when we have many paths (see Graph example)
	rect := Rect{}
	for _, layer := range c.layers {
		rect = rect.Add(layer.Bounds())
	}
	for i, layer := range c.layers {
		switch l := layer.(type) {
		case pathLayer:
			l.path = l.path.Translate(-rect.X+margin, -rect.Y+margin)
			c.layers[i] = l
		case textLayer:
			l.m = Identity.Translate(-rect.X+margin, -rect.Y+margin).Mul(l.m)
			c.layers[i] = l
		case imageLayer:
			l.m = Identity.Translate(-rect.X+margin, -rect.Y+margin).Mul(l.m)
			c.layers[i] = l
		}
	}
	c.width = rect.W + 2*margin
	c.height = rect.H + 2*margin
}

// WriteSVG writes the stored drawing operations in Canvas in the SVG file format.
func (c *Canvas) WriteSVG(w io.Writer) error {
	svg := newSVGWriter(w, c.width, c.height)
	for _, l := range c.layers {
		l.WriteSVG(svg)
	}
	return svg.Close()
}

// WritePDF writes the stored drawing operations in Canvas in the PDF file format.
func (c *Canvas) WritePDF(w io.Writer) error {
	pdf := newPDFWriter(w)
	pdfPage := pdf.NewPage(c.width, c.height)
	for _, l := range c.layers {
		l.WritePDF(pdfPage)
	}
	return pdf.Close()
}

// WriteEPS writes the stored drawing operations in Canvas in the EPS file format.
// Be aware that EPS does not support transparency of colors.
func (c *Canvas) WriteEPS(w io.Writer) error {
	eps := newEPSWriter(w, c.width, c.height)
	for _, l := range c.layers {
		eps.Write([]byte("\n"))
		l.WriteEPS(eps)
	}
	return eps.Close()
}

// WriteImage writes the stored drawing operations in Canvas as a rasterized image with given DPM (dots-per-millimeter). Higher DPM will result in bigger images.
func (c *Canvas) WriteImage(dpm float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.width*dpm+0.5), int(c.height*dpm+0.5)))
	draw.Draw(img, img.Bounds(), image.NewUniform(White), image.Point{}, draw.Src)
	for _, l := range c.layers {
		l.DrawImage(img, dpm)
	}
	return img
}

// WriteImage writes the stored drawing operations in Canvas as a rasterized image with given DPM (dots-per-millimeter). Higher DPM will result in bigger images.
func (c *Canvas) ToOpenGL() *OpenGL {
	ogl := newOpenGL()
	for _, l := range c.layers {
		l.ToOpenGL(ogl)
	}
	return ogl
}
