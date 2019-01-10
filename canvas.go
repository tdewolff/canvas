package canvas

import (
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"io"
	"math"
	"strconv"

	"golang.org/x/image/vector"

	"github.com/jung-kurt/gofpdf"
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

type layer struct {
	t        layerType
	x, y     float64
	color    color.Color
	fontFace FontFace
	path     *Path
	text     string
}

type C struct {
	w, h, dpi float64
	color     color.Color
	fontFace  FontFace

	layers []layer
	fonts  []*Font
}

func New(dpi float64) *C {
	return &C{0.0, 0.0, dpi, color.Black, FontFace{}, []layer{}, []*Font{}}
}

func (c *C) Open(w, h float64) {
	c.w = w
	c.h = h
}

func (c *C) DPI() float64 {
	return c.dpi
}

func (c *C) SetColor(col color.Color) {
	c.color = col
}

func (c *C) SetFont(fontFace FontFace) {
	c.fontFace = fontFace
}

func (c *C) DrawPath(x, y float64, p *Path) {
	c.layers = append(c.layers, layer{pathLayer, x, y, c.color, c.fontFace, p.Copy(), ""})
}

func (c *C) DrawText(x, y float64, s string) {
	c.layers = append(c.layers, layer{textLayer, x, y, c.color, c.fontFace, nil, s})
	c.fonts = append(c.fonts, c.fontFace.font)
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

func (c *C) WritePDF(pdf *gofpdf.Fpdf) {
	pdf.AddPageFormat("P", gofpdf.SizeType{c.w, c.h})
	for _, l := range c.layers {
		r, g, b, a := l.color.RGBA()
		pdf.SetDrawColor(int(r), int(g), int(b))
		pdf.SetFillColor(int(r), int(g), int(b))
		pdf.SetAlpha(float64(a)/0xffff, "Normal")

		if l.t == pathLayer {
			p := l.path.Copy().Translate(l.x, l.y)
			for i := 0; i < len(p.d); {
				cmd := p.d[i]
				switch cmd {
				case MoveToCmd:
					pdf.MoveTo(p.d[i+1], p.d[i+2])
				case LineToCmd:
					pdf.LineTo(p.d[i+1], p.d[i+2])
				case QuadToCmd:
					pdf.CurveTo(p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4])
				case CubeToCmd:
					pdf.CurveBezierCubicTo(p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4], p.d[i+5], p.d[i+6])
				case ArcToCmd:
					x1, y1 := pdf.GetX(), pdf.GetY()
					rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]*math.Pi/180
					largeArc, sweep := fromArcFlags(p.d[i+4])
					x2, y2 := p.d[i+5], p.d[i+6]

					cx, cy, angle1, angle2 := ellipseToCenter(x1, y1, rx, ry, rot, largeArc, sweep, x2, y2)
					pdf.ArcTo(cx, cy, rx, ry, rot, -angle1, -angle2)
				case CloseCmd:
					pdf.ClosePath()
				}
				i += cmdLen(cmd)
			}
			pdf.DrawPath("F")
		} else if l.t == textLayer {
			name, style, size := l.fontFace.Info()
			pdfStyle := ""
			if style&Bold != 0 {
				pdfStyle += "B"
			}
			if style&Italic != 0 {
				pdfStyle += "I"
			}
			pdf.SetFont(name, pdfStyle, size*PtPerMm)
			pdf.Text(l.x, l.y, l.text)
		}
	}
}

func (c *C) WriteImage() *image.RGBA {
	dpm := c.dpi * InchPerMm
	img := image.NewRGBA(image.Rect(0.0, 0.0, int(c.w*dpm), int(c.h*dpm)))
	ras := vector.NewRasterizer(int(c.w*dpm), int(c.h*dpm))

	bg := Rectangle(0.0, 0.0, c.w, c.h)
	c.layers = append([]layer{{pathLayer, 0.0, 0.0, color.White, FontFace{}, bg, ""}}, c.layers...)

	for _, l := range c.layers {
		if l.t == textLayer {
			l.path = l.fontFace.ToPath(l.text)
			l.t = pathLayer
		}

		if l.t == pathLayer {
			p := l.path.Copy().Translate(l.x, l.y)
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
					xpen, ypen := ras.Pen()
					x1, y1 := float64(xpen)/dpm, float64(ypen)/dpm
					rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]*math.Pi/180
					largeArc, sweep := fromArcFlags(p.d[i+4])
					x2, y2 := p.d[i+5], p.d[i+6]

					cx, cy, angle1, angle2 := ellipseToCenter(x1, y1, rx, ry, rot, largeArc, sweep, x2, y2)
					angle1 *= math.Pi / 180
					angle2 *= math.Pi / 180

					// TODO: use dynamic step size
					// from https://github.com/fogleman/gg/blob/master/context.go#L485
					const n = 16
					for i := 0; i < n; i++ {
						p1 := float64(i+0) / n
						p2 := float64(i+1) / n
						a1 := angle1 + (angle2-angle1)*p1
						a2 := angle1 + (angle2-angle1)*p2
						xt0 := cx + rx*math.Cos(a1)
						yt0 := cy + ry*math.Sin(a1)
						xt1 := cx + rx*math.Cos(a1+(a2-a1)/2)
						yt1 := cy + ry*math.Sin(a1+(a2-a1)/2)
						xt2 := cx + rx*math.Cos(a2)
						yt2 := cy + ry*math.Sin(a2)
						ctx := 2*xt1 - xt0/2 - xt2/2
						cty := 2*yt1 - yt0/2 - yt2/2
						ras.QuadTo(float32(ctx*dpm), float32(cty*dpm), float32(xt2*dpm), float32(yt2*dpm))
					}
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
