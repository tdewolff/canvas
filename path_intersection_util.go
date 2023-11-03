package canvas

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

// Intersections amongst the combinations between line, quad, cube, elliptical arcs. We consider four cases: the curves do not cross nor touch (intersections is empty), the curves intersect (and cross), the curves intersect tangentially (touching), or the curves are identical (or parallel in the case of two lines). In the last case we say there are no intersections. As all curves are segments, it is considered a secant intersection when the segments touch but "intent to" cut at their ends (i.e. when position equals to 0 or 1 for either segment).

type intersectionKind int

// AintoB is when A intersects and goes into the left-hand side of B, BintoA is the reverse
const (
	AintoB intersectionKind = 1
	BintoA intersectionKind = 2
)

func (v intersectionKind) String() string {
	if v == AintoB {
		return " AintoB"
	} else if v == BintoA {
		return " BintoA"
	}
	return ""
}

type intersectionParallel int

// Parallel is set when the intersection point is the start of a piece of parallel paths. After the collisions() function the AParallel and BParallel values are used to indicate if following along A is parallel, following along B is parallel, or both.
const (
	NoParallel intersectionParallel = 0
	AParallel  intersectionParallel = 1
	BParallel  intersectionParallel = 2
	Parallel   intersectionParallel = 3
)

func (v intersectionParallel) String() string {
	if v == Parallel {
		return " Parallel"
	} else if v == AParallel {
		return " AParallel"
	} else if v == BParallel {
		return " BParallel"
	}
	return ""
}

type Intersection struct {
	// SegA, SegB, and Parallel are filled/specified only for path intersections, not segment
	Point
	SegA, SegB int
	TA, TB     float64 // position along segment in [0,1]
	DirA, DirB float64 // angle of direction along segment
	Kind       intersectionKind
	Parallel   intersectionParallel // NoParallel or Parallel
	Tangent    bool
}

func (z Intersection) Equals(o Intersection) bool {
	return z.Point.Equals(o.Point) && z.SegA == o.SegA && z.SegB == o.SegB && Equal(z.TA, o.TA) && Equal(z.TB, o.TB) && angleEqual(z.DirA, o.DirA) && angleEqual(z.DirB, o.DirB) && z.Kind == o.Kind && z.Parallel == o.Parallel
}

func (z Intersection) String() string {
	tangent := ""
	if z.Parallel == NoParallel && z.Tangent {
		tangent = " Tangent"
	}
	return fmt.Sprintf("pos={%g,%g} seg={%d,%d} t={%g,%g} dir={%g°,%g°}%v%v%v", z.Point.X, z.Point.Y, z.SegA, z.SegB, z.TA, z.TB, angleNorm(z.DirA)*180.0/math.Pi, angleNorm(z.DirB)*180.0/math.Pi, z.Kind, z.Parallel, tangent)
}

type Intersections []Intersection

// There are intersections.
func (zs Intersections) Has() bool {
	return 0 < len(zs)
}

// HasSecant returns true when there are secant intersections, i.e. the curves intersect and cross (they cut).
func (zs Intersections) HasSecant() bool {
	for _, z := range zs {
		if !z.Tangent {
			return true
		}
	}
	return false
}

// HasTangent returns true when there are tangent intersections, i.e. the curves intersect but don't cross (they touch).
func (zs Intersections) HasTangent() bool {
	for _, z := range zs {
		if z.Tangent {
			return true
		}
	}
	return false
}

func (zs Intersections) String() string {
	sb := strings.Builder{}
	for i, z := range zs {
		if i != 0 {
			fmt.Fprintf(&sb, "\n")
		}
		fmt.Fprintf(&sb, "%v %v", i, z)
	}
	return sb.String()
}

// sortAndWrapEnd sorts intersections for curve A and then curve B, but wraps intersections at the end point of the path (which equals the position of the start of the path) to the front of the list. Length parameters should be the number of segments in A and B respectively.
func (zs Intersections) sortAndWrapEnd(segOffsetA, segOffsetB, lenA, lenB int) {
	pos := func(z Intersection) (float64, float64) {
		posa := float64(z.SegA) + z.TA
		if Equal(z.TA, 1.0) {
			posa -= Epsilon
			if z.SegA == segOffsetA+lenA-1 {
				posa -= float64(lenA - 1) // put end into first segment (moveto)
			}
		}
		posb := float64(z.SegB) + z.TB
		if Equal(z.TB, 1.0) {
			posb -= Epsilon
			if z.SegB == segOffsetB+lenB-1 {
				posb -= float64(lenB - 1) // put end into first segment (moveto)
			}
		}
		return posa, posb
	}

	sort.SliceStable(zs, func(i, j int) bool {
		// sort by P and secondary to Q. Consider a point at the very end of the curve (seg=len-1, t=1) as if it were at the beginning, since it is on the starting point of the path
		posai, posbi := pos(zs[i])
		posaj, posbj := pos(zs[j])
		posi := 1000.0*posai/float64(lenA) + posbi/float64(lenB)
		posj := 1000.0*posaj/float64(lenA) + posbj/float64(lenB)
		return posi < posj
	})
}

// ASort sorts intersections for curve A
func (zs Intersections) ASort() {
	sort.SliceStable(zs, func(i, j int) bool {
		zi, zj := zs[i], zs[j]
		if zi.SegA == zj.SegA {
			if Equal(zi.TA, zj.TA) {
				// A intersects B twice at the same point, sort in case of parallel parts
				// TODO: is this valid?? make sure that sorting is consistent to match with order when intersections are slightly separated. That is, you have outer and inner intersection pairs related to the parallel parts in between, that should be sorted as such (outer incoming, inner incoming, inner outgoing, outer outgoing) over A
				return zi.Kind == BintoA
			}
			return zi.TA < zj.TA
		}
		return zi.SegA < zj.SegA
	})
}

// ArgASort sorts indices of intersections for curve A
func (zs Intersections) ArgASort() []int {
	idx := make([]int, len(zs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		zi, zj := zs[idx[i]], zs[idx[j]]
		if zi.SegA == zj.SegA {
			if Equal(zi.TA, zj.TA) {
				// A intersects B twice at the same point, sort in case of parallel parts
				// TODO: is this valid?? make sure that sorting is consistent to match with order when intersections are slightly separated. That is, you have outer and inner intersection pairs related to the parallel parts in between, that should be sorted as such (outer incoming, inner incoming, inner outgoing, outer outgoing) over A
				return zi.Kind == BintoA
			}
			return zi.TA < zj.TA
		}
		return zi.SegA < zj.SegA
	})
	return idx
}

// ArgBSort sorts indices of intersections for curve B
func (zs Intersections) ArgBSort() []int {
	idx := make([]int, len(zs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		zi, zj := zs[idx[i]], zs[idx[j]]
		if zi.SegB == zj.SegB {
			if Equal(zi.TB, zj.TB) {
				// A intersects B twice at the same point, sort in case of parallel parts
				// TODO: is this valid?? make sure that sorting is consistent to match with order when intersections are slightly separated. That is, you have outer and inner intersection pairs related to the parallel parts in between, that should be sorted as such (outer incoming, inner incoming, inner outgoing, outer outgoing) over B
				return zi.Kind == AintoB
			}
			return zi.TB < zj.TB
		}
		return zi.SegB < zj.SegB
	})
	return idx
}

// intersect for path segments a and b, starting at a0 and b0
func (zs Intersections) appendSegment(segA int, a0 Point, a []float64, segB int, b0 Point, b []float64) Intersections {
	// TODO: add fast check if bounding boxes overlap, below doesn't account for vertical/horizontal lines

	n := len(zs)
	swapCurves := false
	if a[0] == LineToCmd || a[0] == CloseCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineLine(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]})
		} else if b[0] == QuadToCmd {
			zs = zs.LineQuad(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]})
		} else if b[0] == CubeToCmd {
			zs = zs.LineCube(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]}, Point{b[5], b[6]})
		} else if b[0] == ArcToCmd {
			rx := b[1]
			ry := b[2]
			phi := b[3] * math.Pi / 180.0
			large, sweep := toArcFlags(b[4])
			cx, cy, theta0, theta1 := ellipseToCenter(b0.X, b0.Y, rx, ry, phi, large, sweep, b[5], b[6])
			zs = zs.LineEllipse(a0, Point{a[1], a[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
		}
	} else if a[0] == QuadToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineQuad(b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]})
			swapCurves = true
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for quad-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for quad-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for quad-arc")
		}
	} else if a[0] == CubeToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineCube(b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]}, Point{a[5], a[6]})
			swapCurves = true
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
		cx, cy, theta0, theta1 := ellipseToCenter(a0.X, a0.Y, rx, ry, phi, large, sweep, a[5], a[6])
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineEllipse(b0, Point{b[1], b[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			swapCurves = true
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for arc-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for arc-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for arc-arc")
		}
	}

	// swap A and B in the intersection found to match segments A and B of this function
	if swapCurves {
		for i := n; i < len(zs); i++ {
			zs[i].SegA, zs[i].SegB = segA, segB
			zs[i].TA, zs[i].TB = zs[i].TB, zs[i].TA
			zs[i].DirA, zs[i].DirB = zs[i].DirB, zs[i].DirA
			if zs[i].Kind == BintoA {
				zs[i].Kind = AintoB
			} else {
				zs[i].Kind = BintoA
			}
		}
	} else {
		for i := n; i < len(zs); i++ {
			zs[i].SegA, zs[i].SegB = segA, segB
		}
	}
	return zs
}

func (zs Intersections) add(pos Point, ta, tb float64, dira, dirb float64, tangent bool) Intersections {
	// the segment-segment functions check whether ta/tb are between [0.0,1.0+Epsilon], clamp
	if ta < 0.0 {
		ta = 0.0
	} else if 1.0 < ta {
		ta = 1.0
	}
	if tb < 0.0 {
		tb = 0.0
	} else if 1.0 < tb {
		tb = 1.0
	}

	var kind intersectionKind
	var parallel intersectionParallel
	if angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi) {
		parallel = Parallel
	} else {
		if angleNorm(dirb-dira) < math.Pi {
			kind = BintoA // B goes to LHS of A, A goes to RHS of B
		} else {
			kind = AintoB // A goes to LHS of B, B goes to RHS of A
		}
	}
	if parallel == Parallel || Equal(ta, 0.0) || Equal(tb, 0.0) || Equal(ta, 1.0) || Equal(tb, 1.0) {
		tangent = true
	}
	return append(zs, Intersection{
		Point:    pos,
		TA:       ta,
		TB:       tb,
		DirA:     dira,
		DirB:     dirb,
		Kind:     kind,
		Parallel: parallel,
		Tangent:  tangent,
	})
}

// http://www.cs.swan.ac.uk/~cssimon/line_intersection.html
func (zs Intersections) LineLine(a0, a1, b0, b1 Point) Intersections {
	if a0.Equals(a1) || b0.Equals(b1) {
		return zs
	}

	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if Equal(div, 0.0) {
		// parallel
		if Equal(da.PerpDot(b1.Sub(a0)), 0.0) {
			// aligned, rotate to x-axis
			angle0 := da.Angle()
			angle1 := db.Angle()
			a := a0.Rot(-angle0, Point{}).X
			b := a1.Rot(-angle0, Point{}).X
			c := b0.Rot(-angle0, Point{}).X
			d := b1.Rot(-angle0, Point{}).X
			if Interval(a, c, d) && Interval(b, c, d) || Interval(a, d, c) && Interval(b, d, c) {
				// a-b in c-d or a-b == c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), angle0, angle1, true)
				zs = zs.add(a1, 1.0, (b-c)/(d-c), angle0, angle1, true)
			} else if Interval(c, a, b) && Interval(d, a, b) {
				// c-d in a-b
				zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
			} else if Interval(a, c, d) || Interval(a, d, c) {
				// a in c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), angle0, angle1, true)
				if a < d-Epsilon {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
				} else if a < c-Epsilon {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				}
			} else if Interval(b, c, d) || Interval(b, d, c) {
				// b in c-d
				if c < b-Epsilon {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				} else if d < b-Epsilon {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
				}
				zs = zs.add(a1, 1.0, (b-c)/(d-c), angle0, angle1, true)
			}
		}
		return zs
	}

	ta := db.PerpDot(a0.Sub(b0)) / div
	tb := da.PerpDot(a0.Sub(b0)) / div
	if Interval(ta, 0.0, 1.0) && Interval(tb, 0.0, 1.0) {
		zs = zs.add(a0.Interpolate(a1, ta), ta, tb, da.Angle(), db.Angle(), false)
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func (zs Intersections) LineQuad(l0, l1, p0, p1, p2 Point) Intersections {
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

	dira := l1.Sub(l0).Angle()
	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	for _, root := range roots {
		if Interval(root, 0.0, 1.0) {
			var s float64
			pos := quadraticBezierPos(p0, p1, p2, root)
			if horizontal {
				s = (pos.X - l0.X) / (l1.X - l0.X)
			} else {
				s = (pos.Y - l0.Y) / (l1.Y - l0.Y)
			}
			if Interval(s, 0.0, 1.0) {
				deriv := quadraticBezierDeriv(p0, p1, p2, root)
				dirb := deriv.Angle()
				// deviate angle slightly to distinguish between BintoA/AintoB on head-on collision
				if Equal(root, 0.0) || Equal(root, 1.0) || Equal(s, 0.0) || Equal(s, 1.0) {
					deriv2 := quadraticBezierDeriv2(p0, p1, p2)
					if (0.0 <= deriv.PerpDot(deriv2)) == (Equal(root, 0.0) || !Equal(root, 1.0) && Equal(s, 0.0)) {
						dirb += Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
					}
				}
				zs = zs.add(pos, s, root, dira, dirb, Equal(A.Dot(deriv), 0.0))
			}
		}
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func (zs Intersections) LineCube(l0, l1, p0, p1, p2, p3 Point) Intersections {
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

	dira := l1.Sub(l0).Angle()
	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	for _, root := range roots {
		if Interval(root, 0.0, 1.0) {
			var s float64
			pos := cubicBezierPos(p0, p1, p2, p3, root)
			if horizontal {
				s = (pos.X - l0.X) / (l1.X - l0.X)
			} else {
				s = (pos.Y - l0.Y) / (l1.Y - l0.Y)
			}
			if Interval(s, 0.0, 1.0) {
				deriv := cubicBezierDeriv(p0, p1, p2, p3, root)
				dirb := deriv.Angle()
				// deviate angle slightly to distinguish between BintoA/AintoB on head-on collision
				if Equal(root, 0.0) || Equal(root, 1.0) || Equal(s, 0.0) || Equal(s, 1.0) {
					deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
					if (0.0 <= deriv.PerpDot(deriv2)) == (Equal(root, 0.0) || !Equal(root, 1.0) && Equal(s, 0.0)) {
						dirb += Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
					}
				}

				// deviate angle slightly to distinguish between BintoA/AintoB when the line and the cubic bezier are parallel only in the intersection, but the paths do cross
				tangent := Equal(A.Dot(deriv), 0.0)
				if !Equal(root, 0.0) && !Equal(root, 1.0) && (angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi)) {
					dirb = p3.Sub(p0).Angle()
					tangent = false
				}
				zs = zs.add(pos, s, root, dira, dirb, tangent)
			}
		}
	}
	return zs
}

func (zs Intersections) LineEllipse(l0, l1, center, radius Point, phi, theta0, theta1 float64) Intersections {
	dira := l1.Sub(l0).Angle()

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

	for _, root := range roots {
		// get intersection position with center as origin
		var x, y, s float64
		if horizontal {
			x = root
			y = (E - C*x) / D
			s = (x - l0.X) / (l1.X - l0.X)
		} else {
			y = root
			x = (E - D*y) / C
			s = (y - l0.Y) / (l1.Y - l0.Y)
		}

		angle := math.Atan2(y, x)
		if Interval(s, 0.0, 1.0) && angleBetween(angle, theta0, theta1) {
			if theta0 <= theta1 {
				angle = theta0 - Epsilon + angleNorm(angle-theta0+Epsilon)
			} else {
				angle = theta1 - Epsilon + angleNorm(angle-theta1+Epsilon)
			}
			t := (angle - theta0) / (theta1 - theta0)
			pos := Point{x, y}.Rot(phi, Origin).Add(center)
			dirb := ellipseDeriv(radius.X, radius.Y, phi, theta0 <= theta1, angle).Angle()
			// deviate angle slightly to distinguish between BintoA/AintoB on head-on directions
			if Equal(t, 0.0) || Equal(t, 1.0) || Equal(s, 0.0) || Equal(s, 1.0) {
				if (theta0 <= theta1) == (Equal(t, 0.0) || !Equal(t, 1.0) && Equal(s, 0.0)) {
					dirb += Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
				} else {
					dirb -= Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
				}
			}
			zs = zs.add(pos, s, t, dira, dirb, Equal(root, 0.0))
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

func intersectionRayLine(a0, a1, b0, b1 Point) (Point, bool) {
	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if Equal(div, 0.0) {
		// parallel
		return Point{}, false
	}

	tb := da.PerpDot(a0.Sub(b0)) / div
	if Interval(tb, 0.0, 1.0) {
		fmt.Println(tb, b0.Interpolate(b1, tb))
		return b0.Interpolate(b1, tb), true
	}
	return Point{}, false
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
