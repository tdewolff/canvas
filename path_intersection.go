package canvas

import (
	"fmt"
	"math"
	"sort"
)

func (p *Path) And(q *Path) *Path {
	return nil
	//ps, qs, err := cutPair(p, q)
	//if err != nil {
	//	panic(err)
	//} else if ps == nil {
	//	return &Path{}
	//}

	//r := &Path{}
	//state := q.in(ps[0])
	//for i := range ps {
	//	if state {
	//		r = r.Join(ps[i])
	//	} else {
	//		r = r.Join(qs[i])
	//	}
	//	state = !state
	//}
	//r.Close()
	//return r
}

func (p *Path) Not(q *Path) *Path {
	return nil
	//ps, qs, err := cutPair(p, q)
	//if err != nil {
	//	panic(err)
	//} else if ps == nil {
	//	return &Path{}
	//}

	//r := &Path{}
	//state := q.in(ps[0])
	//for i := range ps {
	//	fmt.Println(state)
	//	fmt.Println(ps[i])
	//	fmt.Println(qs[i])
	//	if state {
	//		r = r.Join(qs[i].Reverse())
	//	} else {
	//		r = r.Join(ps[i])
	//	}
	//	state = !state
	//}
	//r.Close()
	//return r
}

func (p *Path) Or(q *Path) *Path {
	zs := p.Intersections(q)
	return p.cut(zs)[0]
}

func (p *Path) Xor(q *Path) *Path {
	zs := p.Intersections(q)
	return p.cut(zs)[0]
}

func (p *Path) Div(q *Path) []*Path {
	return nil
	//zs := p.Intersections(q)
	//if len(zs) == 0 {
	//	return nil
	//}

	//ps := p.cut(zs)
	//qs := q.cut(zs.swapCurves())
	//if len(ps) != len(qs) {
	//	panic("len(ps) != len(qs)")
	//} else if len(ps) == 0 {
	//	panic("len(ps) == 0")
	//}

	//rs := []*Path{}
	//in := p.in(qs[0])
	//for i := range ps {
	//	if in {
	//		rs = append(rs, qs[i])
	//	} else {
	//		rs = append(rs, ps[i])
	//	}
	//}
	//return rs
}

func (p *Path) Cut(q *Path) []*Path {
	zs := p.Intersections(q)
	return p.cut(zs)
}

func (p *Path) in(q *Path) bool {
	fillRule := NonZero // TODO: let user pass, or get from Path?
	q0 := q.StartPos()
	i := 0
	if q.d[0] == MoveToCmd {
		i += cmdLen(MoveToCmd)
	}
	i += cmdLen(q.d[i])
	q1 := Point{q.d[i-3], q.d[i-2]}
	qMid := q0.Interpolate(q1, 0.5)
	return p.Interior(qMid.X, qMid.Y, fillRule)
}

func (p *Path) cut(zs intersections) []*Path {
	var ii int // segment index into p
	var zi int // intersection index into zs
	var start Point
	var iLastMoveTo int
	ps := []*Path{&Path{}}
	ps[0].MoveTo(0.0, 0.0)
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		if cmd == MoveToCmd {
			iLastMoveTo = len(ps) - 1
			if !ps[len(ps)-1].Pos().Equals(Point{p.d[i+1], p.d[i+2]}) {
				ps[len(ps)-1].MoveTo(p.d[i+1], p.d[i+2])
			}
		} else if zi < len(zs) && ii == zs[zi].SegA {
			switch cmd {
			case LineToCmd, CloseCmd:
				for zi < len(zs) && ii == zs[zi].SegA {
					ps[len(ps)-1].LineTo(zs[zi].X, zs[zi].Y)
					ps = append(ps, &Path{})
					ps[len(ps)-1].MoveTo(zs[zi].X, zs[zi].Y)
					zi++
				}
				ps[len(ps)-1].LineTo(p.d[i+1], p.d[i+2])
			case QuadToCmd:
				// TODO: loop zis
				_, a1, a2, b0, b1, b2 := quadraticBezierSplit(start, Point{p.d[i+1], p.d[i+2]}, Point{p.d[i+3], p.d[i+4]}, zs[zi].TA)
				ps[len(ps)-1].QuadTo(a1.X, a1.Y, a2.X, a2.Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(b0.X, b0.Y)
				ps[len(ps)-1].QuadTo(b1.X, b1.Y, b2.X, b2.Y)
			case CubeToCmd:
				// TODO: loop zis
				_, a1, a2, a3, b0, b1, b2, b3 := cubicBezierSplit(start, Point{p.d[i+1], p.d[i+2]}, Point{p.d[i+3], p.d[i+4]}, Point{p.d[i+5], p.d[i+6]}, zs[zi].TA)
				ps[len(ps)-1].CubeTo(a1.X, a1.Y, a2.X, a2.Y, a3.X, a3.Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(b0.X, b0.Y)
				ps[len(ps)-1].CubeTo(b1.X, b1.Y, b2.X, b2.Y, b3.X, b3.Y)
			case ArcToCmd:
				// TODO: loop zis
				rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
				large, sweep := toArcFlags(p.d[i+4])
				end := Point{p.d[i+5], p.d[i+6]}
				cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
				mid, large1, large2, ok := ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, zs[zi].TA)
				if !ok {
					// should never happen
					panic("theta not in elliptic arc range for splitting")
				}

				ps[len(ps)-1].ArcTo(rx, ry, phi*180.0/math.Pi, large1, sweep, mid.X, mid.Y)
				ps = append(ps, &Path{})
				ps[len(ps)-1].MoveTo(mid.X, mid.Y)
				ps[len(ps)-1].ArcTo(rx, ry, phi*180.0/math.Pi, large2, sweep, end.X, end.Y)
			}
		} else if cmd == CloseCmd {
			ps[len(ps)-1].LineTo(p.d[i+1], p.d[i+2])
		} else {
			ps[len(ps)-1].d = append(ps[len(ps)-1].d, p.d[i:i+cmdLen(cmd)]...)
		}
		if cmd == CloseCmd && iLastMoveTo != len(ps)-1 {
			// join close command with last moveto
			ps[iLastMoveTo] = ps[len(ps)-1].Join(ps[iLastMoveTo])
			ps = ps[:len(ps)-1]
		}
		i += cmdLen(cmd)
		start = Point{p.d[i-3], p.d[i-2]}
		ii++
	}
	return ps
}

type pathIntersectionPos struct {
	Point
	seg int
	t   float64
}

type pathIntersection struct {
	intersection
	prevA, nextA *pathIntersection
	prevB, nextB *pathIntersection
}

func cutPath(p, q *Path, z0, z1 intersection) (*Path, *Path) {
	pReverse := z1.SegA < z0.SegA || z1.SegA == z0.SegA && z1.TA < z0.TA
	qReverse := z1.SegB < z0.SegB || z1.SegB == z0.SegB && z1.TB < z0.TB

	// order intersections
	zp0, zp1 := z0, z1
	if pReverse {
		zp0, zp1 = z1, z0
	}
	zq0, zq1 := z0, z1
	if qReverse {
		zq0, zq1 = z1, z0
	}

	seg := 0
	var start Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		if seg == zp0.SegA {
			t0 := zp0.TA
			t1 := 1.0
			if seg == zp1.SegA {
				t1 = zp1.TA
			}
			_, _, _, _ = zp0, zp1, zq0, zq1
			_, _ = t0, t1
			_ = start
			//cutPathSegment(start, p.d[i:i+cmdLen(cmd)], zp0, zp1, t0, t1)
		}
		i += cmdLen(cmd)
		start = Point{p.d[i-3], p.d[i-2]}
		seg++
	}

	return nil, nil
}

func cutPathSegment(p0 Point, p []float64, pos0, pos1 Point, t0, t1 float64) *Path {
	return &Path{}
	//r := &Path{}
	//if p[0] == LineToCmd || p[0] == CloseCmd {
	//	r.MoveTo(pos0.X, pos0.Y)
	//	r.LineTo(

	//}
}

// get intersections for paths p and q sorted for both
func pathIntersections(p, q *Path) *pathIntersection {
	zs := p.Intersections(q)
	if len(zs) == 0 {
		return nil
	} else if len(zs)%2 != 0 {
		panic("len(zs)%2 != 0")
	}

	head := &pathIntersection{
		intersection: zs[0],
	}
	prev := head
	list := []*pathIntersection{head}
	for _, z := range zs[1:] {
		next := &pathIntersection{
			intersection: z,
			prevA:        prev,
		}
		list = append(list, next)
		prev.nextA = next
		prev = next
	}
	head.prevA = prev
	prev.nextA = head

	idxs := zs.swappedArgSort() // sorted indices for intersections of q by p
	for idxQ, idxP := range idxs {
		if 0 < idxQ {
			list[idxP].prevB = list[idxs[idxQ-1]]
		}
		if idxQ < len(idxs)-1 {
			list[idxP].nextB = list[idxs[idxQ+1]]
		}
	}
	list[idxs[0]].prevB = list[idxs[len(idxs)-1]]
	list[idxs[len(idxs)-1]].nextB = list[idxs[0]]
	return head
}

// Intersections for path p by path q, sorted for path p.
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
					zs := intersectSegments(pStart, p.d[i:i+pLen], qStart, q.d[j:j+qLen])
					for k, _ := range zs {
						zs[k].SegA = pI
						zs[k].SegB = qI
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
	sort.Stable(zss)
	return zss
}

// intersect for path segments a and b, starting at a0 and b0
func intersectSegments(a0 Point, a []float64, b0 Point, b []float64) intersections {
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
	SegA, SegB int     // segment indices
	TA, TB     float64 // line or Bézier curve parameter, or arc angle, of intersection
	Tangent    bool    // tangential non-crossing/touching
}

func (z intersection) String() string {
	s := fmt.Sprintf("pos=%v seg={%d,%d} t={%.5g,%.5g}", z.Point, z.SegA, z.SegB, z.TA, z.TB)
	if z.Tangent {
		s += " tangent"
	}
	return s
}

// intersections sorted for curve A
type intersections []intersection

func (zs *intersections) add(pos Point, ta, tb float64, tangent bool) {
	*zs = append(*zs, intersection{
		Point:   pos,
		TA:      ta,
		TB:      tb,
		Tangent: tangent,
	})
}

func (zs intersections) Len() int {
	return len(zs)
}

func (zs intersections) Swap(i, j int) {
	zs[i], zs[j] = zs[j], zs[i]
}

func (zs intersections) Less(i, j int) bool {
	if zs[i].SegA == zs[j].SegA {
		return zs[i].TA < zs[j].TA
	}
	return zs[i].SegA < zs[j].SegA
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

// sort indices of intersections for curve B
type intersectionsSwappedArgSort struct {
	intersections
	idx []int
}

func (zs intersectionsSwappedArgSort) Swap(i, j int) {
	zs.idx[i], zs.idx[j] = zs.idx[j], zs.idx[i]
}

func (zs intersectionsSwappedArgSort) Less(i, j int) bool {
	if zs.intersections[zs.idx[i]].SegB == zs.intersections[zs.idx[j]].SegB {
		return zs.intersections[zs.idx[i]].TB < zs.intersections[zs.idx[j]].TB
	}
	return zs.intersections[zs.idx[i]].SegB < zs.intersections[zs.idx[j]].SegB
}

// get indices of sorted intersections for curve B
func (zs intersections) swappedArgSort() []int {
	idx := make([]int, len(zs))
	for i := range idx {
		idx[i] = i
	}
	sort.Stable(intersectionsSwappedArgSort{zs, idx})
	return idx
}

func (zs intersections) swapCurves() intersections {
	zs2 := make(intersections, len(zs))
	for i, _ := range zs {
		zs2[i].SegA, zs2[i].SegB = zs[i].SegB, zs[i].SegA
		zs2[i].TA, zs2[i].TB = zs[i].TB, zs[i].TA
	}
	return zs2
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
