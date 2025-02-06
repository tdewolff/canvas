package canvas

import (
	"image/color"
	"math"
)

type Pattern interface {
	SetView(Matrix) Pattern
	SetColorSpace(ColorSpace) Pattern
	RenderTo(Renderer, *Path)
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
	Fill      Paint
	Thickness float64
	cell      Matrix
	hatch     Hatcher
}

// Hatcher is a hatch pattern along the cell's axes. The rectangle (x0,y0)-(x1,y1) is expressed in the unit cell's coordinate system, and the returned path should be transformed by the cell to obtain the final hatch pattern.
type Hatcher func(float64, float64, float64, float64) *Path

// NewHatchPattern returns a new hatch pattern.
func NewHatchPattern(ifill interface{}, thickness float64, cell Matrix, hatch Hatcher) *HatchPattern {
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
		Fill:      fill,
		Thickness: thickness,
		cell:      cell,
		hatch:     hatch,
	}
}

// SetView sets the view. Automatically called by Canvas for coordinate system transformations.
func (p *HatchPattern) SetView(view Matrix) Pattern {
	return p
}

// SetColorSpace sets the color space. Automatically called by the rasterizer.
func (p *HatchPattern) SetColorSpace(colorSpace ColorSpace) Pattern {
	if _, ok := colorSpace.(LinearColorSpace); ok {
		return p
	}

	if p.Fill.IsGradient() {
		p.Fill.Gradient.SetColorSpace(colorSpace)
	} else if p.Fill.IsColor() {
		p.Fill.Color = colorSpace.ToLinear(p.Fill.Color)
	}
	return p
}

// Tile tiles the hatch pattern within the clipping path.
func (p *HatchPattern) Tile(clip *Path) *Path {
	dst := clip.FastBounds()

	// find extremes along cell axes
	invCell := p.cell.Inv()
	points := []Point{
		invCell.Dot(Point{dst.X0 - p.Thickness, dst.Y0 - p.Thickness}),
		invCell.Dot(Point{dst.X1 + p.Thickness, dst.Y0 - p.Thickness}),
		invCell.Dot(Point{dst.X1 + p.Thickness, dst.Y1 + p.Thickness}),
		invCell.Dot(Point{dst.X0 - p.Thickness, dst.Y1 + p.Thickness}),
	}
	x0, x1 := points[0].X, points[0].X
	y0, y1 := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		x0 = math.Min(x0, point.X)
		x1 = math.Max(x1, point.X)
		y0 = math.Min(y0, point.Y)
		y1 = math.Max(y1, point.Y)
	}

	hatch := p.hatch(x0, y0, x1, y1)
	hatch = hatch.Transform(p.cell)
	hatch = hatch.And(clip)
	if p.Thickness != 0.0 {
		hatch = hatch.Stroke(p.Thickness, ButtCap, MiterJoin, 0.01)
	}
	return hatch
}

// RenderTo tiles the hatch pattern to the clipping path and renders it to the renderer.
func (p *HatchPattern) RenderTo(r Renderer, clip *Path) {
	hatch := p.Tile(clip)
	r.RenderPath(hatch, Style{Fill: p.Fill}, Identity)
}

// NewLineHatch returns a new line hatch pattern with lines at an angle with a spacing of distance. Thickness is the stroke thickness applied to the shape; stroking is ignored with thickness is zero.
func NewLineHatch(ifill interface{}, angle, distance, thickness float64) *HatchPattern {
	cell := Identity.Rotate(angle).Scale(distance, distance)
	return NewHatchPattern(ifill, thickness, cell, func(x0, y0, x1, y1 float64) *Path {
		p := &Path{}
		for y := math.Floor(y0); y <= y1; y += 1.0 {
			p.MoveTo(x0, y)
			p.LineTo(x1, y)
		}
		return p
	})
}

// NewCrossHatch returns a new cross hatch pattern of two regular line hatches at different angles and with different distance intervals. Thickness is the stroke thickness applied to the shape; stroking is ignored with thickness is zero.
func NewCrossHatch(ifill interface{}, angle0, angle1, distance0, distance1, thickness float64) *HatchPattern {
	cell := PrimitiveCell(
		Point{distance0, 0.0}.Rot(angle0*math.Pi/180.0, Origin),
		Point{distance1, 0.0}.Rot(angle1*math.Pi/180.0, Origin),
	)
	return NewHatchPattern(ifill, thickness, cell, func(x0, y0, x1, y1 float64) *Path {
		p := &Path{}
		for y := math.Floor(y0); y <= y1; y += 1.0 {
			p.MoveTo(x0, y)
			p.LineTo(x1, y)
		}
		for x := math.Floor(x0); x <= x1; x += 1.0 {
			p.MoveTo(x, y0)
			p.LineTo(x, y1)
		}
		return p
	})
}

// NewShapeHatch returns a new shape hatch that repeats the given shape over a rhombus primitive cell with sides of length distance. Thickness is the stroke thickness applied to the shape; stroking is ignored with thickness is zero.
func NewShapeHatch(ifill interface{}, shape *Path, distance, thickness float64) *HatchPattern {
	d := distance * math.Sin(60.0*math.Pi/180.0)
	cell := SquareCell(1.0)
	return NewHatchPattern(ifill, thickness, cell, func(x0, y0, x1, y1 float64) *Path {
		p := &Path{}
		for y := math.Floor(y0/distance) * distance; y <= y1; y += 2.0 * d {
			for x := math.Floor(x0/distance) * distance; x <= x1; x += distance {
				p = p.Append(shape.Copy().Translate(x, y))
			}
			for x := (math.Floor(x0/distance) + 0.5) * distance; x <= x1; x += distance {
				p = p.Append(shape.Copy().Translate(x, y+d))
			}
		}
		return p
	})
}
