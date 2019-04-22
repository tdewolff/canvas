package canvas

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
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

var (
	Black   = color.RGBA{0, 0, 0, 255}
	White   = color.RGBA{0, 0, 0, 255}
	Grey    = color.RGBA{128, 128, 128, 255}
	Red     = color.RGBA{255, 0, 0, 255}
	Lime    = color.RGBA{0, 255, 0, 255}
	Blue    = color.RGBA{0, 0, 255, 255}
	Yellow  = color.RGBA{255, 255, 0, 255}
	Magenta = color.RGBA{255, 0, 255, 255}
	Cyan    = color.RGBA{0, 255, 255, 255}
)

func writeCSSColor(w io.Writer, c color.Color) {
	r, g, b, a := c.RGBA()
	rgba := [4]byte{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	if rgba[3] == 0xff {
		buf := make([]byte, 7)
		buf[0] = '#'
		hex.Encode(buf[1:], rgba[:3])
		w.Write(buf)
	} else {
		buf := make([]byte, 0, 24)
		buf = append(buf, []byte("rgba(")...)
		buf = strconv.AppendInt(buf, int64(rgba[0]), 10)
		buf = append(buf, ',')
		buf = strconv.AppendInt(buf, int64(rgba[1]), 10)
		buf = append(buf, ',')
		buf = strconv.AppendInt(buf, int64(rgba[2]), 10)
		buf = append(buf, ',')
		buf = strconv.AppendFloat(buf, float64(rgba[3])/0xff, 'g', 4, 64)
		buf = append(buf, ')')
		w.Write(buf)
	}
}

func writePSColor(w io.Writer, c color.Color) {
	r, g, b, _ := c.RGBA()
	rf, gf, bf := float64(r)/65535.0, float64(g)/65535.0, float64(b)/65535.0

	writeFloat64(w, rf)
	w.Write([]byte(" "))
	writeFloat64(w, gf)
	w.Write([]byte(" "))
	writeFloat64(w, bf)
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
	t        layerType
	x, y     float64
	color    color.Color
	fontFace FontFace
	path     *Path
	text     string
}

type C struct {
	w, h     float64
	color    color.Color
	fontFace FontFace

	layers []layer
	fonts  []*Font
}

func New() *C {
	return &C{0.0, 0.0, color.Black, FontFace{}, []layer{}, []*Font{}}
}

func (c *C) Open(w, h float64) {
	c.w = w
	c.h = h
}

func (c *C) SetColor(col color.Color) {
	c.color = col
}

func (c *C) SetFont(fontFace FontFace) {
	c.fontFace = fontFace
}

func (c *C) DrawPath(x, y float64, p *Path) {
	p = p.Copy()
	c.layers = append(c.layers, layer{pathLayer, x, y, c.color, c.fontFace, p, ""})
}

func (c *C) DrawText(x, y float64, s string) {
	c.layers = append(c.layers, layer{textLayer, x, y, c.color, c.fontFace, nil, s})
	c.fonts = append(c.fonts, c.fontFace.f)
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
		for _, f := range c.fonts {
			w.Write([]byte("@font-face { font-family: '"))
			w.Write([]byte(f.name))
			w.Write([]byte("'; src: url('data:"))
			w.Write([]byte(f.mimetype))
			w.Write([]byte(";base64,"))
			b64 := base64.NewEncoder(base64.StdEncoding, w)
			b64.Write(f.raw)
			b64.Close()
			w.Write([]byte("'); }\n"))
			w.Write([]byte("</style></defs>"))
		}
	}
	for _, l := range c.layers {
		if l.t == pathLayer {
			p := l.path.Copy().Translate(l.x, l.y)
			w.Write([]byte("<path d=\""))
			w.Write([]byte(p.String()))
			if l.color != color.Black {
				w.Write([]byte("\" fill=\""))
				writeCSSColor(w, l.color)
			}
			w.Write([]byte("\"/>"))
		} else if l.t == textLayer {
			// TODO: use tspan for newlines
			name, style, size := l.fontFace.Info()
			w.Write([]byte("<text x=\""))
			writeFloat64(w, l.x)
			w.Write([]byte("\" y=\""))
			writeFloat64(w, l.y)
			w.Write([]byte("\" font-family=\""))
			w.Write([]byte(name))
			w.Write([]byte("\" font-size=\""))
			writeFloat64(w, size)
			if style&Italic != 0 {
				w.Write([]byte("\" font-style=\"italic"))
			}
			if style&Bold != 0 {
				w.Write([]byte("\" font-weight=\"bold"))
			}
			if c.color != color.Black {
				w.Write([]byte("\" fill=\""))
				writeCSSColor(w, l.color)
			}
			w.Write([]byte("\">"))
			w.Write([]byte(l.text))
			w.Write([]byte("</text>"))
		}
	}
	w.Write([]byte("</svg>"))
}

func (c *C) WritePDF(writer io.Writer) error {
	w := NewPDFWriter(writer, 300, 300)

	b := &bytes.Buffer{}
	for _, l := range c.layers {
		if l.t == textLayer {
			// TODO: embed fonts and draw text
			l.path = l.fontFace.ToPath(l.text)
			l.t = pathLayer
		}

		if l.t == pathLayer {
			l.path.Translate(150.0, 150.0)
			b.WriteString(" ")
			writePSColor(b, l.color)
			b.WriteString(" rg ")
			b.WriteString(l.path.ToPDF())
		}
	}

	w.WriteObject(PDFStream{
		filters: []PDFFilter{PDFFilterFlate},
		b:       b.Bytes(),
	})
	return w.Close()
}

func (c *C) WriteImage(dpi float64) *image.RGBA {
	dpm := dpi * InchPerMm
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.w*dpm), int(c.h*dpm)))
	ras := vector.NewRasterizer(int(c.w*dpm), int(c.h*dpm))

	bg := Rectangle(0.0, 0.0, c.w, c.h)
	layers := append([]layer{{pathLayer, 0.0, 0.0, color.White, FontFace{}, bg, ""}}, c.layers...)

	for _, l := range layers {
		if l.t == textLayer {
			l.path = l.fontFace.ToPath(l.text)
			l.t = pathLayer
		}

		if l.t == pathLayer {
			p := l.path.Copy().Translate(l.x, l.y)
			p.Replace(nil, nil, ellipseToBeziers)

			for i := 0; i < len(p.d); {
				cmd := p.d[i]
				switch cmd {
				case MoveToCmd:
					ras.MoveTo(float32(p.d[i+1]*dpm), float32(p.d[i+2]*dpm))
				case LineToCmd:
					ras.LineTo(float32(p.d[i+1]*dpm), float32(p.d[i+2]*dpm))
				case QuadToCmd:
					ras.QuadTo(float32(p.d[i+1]*dpm), float32(p.d[i+2]*dpm), float32(p.d[i+3]*dpm), float32(p.d[i+4]*dpm))
				case CubeToCmd:
					ras.CubeTo(float32(p.d[i+1]*dpm), float32(p.d[i+2]*dpm), float32(p.d[i+3]*dpm), float32(p.d[i+4]*dpm), float32(p.d[i+5]*dpm), float32(p.d[i+6]*dpm))
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
			ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.color), image.Point{})
			ras.Reset(size.X, size.Y)
		}
	}
	return img
}

func (c *C) WriteEPS(w io.Writer) {
	w.Write([]byte("%!PS-Adobe-3.0 EPSF-3.0\n%%BoundingBox: 0 0 "))
	writeFloat64(w, c.w)
	w.Write([]byte(" "))
	writeFloat64(w, c.h)
	w.Write([]byte("\n"))

	// TODO: generate preview

	bg := Rectangle(0.0, 0.0, c.w, c.h)
	layers := append([]layer{{pathLayer, 0.0, 0.0, color.White, FontFace{}, bg, ""}}, c.layers...)

	for _, l := range layers {
		writePSColor(w, l.color)
		w.Write([]byte(" setrgbcolor\n"))

		if l.t == textLayer {
			// TODO: embed fonts (convert TTF to Type 42) and draw text
			l.path = l.fontFace.ToPath(l.text)
			l.t = pathLayer
		}

		if l.t == pathLayer {
			p := l.path.Copy().Translate(l.x, l.y).Scale(1.0, -1.0).Translate(0.0, c.h)
			w.Write([]byte(p.ToPS()))
			w.Write([]byte(" fill\n"))
		}
	}
}
