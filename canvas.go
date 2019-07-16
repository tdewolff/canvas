package canvas

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/vector"
)

const mmPerPt = 0.3527777777777778
const ptPerMm = 2.8346456692913384
const mmPerInch = 25.4
const inchPerMm = 1 / 25.4

// Canvas holds the intermediate drawing state, accumulating all the layers (draw actions) and keeping track of the draw state. It allows for exporting to various target formats and using their native stroking and text features.
type Canvas struct {
	w, h   float64 // TODO: allow to set later on? Automatic fitting of entire canvas?
	layers []layer
	fonts  map[*Font]bool
	drawState
	stack []drawState
}

// New returns a new Canvas of given width and height in mm.
func New(w, h float64) *Canvas {
	return &Canvas{w, h, []layer{}, map[*Font]bool{}, defaultDrawState, nil}
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

// ComposeView multiplies the current affine transformation matrix by the given one.
func (c *Canvas) ComposeView(m Matrix) {
	c.m = m.Mul(c.m)
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
	c.dashOffset = dashOffset
	c.dashes = dashes
}

// DrawPath draws a path at position (x,y) using the current draw state.
func (c *Canvas) DrawPath(x, y float64, path *Path) {
	if c.fillColor.A == 0 && (c.strokeColor.A == 0 || c.strokeWidth == 0.0) {
		return
	}
	if !path.Empty() {
		path = path.Transform(Identity.Translate(x, y).Mul(c.m))
		c.drawState.fillRule = FillRule
		c.layers = append(c.layers, pathLayer{path, c.drawState})
	}
}

// DrawText draws text at position (x,y) using the current draw state. In particular, it only uses the current affine transformation matrix.
func (c *Canvas) DrawText(x, y float64, text *Text) {
	for font := range text.fonts {
		c.fonts[font] = true
	}
	// TODO: skip if empty
	c.layers = append(c.layers, textLayer{text, Identity.Translate(x, y).Mul(c.m)})
}

// ImageEncoding defines whether the embedded image shall be embedded as Lossless (typically PNG) or Lossy (typically JPG).
type ImageEncoding int

// see ImageEncoding
const (
	Lossless ImageEncoding = iota
	Lossy
)

// DrawImage draws an image at position (x,y), using an image encoding (Lossy or Lossless) and DPM (dots-per-millimeter). A higher DPM will draw a smaller image.
func (c *Canvas) DrawImage(x, y float64, img image.Image, enc ImageEncoding, dpm float64) {
	if img.Bounds().Size().Eq(image.Point{}) {
		return
	}
	m := Identity.Mul(c.m).Translate(x, y).Scale(1/dpm, 1/dpm)
	c.layers = append(c.layers, imageLayer{img, enc, m})
}

////////////////////////////////////////////////////////////////

// WriteSVG writes the stored drawing operations in Canvas in the SVG file format.
func (c *Canvas) WriteSVG(w io.Writer) {
	fmt.Fprintf(w, `<svg version="1.1" width="%.5g" height="%.5g" viewBox="0 0 %.5g %.5g" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, c.w, c.h, c.w, c.h)
	if len(c.fonts) > 0 {
		fmt.Fprintf(w, "<defs><style>")
		for f := range c.fonts {
			mimetype, raw := f.Raw()
			fmt.Fprintf(w, "\n@font-face{font-family:'%s';src:url('data:%s;base64,", f.name, mimetype)
			encoder := base64.NewEncoder(base64.StdEncoding, w)
			encoder.Write(raw)
			encoder.Close()
			fmt.Fprintf(w, "');}")
		}
		fmt.Fprintf(w, "\n</style></defs>")
	}
	for _, l := range c.layers {
		l.WriteSVG(w, c.h)
	}
	fmt.Fprintf(w, "</svg>")
}

// WritePDF writes the stored drawing operations in Canvas in the PDF file format.
func (c *Canvas) WritePDF(w io.Writer) error {
	pdf := newPDFWriter(w)
	pdfpage := pdf.NewPage(c.w, c.h)
	for _, l := range c.layers {
		l.WritePDF(pdfpage)
	}
	return pdf.Close()
}

// WriteEPS writes the stored drawing operations in Canvas in the EPS file format.
// Be aware that EPS does not support transparency of colors.
func (c *Canvas) WriteEPS(w io.Writer) {
	eps := newEPSWriter(w, c.w, c.h)
	for _, l := range c.layers {
		eps.Write([]byte("\n"))
		l.WriteEPS(eps)
	}
}

// WriteImage writes the stored drawing operations in Canvas as a rasterized image with given DPM (dots-per-millimeter). Higher DPM will result in bigger images.
func (c *Canvas) WriteImage(dpm float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.w*dpm+0.5), int(c.h*dpm+0.5)))
	draw.Draw(img, img.Bounds(), image.NewUniform(White), image.Point{}, draw.Src)
	for _, l := range c.layers {
		l.WriteImage(img, dpm)
	}
	return img
}

////////////////////////////////////////////////////////////////

type layer interface {
	WriteSVG(io.Writer, float64)
	WritePDF(*pdfPageWriter)
	WriteEPS(*epsWriter)
	WriteImage(*image.RGBA, float64)
}

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

////////////////////////////////////////////////////////////////

type pathLayer struct {
	path *Path
	drawState
}

func (l pathLayer) WriteSVG(w io.Writer, h float64) {
	fill := l.fillColor.A != 0
	stroke := l.strokeColor.A != 0 && 0.0 < l.strokeWidth

	p := l.path.Transform(Identity.Translate(0.0, h).ReflectY())
	fmt.Fprintf(w, `<path d="%s`, p.ToSVG())

	strokeUnsupported := false
	if arcs, ok := l.strokeJoiner.(arcsJoiner); ok && math.IsNaN(arcs.limit) {
		strokeUnsupported = true
	} else if miter, ok := l.strokeJoiner.(miterJoiner); ok {
		if math.IsNaN(miter.limit) {
			strokeUnsupported = true
		} else if _, ok := miter.gapJoiner.(bevelJoiner); !ok {
			strokeUnsupported = true
		}
	}

	if !stroke {
		if fill {
			if l.fillColor != Black {
				fmt.Fprintf(w, `" fill="%s`, toCSSColor(l.fillColor))
			}
			if l.fillRule == EvenOdd {
				fmt.Fprintf(w, `" fill-rule="evenodd`)
			}
		} else {
			fmt.Fprintf(w, `" fill="none`)
		}
	} else {
		style := &strings.Builder{}
		if fill {
			if l.fillColor != Black {
				fmt.Fprintf(style, ";fill:%s", toCSSColor(l.fillColor))
			}
			if l.fillRule == EvenOdd {
				fmt.Fprintf(style, ";fill-rule:evenodd")
			}
		} else {
			fmt.Fprintf(style, ";fill:none")
		}
		if stroke && !strokeUnsupported {
			fmt.Fprintf(style, `;stroke:%s`, toCSSColor(l.strokeColor))
			if l.strokeWidth != 1.0 {
				fmt.Fprintf(style, ";stroke-width:%.5g", l.strokeWidth)
			}
			if _, ok := l.strokeCapper.(roundCapper); ok {
				fmt.Fprintf(style, ";stroke-linecap:round")
			} else if _, ok := l.strokeCapper.(squareCapper); ok {
				fmt.Fprintf(style, ";stroke-linecap:square")
			} else if _, ok := l.strokeCapper.(buttCapper); !ok {
				panic("SVG: line cap not support")
			}
			if _, ok := l.strokeJoiner.(bevelJoiner); ok {
				fmt.Fprintf(style, ";stroke-linejoin:bevel")
			} else if _, ok := l.strokeJoiner.(roundJoiner); ok {
				fmt.Fprintf(style, ";stroke-linejoin:round")
			} else if arcs, ok := l.strokeJoiner.(arcsJoiner); ok && !math.IsNaN(arcs.limit) {
				fmt.Fprintf(style, ";stroke-linejoin:arcs")
				if arcs.limit != 4.0 {
					fmt.Fprintf(style, ";stroke-miterlimit:%.5g", arcs.limit)
				}
			} else if miter, ok := l.strokeJoiner.(miterJoiner); ok && !math.IsNaN(miter.limit) {
				// a miter line join is the default
				if miter.limit != 4.0 {
					fmt.Fprintf(style, ";stroke-miterlimit:%.5g", miter.limit)
				}
			} else {
				panic("SVG: line join not support")
			}

			if 0 < len(l.dashes) {
				fmt.Fprintf(style, ";stroke-dasharray:%.5g", l.dashes[0])
				for _, dash := range l.dashes[1:] {
					fmt.Fprintf(style, " %.5g", dash)
				}
				if 0.0 != l.dashOffset {
					fmt.Fprintf(style, ";stroke-dashoffset:%.5g", l.dashOffset)
				}
			}
		}
		if 0 < style.Len() {
			fmt.Fprintf(w, `" style="%s`, style.String()[1:])
		}
	}
	fmt.Fprintf(w, `"/>`)

	if stroke && strokeUnsupported {
		// stroke settings unsupported by PDF, draw stroke explicitly
		if 0 < len(l.dashes) {
			p = p.Dash(l.dashOffset, l.dashes...)
		}
		p = p.Stroke(l.strokeWidth, l.strokeCapper, l.strokeJoiner)
		fmt.Fprintf(w, `<path d="%s`, p.ToSVG())
		if l.strokeColor != Black {
			fmt.Fprintf(w, `" fill="%s`, toCSSColor(l.strokeColor))
		}
		if l.fillRule == EvenOdd {
			fmt.Fprintf(w, `" fill-rule="evenodd`)
		}
		fmt.Fprintf(w, `"/>`)
	}
}

func (l pathLayer) WritePDF(w *pdfPageWriter) {
	fill := l.fillColor.A != 0
	stroke := l.strokeColor.A != 0 && 0.0 < l.strokeWidth

	closed := false
	data := l.path.ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	differentAlpha := fill && stroke && l.fillColor.A != l.strokeColor.A

	// PDFs don't support the arcs joiner, miter joiner (not clipped), or miter-clip joiner with non-bevel fallback
	// TODO: PDF does not support connecting first and last dashes if path is closed
	strokeUnsupported := false
	if _, ok := l.strokeJoiner.(arcsJoiner); ok {
		strokeUnsupported = true
	} else if miter, ok := l.strokeJoiner.(miterJoiner); ok {
		if math.IsNaN(miter.limit) {
			strokeUnsupported = true
		} else if _, ok := miter.gapJoiner.(bevelJoiner); !ok {
			strokeUnsupported = true
		}
	}

	if !stroke || !strokeUnsupported {
		if fill && !stroke {
			w.SetFillColor(l.fillColor)
			w.Write([]byte(" "))
			w.Write([]byte(data))
			w.Write([]byte(" f"))
			if l.fillRule == EvenOdd {
				w.Write([]byte("*"))
			}
		} else if !fill && stroke {
			w.SetStrokeColor(l.strokeColor)
			w.SetLineWidth(l.strokeWidth)
			w.SetLineCap(l.strokeCapper)
			w.SetLineJoin(l.strokeJoiner)
			w.SetDashes(l.dashOffset, l.dashes)
			w.Write([]byte(" "))
			w.Write([]byte(data))
			if closed {
				w.Write([]byte(" s"))
			} else {
				w.Write([]byte(" S"))
			}
			if l.fillRule == EvenOdd {
				w.Write([]byte("*"))
			}
		} else if fill && stroke {
			if !differentAlpha {
				w.SetFillColor(l.fillColor)
				w.SetStrokeColor(l.strokeColor)
				w.SetLineWidth(l.strokeWidth)
				w.SetLineCap(l.strokeCapper)
				w.SetLineJoin(l.strokeJoiner)
				w.SetDashes(l.dashOffset, l.dashes)
				w.Write([]byte(" "))
				w.Write([]byte(data))
				if closed {
					w.Write([]byte(" b"))
				} else {
					w.Write([]byte(" B"))
				}
				if l.fillRule == EvenOdd {
					w.Write([]byte("*"))
				}
			} else {
				w.SetFillColor(l.fillColor)
				w.Write([]byte(" "))
				w.Write([]byte(data))
				w.Write([]byte(" f"))
				if l.fillRule == EvenOdd {
					w.Write([]byte("*"))
				}

				w.SetStrokeColor(l.strokeColor)
				w.SetLineWidth(l.strokeWidth)
				w.SetLineCap(l.strokeCapper)
				w.SetLineJoin(l.strokeJoiner)
				w.SetDashes(l.dashOffset, l.dashes)
				w.Write([]byte(" "))
				w.Write([]byte(data))
				if closed {
					w.Write([]byte(" s"))
				} else {
					w.Write([]byte(" S"))
				}
				if l.fillRule == EvenOdd {
					w.Write([]byte("*"))
				}
			}
		}
	} else {
		// stroke && strokeUnsupported
		if fill {
			w.SetFillColor(l.fillColor)
			w.Write([]byte(" "))
			w.Write([]byte(data))
			w.Write([]byte(" f"))
			if l.fillRule == EvenOdd {
				w.Write([]byte("*"))
			}
		}

		// stroke settings unsupported by PDF, draw stroke explicitly
		strokePath := l.path
		if 0 < len(l.dashes) {
			strokePath = strokePath.Dash(l.dashOffset, l.dashes...)
		}
		strokePath = strokePath.Stroke(l.strokeWidth, l.strokeCapper, l.strokeJoiner)

		w.SetFillColor(l.strokeColor)
		w.Write([]byte(" "))
		w.Write([]byte(strokePath.ToPDF()))
		w.Write([]byte(" f"))
		if l.fillRule == EvenOdd {
			w.Write([]byte("*"))
		}
	}
}

func (l pathLayer) WriteEPS(w *epsWriter) {
	// TODO: EPS test ellipse, rotations etc
	w.SetColor(l.fillColor)
	w.Write([]byte(" "))
	w.Write([]byte(l.path.ToPS()))
	w.Write([]byte(" fill"))
	// TODO: EPS add drawState support
}

func (l pathLayer) WriteImage(img *image.RGBA, dpm float64) {
	// TODO: use fillRule
	w, h := img.Bounds().Size().X, img.Bounds().Size().Y
	if l.fillColor.A != 0 {
		ras := vector.NewRasterizer(w, h)
		l.path.ToRasterizer(ras, dpm)
		size := ras.Size()
		ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.fillColor), image.Point{})
	}
	if l.strokeColor.A != 0 && 0.0 < l.strokeWidth {
		strokePath := l.path
		if 0 < len(l.dashes) {
			strokePath = strokePath.Dash(l.dashOffset, l.dashes...)
		}
		strokePath = strokePath.Stroke(l.strokeWidth, l.strokeCapper, l.strokeJoiner)

		ras := vector.NewRasterizer(w, h)
		strokePath.ToRasterizer(ras, dpm)
		size := ras.Size()
		ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.strokeColor), image.Point{})
	}
}

////////////////////////////////////////////////////////////////

type textLayer struct {
	*Text
	viewport Matrix
}

func (l textLayer) WriteSVG(w io.Writer, h float64) {
	l.Text.WriteSVG(w, h, l.viewport)
}

func (l textLayer) WritePDF(w *pdfPageWriter) {
	l.Text.WritePDF(w, l.viewport)
}

func (l textLayer) WriteEPS(w *epsWriter) {
	// TODO: EPS write text
	paths, colors := l.ToPaths()
	for i, path := range paths {
		state := defaultDrawState
		state.fillColor = colors[i]
		pathLayer{path.Transform(l.viewport), state}.WriteEPS(w)
	}
}

func (l textLayer) WriteImage(img *image.RGBA, dpm float64) {
	//l.Text.WriteImage(img, dpm)
	paths, colors := l.ToPaths()
	for i, path := range paths {
		state := defaultDrawState
		state.fillColor = colors[i]
		pathLayer{path.Transform(l.viewport), state}.WriteImage(img, dpm)
	}
}

////////////////////////////////////////////////////////////////

type imageLayer struct {
	img image.Image
	enc ImageEncoding
	m   Matrix
}

func (l imageLayer) WriteSVG(w io.Writer, h float64) {
	mimetype := "image/png"
	if l.enc == Lossy {
		mimetype = "image/jpg"
	}

	m := l.m.Translate(0.0, float64(l.img.Bounds().Size().Y))
	fmt.Fprintf(w, `<image transform="%v" width="%d" height="%d" xlink:href="data:%s;base64,`,
		m.ToSVG(h), l.img.Bounds().Size().X, l.img.Bounds().Size().Y, mimetype)

	encoder := base64.NewEncoder(base64.StdEncoding, w)
	if l.enc == Lossy {
		if err := jpeg.Encode(encoder, l.img, nil); err != nil {
			panic(err)
		}
	} else {
		if err := png.Encode(encoder, l.img); err != nil {
			panic(err)
		}
	}
	if err := encoder.Close(); err != nil {
		panic(err)
	}

	fmt.Fprintf(w, `"/>`)
}

func (l imageLayer) WritePDF(w *pdfPageWriter) {
	w.DrawImage(l.img, l.enc, l.m)
}

func (l imageLayer) WriteEPS(w *epsWriter) {
	// TODO: EPS write image
}

func (l imageLayer) WriteImage(img *image.RGBA, dpm float64) {
	m := l.m.Scale(dpm, dpm)
	h := float64(img.Bounds().Size().Y)
	origin := l.m.Dot(Point{0, float64(l.img.Bounds().Size().Y)})
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X * dpm, -m[1][0], m[1][1], h - origin.Y*dpm}
	draw.CatmullRom.Transform(img, aff3, l.img, l.img.Bounds(), draw.Over, nil)
}
