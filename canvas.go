package canvas

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
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

func writePSColor(w io.Writer, color color.RGBA) {
	writeFloat64(w, float64(color.R))
	w.Write([]byte(" "))
	writeFloat64(w, float64(color.G))
	w.Write([]byte(" "))
	writeFloat64(w, float64(color.B))
}

func writeFloat64(w io.Writer, f float64) {
	buf := []byte{}
	w.Write(strconv.AppendFloat(buf, f, 'g', 5, 64))
}

////////////////////////////////////////////////////////////////

type layerType int

const (
	pathLayer layerType = iota
	textLayer
)

// layer is either a path or text, in which case only the respective field is set
type layer struct {
	t         layerType
	x, y, rot float64
	color     color.RGBA
	path      *Path
	text      *Text
}

type C struct {
	w, h  float64
	color color.RGBA

	layers []layer
	fonts  map[*Font]bool
}

func New(w, h float64) *C {
	return &C{w, h, Black, []layer{}, map[*Font]bool{}}
}

func (c *C) SetColor(color color.RGBA) {
	c.color = color
}

func (c *C) DrawPath(x, y, rot float64, p *Path) {
	p = p.Copy()
	c.layers = append(c.layers, layer{pathLayer, x, y, rot, c.color, p, nil})
}

func (c *C) DrawText(x, y, rot float64, text *Text) {
	for font := range text.fonts {
		c.fonts[font] = true
	}
	c.layers = append(c.layers, layer{textLayer, x, y, rot, c.color, nil, text})
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
		if l.t == pathLayer {
			p := l.path.Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y).Scale(1.0, -1.0).Translate(0.0, c.h)
			w.Write([]byte("<path d=\""))
			w.Write([]byte(p.ToSVG()))
			if l.color != Black {
				w.Write([]byte("\" fill=\""))
				writeCSSColor(w, l.color)
			}
			w.Write([]byte("\"/>"))
		} else if l.t == textLayer {
			w.Write([]byte(l.text.ToSVG(l.x, c.h-l.y, l.rot, l.color)))
		}
	}
	w.Write([]byte("</svg>"))
}

func (c *C) WritePDF(writer io.Writer) error {
	pdf := NewPDFWriter(writer, c.w, c.h)

	color := Black
	buf := &bytes.Buffer{}
	for _, l := range c.layers {
		if l.color.R != color.R || l.color.G != color.G || l.color.B != color.B {
			buf.WriteString(" ")
			writePSColor(buf, l.color)
			buf.WriteString(" rg")
		}
		if color.A != 255 {
			gs := pdf.GetOpacityGS(float64(color.A) / 255.0)
			buf.WriteString(fmt.Sprintf(" q /%v gs", gs))
		}
		if l.color != color {
			color = l.color
		}

		if l.t == textLayer {
			// TODO: embed fonts and draw text
			p := l.text.ToPath().Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			buf.WriteString(" ")
			buf.WriteString(p.ToPDF())
			p = l.text.ToPathDecorations().Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			buf.WriteString(" ")
			buf.WriteString(p.ToPDF())
		} else if l.t == pathLayer {
			p := l.path.Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			buf.WriteString(" ")
			buf.WriteString(p.ToPDF())
		}
		if color.A != 255 {
			buf.WriteString(" Q")
		}
	}

	_, _ = buf.ReadByte() // discard first space
	pdf.WriteObject(PDFStream{
		filters: []PDFFilter{},
		b:       buf.Bytes(),
	})
	return pdf.Close()
}

func (c *C) WriteImage(dpi float64) *image.RGBA {
	dpm := dpi * InchPerMm
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.w*dpm), int(c.h*dpm)))
	ras := vector.NewRasterizer(int(c.w*dpm), int(c.h*dpm))

	bg := Rectangle(0.0, 0.0, c.w, c.h)
	layers := append([]layer{{pathLayer, 0.0, 0.0, 0.0, White, bg, nil}}, c.layers...)
	dy := float32(c.h * dpm)

	draw := func(p *Path, color color.RGBA) {
		p.Replace(nil, nil, ellipseToBeziers)
		for i := 0; i < len(p.d); {
			cmd := p.d[i]
			switch cmd {
			case MoveToCmd:
				ras.MoveTo(float32(p.d[i+1]*dpm), dy-float32(p.d[i+2]*dpm))
			case LineToCmd:
				ras.LineTo(float32(p.d[i+1]*dpm), dy-float32(p.d[i+2]*dpm))
			case QuadToCmd:
				ras.QuadTo(float32(p.d[i+1]*dpm), dy-float32(p.d[i+2]*dpm), float32(p.d[i+3]*dpm), dy-float32(p.d[i+4]*dpm))
			case CubeToCmd:
				ras.CubeTo(float32(p.d[i+1]*dpm), dy-float32(p.d[i+2]*dpm), float32(p.d[i+3]*dpm), dy-float32(p.d[i+4]*dpm), float32(p.d[i+5]*dpm), dy-float32(p.d[i+6]*dpm))
			case ArcToCmd:
				panic("arcs should have been replaced")
			case CloseCmd:
				ras.ClosePath()
			}
			i += cmdLen(cmd)
		}
		if len(p.d) > 2 && p.d[len(p.d)-3] != CloseCmd {
			// implicitly close path
			ras.ClosePath()
		}
		size := ras.Size()
		ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(color), image.Point{})
		ras.Reset(size.X, size.Y)
	}

	for _, l := range layers {
		if l.t == textLayer {
			p := l.text.ToPath().Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			draw(p, l.color)
			p = l.text.ToPathDecorations().Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			draw(p, l.color)
		} else if l.t == pathLayer {
			p := l.path.Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			draw(p, l.color)
		}
	}
	return img
}

// WriteEPS writes out the image in the EPS file format.
// Be aware that EPS does not support transparency of colors.
// TODO: test ellipse, rotations etc
func (c *C) WriteEPS(w io.Writer) {
	w.Write([]byte("%!PS-Adobe-3.0 EPSF-3.0\n%%BoundingBox: 0 0 "))
	writeFloat64(w, c.w)
	w.Write([]byte(" "))
	writeFloat64(w, c.h)
	w.Write([]byte("\n"))

	// TODO: generate preview

	bg := Rectangle(0.0, 0.0, c.w, c.h)
	layers := append([]layer{{pathLayer, 0.0, 0.0, 0.0, White, bg, nil}}, c.layers...)

	color := Black
	for _, l := range layers {
		if l.color != color {
			w.Write([]byte(" "))
			writePSColor(w, l.color)
			w.Write([]byte(" setrgbcolor"))
			color = l.color
		}

		if l.t == textLayer {
			// TODO: embed fonts (convert TTF to Type 42) and draw text
			p := l.text.ToPath().Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			w.Write([]byte(" "))
			w.Write([]byte(p.ToPS()))
			p = l.text.ToPathDecorations().Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			w.Write([]byte(" "))
			w.Write([]byte(p.ToPS()))
		} else if l.t == pathLayer {
			p := l.path.Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			w.Write([]byte(" "))
			w.Write([]byte(p.ToPS()))
		}
	}
}

// TODO: check use cases and improve and test functionality. Shortcut for Cartesian / upper-left origin systems?
type View struct {
	C
	x0, y0         float64
	xScale, yScale float64
}

func NewView(c *C, x, y, w, h float64) *View {
	return &View{
		C:      *c,
		x0:     x,
		y0:     y,
		xScale: c.w / w,
		yScale: c.h / h,
	}
}

func (v *View) DrawPath(x, y, rot float64, p *Path) {
	p = p.Copy()
	p.Scale(v.xScale, v.yScale)
	v.C.DrawPath(v.x0+v.xScale*x, v.y0+v.yScale*y, rot, p)
}

func (v *View) DrawText(x, y, rot float64, t *Text) {
	v.C.DrawText(v.x0+v.xScale*x, v.y0+v.yScale*y, rot, t)
}
