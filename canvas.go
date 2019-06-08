package canvas

import (
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"io"
	"strconv"

	"golang.org/x/image/vector"
)

const MmPerPt = 0.3527777777777778
const PtPerMm = 2.8346456692913384
const MmPerInch = 25.4
const InchPerMm = 1 / 25.4

func writeCSSColor(w io.Writer, color color.RGBA) {
	if color.A == 255 {
		buf := make([]byte, 7)
		buf[0] = '#'
		hex.Encode(buf[1:], []byte{color.R, color.G, color.B})
		w.Write(buf)
	} else {
		buf := make([]byte, 0, 24)
		buf = append(buf, []byte("rgba(")...)
		buf = strconv.AppendInt(buf, int64(color.R), 10)
		buf = append(buf, ',')
		buf = strconv.AppendInt(buf, int64(color.G), 10)
		buf = append(buf, ',')
		buf = strconv.AppendInt(buf, int64(color.B), 10)
		buf = append(buf, ',')
		buf = strconv.AppendFloat(buf, float64(color.A)/255.0, 'g', 4, 64)
		buf = append(buf, ')')
		w.Write(buf)
	}
}

func writeFloat64(w io.Writer, f float64) {
	buf := []byte{}
	w.Write(strconv.AppendFloat(buf, f, 'g', 5, 64))
}

////////////////////////////////////////////////////////////////

type layer interface {
	WriteSVG(io.Writer, float64)
	WritePDF(*PDFPageWriter)
	WriteEPS(*EPSWriter)
	WriteImage(*image.RGBA, float64, float64, float64)
}

type C struct {
	w, h   float64
	layers []layer
	fonts  map[*Font]bool
	// transformation matrix
	pathState
}

func New(w, h float64) *C {
	return &C{w, h, []layer{}, map[*Font]bool{}, pathState{}}
}

func (c *C) SetFill(color color.RGBA) {
	c.fillColor = color
}

func (c *C) SetStroke(color color.RGBA, width float64, capper Capper, joiner Joiner) {
	c.strokeColor = color
	c.strokeWidth = width
	c.strokeCapper = capper
	c.strokeJoiner = joiner
}

func (c *C) SetDashes(dashOffset float64, dashes ...float64) {
	c.dashOffset = dashOffset
	c.dashes = dashes
}

func (c *C) DrawPath(x, y float64, path *Path) {
	path = path.Copy().Translate(x, y)
	c.layers = append(c.layers, pathLayer{path, c.pathState})
}

func (c *C) DrawText(x, y float64, text *Text) {
	for font := range text.fonts {
		c.fonts[font] = true
	}
	c.layers = append(c.layers, textLayer{text, x, y, 0.0})
}

func (c *C) WriteSVG(w io.Writer) {
	w.Write([]byte("<svg xmlns=\"http://www.w3.org/2000/svg\" version=\"1.1\" shape-rendering=\"geometricPrecision\" width=\""))
	writeFloat64(w, c.w)
	w.Write([]byte("mm\" height=\""))
	writeFloat64(w, c.h)
	w.Write([]byte("mm\" viewBox=\"0 0 "))
	writeFloat64(w, c.w)
	w.Write([]byte(" "))
	writeFloat64(w, c.h)
	w.Write([]byte("\">"))
	if len(c.fonts) > 0 {
		w.Write([]byte("<defs><style>"))
		for f := range c.fonts {
			w.Write([]byte("@font-face { font-family: '"))
			w.Write([]byte(f.name))
			w.Write([]byte("'; src: url('"))
			w.Write([]byte(f.ToDataURI()))
			w.Write([]byte("'); }\n"))
		}
		w.Write([]byte("</style></defs>"))
	}
	for _, l := range c.layers {
		l.WriteSVG(w, c.h)
	}
	w.Write([]byte("</svg>"))
}

func (c *C) WritePDF(w io.Writer) error {
	pdf := NewPDFWriter(w)
	pdfpage := pdf.NewPage(c.w, c.h)
	for _, l := range c.layers {
		l.WritePDF(pdfpage)
	}
	return pdf.Close()
}

// WriteEPS writes out the image in the EPS file format.
// Be aware that EPS does not support transparency of colors.
// TODO: test ellipse, rotations etc
func (c *C) WriteEPS(w io.Writer) {
	eps := NewEPSWriter(w, c.w, c.h)
	for _, l := range c.layers {
		eps.Write([]byte("\n"))
		l.WriteEPS(eps)
	}
}

func (c *C) WriteImage(dpi float64) *image.RGBA {
	dpm := dpi * InchPerMm
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.w*dpm+0.5), int(c.h*dpm+0.5)))
	draw.Draw(img, img.Bounds(), image.NewUniform(White), image.Point{}, draw.Src)
	for _, l := range c.layers {
		l.WriteImage(img, dpm, c.w, c.h)
	}
	return img
}

type pathLayer struct {
	path *Path
	pathState
}

type pathState struct {
	fillColor, strokeColor color.RGBA
	strokeWidth            float64
	strokeCapper           Capper
	strokeJoiner           Joiner
	dashOffset             float64
	dashes                 []float64
}

func (l pathLayer) WriteSVG(w io.Writer, h float64) {
	p := l.path.Copy().Scale(1.0, -1.0).Translate(0.0, h)
	w.Write([]byte("<path d=\""))
	w.Write([]byte(p.ToSVG()))
	if l.fillColor != Black {
		w.Write([]byte("\" fill=\""))
		writeCSSColor(w, l.fillColor)
	}
	w.Write([]byte("\"/>"))
}

func (l pathLayer) WritePDF(w *PDFPageWriter) {
	w.SetColor(l.fillColor)
	w.Write([]byte(" "))
	w.Write([]byte(l.path.ToPDF()))
	w.Write([]byte(" f"))
}

// WriteEPS writes out the image in the EPS file format.
// Be aware that EPS does not support transparency of colors.
// TODO: test ellipse, rotations etc
func (l pathLayer) WriteEPS(w *EPSWriter) {
	w.SetColor(l.fillColor)
	w.Write([]byte(" "))
	w.Write([]byte(l.path.ToPS()))
	w.Write([]byte(" fill"))
}

func (l pathLayer) WriteImage(img *image.RGBA, dpm, w, h float64) {
	ras := vector.NewRasterizer(int(w*dpm+0.5), int(h*dpm+0.5))
	l.path.ToRasterizer(ras, dpm, w, h)
	size := ras.Size()
	ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.fillColor), image.Point{})
}

type textLayer struct {
	*Text
	x, y, rot float64
}

func (l textLayer) WriteSVG(w io.Writer, h float64) {
	l.Text.WriteSVG(w, l.x, h-l.y, l.rot)
}

func (l textLayer) WritePDF(w *PDFPageWriter) {
	// TODO
	paths, colors := l.ToPaths()
	for i, path := range paths {
		path.Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
		pathLayer{path, pathState{fillColor: colors[i]}}.WritePDF(w)
	}
}

func (l textLayer) WriteEPS(w *EPSWriter) {
	// TODO
	paths, colors := l.ToPaths()
	for i, path := range paths {
		path.Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
		pathLayer{path, pathState{fillColor: colors[i]}}.WriteEPS(w)
	}
}

func (l textLayer) WriteImage(img *image.RGBA, dpm, w, h float64) {
	paths, colors := l.ToPaths()
	for i, path := range paths {
		path.Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
		pathLayer{path, pathState{fillColor: colors[i]}}.WriteImage(img, dpm, w, h)
	}
}
