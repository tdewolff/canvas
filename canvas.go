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

func cssColor(c color.Color) []byte {
	r, g, b, a := c.RGBA()
	rgba := [4]byte{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	if rgba[3] == 0xff {
		buf := make([]byte, 7)
		buf[0] = '#'
		hex.Encode(buf[1:], rgba[:3])
		return buf
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
		return buf
	}
}

////////////////////////////////////////////////////////////////

// C is a target independent interface for drawing paths and text.
type C interface {
	Open(float64, float64)
	DPI() float64

	SetColor(color.Color)
	SetFont(FontFace)

	DrawPath(float64, float64, *Path)
	DrawText(float64, float64, string)
}

////////////////////////////////////////////////////////////////

type SVG struct {
	w io.Writer

	color    color.Color
	fontFace FontFace
}

func NewSVG(w io.Writer) *SVG {
	return &SVG{w, color.Black, FontFace{}}
}

func (c *SVG) Open(w, h float64) {
	c.w.Write([]byte("<svg xmlns=\"http://www.w3.org/2000/svg\" version=\"1.1\" shape-rendering=\"geometricPrecision\" width=\""))
	c.writeF(w)
	c.w.Write([]byte("mm\" height=\""))
	c.writeF(h)
	c.w.Write([]byte("mm\" viewBox=\"0 0 "))
	c.writeF(w)
	c.w.Write([]byte(" "))
	c.writeF(h)
	c.w.Write([]byte("\">"))
}

func (c *SVG) EmbedFont(font Font) {
	c.w.Write([]byte("<defs><style>"))
	c.w.Write([]byte("@font-face { font-family: '"))
	c.w.Write([]byte(font.name))
	c.w.Write([]byte("'; src: url('data:"))
	c.w.Write([]byte(font.mimetype))
	c.w.Write([]byte(";base64,"))
	b64 := base64.NewEncoder(base64.StdEncoding, c.w)
	b64.Write(font.raw)
	b64.Close()
	c.w.Write([]byte("'); }\n"))
	c.w.Write([]byte("</style></defs>"))
}

func (c *SVG) DPI() float64 {
	return 72.0
}

func (c *SVG) Close() {
	c.w.Write([]byte("</svg>"))
}

func (c *SVG) SetColor(col color.Color) {
	c.color = col
}

func (c *SVG) SetFont(fontFace FontFace) {
	c.fontFace = fontFace
}

func (c *SVG) writeF(f float64) {
	buf := []byte{}
	c.w.Write(strconv.AppendFloat(buf, f, 'g', 5, 64))
}

func (c *SVG) DrawPath(x, y float64, p *Path) {
	p = p.Copy().Translate(x, y)
	c.w.Write([]byte("<path d=\""))
	c.w.Write([]byte(p.String()))
	if c.color != color.Black {
		c.w.Write([]byte("\" fill=\""))
		c.w.Write(cssColor(c.color))
	}
	c.w.Write([]byte("\"/>"))
}

func (c *SVG) DrawText(x, y float64, s string) {
	// TODO: use tspan for newlines
	name, style, size := c.fontFace.Info()
	c.w.Write([]byte("<text x=\""))
	c.writeF(x)
	c.w.Write([]byte("\" y=\""))
	c.writeF(y)
	c.w.Write([]byte("\" font-family=\""))
	c.w.Write([]byte(name))
	c.w.Write([]byte("\" font-size=\""))
	c.writeF(size)
	if style&Italic != 0 {
		c.w.Write([]byte("\" font-style=\"italic"))
	}
	if style&Bold != 0 {
		c.w.Write([]byte("\" font-weight=\"bold"))
	}
	if c.color != color.Black {
		c.w.Write([]byte("\" fill=\""))
		c.w.Write(cssColor(c.color))
	}
	c.w.Write([]byte("\">"))
	c.w.Write([]byte(s))
	c.w.Write([]byte("</text>"))
}

////////////////////////////////////////////////////////////////

type PDF struct {
	f *gofpdf.Fpdf
}

func NewPDF(f *gofpdf.Fpdf) *PDF {
	return &PDF{f}
}

func (c *PDF) Open(w, h float64) {
	c.f.AddPageFormat("P", gofpdf.SizeType{w, h})
}

func (c *PDF) DPI() float64 {
	return 72.0
}

func (c *PDF) SetColor(col color.Color) {
	r, g, b, a := col.RGBA()
	c.f.SetDrawColor(int(r), int(g), int(b))
	c.f.SetFillColor(int(r), int(g), int(b))
	c.f.SetAlpha(float64(a)/0xffff, "Normal")
}

func (c *PDF) SetFont(fontFace FontFace) {
	name, style, size := fontFace.Info()
	pdfStyle := ""
	if style&Bold != 0 {
		pdfStyle += "B"
	}
	if style&Italic != 0 {
		pdfStyle += "I"
	}
	c.f.SetFont(name, pdfStyle, size*PtPerMm)
}

func (c *PDF) DrawPath(x, y float64, p *Path) {
	p = p.Copy().Translate(x, y)
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			c.f.MoveTo(p.d[i+1], p.d[i+2])
		case LineToCmd:
			c.f.LineTo(p.d[i+1], p.d[i+2])
		case QuadToCmd:
			c.f.CurveTo(p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4])
		case CubeToCmd:
			c.f.CurveBezierCubicTo(p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4], p.d[i+5], p.d[i+6])
		case ArcToCmd:
			x1 := c.f.GetX()
			y1 := c.f.GetY()
			rx := p.d[i+1]
			ry := p.d[i+2]
			rot := p.d[i+3] * math.Pi / 180
			largeArc, sweep := fromArcFlags(p.d[i+4])
			x2 := p.d[i+5]
			y2 := p.d[i+6]

			cx, cy, angle1, angle2 := ellipseToCenter(x1, y1, rx, ry, rot, largeArc, sweep, x2, y2)
			c.f.ArcTo(cx, cy, rx, ry, rot, -angle1, -angle2)
		case CloseCmd:
			c.f.ClosePath()
		}
		i += cmdLen(cmd)
	}
	c.f.DrawPath("F")
}

func (c *PDF) DrawText(x, y float64, s string) {
	c.f.Text(x, y, s)
}

////////////////////////////////////////////////////////////////

type Image struct {
	img *image.RGBA
	r   *vector.Rasterizer
	dpm float64

	color    color.Color
	fontFace FontFace
}

func NewImage(dpi float64) *Image {
	return &Image{nil, nil, dpi * InchPerMm, color.Black, FontFace{}}
}

func (c *Image) Image() *image.RGBA {
	return c.img
}

func (c *Image) Open(w, h float64) {
	c.img = image.NewRGBA(image.Rect(0, 0, int(w*c.dpm), int(h*c.dpm)))
	c.r = vector.NewRasterizer(int(w*c.dpm), int(h*c.dpm))

	p := Rectangle(0, 0, w, h)
	c.SetColor(color.White)
	c.DrawPath(0, 0, p)
	c.SetColor(color.Black)
}

func (c *Image) DPI() float64 {
	return c.dpm * MmPerInch
}

func (c *Image) SetColor(col color.Color) {
	c.color = col
}

func (c *Image) SetFont(fontFace FontFace) {
	c.fontFace = fontFace
}

func (c *Image) DrawPath(x, y float64, p *Path) {
	p = p.Copy().Translate(x, y)
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			c.r.MoveTo(float32(p.d[i+1]*c.dpm), float32(p.d[i+2]*c.dpm))
		case LineToCmd:
			c.r.LineTo(float32(p.d[i+1]*c.dpm), float32(p.d[i+2]*c.dpm))
		case QuadToCmd:
			c.r.QuadTo(float32(p.d[i+1]*c.dpm), float32(p.d[i+2]*c.dpm), float32(p.d[i+3]*c.dpm), float32(p.d[i+4]*c.dpm))
		case CubeToCmd:
			c.r.CubeTo(float32(p.d[i+1]*c.dpm), float32(p.d[i+2]*c.dpm), float32(p.d[i+3]*c.dpm), float32(p.d[i+4]*c.dpm), float32(p.d[i+5]*c.dpm), float32(p.d[i+6]*c.dpm))
		case ArcToCmd:
			xpen, ypen := c.r.Pen()
			x1 := float64(xpen) / c.dpm
			y1 := float64(ypen) / c.dpm
			rx := p.d[i+1]
			ry := p.d[i+2]
			rot := p.d[i+3] * math.Pi / 180
			largeArc, sweep := fromArcFlags(p.d[i+4])
			x2 := p.d[i+5]
			y2 := p.d[i+6]

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
				c.r.QuadTo(float32(ctx*c.dpm), float32(cty*c.dpm), float32(xt2*c.dpm), float32(yt2*c.dpm))
			}
		case CloseCmd:
			c.r.ClosePath()
		}
		i += cmdLen(cmd)
	}
	if len(p.d) > 2 && p.d[len(p.d)-3] != CloseCmd {
		// implicitly close path
		c.r.ClosePath()
	}
	size := c.r.Size()
	c.r.Draw(c.img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(c.color), image.Point{})
	c.r.Reset(size.X, size.Y)
}

func (c *Image) DrawText(x, y float64, s string) {
	c.DrawPath(x, y, c.fontFace.ToPath(s))
}
