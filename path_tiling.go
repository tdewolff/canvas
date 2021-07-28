package canvas

import (
	"math"

	"github.com/ByteArena/poly2tri-go"
)

type Tiler struct {
	A, B Point
	Ms   []Matrix
}

func P1(x, y, rot float64) Tiler {
	return Tiler{
		Point{x, 0.0},
		Point{y, 0.0}.Rot(rot*math.Pi/180.0, Point{0.0, 0.0}),
		[]Matrix{
			Identity,
		},
	}
}

func Pm(x, y float64) Tiler {
	return Tiler{
		Point{x, 0.0},
		Point{0.0, y},
		[]Matrix{
			Identity,
			Identity.ReflectXAbout(x / 2.0),
		},
	}
}

func Pg(x, y float64) Tiler {
	return Tiler{
		Point{x, 0.0},
		Point{0.0, y},
		[]Matrix{
			Identity,
			Identity.Translate(x/2.0, 0.0).ReflectYAbout(y / 2.0),
		},
	}
}

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

func (p *Path) Tile(n, m int, tiler Tiler) *Path {
	a, b, ms := tiler.A, tiler.B, tiler.Ms
	pm := &Path{}
	for _, m := range ms {
		pm = pm.Append(p.Transform(m))
	}

	pt := &Path{}
	for j := 0; j < m; j++ {
		for i := 0; i < n; i++ {
			pos := a.Mul(float64(i)).Add(b.Mul(float64(j)))
			pt = pt.Append(pm.Translate(pos.X, pos.Y))
		}
	}
	return pt
}

// Triangulate tessellates the path and returns the triangles that fill the path. WIP
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
