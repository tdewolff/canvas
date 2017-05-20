package canvas

import (
	"bytes"
	"encoding/hex"
	"image"
	"image/color"
	"io"
	"math"
	"strconv"

	"golang.org/x/image/font"
	"golang.org/x/image/vector"

	"github.com/jung-kurt/gofpdf"
)

var Epsilon float64 = 1e-10

func cssColor(c color.Color) []byte {
	const m = 1<<16 - 1
	r, g, b, a := c.RGBA()
	rgba := [4]byte{uint8(r / m >> 8), uint8(g / m >> 8), uint8(b / m >> 8), uint8(a / m >> 8)}
	if a == 0xffff {
		buf := make([]byte, 7)
		buf[0] = '#'
		hex.Encode(buf[1:], rgba[:3])
		return buf
	} else {
		buf := make([]byte, 0, 24)
		buf = append(buf, []byte("rgba(")...)
		buf = strconv.AppendInt(buf, int64(r), 10)
		buf = append(buf, ',')
		buf = strconv.AppendInt(buf, int64(g), 10)
		buf = append(buf, ',')
		buf = strconv.AppendInt(buf, int64(b), 10)
		buf = append(buf, ',')
		buf = strconv.AppendFloat(buf, float64(a)/0xffff, 'g', 4, 64)
		buf = append(buf, ')')
		return buf
	}
}

////////////////////////////////////////////////////////////////

type C interface {
	Open(float64, float64)
	Close()

	PushLayer(string)
	PopLayer()

	SetColor(color.Color)
	SetFont(string, float64)

	LineHeight() float64
	TextWidth(string) float64

	DrawPath(*Path)
	DrawRect(float64, float64, float64, float64)
	DrawEllipse(float64, float64, float64, float64)
	DrawText(float64, float64, string)
}

////////////////////////////////////////////////////////////////

type SVG struct {
	Fonts

	w io.Writer
	b bytes.Buffer

	prec int
	buf  []byte

	color color.Color

	webFonts []string
}

func NewSVG(w io.Writer, prec int) *SVG {
	return &SVG{*NewFonts(), w, bytes.Buffer{}, prec, make([]byte, 0, 8), color.Black, []string{}}
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
	if len(c.webFonts) > 0 {
		c.w.Write([]byte("<defs><style>"))
		for _, url := range c.webFonts {
			c.w.Write([]byte("@import url('"))
			c.w.Write([]byte(url))
			c.w.Write([]byte("');"))
		}
		c.w.Write([]byte("</style></defs>"))
	}
}

func (c *SVG) Close() {
	c.w.Write([]byte("</svg>"))
}

func (c *SVG) AddWebFont(url string) {
	c.webFonts = append(c.webFonts, url)
}

func (c *SVG) PushLayer(name string) {
	c.w.Write([]byte("<g class=\"" + name + "\">"))
}

func (c *SVG) PopLayer() {
	c.w.Write([]byte("</g>"))
}

func (c *SVG) SetColor(color color.Color) {
	c.color = color
}

func (c *SVG) writeF(f float64) {
	c.w.Write(strconv.AppendFloat(c.buf[:0], f, 'g', c.prec, 64))
}

func (c *SVG) DrawPath(p *Path) {
	c.w.Write([]byte("<path d=\""))
	i := 0
	x0, y0 := 0.0, 0.0
	x, y := 0.0, 0.0
	for _, cmd := range p.cmds {
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+0], p.d[i+1]
			c.w.Write([]byte("M"))
			c.writeF(p.d[i+0])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+1])
			x0, y0 = p.d[i+0], p.d[i+1]
			i += 2
		case LineToCmd:
			x, y = p.d[i+0], p.d[i+1]
			if math.Abs(y) < Epsilon {
				c.w.Write([]byte("H"))
				c.writeF(x)
			} else if math.Abs(x) < Epsilon {
				c.w.Write([]byte("V"))
				c.writeF(y)
			} else {
				c.w.Write([]byte("L"))
				c.writeF(x)
				c.w.Write([]byte(" "))
				c.writeF(y)
			}
			i += 2
		case QuadToCmd:
			x, y = p.d[i+1], p.d[i+2]
			c.w.Write([]byte("Q"))
			c.writeF(p.d[i+0])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+1])
			c.w.Write([]byte(" "))
			c.writeF(x)
			c.w.Write([]byte(" "))
			c.writeF(y)
			i += 4
		case CubeToCmd:
			x, y = p.d[i+4], p.d[i+5]
			c.w.Write([]byte("C"))
			c.writeF(p.d[i+0])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+1])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+2])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+3])
			c.w.Write([]byte(" "))
			c.writeF(x)
			c.w.Write([]byte(" "))
			c.writeF(y)
			i += 6
		case ArcToCmd:
			x, y = p.d[i+5], p.d[i+6]
			c.w.Write([]byte("A"))
			c.writeF(p.d[i+0])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+1])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+2])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+3])
			c.w.Write([]byte(" "))
			c.writeF(p.d[i+4])
			c.w.Write([]byte(" "))
			c.writeF(x)
			c.w.Write([]byte(" "))
			c.writeF(y)
			i += 7
		case CloseCmd:
			c.w.Write([]byte("Z"))
			x, y = x0, y0
		}
	}
	if c.color != color.Black {
		c.w.Write([]byte("\" fill=\""))
		c.w.Write(cssColor(c.color))
	}
	c.w.Write([]byte("\"/>"))
}

func (c *SVG) DrawRect(x, y, w, h float64) {
	p := &Path{}
	p.Rect(x, y, w, h)
	c.DrawPath(p)
}

func (c *SVG) DrawEllipse(x, y, rx, ry float64) {
	p := &Path{}
	p.Ellipse(x, y, rx, ry)
	c.DrawPath(p)
}

func (c *SVG) DrawText(x, y float64, s string) {
	c.w.Write([]byte("<text x=\""))
	c.writeF(x)
	c.w.Write([]byte("\" y=\""))
	c.writeF(y)
	c.w.Write([]byte("\" font-family=\""))
	c.w.Write([]byte(c.font))
	c.w.Write([]byte("\" font-size=\""))
	c.writeF(c.fontsize * 0.352778)
	if c.fontstyle&Italic != 0 {
		c.w.Write([]byte("\" font-style=\"italic"))
	}
	if c.fontstyle&Bold != 0 {
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

func (c *PDF) Close() {
}

func (c *PDF) PushLayer(name string) {
	id := c.f.AddLayer(name, true)
	c.f.BeginLayer(id)
}

func (c *PDF) PopLayer() {
	c.f.EndLayer()
}

func (c *PDF) SetColor(color color.Color) {
	r, g, b, a := color.RGBA()
	c.f.SetDrawColor(int(r), int(g), int(b))
	c.f.SetFillColor(int(r), int(g), int(b))
	c.f.SetAlpha(float64(a)/0xffff, "Normal")
}

func (c *PDF) SetFont(name string, size float64) {
	c.f.SetFont(name, "", size)
}

func (c *PDF) LineHeight() float64 {
	pt, _ := c.f.GetFontSize()
	return pt * 0.352778
}

func (c *PDF) TextWidth(s string) float64 {
	return c.f.GetStringWidth(s)
}

func (c *PDF) DrawPath(p *Path) {
	i := 0
	for _, cmd := range p.cmds {
		switch cmd {
		case MoveToCmd:
			c.f.MoveTo(p.d[i+0], p.d[i+1])
			i += 2
		case LineToCmd:
			c.f.LineTo(p.d[i+0], p.d[i+1])
			i += 2
		case QuadToCmd:
			c.f.CurveTo(p.d[i+0], p.d[i+1], p.d[i+2], p.d[i+3])
			i += 4
		case CubeToCmd:
			c.f.CurveBezierCubicTo(p.d[i+0], p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4], p.d[i+5])
			i += 6
		case ArcToCmd:
			x1 := c.f.GetX()
			y1 := c.f.GetY()
			rx := p.d[i+0]
			ry := p.d[i+1]
			rot := p.d[i+2] * math.Pi / 180
			large := p.d[i+3] == 1.0
			sweep := p.d[i+4] == 1.0
			x2 := p.d[i+5]
			y2 := p.d[i+6]

			cx, cy, angle1, angle2 := arcToCenter(x1, y1, rx, ry, rot, large, sweep, x2, y2)
			c.f.ArcTo(cx, cy, rx, ry, rot, -angle1, -angle2)
			i += 7
		case CloseCmd:
			c.f.ClosePath()
		}
	}
	c.f.DrawPath("F")
}

func (c *PDF) DrawRect(x, y, w, h float64) {
	p := &Path{}
	p.Rect(x, y, w, h)
	c.DrawPath(p)
}

func (c *PDF) DrawEllipse(x, y, rx, ry float64) {
	p := &Path{}
	p.Ellipse(x, y, rx, ry)
	c.DrawPath(p)
}

func (c *PDF) DrawText(x, y float64, s string) {
	c.f.Text(x, y, s)
}

////////////////////////////////////////////////////////////////

type Image struct {
	Fonts

	img *image.RGBA
	r   *vector.Rasterizer

	dpm      float64
	color    color.Color
	fontface font.Face
}

func NewImage(dpi float64) *Image {
	return &Image{*NewFonts(), nil, nil, dpi / 25.4, color.Black, nil}
}

func (c *Image) Image() *image.RGBA {
	return c.img
}

func (c *Image) Open(w, h float64) {
	c.img = image.NewRGBA(image.Rect(0, 0, int(w*c.dpm), int(h*c.dpm)))
	c.r = vector.NewRasterizer(int(w*c.dpm), int(h*c.dpm))

	c.SetColor(color.White)
	c.DrawRect(0, 0, w, h)
	c.SetColor(color.Black)
}

func (c *Image) Close() {
}

func (c *Image) PushLayer(name string) {
}

func (c *Image) PopLayer() {
}

func (c *Image) SetColor(color color.Color) {
	c.color = color
}

func (c *Image) DrawPath(p *Path) {
	i := 0
	for _, cmd := range p.cmds {
		switch cmd {
		case MoveToCmd:
			c.r.MoveTo(toF32Vec(p.d[i+0]*c.dpm, p.d[i+1]*c.dpm))
			i += 2
		case LineToCmd:
			c.r.LineTo(toF32Vec(p.d[i+0]*c.dpm, p.d[i+1]*c.dpm))
			i += 2
		case QuadToCmd:
			c.r.QuadTo(toF32Vec(p.d[i+0]*c.dpm, p.d[i+1]*c.dpm), toF32Vec(p.d[i+2]*c.dpm, p.d[i+3]*c.dpm))
			i += 4
		case CubeToCmd:
			c.r.CubeTo(toF32Vec(p.d[i+0]*c.dpm, p.d[i+1]*c.dpm), toF32Vec(p.d[i+2]*c.dpm, p.d[i+3]*c.dpm), toF32Vec(p.d[i+4]*c.dpm, p.d[i+5]*c.dpm))
			i += 6
		case ArcToCmd:
			x1 := float64(c.r.Pen()[0]) / c.dpm
			y1 := float64(c.r.Pen()[1]) / c.dpm
			rx := p.d[i+0]
			ry := p.d[i+1]
			rot := p.d[i+2] * math.Pi / 180
			large := p.d[i+3] == 1.0
			sweep := p.d[i+4] == 1.0
			x2 := p.d[i+5]
			y2 := p.d[i+6]

			cx, cy, angle1, angle2 := arcToCenter(x1, y1, rx, ry, rot, large, sweep, x2, y2)
			angle1 *= math.Pi / 180
			angle2 *= math.Pi / 180

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
				c.r.QuadTo(toF32Vec(ctx*c.dpm, cty*c.dpm), toF32Vec(xt2*c.dpm, yt2*c.dpm))
			}
			i += 7
		case CloseCmd:
			c.r.ClosePath()
		}
	}
	size := c.r.Size()
	c.r.Draw(c.img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(c.color), image.Point{})
	c.r.Reset(size.X, size.Y)
}

func (c *Image) DrawRect(x, y, w, h float64) {
	p := &Path{}
	p.Rect(x, y, w, h)
	c.DrawPath(p)
}

func (c *Image) DrawEllipse(x, y, rx, ry float64) {
	p := &Path{}
	p.Ellipse(x, y, rx, ry)
	c.DrawPath(p)
}

func (c *Image) DrawText(x, y float64, s string) {
	d := font.Drawer{c.img, image.NewUniform(c.color), c.Fonts.fontface, toP26_6(x*c.dpm, y*c.dpm)}
	d.DrawString(s)
}
