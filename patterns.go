package canvas

import (
	"image/color"
	"math"
)

type Pattern interface {
	ClipTo(Renderer, *Path)
}

//type CanvasPattern struct {
//	c    *Canvas
//	cell Matrix
//}
//
//func NewPattern(c *Canvas, cell Matrix) *CanvasPattern {
//	return &CanvasPattern{
//		c:    c,
//		cell: cell,
//	}
//}
//
//func (p *CanvasPattern) ClipTo(r Renderer, clip *Path) {
//	//fmt.Println("src", p.c.Size())
//	//fmt.Println("dst", r.Size())
//	//fmt.Println("matrix", p.m)
//	// TODO: tile
//	p.c.RenderViewTo(r, p.cell)
//}

//type ImagePattern struct {
//	img  *image.RGBA
//	cell Matrix
//}
//
//func NewImagePattern() *ImagePattern {
//	return &ImagePattern{}
//}
//
//func (p *ImagePattern) ClipTo(r Renderer, clip *Path) {
//}

// Hatch pattern is a filling hatch pattern.
type HatchPattern struct {
	Fill Paint
	Hatcher
}

// Hatcher is a hatch pattern along the cell's axes.
type Hatcher interface {
	Cell() Matrix
	Hatch(float64, float64, float64, float64) *Path
}

// NewHatchPattern returns a new hatch pattern.
func NewHatchPattern(ifill interface{}, hatcher Hatcher) *HatchPattern {
	var fill Paint
	if paint, ok := ifill.(Paint); ok {
		fill = paint
	} else if pattern, ok := ifill.(Pattern); ok {
		fill = Paint{Pattern: pattern}
	} else if gradient, ok := ifill.(Gradient); ok {
		fill = Paint{Gradient: gradient}
	} else if col, ok := ifill.(color.Color); ok {
		fill = Paint{Color: rgbaColor(col)}
	}
	if fill.IsPattern() {
		panic("hatch paint cannot be pattern")
	}
	return &HatchPattern{
		Fill:    fill,
		Hatcher: hatcher,
	}
}

// Tile tiles the hatch pattern within the clipping path.
func (p *HatchPattern) Tile(clip *Path) *Path {
	cell := p.Cell()
	dst := clip.FastBounds()

	// find extremes along cell axes
	invCell := cell.Inv()
	points := []Point{
		invCell.Dot(Point{dst.X, dst.Y}),
		invCell.Dot(Point{dst.X + dst.W, dst.Y}),
		invCell.Dot(Point{dst.X + dst.W, dst.Y + dst.H}),
		invCell.Dot(Point{dst.X, dst.Y + dst.H}),
	}
	x0, x1 := points[0].X, points[0].X
	y0, y1 := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		x0 = math.Min(x0, point.X)
		x1 = math.Max(x1, point.X)
		y0 = math.Min(y0, point.Y)
		y1 = math.Max(y1, point.Y)
	}

	hatch := p.Hatch(x0, y0, x1, y1)
	hatch = hatch.Transform(cell)
	return hatch.And(clip)
}

// ClipTo tiles the hatch pattern to the clipping path and renders it to the renderer.
func (p *HatchPattern) ClipTo(r Renderer, clip *Path) {
	hatch := p.Tile(clip)
	r.RenderPath(hatch, Style{Fill: p.Fill}, Identity)
}

// LineHatch is a hatch pattern of lines at an angle, spacing distance, and thickness.
type LineHatch struct {
	Angle     float64
	Distance  float64
	Thickness float64
}

// NewLineHatch returns a new line hatch pattern.
func NewLineHatch(ifill interface{}, angle, distance, thickness float64) *HatchPattern {
	return NewHatchPattern(ifill, &LineHatch{
		Angle:     angle,
		Distance:  distance,
		Thickness: thickness,
	})
}

// Cell is the primitive cell along which the pattern repeats.
func (h *LineHatch) Cell() Matrix {
	return Identity.Rotate(h.Angle).Scale(h.Distance, h.Distance)
}

// Hatch tiles the hatch pattern in unit cells. That is, the rectangle (x0,y0)-(x1,y1) is expressed in the unit cell's coordinate system, and the returned path should be transformed by the cell.
func (h *LineHatch) Hatch(x0, y0, x1, y1 float64) *Path {
	y0 -= h.Thickness / 2.0
	y1 += h.Thickness / 2.0

	p := &Path{}
	for y := math.Floor(y0); y < y1; y += 1.0 {
		p.MoveTo(x0, y)
		p.LineTo(x1, y)
	}
	return p.Stroke(h.Thickness, ButtCap, MiterJoin, 0.01)
}
