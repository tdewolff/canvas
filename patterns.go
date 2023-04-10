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

// Gradient is a gradient pattern for filling.
type Gradient interface {
	SetColorSpace(ColorSpace) Gradient
	At(float64, float64) color.RGBA
}

// Stop is a color and offset for gradient patterns.
type Stop struct {
	Offset float64
	Color  color.RGBA
}

// Stops are the colors and offsets for gradient patterns, sorted by offset.
type Stops []Stop

// Add adds a new color stop to a gradient.
func (stops *Stops) Add(t float64, color color.RGBA) {
	stop := Stop{math.Min(math.Max(t, 0.0), 1.0), color}
	// insert or replace stop and keep sort order
	for i := range *stops {
		if Equal((*stops)[i].Offset, stop.Offset) {
			(*stops)[i] = stop
			return
		} else if stop.Offset < (*stops)[i].Offset {
			*stops = append((*stops)[:i], append(Stops{stop}, (*stops)[i:]...)...)
			return
		}
	}
	*stops = append(*stops, stop)
}

// At returns the color at position t ∈ [0,1].
func (stops Stops) At(t float64) color.RGBA {
	if len(stops) == 0 {
		return Transparent
	} else if t <= 0.0 || len(stops) == 1 {
		return stops[0].Color
	} else if 1.0 <= t {
		return stops[len(stops)-1].Color
	}
	for i, stop := range stops[1:] {
		if t < stop.Offset {
			t = (t - stops[i].Offset) / (stop.Offset - stops[i].Offset)
			return colorLerp(stops[i].Color, stop.Color, t)
		}
	}
	return stops[len(stops)-1].Color
}

func colorLerp(c0, c1 color.RGBA, t float64) color.RGBA {
	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()
	return color.RGBA{
		lerp(r0, r1, t),
		lerp(g0, g1, t),
		lerp(b0, b1, t),
		lerp(a0, a1, t),
	}
}

func lerp(a, b uint32, t float64) uint8 {
	return uint8(uint32((1.0-t)*float64(a)+t*float64(b)) >> 8)
}

// LinearGradient is a linear gradient pattern between the given start and end points. The color at offset 0 corresponds to the start position, and offset 1 to the end position. Start and end points are in the canvas's coordinate system.
type LinearGradient struct {
	Start, End Point
	Stops

	d  Point
	d2 float64
}

// NewLinearGradient returns a new linear gradient pattern.
func NewLinearGradient(start, end Point) *LinearGradient {
	d := end.Sub(start)
	return &LinearGradient{
		Start: start,
		End:   end,

		d:  d,
		d2: d.Dot(d),
	}
}

// SetColorSpace returns the linear gradient with the given color space. Automatically called by the rasterizer.
func (g *LinearGradient) SetColorSpace(colorSpace ColorSpace) Gradient {
	if _, ok := colorSpace.(LinearColorSpace); ok {
		return g
	}
	gradient := &(*g)
	for i := range gradient.Stops {
		gradient.Stops[i].Color = colorSpace.ToLinear(gradient.Stops[i].Color)
	}
	return gradient
}

// At returns the color at position (x,y).
func (g *LinearGradient) At(x, y float64) color.RGBA {
	if len(g.Stops) == 0 {
		return Transparent
	}

	p := Point{x, y}.Sub(g.Start)
	if Equal(g.d.Y, 0.0) && !Equal(g.d.X, 0.0) {
		return g.Stops.At(p.X / g.d.X) // horizontal
	} else if !Equal(g.d.Y, 0.0) && Equal(g.d.X, 0.0) {
		return g.Stops.At(p.Y / g.d.Y) // vertical
	}
	t := p.Dot(g.d) / g.d2
	return g.Stops.At(t)
}

// RadialGradient is a radial gradient pattern between two circles defined by their center points and radii. Color stop at offset 0 corresponds to the first circle and offset 1 to the second circle.
type RadialGradient struct {
	C0, C1 Point
	R0, R1 float64
	Stops

	cd    Point
	dr, a float64
}

// NewRadialGradient returns a new radial gradient pattern.
func NewRadialGradient(c0 Point, r0 float64, c1 Point, r1 float64) *RadialGradient {
	cd := c1.Sub(c0)
	dr := r1 - r0
	return &RadialGradient{
		C0: c0,
		R0: r0,
		C1: c1,
		R1: r1,

		cd: cd,
		dr: dr,
		a:  cd.Dot(cd) - dr*dr,
	}
}

// SetColorSpace returns the linear gradient with the given color space. Automatically called by the rasterizer.
func (g *RadialGradient) SetColorSpace(colorSpace ColorSpace) Gradient {
	if _, ok := colorSpace.(LinearColorSpace); ok {
		return g
	}
	gradient := *g
	for i := range gradient.Stops {
		gradient.Stops[i].Color = colorSpace.ToLinear(gradient.Stops[i].Color)
	}
	return &gradient
}

// At returns the color at position (x,y).
func (g *RadialGradient) At(x, y float64) color.RGBA {
	if len(g.Stops) == 0 {
		return Transparent
	}

	// see reference implementation of pixman-radial-gradient
	// https://github.com/servo/pixman/blob/master/pixman/pixman-radial-gradient.c#L161
	pd := Point{x, y}.Sub(g.C0)
	b := pd.Dot(g.cd) + g.R0*g.dr
	c := pd.Dot(pd) - g.R0*g.R0
	t0, t1 := solveQuadraticFormula(g.a, -2.0*b, c)
	if !math.IsNaN(t1) {
		return g.Stops.At(t1)
	} else if !math.IsNaN(t0) {
		return g.Stops.At(t0)
	}
	return Transparent
}

// ImagePattern is an image tiling pattern of an image drawn from an origin with a certain resolution. Higher resolution will give smaller tilings.
//type ImagePattern struct {
//	img    *image.RGBA
//	res    Resolution
//	origin Point
//}
//
//// NewImagePattern returns a new image pattern.
//func NewImagePattern(iimg image.Image, res Resolution, origin Point) *ImagePattern {
//	img, ok := iimg.(*image.RGBA)
//	if !ok {
//		bounds := iimg.Bounds()
//		img = image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
//		draw.Draw(img, img.Bounds(), iimg, bounds.Min, draw.Src)
//	}
//	return &ImagePattern{
//		img:    img,
//		res:    res,
//		origin: origin,
//	}
//}
//
//// SetColorSpace returns the linear gradient with the given color space. Automatically called by the rasterizer.
//func (p *ImagePattern) SetColorSpace(colorSpace ColorSpace) Pattern {
//	if _, ok := colorSpace.(LinearColorSpace); ok {
//		return p
//	}
//	// TODO: optimize
//	pattern := *p
//	bounds := p.img.Bounds()
//	pattern.img = image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
//	draw.Draw(pattern.img, pattern.img.Bounds(), p.img, bounds.Min, draw.Src)
//	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
//		for x := bounds.Min.X; x < bounds.Max.X; x++ {
//			col := pattern.img.RGBAAt(x, y)
//			col = colorSpace.ToLinear(col)
//			pattern.img.SetRGBA(x, y, col)
//		}
//	}
//	return &pattern
//}
//
//// At returns the color at position (x,y).
//func (p *ImagePattern) At(x, y float64) color.RGBA {
//	x = (x - p.origin.X) * p.res.DPMM()
//	y = (y - p.origin.Y) * p.res.DPMM()
//
//	var s [4]uint8
//	ix0, iy0 := int(x), int(y)
//	fx, fy := x-float64(ix0), y-float64(iy0)
//	ix0 = ix0 % p.img.Bounds().Dx()
//	iy0 = iy0 % p.img.Bounds().Dy()
//	ix1 := (ix0 + 1) % p.img.Bounds().Dx()
//	iy1 := (iy0 + 1) % p.img.Bounds().Dy()
//	d00 := p.img.PixOffset(ix0, iy0)
//	d10 := p.img.PixOffset(ix1, iy0)
//	d01 := p.img.PixOffset(ix0, iy1)
//	d11 := p.img.PixOffset(ix1, iy1)
//	for i := 0; i < 4; i++ {
//		s[i] = uint8((1.0-fy)*((1.0-fx)*float64(p.img.Pix[d00+i])+fx*float64(p.img.Pix[d10+i])) + fy*((1.0-fx)*float64(p.img.Pix[d01+i])+fx*float64(p.img.Pix[d11+i])) + 0.5)
//	}
//	return color.RGBA{s[0], s[1], s[2], s[3]}
//}
