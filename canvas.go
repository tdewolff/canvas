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

var (
	Black            color.Color = color.RGBA{0, 0, 0, 255}
	Grey             color.Color = color.RGBA{128, 128, 128, 255}
	White            color.Color = color.RGBA{0, 0, 0, 255}
	Red              color.Color = color.RGBA{255, 0, 0, 255}
	Lime             color.Color = color.RGBA{0, 255, 0, 255}
	Blue             color.Color = color.RGBA{0, 0, 255, 255}
	Yellow           color.Color = color.RGBA{255, 255, 0, 255}
	Magenta          color.Color = color.RGBA{255, 0, 255, 255}
	Cyan             color.Color = color.RGBA{0, 255, 255, 255}
	BlackTransparent color.Color = color.RGBA{0, 0, 0, 128}
	DimGrey          color.Color = color.RGBA{105, 105, 105, 255}
	DarkGrey         color.Color = color.RGBA{169, 169, 169, 255}
	Silver           color.Color = color.RGBA{192, 192, 192, 255}
	LightGrey        color.Color = color.RGBA{211, 211, 211, 255}
	Gainsboro        color.Color = color.RGBA{220, 220, 220, 255}
	WhiteSmoke       color.Color = color.RGBA{245, 245, 245, 255}
	SteelBlue        color.Color = color.RGBA{70, 130, 180, 255}
	SlateGrey        color.Color = color.RGBA{112, 128, 144, 255}
	LightSteelBlue   color.Color = color.RGBA{176, 196, 222, 255}
	LightSlateGrey   color.Color = color.RGBA{119, 136, 153, 255}
	DarkSlateBlue    color.Color = color.RGBA{72, 61, 139, 255}
	DarkSlateGrey    color.Color = color.RGBA{47, 79, 79, 255}
	OrangeRed        color.Color = color.RGBA{255, 69, 0, 255}
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
	t         layerType
	x, y, rot float64
	color     color.Color
	path      *Path
	text      *Text
}

type C struct {
	w, h  float64
	color color.Color

	layers []layer
	fonts  map[*Font]bool
}

func New(w, h float64) *C {
	return &C{w, h, color.Black, []layer{}, map[*Font]bool{}}
}

func (c *C) SetColor(col color.Color) {
	c.color = col
}

func (c *C) DrawPath(x, y, rot float64, p *Path) {
	p = p.Copy()
	c.layers = append(c.layers, layer{pathLayer, x, y, rot, c.color, p, nil})
}

func (c *C) DrawText(x, y, rot float64, text *Text) {
	for _, font := range text.fonts {
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
			if l.color != color.Black {
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
		R, G, B, _ := color.RGBA()
		r, g, b, a := l.color.RGBA()
		if r != R || g != G || b != B {
			buf.WriteString(" ")
			writePSColor(buf, l.color)
			buf.WriteString(" rg")
		}
		if a != 65535.0 {
			gs := pdf.GetOpacityGS(float64(a) / 65535.0)
			buf.WriteString(fmt.Sprintf(" q /%v gs", gs))
		}
		if l.color != color {
			color = l.color
		}

		if l.t == textLayer {
			// TODO: embed fonts and draw text
			l.path = l.text.ToPath()
			l.t = pathLayer
		}

		if l.t == pathLayer {
			p := l.path.Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
			buf.WriteString(" ")
			buf.WriteString(p.ToPDF())
		}
		if a != 65535.0 {
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
	layers := append([]layer{{pathLayer, 0.0, 0.0, 0.0, color.White, bg, nil}}, c.layers...)

	dy := float32(c.h * dpm)
	for _, l := range layers {
		if l.t == textLayer {
			l.path = l.text.ToPath()
			l.t = pathLayer
		}

		if l.t == pathLayer {
			p := l.path.Copy().Rotate(l.rot, 0.0, 0.0).Translate(l.x, l.y)
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
			ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.color), image.Point{})
			ras.Reset(size.X, size.Y)
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
	layers := append([]layer{{pathLayer, 0.0, 0.0, 0.0, color.White, bg, nil}}, c.layers...)

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
			l.path = l.text.ToPath()
			l.t = pathLayer
		}

		if l.t == pathLayer {
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
