package canvas

import (
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type Output int

const (
	SVG Output = iota
	PDF
	PNG
	JPG
	GIF
)

type ChartRenderer struct {
	c            *Canvas
	output       Output
	dpi          float64
	p            *Path
	font         *FontFamily
	fontSize     float64
	fontColor    drawing.Color
	textRotation float64
}

func NewChartRenderer(output Output) func(int, int) (chart.Renderer, error) {
	return func(w, h int) (chart.Renderer, error) {
		font := NewFontFamily("font")
		font.LoadLocalFont("Arimo", FontRegular)

		c := New(float64(w), float64(h))
		c.SetCoordinateSystem(CartesianQuadrant4)
		return &ChartRenderer{
			c:      c,
			output: output,
			dpi:    72.0,
			p:      &Path{},
			font:   font,
		}, nil
	}
}

func (r *ChartRenderer) ResetStyle() {
	r.c.ResetStyle()
	r.textRotation = 0.0
}

func (r *ChartRenderer) GetDPI() float64 {
	return r.dpi
}

func (r *ChartRenderer) SetDPI(dpi float64) {
	r.dpi = dpi
}

func (r *ChartRenderer) SetClassName(name string) {
}

func (r *ChartRenderer) SetStrokeColor(col drawing.Color) {
	r.c.SetStrokeColor(col)
}

func (r *ChartRenderer) SetFillColor(col drawing.Color) {
	r.c.SetFillColor(col)
}

func (r *ChartRenderer) SetStrokeWidth(width float64) {
	r.c.SetStrokeWidth(width)
}

func (r *ChartRenderer) SetStrokeDashArray(dashArray []float64) {
	r.c.SetDashes(0.0, dashArray...)
}

func (r *ChartRenderer) MoveTo(x, y int) {
	r.p.MoveTo(float64(x), float64(y))
}

func (r *ChartRenderer) LineTo(x, y int) {
	r.p.LineTo(float64(x), float64(y))
}

func (r *ChartRenderer) QuadCurveTo(cx, cy, x, y int) {
	r.p.QuadTo(float64(cx), float64(cy), float64(x), float64(y))
}

func (r *ChartRenderer) ArcTo(cx, cy int, rx, ry, startAngle, delta float64) {
	fmt.Println("arc")
}

func (r *ChartRenderer) Close() {
	r.p.Close()
	r.p.MoveTo(0.0, 0.0)
}

func (r *ChartRenderer) Stroke() {
	r.c.PushStyle()
	r.c.SetFillColor(Transparent)
	r.c.DrawPath(0.0, 0.0, r.p)
	r.c.PopStyle()
	r.p = &Path{}
}

func (r *ChartRenderer) Fill() {
	r.c.PushStyle()
	r.c.SetStrokeColor(Transparent)
	r.c.DrawPath(0.0, 0.0, r.p)
	r.c.PopStyle()
	r.p = &Path{}
}

func (r *ChartRenderer) FillStroke() {
	r.c.DrawPath(0.0, 0.0, r.p)
	r.p = &Path{}
}

func (r *ChartRenderer) Circle(radius float64, x, y int) {
	r.c.DrawPath(float64(x), float64(y), Circle(radius))
}

func (r *ChartRenderer) SetFont(font *truetype.Font) {
	if font != nil {
		//fmt.Println(font.Name(1))
	}
}

func (r *ChartRenderer) SetFontColor(col drawing.Color) {
	r.fontColor = col
}

func (r *ChartRenderer) SetFontSize(size float64) {
	r.fontSize = size
}

func (r *ChartRenderer) Text(body string, x, y int) {
	face := r.font.Face(r.fontSize*ptPerMm*r.dpi/72.0, r.fontColor, FontRegular, FontNormal)
	r.c.PushStyle()
	r.c.SetFillColor(r.fontColor)
	r.c.DrawText(float64(x), float64(y), NewTextLine(face, body, Left))
	r.c.PopStyle()
}

func (r *ChartRenderer) MeasureText(body string) chart.Box {
	p, _ := r.font.Face(r.fontSize*ptPerMm*r.dpi/72.0, r.fontColor, FontRegular, FontNormal).ToPath(body)
	bounds := p.Bounds()
	return chart.Box{Left: int(bounds.X + 0.5), Top: int(bounds.Y + 0.5), Right: int((bounds.W + bounds.X) + 0.5), Bottom: int((bounds.H + bounds.Y) + 0.5)}
}

func (r *ChartRenderer) SetTextRotation(radian float64) {
	r.textRotation = radian
}

func (r *ChartRenderer) ClearTextRotation() {
	r.textRotation = 0.0
}

func (r *ChartRenderer) Save(w io.Writer) error {
	fmt.Println("save")
	switch r.output {
	case SVG:
		return r.c.WriteSVG(w)
	case PDF:
		return r.c.WritePDF(w)
	case PNG:
		img := r.c.WriteImage(r.dpi * inchPerMm)
		return png.Encode(w, img)
	case JPG:
		img := r.c.WriteImage(r.dpi * inchPerMm)
		return jpeg.Encode(w, img, &jpeg.Options{})
	case GIF:
		img := r.c.WriteImage(r.dpi * inchPerMm)
		return gif.Encode(w, img, &gif.Options{})
	}
	return fmt.Errorf("unknown output format")
}
