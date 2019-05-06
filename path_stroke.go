package canvas

import (
	"math"
)

// NOTE: implementation mostly taken from github.com/golang/freetype/raster/stroke.go

// Capper implements Cap, with rhs the path to append to, halfWidth the half width of the stroke,
// pivot the pivot point around which to construct a cap, and n0 the normal at the start of the path.
// The length of n0 is equal to the halfWidth.
type Capper interface {
	Cap(*Path, float64, Point, Point)
}

type CapperFunc func(*Path, float64, Point, Point)

func (f CapperFunc) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	f(p, halfWidth, pivot, n0)
}

// RoundCapper caps the start or end of a path by a round cap.
var RoundCapper Capper = CapperFunc(roundCapper)

func roundCapper(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.ArcTo(halfWidth, halfWidth, 0, false, true, end.X, end.Y)
}

// ButtCapper caps the start or end of a path by a butt cap.
var ButtCapper Capper = CapperFunc(buttCapper)

func buttCapper(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.LineTo(end.X, end.Y)
}

// SquareCapper caps the start or end of a path by a square cap.
var SquareCapper Capper = CapperFunc(squareCapper)

func squareCapper(p *Path, halfWidth float64, pivot, n0 Point) {
	e := n0.Rot90CCW()
	corner1 := pivot.Add(e).Add(n0)
	corner2 := pivot.Add(e).Sub(n0)
	end := pivot.Sub(n0)
	p.LineTo(corner1.X, corner1.Y)
	p.LineTo(corner2.X, corner2.Y)
	p.LineTo(end.X, end.Y)
}

////////////////

// Joiner implements Join, with rhs the right path and lhs the left path to append to, pivot the intersection of both
// path elements, n0 and n1 the normals at the start and end of the path respectively.
// The length of n0 and n1 are equal to the halfWidth.
type Joiner interface {
	Join(*Path, *Path, float64, Point, Point, Point, float64, float64)
}

type JoinerFunc func(*Path, *Path, float64, Point, Point, Point, float64, float64)

func (f JoinerFunc) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	f(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
}

// BevelJoiner connects two path elements by a linear join.
var BevelJoiner Joiner = JoinerFunc(bevelJoiner)

func bevelJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

// RoundJoiner connects two path elements by a round join.
var RoundJoiner Joiner = JoinerFunc(roundJoiner)

func roundJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	cw := n0.Rot90CW().Dot(n1) >= 0
	if cw { // bend to the right, ie. CW
		rhs.LineTo(rEnd.X, rEnd.Y)
		lhs.ArcTo(halfWidth, halfWidth, 0, false, false, lEnd.X, lEnd.Y)
	} else { // bend to the left, ie. CCW
		rhs.ArcTo(halfWidth, halfWidth, 0, false, true, rEnd.X, rEnd.Y)
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

var MiterJoiner Joiner = JoinerFunc(miterJoiner)

func miterJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	} else if n0.Equals(n1.Neg()) {
		bevelJoiner(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	cw := n0.Rot90CW().Dot(n1) >= 0
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	theta := n0.Angle(n1) / 2.0
	d := hw / math.Cos(theta)
	mid := pivot.Add(n0.Add(n1).Norm(d))

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	if cw { // bend to the right, ie. CW
		lhs.LineTo(mid.X, mid.Y)
	} else {
		rhs.LineTo(mid.X, mid.Y)
	}
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

var ArcsJoiner Joiner = JoinerFunc(arcsJoiner)

// https://math.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
func circleCircleIntersection(c0 Point, r0 float64, c1 Point, r1 float64) (Point, Point, bool) {
	R := c0.Sub(c1).Length()
	if R < math.Abs(r0-r1) || r0+r1 < R {
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

// http://mathworld.wolfram.com/Circle-LineIntersection.html
func circleLineIntersection(c Point, r float64, pivot, n Point) (Point, Point, bool) {
	d := n.Rot90CCW().Norm(1.0) // along line direction, anchored in pivot, its length is 1
	D := pivot.Sub(c).PerpDot(d)
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

func arcsJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	} else if n0.Equals(n1.Neg()) {
		bevelJoiner(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	} else if math.IsNaN(r0) && math.IsNaN(r1) {
		miterJoiner(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	cw := n0.Rot90CW().Dot(n1) >= 0
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	// r is the radius of the original curve, R the radius of the stroke curve, c are the centers of the circles
	c0 := pivot.Add(n0.Norm(-r0))
	c1 := pivot.Add(n1.Norm(-r1))
	R0, R1 := math.Abs(r0+hw), math.Abs(r1+hw)

	var i0, i1 Point
	var ok bool
	if math.IsNaN(r0) {
		line := pivot.Add(n0)
		if cw {
			line = pivot.Sub(n0)
		}
		i0, i1, ok = circleLineIntersection(c1, R1, line, n0)
	} else if math.IsNaN(r1) {
		line := pivot.Add(n1)
		if cw {
			line = pivot.Sub(n1)
		}
		i0, i1, ok = circleLineIntersection(c0, R0, line, n1)
	} else {
		i0, i1, ok = circleCircleIntersection(c0, R0, c1, R1)
	}
	if !ok {
		// no intersection, default to bevel
		bevelJoiner(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}

	// find the closest intersection when following the arc (using either arc r0 or r1 with center c0 or c1 respectively)
	c, rcw := c0, r0 < 0.0
	if math.IsNaN(r0) {
		c, rcw = c1, r1 >= 0.0
	}
	thetaPivot := pivot.Sub(c).Radial()
	dtheta0 := i0.Sub(c).Radial() - thetaPivot
	dtheta1 := i1.Sub(c).Radial() - thetaPivot
	if rcw { // r runs clockwise, so look the other way around
		dtheta0 = -dtheta0
		dtheta1 = -dtheta1
	}
	mid := i0
	if angleNorm(dtheta1) < angleNorm(dtheta0) {
		mid = i1
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	if cw { // bend to the right, ie. CW
		rhs.LineTo(rEnd.X, rEnd.Y)
		if math.IsNaN(r0) {
			lhs.LineTo(mid.X, mid.Y)
		} else {
			lhs.ArcTo(R0, R0, 0.0, false, r0 > 0.0, mid.X, mid.Y)
		}
		if math.IsNaN(r1) {
			lhs.LineTo(lEnd.X, lEnd.Y)
		} else {
			lhs.ArcTo(R1, R1, 0.0, false, r1 > 0.0, lEnd.X, lEnd.Y)
		}
	} else { // bend to the left, ie. CCW
		if math.IsNaN(r0) {
			rhs.LineTo(mid.X, mid.Y)
		} else {
			rhs.ArcTo(R0, R0, 0.0, false, r0 > 0.0, mid.X, mid.Y)
		}
		if math.IsNaN(r1) {
			rhs.LineTo(rEnd.X, rEnd.Y)
		} else {
			rhs.ArcTo(R1, R1, 0.0, false, r1 > 0.0, rEnd.X, rEnd.Y)
		}
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

type pathState struct {
	cmd    float64
	p0, p1 Point   // position of start and end
	n0, n1 Point   // normal of start and end
	r0, r1 float64 // radius of start and end

	cp1, cp2                    Point   // Beziérs
	rx, ry, rot, theta0, theta1 float64 // arcs
	largeArc, sweep             bool    // arcs
}

// offsetSegment returns the rhs and lhs paths from offsetting a path segment
// it closes rhs and lhs when p is closed as well
func offsetSegment(p *Path, halfWidth float64, cr Capper, jr Joiner) (*Path, *Path) {
	closed := false
	states := []pathState{}
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
		case LineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			n := end.Sub(start).Rot90CW().Norm(halfWidth)
			states = append(states, pathState{
				cmd: LineToCmd,
				p0:  start,
				p1:  end,
				n0:  n,
				n1:  n,
				r0:  math.NaN(),
				r1:  math.NaN(),
			})
		case QuadToCmd, CubeToCmd:
			var cp1, cp2 Point
			if cmd == QuadToCmd {
				c := Point{p.d[i+1], p.d[i+2]}
				end = Point{p.d[i+3], p.d[i+4]}
				cp1, cp2 = quadraticToCubicBezier(start, c, end)
			} else {
				cp1 = Point{p.d[i+1], p.d[i+2]}
				cp2 = Point{p.d[i+3], p.d[i+4]}
				end = Point{p.d[i+5], p.d[i+6]}
			}
			n0 := cubicBezierNormal(start, cp1, cp2, end, 0.0, halfWidth)
			n1 := cubicBezierNormal(start, cp1, cp2, end, 1.0, halfWidth)
			r0 := cubicBezierRadius(start, cp1, cp2, end, 0.0)
			r1 := cubicBezierRadius(start, cp1, cp2, end, 1.0)
			states = append(states, pathState{
				cmd: CubeToCmd,
				p0:  start,
				p1:  end,
				n0:  n0,
				n1:  n1,
				r0:  r0,
				r1:  r1,
				cp1: cp1,
				cp2: cp2,
			})
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, largeArc, sweep, end.X, end.Y)
			n0 := ellipseNormal(phi, sweep, theta0, halfWidth)
			n1 := ellipseNormal(phi, sweep, theta1, halfWidth)
			r0 := ellipseRadius(rx, ry, phi, sweep, theta0)
			r1 := ellipseRadius(rx, ry, phi, sweep, theta1)
			states = append(states, pathState{
				cmd:      ArcToCmd,
				p0:       start,
				p1:       end,
				n0:       n0,
				n1:       n1,
				r0:       r0,
				r1:       r1,
				rx:       rx,
				ry:       ry,
				rot:      phi * 180.0 / math.Pi,
				theta0:   theta0,
				theta1:   theta1,
				largeArc: largeArc,
				sweep:    sweep,
			})
		case CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			if !equal(start.X, end.X) || !equal(start.Y, end.Y) {
				n := end.Sub(start).Rot90CW().Norm(halfWidth)
				states = append(states, pathState{
					cmd: CloseCmd,
					p0:  start,
					p1:  end,
					n0:  n,
					n1:  n,
					r0:  math.NaN(),
					r1:  math.NaN(),
				})
			}
			closed = true
		}
		start = end
		i += cmdLen(cmd)
	}
	if len(states) == 0 || len(states) == 1 && states[0].cmd == CloseCmd {
		return nil, nil
	}

	rhs, lhs := &Path{}, &Path{}
	rStart := states[0].p0.Add(states[0].n0)
	lStart := states[0].p0.Sub(states[0].n0)
	rhs.MoveTo(rStart.X, rStart.Y)
	lhs.MoveTo(lStart.X, lStart.Y)

	// TODO: fix if there is no space for Joiner when stroke is too thick
	for i, cur := range states {
		switch cur.cmd {
		case LineToCmd:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			rhs.LineTo(rEnd.X, rEnd.Y)
			lhs.LineTo(lEnd.X, lEnd.Y)
		case CubeToCmd:
			rhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, halfWidth, Tolerance))
			lhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, -halfWidth, Tolerance))
		case ArcToCmd:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			if !cur.sweep { // bend to the right, ie. CW
				rhs.ArcTo(cur.rx-halfWidth, cur.ry-halfWidth, cur.rot, cur.largeArc, cur.sweep, rEnd.X, rEnd.Y)
				lhs.ArcTo(cur.rx+halfWidth, cur.ry+halfWidth, cur.rot, cur.largeArc, cur.sweep, lEnd.X, lEnd.Y)
			} else { // bend to the left, ie. CCW
				rhs.ArcTo(cur.rx+halfWidth, cur.ry+halfWidth, cur.rot, cur.largeArc, cur.sweep, rEnd.X, rEnd.Y)
				lhs.ArcTo(cur.rx-halfWidth, cur.ry-halfWidth, cur.rot, cur.largeArc, cur.sweep, lEnd.X, lEnd.Y)
			}
		case CloseCmd:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			rhs.LineTo(rEnd.X, rEnd.Y)
			lhs.LineTo(lEnd.X, lEnd.Y)
		}

		if i+1 < len(states) || closed {
			var next pathState
			if i+1 < len(states) {
				next = states[i+1]
			} else {
				next = states[0]
			}
			jr.Join(rhs, lhs, halfWidth, cur.p1, cur.n1, next.n0, cur.r1, next.r0)
		}
	}
	lhs = lhs.Reverse()
	if closed {
		rhs.Close()
		lhs.Close()
		return rhs, lhs
	}
	cr.Cap(rhs, halfWidth, states[len(states)-1].p1, states[len(states)-1].n1)
	rhs.Join(lhs)
	cr.Cap(rhs, halfWidth, states[0].p0, states[0].n0.Neg())
	rhs.Close()
	return rhs, nil
}

// Offset offsets the path to expand by w. If w is negative it will contract (buggy).
func (p *Path) Offset(w float64) *Path {
	if w == 0.0 {
		return p
	}

	q := &Path{}
	expand := w > 0.0
	for _, ps := range p.Split() {
		if !ps.Closed() {
			continue
		}
		rhs, lhs := offsetSegment(p, w, ButtCapper, RoundJoiner)
		if rhs != nil { // lhs is also nil, as path is closed
			if expand == ps.CCW() {
				q.Append(rhs)
			} else {
				q.Append(lhs)
			}
		}
	}
	return q
}

// Stroke converts a path into a stroke of width w. It uses cr to cap the start and end of the path, and jr to
// join all path elemtents. If the path closes itself, it will use a join between the start and end instead of capping them.
// The tolerance is the maximum deviation from the original path when flattening Beziers and optimizing the stroke.
func (p *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	sp := &Path{}
	halfWidth := w / 2.0
	for _, ps := range p.Split() {
		rhs, lhs := offsetSegment(ps, halfWidth, cr, jr)
		if rhs != nil {
			sp.Append(rhs)
		}
		if lhs != nil {
			sp.Append(lhs)
		}
	}
	return sp
}
