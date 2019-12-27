package canvas

import (
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type Output int

const (
	OutputSVG Output = iota
	OutputPDF
	OutputEPS
	OutputPNG
	OutputJPG
	OutputGIF
)

type ChartRenderer struct {
	c            *canvas
	output       Output
	dpi          float64
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
		return &ChartRenderer{
			c:      c,
			output: output,
			dpi:    72.0,
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
	// TODO
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
	r.c.MoveTo(float64(x), r.c.H-float64(y))
}

func (r *ChartRenderer) LineTo(x, y int) {
	r.c.LineTo(float64(x), r.c.H-float64(y))
}

func (r *ChartRenderer) QuadCurveTo(cx, cy, x, y int) {
	r.c.QuadTo(float64(cx), r.c.H-float64(cy), float64(x), r.c.H-float64(y))
}

func (r *ChartRenderer) ArcTo(cx, cy int, rx, ry, startAngle, delta float64) {
	startAngle *= 180.0 / math.Pi
	delta *= 180.0 / math.Pi

	start := ellipsePos(rx, -ry, 0.0, float64(cx), r.c.H-float64(cy), startAngle)
	if r.c.Empty() {
		r.c.MoveTo(start.X, r.c.H-start.Y)
	} else {
		r.c.LineTo(start.X, r.c.H-start.Y)
	}
	r.c.Arc(rx, ry, 0.0, startAngle, startAngle+delta)
}

func (r *ChartRenderer) Close() {
	r.c.ClosePath()
	r.c.MoveTo(0.0, 0.0)
}

func (r *ChartRenderer) Stroke() {
	r.c.Stroke()
}

func (r *ChartRenderer) Fill() {
	r.c.Fill()
}

func (r *ChartRenderer) FillStroke() {
	r.c.FillStroke()
}

func (r *ChartRenderer) Circle(radius float64, x, y int) {
	r.c.DrawPath(float64(x), float64(y), Circle(radius))
}

func (r *ChartRenderer) SetFont(font *truetype.Font) {
	// TODO
}

func (r *ChartRenderer) SetFontColor(col drawing.Color) {
	r.fontColor = col
}

func (r *ChartRenderer) SetFontSize(size float64) {
	r.fontSize = size
}

func (r *ChartRenderer) Text(body string, x, y int) {
	face := r.font.Face(r.fontSize*ptPerMm*r.dpi/72.0, r.fontColor, FontRegular, FontNormal)
	r.c.Push()
	r.c.SetFillColor(r.fontColor)
	r.c.ComposeView(Identity.Rotate(-r.textRotation * 180.0 / math.Pi))
	r.c.DrawText(float64(x), r.c.H-float64(y), NewTextLine(face, body, Left))
	r.c.Pop()
}

func (r *ChartRenderer) MeasureText(body string) chart.Box {
	p, _ := r.font.Face(r.fontSize*ptPerMm*r.dpi/72.0, r.fontColor, FontRegular, FontNormal).ToPath(body)
	bounds := p.Bounds()
	bounds = bounds.Transform(Identity.Rotate(-r.textRotation * 180.0 / math.Pi))
	return chart.Box{Left: int(bounds.X + 0.5), Top: int(bounds.Y + 0.5), Right: int((bounds.W + bounds.X) + 0.5), Bottom: int((bounds.H + bounds.Y) + 0.5)}
}

func (r *ChartRenderer) SetTextRotation(radian float64) {
	r.textRotation = radian
}

func (r *ChartRenderer) ClearTextRotation() {
	r.textRotation = 0.0
}

func (r *ChartRenderer) Save(w io.Writer) error {
	switch r.output {
	case OutputSVG:
		svg := SVG(w, r.c.W, r.c.H)
		r.c.Render(svg)
		return svg.Close()
	case OutputPDF:
		pdf := PDF(w, r.c.W, r.c.H)
		r.c.Render(pdf)
		return pdf.Close()
	case OutputEPS:
		eps := EPS(w, r.c.W, r.c.H)
		r.c.Render(eps)
		return nil
	case OutputPNG:
		img := r.c.WriteImage(r.dpi * inchPerMm)
		if err := png.Encode(w, img); err != nil {
			return err
		}
		return nil
	case OutputJPG:
		img := r.c.WriteImage(r.dpi * inchPerMm)
		if err := jpeg.Encode(w, img, nil); err != nil {
			return err
		}
		return nil
	case OutputGIF:
		img := r.c.WriteImage(r.dpi * inchPerMm)
		if err := gif.Encode(w, img, nil); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unknown output format")
}
