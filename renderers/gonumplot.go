package renderers

import (
	"image"
	"image/color"
	"math"

	"github.com/Seanld/canvas"
	canvasFont "github.com/tdewolff/font"
	"golang.org/x/image/font"
	gonumFont "gonum.org/v1/plot/font"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// GonumPlot is a github.com/gonum/plot/vg renderer.
type GonumPlot struct {
	ctx   *canvas.Context
	fonts map[string]*canvas.FontFamily
}

// NewGonumPlot returns a new github.com/gonum/plot/vg renderer.
func NewGonumPlot(r canvas.Renderer) draw.Canvas {
	c := &GonumPlot{
		ctx:   canvas.NewContext(r),
		fonts: map[string]*canvas.FontFamily{},
	}
	return draw.New(c)
}

// Size returns the width and height of the canvas.
func (r *GonumPlot) Size() (vg.Length, vg.Length) {
	width, height := r.ctx.Size()
	return vg.Length(width * ptPerMm), vg.Length(height * ptPerMm)
}

// SetLineWidth sets the width of stroked paths.
// If the width is not positive then stroked lines
// are not drawn.
//
// The initial line width is 1 point.
func (r *GonumPlot) SetLineWidth(length vg.Length) {
	r.ctx.SetStrokeWidth(float64(length * mmPerPt))
}

// SetLineDash sets the dash pattern for lines.
// The pattern slice specifies the lengths of
// alternating dashes and gaps, and the offset
// specifies the distance into the dash pattern
// to start the dash.
//
// The initial dash pattern is a solid line.
func (r *GonumPlot) SetLineDash(pattern []vg.Length, offset vg.Length) {
	array := make([]float64, 0, len(pattern))
	for _, dash := range pattern {
		array = append(array, float64(dash*mmPerPt))
	}
	r.ctx.SetDashes(float64(offset*mmPerPt), array...)
}

// SetColor sets the current drawing color.
// Note that fill color and stroke color are
// the same, so if you want different fill
// and stroke colors then you must set a color,
// draw fills, set a new color and then draw lines.
//
// The initial color is black.  If SetColor is
// called with a nil color then black is used.
func (r *GonumPlot) SetColor(col color.Color) {
	if col == nil {
		col = color.Black
	}
	r.ctx.SetFillColor(col)
	r.ctx.SetStrokeColor(col)
}

// Rotate applies a rotation transform to the
// context.  The parameter is specified in
// radians.
func (r *GonumPlot) Rotate(rad float64) {
	r.ctx.Rotate(rad * 180.0 / math.Pi)
}

// Translate applies a translational transform
// to the context.
func (r *GonumPlot) Translate(pt vg.Point) {
	r.ctx.Translate(float64(pt.X*mmPerPt), float64(pt.Y*mmPerPt))
}

// Scale applies a scaling transform to the
// context
func (r *GonumPlot) Scale(x, y float64) {
	r.ctx.Scale(x, y)
}

// Push saves the current line width, the
// current dash pattern, the current
// transforms, and the current color
// onto a stack so that the state can later
// be restored by calling Pop().
func (r *GonumPlot) Push() {
	r.ctx.Push()
}

// Pop restores the context saved by the
// corresponding call to Push().
func (r *GonumPlot) Pop() {
	r.ctx.Pop()
}

func (r *GonumPlot) addPath(path vg.Path) {
	for _, comp := range path {
		switch comp.Type {
		case vg.MoveComp:
			r.ctx.MoveTo(float64(comp.Pos.X*mmPerPt), float64(comp.Pos.Y*mmPerPt))
		case vg.LineComp:
			r.ctx.LineTo(float64(comp.Pos.X*mmPerPt), float64(comp.Pos.Y*mmPerPt))
		case vg.ArcComp:
			r.ctx.Arc(float64(comp.Radius*mmPerPt), float64(comp.Radius*mmPerPt), 0.0, float64(comp.Start)*180.0/math.Pi, float64(comp.Start+comp.Angle)*180.0/math.Pi)
		case vg.CurveComp:
			r.ctx.CubeTo(float64(comp.Control[0].X*mmPerPt), float64(comp.Control[0].Y*mmPerPt), float64(comp.Control[1].X*mmPerPt), float64(comp.Control[1].Y*mmPerPt), float64(comp.Pos.X*mmPerPt), float64(comp.Pos.Y*mmPerPt))
		case vg.CloseComp:
			r.ctx.Close()
		}
	}
}

// Stroke strokes the given path.
func (r *GonumPlot) Stroke(path vg.Path) {
	r.addPath(path)
	r.ctx.Stroke()
}

// Fill fills the given path.
func (r *GonumPlot) Fill(path vg.Path) {
	r.addPath(path)
	r.ctx.Fill()
}

// FillString fills in text at the specified
// location using the given font.
// If the font size is zero, the text is not drawn.
func (r *GonumPlot) FillString(f gonumFont.Face, pt vg.Point, text string) {
	style := canvas.FontRegular
	switch f.Font.Weight {
	case font.WeightThin:
		style = canvas.FontThin
	case font.WeightExtraLight:
		style = canvas.FontExtraLight
	case font.WeightLight:
		style = canvas.FontLight
	case font.WeightMedium:
		style = canvas.FontMedium
	case font.WeightSemiBold:
		style = canvas.FontSemiBold
	case font.WeightBold:
		style = canvas.FontBold
	case font.WeightExtraBold:
		style = canvas.FontExtraBold
	case font.WeightBlack:
		style = canvas.FontBlack
	}
	if f.Font.Style == font.StyleItalic || f.Font.Style == font.StyleOblique {
		style |= canvas.FontItalic
	}

	fontFamily := r.fonts[f.Name()]
	if fontFamily == nil {
		fontFamily = canvas.NewFontFamily(f.Name())
		if err := fontFamily.LoadFont(canvasFont.FromGoSFNT(f.Face), 0, canvas.FontRegular); err != nil {
			panic(err)
		}
		r.fonts[f.Name()] = fontFamily
	}

	face := fontFamily.Face(float64(f.Font.Size), r.ctx.Style.Fill.Color, style, canvas.FontNormal)
	r.ctx.DrawText(float64(pt.X*mmPerPt), float64(pt.Y*mmPerPt), canvas.NewTextLine(face, text, canvas.Left))
}

// DrawImage draws the image, scaled to fit
// the destination rectangle.
func (r *GonumPlot) DrawImage(rect vg.Rectangle, img image.Image) {
	size := img.Bounds().Size()
	if size.Eq(image.Point{}) {
		return
	}

	x, y := float64(rect.Min.X*mmPerPt), float64(rect.Min.Y*mmPerPt)
	w, h := float64(rect.Max.X*mmPerPt)-x, float64(rect.Max.Y*mmPerPt)-y

	coord := r.ctx.CoordView().Dot(canvas.Point{x, y})
	m := r.ctx.View().Translate(coord.X, coord.Y).Scale(w/float64(size.X), h/float64(size.Y))
	r.ctx.RenderImage(img, m)
}
