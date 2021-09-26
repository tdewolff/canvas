package canvas

import (
	"fmt"
	"math"
	"sort"
)

func (p *Path) And(q *Path) []*Path {
	zs := p.Intersections(q)
	return p.cut(zs)
}

func (p *Path) Not(q *Path) []*Path {
	zs := p.Intersections(q)
	return p.cut(zs)
}

func (p *Path) Or(q *Path) []*Path {
	zs := p.Intersections(q)
	return p.cut(zs)
}

func (p *Path) Xor(q *Path) []*Path {
	zs := p.Intersections(q)
	return p.cut(zs)
}

func (p *Path) Div(q *Path) []*Path {
	zs := p.Intersections(q)
	ps := p.cut(zs)
	qs := q.cut(zs.swapCurves())
	_ = qs
	return ps
}

func (p *Path) Cut(q *Path) []*Path {
	zs := p.Intersections(q)
	return p.cut(zs)
}

func (p *Path) cut(zs intersections) []*Path {
	sort.Sort(zs)

	var j int  // copied-until index into p.d
	var ii int // segment index into p
	var zi int // intersection index into zs
	var start Point
	var iLastMoveTo int
	ps := []*Path{&Path{}}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		if cmd == MoveToCmd {
			iLastMoveTo = len(ps) - 1
		} else if zi < len(zs) && ii == zs[zi].Ia {
			ps[len(ps)-1].d = append(ps[len(ps)-1].d, p.d[j:i]...)
			j = i + cmdLen(cmd)

			switch cmd {
			case CloseCmd:
				ps[len(ps)-1].LineTo(zs[zi].X, zs[zi].Y)
				q := &Path{}
				q.MoveTo(zs[zi].X, zs[zi].Y)
				q.LineTo(p.d[i+1], p.d[i+2])
				ps[iLastMoveTo] = q.Join(ps[iLastMoveTo])
			case LineToCmd:
				ps[len(ps)-1].LineTo(zs[zi].X, zs[zi].Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(zs[zi].X, zs[zi].Y)
				ps[len(ps)-1].LineTo(p.d[i+1], p.d[i+2])
			case QuadToCmd:
				_, a1, a2, b0, b1, b2 := quadraticBezierSplit(start, Point{p.d[i+1], p.d[i+2]}, Point{p.d[i+3], p.d[i+4]}, zs[zi].Ta)
				ps[len(ps)-1].QuadTo(a1.X, a1.Y, a2.X, a2.Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(b0.X, b0.Y)
				ps[len(ps)-1].QuadTo(b1.X, b1.Y, b2.X, b2.Y)
			case CubeToCmd:
				_, a1, a2, a3, b0, b1, b2, b3 := cubicBezierSplit(start, Point{p.d[i+1], p.d[i+2]}, Point{p.d[i+3], p.d[i+4]}, Point{p.d[i+5], p.d[i+6]}, zs[zi].Ta)
				ps[len(ps)-1].CubeTo(a1.X, a1.Y, a2.X, a2.Y, a3.X, a3.Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(b0.X, b0.Y)
				ps[len(ps)-1].CubeTo(b1.X, b1.Y, b2.X, b2.Y, b3.X, b3.Y)
			case ArcToCmd:
				rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
				large, sweep := toArcFlags(p.d[i+4])
				end := Point{p.d[i+5], p.d[i+6]}
				cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
				mid, large1, large2, ok := ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, zs[zi].Ta)
				if !ok {
					// should never happen
					panic("theta not in elliptic arc range for splitting")
				}

				ps[len(ps)-1].ArcTo(rx, ry, phi*180.0/math.Pi, large1, sweep, mid.X, mid.Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(mid.X, mid.Y)
				ps[len(ps)-1].ArcTo(rx, ry, phi*180.0/math.Pi, large2, sweep, end.X, end.Y)
			}
			zi++
		}
		i += cmdLen(cmd)
		start = Point{p.d[i-3], p.d[i-2]}
		ii++
	}
	return ps
}

func (p *Path) Intersections(q *Path) intersections {
	// TODO: uses O(N^2), try sweep line or bently-ottman to reduce to O((N+K) log N)
	zss := intersections{}
	var pI, qI int
	var pStart, qStart Point
	for i := 0; i < len(p.d); {
		pLen := cmdLen(p.d[i])
		if p.d[i] != MoveToCmd {
			qI = 0
			qStart = Origin
			for j := 0; j < len(q.d); {
				qLen := cmdLen(q.d[j])
				if q.d[j] != MoveToCmd {
					zs := intersectSegments(pStart, qStart, p.d[i:i+pLen], q.d[j:j+qLen])
					for k, _ := range zs {
						zs[k].Ia = pI
						zs[k].Ib = qI
					}
					zss = append(zss, zs...)
				}
				j += qLen
				qStart = Point{q.d[j-3], q.d[j-2]}
				qI++
			}
		}
		i += pLen
		pStart = Point{p.d[i-3], p.d[i-2]}
		pI++
	}
	return zss
}

func intersectSegments(a0, b0 Point, a, b []float64) intersections {
	// TODO: add fast check if bounding boxes overlap
	// check if approximated bounding boxes overlap
	//axmin, axmax := math.Min(a0.X, a[len(a)-3]), math.Max(a0.X, a[len(a)-3])
	//aymin, aymax := math.Min(a0.Y, a[len(a)-2]), math.Max(a0.Y, a[len(a)-2])
	//if a[0] == QuadToCmd {
	//	axmin, axmax = math.Min(axmin, a[len(a)-5]), math.Max(axmax, a[len(a)-5])
	//	aymin, aymax = math.Min(aymin, a[len(a)-4]), math.Max(aymax, a[len(a)-4])
	//} else if a[0] == CubeToCmd {
	//	axmin, axmax = math.Min(axmin, a[len(a)-7]), math.Max(axmax, a[len(a)-7])
	//	aymin, aymax = math.Min(aymin, a[len(a)-6]), math.Max(aymax, a[len(a)-6])
	//	axmin, axmax = math.Min(axmin, a[len(a)-5]), math.Max(axmax, a[len(a)-5])
	//	aymin, aymax = math.Min(aymin, a[len(a)-4]), math.Max(aymax, a[len(a)-4])
	//} else if a[0] == ArcToCmd {
	//}

	if a[0] == LineToCmd || a[0] == CloseCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			return intersectionLineLine(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]})
		} else if b[0] == QuadToCmd {
			return intersectionLineQuad(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]})
		} else if b[0] == CubeToCmd {
			return intersectionLineCube(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]}, Point{b[5], b[6]})
		} else if b[0] == ArcToCmd {
			rx := b[1]
			ry := b[2]
			phi := b[3] * math.Pi / 180.0
			large, sweep := toArcFlags(b[4])
			cx, cy, theta0, theta1 := ellipseToCenter(b0.X, b0.Y, rx, ry, phi, large, sweep, b[5], b[6])
			return intersectionLineEllipse(a0, Point{a[1], a[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
		}
	} else if a[0] == QuadToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			return intersectionLineQuad(b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]}).swapCurves()
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for quad-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for quad-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for quad-arc")
		}
	} else if a[0] == CubeToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			return intersectionLineCube(b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]}, Point{a[5], a[6]}).swapCurves()
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for cube-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for cube-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for cube-arc")
		}
	} else if a[0] == ArcToCmd {
		rx := a[1]
		ry := a[2]
		phi := a[3] * math.Pi / 180.0
		large, sweep := toArcFlags(a[4])
		cx, cy, theta0, theta1 := ellipseToCenter(b0.X, b0.Y, rx, ry, phi, large, sweep, a[5], a[6])
		if b[0] == LineToCmd || b[0] == CloseCmd {
			return intersectionLineEllipse(b0, Point{b[1], b[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1).swapCurves()
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for arc-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for arc-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for arc-arc")
		}
	}
	return intersections{} // has MoveCmd
}

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

// Intersections amongst the combinations between line, quad, cube, elliptical arcs. We consider four cases: the curves do not cross nor touch (intersections is empty), the curves intersect (and cross), the curves intersect tangentially (touching), or the curves are identical (or parallel in the case of two lines). In the last case we say there are no intersections. As all curves are segments, it is considered a secant intersection when the segments touch but "intent to" cut at their ends (i.e. when s or t equals to 0 or 1 for either segment).

type intersection struct {
	Point
	Ia, Ib  int     // segment index (used for whole paths only
	Ta, Tb  float64 // line or Bézier curve parameter, or arc angle, of intersection
	Tangent bool    // tangential non-crossing/touching
}

func (z intersection) String() string {
	s := fmt.Sprintf("pos=%v i={%d,%d} t={%.5g,%.5g}", z.Point, z.Ia, z.Ib, z.Ta, z.Tb)
	if z.Tangent {
		s += " tangent"
	}
	return s
}

type intersections []intersection

func (zs *intersections) add(pos Point, ta, tb float64, tangent bool) {
	*zs = append(*zs, intersection{
		Point:   pos,
		Ta:      ta,
		Tb:      tb,
		Tangent: tangent,
	})
}

func (zs intersections) swapCurves() intersections {
	for i, _ := range zs {
		zs[i].Ia, zs[i].Ib = zs[i].Ib, zs[i].Ia
		zs[i].Ta, zs[i].Tb = zs[i].Tb, zs[i].Ta
	}
	return zs
}

func (zs intersections) Len() int {
	return len(zs)
}

func (zs intersections) Swap(i, j int) {
	zs[i], zs[j] = zs[j], zs[i]
}

func (zs intersections) Less(i, j int) bool {
	if zs[i].Ia == zs[j].Ia {
		return zs[i].Ta < zs[j].Tb
	}
	return zs[i].Ia < zs[j].Ib
}

// There are intersections.
func (zs intersections) Has() bool {
	return 0 < len(zs)
}

// There are secants, i.e. the curves intersect and cross (they cut).
func (zs intersections) HasSecant() bool {
	for _, z := range zs {
		if !z.Tangent {
			return true
		}
	}
	return false
}

// There are tangents, i.e. the curves intersect but don't cross (they touch).
func (zs intersections) HasTangent() bool {
	for _, z := range zs {
		if z.Tangent {
			return true
		}
	}
	return false
}

// http://www.cs.swan.ac.uk/~cssimon/line_intersection.html
func intersectionLineLine(a0, a1, b0, b1 Point) intersections {
	zs := intersections{}
	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if Equal(div, 0.0) {
		return zs
	}

	ta := db.PerpDot(a0.Sub(b0)) / div
	tb := da.PerpDot(a0.Sub(b0)) / div
	if 0.0 <= ta && ta <= 1.0 && 0.0 <= tb && tb <= 1.0 {
		zs.add(a0.Interpolate(a1, ta), ta, tb, false)
	}
	return zs
}

// http://mathworld.wolfram.com/Circle-LineIntersection.html
func intersectionRayCircle(l0, l1, c Point, r float64) (Point, Point, bool) {
	d := l1.Sub(l0).Norm(1.0) // along line direction, anchored in l0, its length is 1
	D := l0.Sub(c).PerpDot(d)
	discriminant := r*r - D*D
	if discriminant < 0 {
		return Point{}, Point{}, false
	}
	discriminant = math.Sqrt(discriminant)

	ax := D * d.Y
	bx := d.X * discriminant
	if d.Y < 0.0 {
		bx = -bx
	}
	ay := -D * d.X
	by := math.Abs(d.Y) * discriminant
	return c.Add(Point{ax + bx, ay + by}), c.Add(Point{ax - bx, ay - by}), true
}

// https://math.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
func intersectionCircleCircle(c0 Point, r0 float64, c1 Point, r1 float64) (Point, Point, bool) {
	R := c0.Sub(c1).Length()
	if R < math.Abs(r0-r1) || r0+r1 < R || c0.Equals(c1) {
		return Point{}, Point{}, false
	}
	R2 := R * R

	k := r0*r0 - r1*r1
	a := 0.5
	b := 0.5 * k / R2
	c := 0.5 * math.Sqrt(2.0*(r0*r0+r1*r1)/R2-k*k/(R2*R2)-1.0)

	i0 := c0.Add(c1).Mul(a)
	i1 := c1.Sub(c0).Mul(b)
	i2 := Point{c1.Y - c0.Y, c0.X - c1.X}.Mul(c)
	return i0.Add(i1).Add(i2), i0.Add(i1).Sub(i2), true
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineQuad(l0, l1, p0, p1, p2 Point) intersections {
	// write line as A.X = bias
	A := Point{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p0.Sub(p1.Mul(2.0)).Add(p2))
	b := A.Dot(p1.Sub(p0).Mul(2.0))
	c := A.Dot(p0) - bias

	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
		}
	}

	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	if horizontal {
		if l1.X < l0.X {
			l0, l1 = l1, l0
		}
	} else if l1.Y < l0.Y {
		l0, l1 = l1, l0
	}

	zs := intersections{}
	for _, root := range roots {
		if 0.0 <= root && root <= 1.0 {
			pos := quadraticBezierPos(p0, p1, p2, root)
			dif := A.Dot(quadraticBezierDeriv(p0, p1, p2, root))
			if horizontal {
				if l0.X <= pos.X && pos.X <= l1.X {
					zs.add(pos, (pos.X-l0.X)/(l1.X-l0.X), root, Equal(dif, 0.0))
				}
			} else if l0.Y <= pos.Y && pos.Y <= l1.Y {
				zs.add(pos, (pos.Y-l0.Y)/(l1.Y-l0.Y), root, Equal(dif, 0.0))
			}
		}
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineCube(l0, l1, p0, p1, p2, p3 Point) intersections {
	// write line as A.X = bias
	A := Point{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p3.Sub(p0).Add(p1.Mul(3.0)).Sub(p2.Mul(3.0)))
	b := A.Dot(p0.Mul(3.0).Sub(p1.Mul(6.0)).Add(p2.Mul(3.0)))
	c := A.Dot(p1.Mul(3.0).Sub(p0.Mul(3.0)))
	d := A.Dot(p0) - bias

	roots := []float64{}
	r0, r1, r2 := solveCubicFormula(a, b, c, d)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
			if !math.IsNaN(r2) {
				roots = append(roots, r2)
			}
		}
	}

	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	if horizontal {
		if l1.X < l0.X {
			l0, l1 = l1, l0
		}
	} else if l1.Y < l0.Y {
		l0, l1 = l1, l0
	}

	zs := intersections{}
	for _, root := range roots {
		if 0.0 <= root && root <= 1.0 {
			pos := cubicBezierPos(p0, p1, p2, p3, root)
			dif := A.Dot(cubicBezierDeriv(p0, p1, p2, p3, root))
			if horizontal {
				if l0.X <= pos.X && pos.X <= l1.X {
					zs.add(pos, (pos.X-l0.X)/(l1.X-l0.X), root, Equal(dif, 0.0))
				}
			} else if l0.Y <= pos.Y && pos.Y <= l1.Y {
				zs.add(pos, (pos.Y-l0.Y)/(l1.Y-l0.Y), root, Equal(dif, 0.0))
			}
		}
	}
	return zs
}

func intersectionLineEllipse(l0, l1, center, radius Point, phi, theta0, theta1 float64) intersections {
	// we take the ellipse center as the origin and counter-rotate by phi
	l0 = l0.Sub(center).Rot(-phi, Origin)
	l1 = l1.Sub(center).Rot(-phi, Origin)

	// write ellipse as Ax^2 + By^2 = 1 and line as Cx + Dy = E
	A := 1.0 / (radius.X * radius.X)
	B := 1.0 / (radius.Y * radius.Y)
	C := l1.Y - l0.Y
	D := l0.X - l1.X
	E := l0.Dot(Point{C, D})

	// rewrite as a polynomial by substituting x or y: ax^2 + bx + c = 0
	var a, b, c float64
	horizontal := math.Abs(C) <= math.Abs(D)
	if horizontal {
		a = A*D*D + B*C*C
		b = -2.0 * B * E * C
		c = B*E*E - D*D
	} else {
		a = B*C*C + A*D*D
		b = -2.0 * A * E * D
		c = A*E*E - C*C
	}

	// find solutions
	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) && !Equal(r0, r1) {
			roots = append(roots, r1)
		}
	}

	zs := intersections{}
	for _, root := range roots {
		// get intersection position with center as origin
		var x, y, s float64
		if horizontal {
			x = root
			y = (E - C*x) / D
			s = (x - l0.X) / (l1.X - l0.X)
		} else {
			y = root
			x = (E - D*x) / C
			s = (y - l0.Y) / (l1.Y - l0.Y)
		}

		angle := math.Atan2(y, x)
		if 0.0 <= s && s <= 1.0 && angleBetween(angle, theta0, theta1) {
			t := angleNorm(angle-theta0) / angleNorm(theta1-theta0)
			if theta1 < theta0 {
				t = 2.0 - t
			}
			pos := Point{x, y}.Rot(phi, Origin).Add(center)
			zs.add(pos, s, t, Equal(root, 0.0))
		}
	}
	return zs
}

// TODO: bezier-bezier intersection
// TODO: bezier-ellipse intersection
// TODO: ellipse-ellipse intersection

// For Bézier-Bézier interesections:
// see T.W. Sederberg, "Computer Aided Geometric Design", 2012
// see T.W. Sederberg and T. Nishita, "Curve intersection using Bézier clipping", 1990
// see T.W. Sederberg and S.R. Parry, "Comparison of three curve intersection algorithms", 1986
