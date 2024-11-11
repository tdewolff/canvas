package canvas

import (
	"math"

	"github.com/ByteArena/poly2tri-go"
)

// PrimitiveCell is a (primitive) cell used for tiling.
func PrimitiveCell(a, b Point) Matrix {
	A := a.Length()
	B := a.PerpDot(b) / A
	s := a.Dot(b) / A / B
	return Identity.Rotate(a.Angle()*180.0/math.Pi).Shear(s, 0.0).Scale(A, B)
}

// SquareCell is a square cell with sides of length a used for tiling.
func SquareCell(a float64) Matrix {
	return Identity.Scale(a, a)
}

// RectangleCell is a rectangular cell with width a and height b used for tiling.
func RectangleCell(a, b float64) Matrix {
	return Identity.Scale(a, b)
}

// RhombusCell is a rhombus cell with sides of length a at an angle of 120 degrees used for tiling.
func RhombusCell(a float64) Matrix {
	return PrimitiveCell(Point{a, 0.0}, Point{a, 0.0}.Rot(120.0, Origin))
}

// ParallelogramCell is a paralellogram cell with sides of length a and b at an angle of rot degrees used for tiling.
func ParallelogramCell(a, b, rot float64) Matrix {
	return PrimitiveCell(Point{a, 0.0}, Point{b, 0.0}.Rot(rot, Origin))
}

// TileRectangle tiles the given cell (determines the axes along which cells are repeated) onto the rectangle dst (bounds of clipping path), where cells are filled by rectangle src (bounds of object to be tiled).
func TileRectangle(cell Matrix, dst, src Rect) []Matrix {
	// find extremes along cell axes
	invCell := cell.Inv()
	points := []Point{
		invCell.Dot(Point{dst.X0, dst.Y0}),
		invCell.Dot(Point{dst.X1, dst.Y0}),
		invCell.Dot(Point{dst.X1, dst.Y1}),
		invCell.Dot(Point{dst.X0, dst.Y1}),
	}
	x0, x1 := points[0].X, points[0].X
	y0, y1 := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		x0 = math.Min(x0, point.X)
		x1 = math.Max(x1, point.X)
		y0 = math.Min(y0, point.Y)
		y1 = math.Max(y1, point.Y)
	}

	// add/subtract when overflowing/underflowing cell
	cellBounds := src.Transform(invCell)
	x0 -= cellBounds.X1 - 1.0
	y0 -= cellBounds.Y1 - 1.0
	x1 -= cellBounds.X0
	y1 -= cellBounds.Y0

	// collect all positions
	cells := []Matrix{}
	for y := math.Floor(y0); y < y1; y += 1.0 {
		for x := math.Floor(x0); x < x1; x += 1.0 {
			p := cell.Dot(Point{x, y})
			if src.Translate(p.X, p.Y).Overlaps(dst) {
				cells = append(cells, cell.Translate(x, y))
			}
		}
	}
	return cells
}

//// P1 wallpaper group
//var P1 = []Matrix{Identity}
//
//// Pm wallpaper group
//var Pm = []Matrix{
//	Identity,
//	Identity.ReflectYAbout(0.5),
//}
//
//// Pg wallpaper group
//var Pg = []Matrix{
//	Identity,
//	Identity.Translate(0.5, 0.0).ReflectYAbout(0.5),
//}

//func Cm(x, y float64) Tiler {
//	return Tiler{
//		Point{x, 0.0},
//		Point{0.0, 2.0 * y},
//		[]Matrix{
//			Identity,
//			Identity.Translate(x/2.0, 0.0).ReflectYAbout(y / 2.0),
//			Identity.Translate(x/2.0, y),
//			Identity.ReflectYAbout(y),
//		},
//	}
//}

//func Pm(x, y float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, y}
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x, pos.Y).ReflectX(),
//		}
//	}
//}
//
//func Cm(x, y float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, y}
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x/2.0, pos.Y).ReflectX(),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y/2.0),
//			Identity.Translate(pos.X+x, pos.Y+y/2.0).ReflectX(),
//		}
//	}
//}
//
//func P2(x, y, rot float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{y, 0.0}.Rot(rot*math.Pi/180.0, Point{0.0, 0.0})
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x, pos.Y+y).Rotate(180),
//		}
//	}
//}
//
//func Pgg(x, y float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, y}
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y/2.0).Rotate(180),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y).ReflectY(),
//			Identity.Translate(pos.X+x, pos.Y+y/2.0).ReflectX(),
//		}
//	}
//}
//
//func Pmm(x, y float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, y}
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x, pos.Y).ReflectX(),
//			Identity.Translate(pos.X, pos.Y+y).ReflectY(),
//			Identity.Translate(pos.X+x, pos.Y+y).Rotate(180),
//		}
//	}
//}
//
//func Cmm(x, y float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, y}
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x, pos.Y).ReflectX(),
//			Identity.Translate(pos.X, pos.Y+y).ReflectY(),
//			Identity.Translate(pos.X+x, pos.Y+y).Rotate(180),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y/2.0).Rotate(180),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y/2.0).ReflectY(),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y/2.0).ReflectX(),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y/2.0),
//		}
//	}
//}
//
//func Pmg(x, y float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, y}
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X+x/2.0, pos.Y).ReflectX(),
//			Identity.Translate(pos.X+x/2.0, pos.Y+y).ReflectY(),
//			Identity.Translate(pos.X+x, pos.Y+y).Rotate(180),
//		}
//	}
//}
//
//func P4(x float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, x}
//	d := x / 2.0
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(90, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(270, d, d),
//		}
//	}
//}
//
//func P4m(x float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, x}
//	d := x / 2.0
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(90, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(270, d, d),
//			Identity.Translate(pos.X, pos.Y).ReflectXAbout(d).RotateAbout(90, d, d),
//			Identity.Translate(pos.X, pos.Y).ReflectXAbout(d),
//			Identity.Translate(pos.X, pos.Y).ReflectYAbout(d),
//			Identity.Translate(pos.X, pos.Y).ReflectYAbout(d).RotateAbout(90, d, d),
//		}
//	}
//}
//
//func P4g(x float64) Tiler {
//	a := Point{x, 0.0}
//	b := Point{0.0, x}
//	d := x / 2.0
//	return func(i, j int) []Matrix {
//		pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
//		return []Matrix{
//			Identity.Translate(pos.X, pos.Y),
//			Identity.Translate(pos.X, pos.Y).ReflectXAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).ReflectYAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, x/4.0, x/4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(90, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(90, d, d).ReflectXAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(90, d, d).ReflectYAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(90, d, d).RotateAbout(180, x/4.0, x/4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, d, d).ReflectXAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, d, d).ReflectYAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(180, d, d).RotateAbout(180, x/4.0, x/4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(270, d, d),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(270, d, d).ReflectXAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(270, d, d).ReflectYAbout(x / 4.0),
//			Identity.Translate(pos.X, pos.Y).RotateAbout(270, d, d).RotateAbout(180, x/4.0, x/4.0),
//		}
//	}
//}

//func P3(d float64) Tiler {
//	a := Point{d, 0.0}
//	b := Point{d, 0.0}.Rot(60.0*math.Pi/180.0, Point{0.0, 0.0})
//	return func() (Point, Point, []Matrix) {
//		return a, b, []Matrix{
//			Identity,
//			Identity.Rotate(120.0),
//			Identity.Rotate(240.0),
//		}
//	}
//}

// WallpaperGroup applies the symmetry transformations for the given plane symmetry (or wallpaper) group within the given primitive cell.
//func (p *Path) WallpaperGroup(group []Matrix, cell Matrix) *Path {
//	// apply isometries
//	r := &Path{}
//	invCell := cell.Inv()
//	for _, isometry := range group {
//		rev := (isometry[0][0] < 0.0) != (isometry[1][1] < 0.0)
//		isometry = Identity.Mul(cell).Mul(isometry).Mul(invCell)
//		if p.Closed() && rev {
//			r = r.Append(p.Copy().Transform(isometry).Reverse())
//		} else {
//			r = r.Append(p.Copy().Transform(isometry))
//		}
//	}
//	return r
//}

// Tile tiles a path within a clipping path using the given primitive cell.
func (p *Path) Tile(clip *Path, cell Matrix) *Path {
	// get path overflow out of cell
	bounds := p.FastBounds()
	clipBounds := clip.FastBounds()
	cells := TileRectangle(cell, clipBounds, bounds)

	// append all tiles
	r := &Path{}
	for _, cell := range cells {
		pos := cell.Dot(Origin)
		r = r.Append(p.Copy().Translate(pos.X, pos.Y))
	}
	return r.And(clip)
}

// Triangulate tessellates the path with triangles that fill the path. WIP
func (p *Path) Triangulate() ([][3]Point, [][5]Point) {
	p = p.ReplaceArcs()

	beziers := [][5]Point{}
	contour := []*poly2tri.Point{}
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd, LineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
		case QuadToCmd:
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			cp1, cp2 := quadraticToCubicBezier(start, cp, end)
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
			beziers = append(beziers, [5]Point{start, cp1, cp2, end, {1.0, 1.0}})
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
			beziers = append(beziers, [5]Point{start, cp1, cp2, end, {1.0, 1.0}})
		case ArcToCmd:
			panic("arcs should have been replaced")
		}
		i += cmdLen(cmd)
		start = end
	}

	swctx := poly2tri.NewSweepContext(contour, false)
	swctx.Triangulate()

	triangles := [][3]Point{}
	for _, tr := range swctx.GetTriangles() {
		p0 := Point{tr.Points[0].X, tr.Points[0].Y}
		p1 := Point{tr.Points[1].X, tr.Points[1].Y}
		p2 := Point{tr.Points[2].X, tr.Points[2].Y}
		triangles = append(triangles, [3]Point{p0, p1, p2})
	}
	return triangles, beziers
}
