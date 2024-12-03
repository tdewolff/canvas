package renderers

import (
	"io"
	"math"

	"github.com/golang/freetype/truetype"
	"github.com/Seanld/canvas"
	"github.com/tdewolff/font"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// GoChart is a github.com/wcharczuk/go-chart renderer.
type GoChart struct {
	c            *canvas.Canvas
	ctx          *canvas.Context
	height       float64
	writer       canvas.Writer
	dpi          float64
	font         *canvas.FontFamily
	fontSize     float64
	fontColor    drawing.Color
	textRotation float64

	fonts map[string]*canvas.FontFamily
}

// NewGoChart returns a new github.com/wcharczuk/go-chart renderer.
func NewGoChart(writer canvas.Writer) func(int, int) (chart.Renderer, error) {
	return func(w, h int) (chart.Renderer, error) {
		c := canvas.New(float64(w)*mmPerPx, float64(h)*mmPerPx)
		gochart := &GoChart{
			c:         c,
			ctx:       canvas.NewContext(c),
			height:    float64(h) * mmPerPx,
			writer:    writer,
			dpi:       chart.DefaultDPI,
			fontSize:  12.0, // uses default of github.com/golang/freetype/truetype
			fontColor: drawing.ColorTransparent,
			fonts:     map[string]*canvas.FontFamily{},
		}
		gochart.ctx.SetFillColor(canvas.Transparent)
		gochart.ctx.SetStrokeWidth(chart.DefaultStrokeWidth * mmPerPx)

		f, err := chart.GetDefaultFont()
		if err != nil {
			return nil, err
		}
		gochart.SetFont(f)
		return gochart, nil
	}
}

// ResetStyle resets any style related settings of the renderer.
func (r *GoChart) ResetStyle() {
	r.ctx.ResetStyle()
	r.textRotation = 0.0
}

// GetDPI gets the DPI for the renderer.
func (r *GoChart) GetDPI() float64 {
	return r.dpi
}

// SetDPI sets the DPI for the renderer.
func (r *GoChart) SetDPI(dpi float64) {
	r.dpi = dpi
}

// SetClassName sets the current class name.
func (r *GoChart) SetClassName(name string) {
	// TODO: SetClassName
}

// SetStrokeColor sets the current stroke color.
func (r *GoChart) SetStrokeColor(col drawing.Color) {
	r.ctx.SetStrokeColor(col)
}

// SetFillColor sets the current fill color.
func (r *GoChart) SetFillColor(col drawing.Color) {
	r.ctx.SetFillColor(col)
}

// SetStrokeWidth sets the stroke width.
func (r *GoChart) SetStrokeWidth(width float64) {
	r.ctx.SetStrokeWidth(width * mmPerPx)
}

// SetStrokeDashArray sets the stroke dash array.
func (r *GoChart) SetStrokeDashArray(dashArray []float64) {
	dashArray2 := make([]float64, len(dashArray))
	for i := 0; i < len(dashArray); i++ {
		dashArray2[i] = dashArray[i] * mmPerPx
	}
	r.ctx.SetDashes(0.0, dashArray2...)
}

// MoveTo moves the cursor to a given point.
func (r *GoChart) MoveTo(x, y int) {
	r.ctx.MoveTo(float64(x)*mmPerPx, r.height-float64(y)*mmPerPx)
}

// LineTo both starts a shape and draws a line to a given point from the previous point.
func (r *GoChart) LineTo(x, y int) {
	r.ctx.LineTo(float64(x)*mmPerPx, r.height-float64(y)*mmPerPx)
}

// QuadCurveTo draws a quad curve. cx and cy represent the Bézier control points.
func (r *GoChart) QuadCurveTo(cx, cy, x, y int) {
	r.ctx.QuadTo(float64(cx)*mmPerPx, r.height-float64(cy)*mmPerPx, float64(x)*mmPerPx, r.height-float64(y)*mmPerPx)
}

// ArcTo draws an arc with a given center (cx,cy) a given set of radii (rx,ry), a startAngle and delta (in radians).
func (r *GoChart) ArcTo(cx, cy int, rx, ry, startAngle, delta float64) {
	startAngle = 2.0*math.Pi - startAngle
	delta = -delta

	start := canvas.EllipsePos(rx*mmPerPx, ry*mmPerPx, 0.0, float64(cx)*mmPerPx, r.height-float64(cy)*mmPerPx, startAngle)
	if r.c.Empty() {
		r.ctx.MoveTo(start.X, start.Y)
	} else {
		r.ctx.LineTo(start.X, start.Y)
	}

	startAngle *= 180.0 / math.Pi
	delta *= 180.0 / math.Pi
	r.ctx.Arc(rx*mmPerPx, ry*mmPerPx, 0.0, startAngle, startAngle+delta)
}

// Close finalizes a shape as drawn by LineTo.
func (r *GoChart) Close() {
	r.ctx.Close()
	r.ctx.MoveTo(0.0, 0.0)
}

// Stroke strokes the path.
func (r *GoChart) Stroke() {
	r.ctx.Stroke()
}

// Fill fills the path, but does not stroke.
func (r *GoChart) Fill() {
	r.ctx.Fill()
}

// FillStroke fills and strokes a path.
func (r *GoChart) FillStroke() {
	r.ctx.FillStroke()
}

// Circle draws a circle at the given coords with a given radius.
func (r *GoChart) Circle(radius float64, x, y int) {
	r.ctx.DrawPath(float64(x)*mmPerPx, r.height-float64(y)*mmPerPx, canvas.Circle(radius*mmPerPx))
}

// SetFont sets a font for a text field.
func (r *GoChart) SetFont(f *truetype.Font) {
	if f == nil {
		r.font = nil
		return
	}

	name := f.Name(truetype.NameIDFontFamily)
	r.font = r.fonts[name]
	if r.font == nil {
		r.font = canvas.NewFontFamily(name)
		if err := r.font.LoadFont(font.FromGoFreetype(f), 0, canvas.FontRegular); err != nil {
			panic(err)
		}
		r.fonts[name] = r.font
	}
}

// SetFontColor sets a font's color.
func (r *GoChart) SetFontColor(col drawing.Color) {
	r.fontColor = col
}

// SetFontSize sets the font size for a text field.
func (r *GoChart) SetFontSize(size float64) {
	r.fontSize = size
}

// Text draws a text blob.
func (r *GoChart) Text(body string, x, y int) {
	if r.font == nil {
		return
	}

	face := r.font.Face(r.fontSize*ptPerMm*mmPerPx*r.dpi/72.0, r.fontColor, canvas.FontRegular, canvas.FontNormal)
	r.ctx.Push()
	r.ctx.ComposeView(canvas.Identity.Rotate(-r.textRotation * 180.0 / math.Pi))
	r.ctx.DrawText(float64(x)*mmPerPx, r.height-float64(y)*mmPerPx, canvas.NewTextLine(face, body, canvas.Left))
	r.ctx.Pop()
}

// MeasureText measures text.
func (r *GoChart) MeasureText(body string) chart.Box {
	if r.font == nil {
		return chart.Box{}
	}

	face := r.font.Face(r.fontSize*ptPerMm*r.dpi/72.0, r.fontColor, canvas.FontRegular, canvas.FontNormal)
	width := face.TextWidth(body)
	return chart.Box{Right: int(math.Ceil(width)), Bottom: int(r.fontSize * r.dpi / 72.0)}
}

// SetTextRotation sets a rotation for drawing elements.
func (r *GoChart) SetTextRotation(radian float64) {
	r.textRotation = radian
}

// ClearTextRotation clears rotation.
func (r *GoChart) ClearTextRotation() {
	r.textRotation = 0.0
}

// Save writes the image to the given writer.
func (r *GoChart) Save(w io.Writer) error {
	return r.writer(w, r.c)
}
