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
	Join(*Path, *Path, float64, Point, Point, Point, float64, float64) bool
}

type JoinerFunc func(*Path, *Path, float64, Point, Point, Point, float64, float64) bool

func (f JoinerFunc) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) bool {
	return f(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
}

// BevelJoiner connects two path elements by a linear join.
var BevelJoiner Joiner = JoinerFunc(bevelJoiner)

func bevelJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) bool {
	if n0.Equals(n1) {
		return false
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
	return true
}

// RoundJoiner connects two path elements by a round join.
var RoundJoiner Joiner = JoinerFunc(roundJoiner)

func roundJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) bool {
	if n0.Equals(n1) {
		return false
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	cw := n0.Rot90CW().Dot(n1) >= 0.0
	if cw { // bend to the right, ie. CW
		rhs.LineTo(rEnd.X, rEnd.Y)
		lhs.ArcTo(halfWidth, halfWidth, 0.0, false, false, lEnd.X, lEnd.Y)
	} else { // bend to the left, ie. CCW
		rhs.ArcTo(halfWidth, halfWidth, 0.0, false, true, rEnd.X, rEnd.Y)
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
	return true
}

var MiterJoiner Joiner = miterJoiner{BevelJoiner, math.NaN()}

func MiterClipJoiner(gapJoiner Joiner, limit float64) Joiner {
	return miterJoiner{gapJoiner, limit}
}

type miterJoiner struct {
	gapJoiner Joiner
	limit     float64
}

func (j miterJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) bool {
	if n0.Equals(n1) {
		return false
	} else if n0.Equals(n1.Neg()) {
		return bevelJoiner(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
	}
	limit := math.Max(j.limit, halfWidth*1.001) // otherwise nearly linear joins will also get clipped

	cw := n0.Rot90CW().Dot(n1) >= 0.0
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	theta := n0.AngleBetween(n1) / 2.0
	d := hw / math.Cos(theta)
	if !math.IsNaN(j.limit) && math.Abs(d) > limit {
		return j.gapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
	}
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
	return true
}

var ArcsJoiner Joiner = arcsJoiner{BevelJoiner, math.NaN()}

func ArcsClipJoiner(gapJoiner Joiner, limit float64) Joiner {
	return arcsJoiner{gapJoiner, limit}
}

type arcsJoiner struct {
	gapJoiner Joiner
	limit     float64
}

func (j arcsJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) bool {
	if n0.Equals(n1) {
		return false
	} else if n0.Equals(n1.Neg()) {
		return bevelJoiner(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
	} else if math.IsNaN(r0) && math.IsNaN(r1) {
		return miterJoiner{j.gapJoiner, j.limit}.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
	}
	limit := math.Max(j.limit, halfWidth*1.001) // otherwise nearly linear joins will also get clipped

	cw := n0.Rot90CW().Dot(n1) >= 0.0
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	// r is the radius of the original curve, R the radius of the stroke curve, c are the centers of the circles
	c0 := pivot.Add(n0.Norm(-r0))
	c1 := pivot.Add(n1.Norm(-r1))
	R0, R1 := math.Abs(r0+hw), math.Abs(r1+hw)

	// TODO: can simplify if intersection returns angles too?
	var i0, i1 Point
	var ok bool
	if math.IsNaN(r0) {
		line := pivot.Add(n0)
		if cw {
			line = pivot.Sub(n0)
		}
		i0, i1, ok = intersectionRayCircle(line, line.Add(n0.Rot90CCW()), c1, R1)
	} else if math.IsNaN(r1) {
		line := pivot.Add(n1)
		if cw {
			line = pivot.Sub(n1)
		}
		i0, i1, ok = intersectionRayCircle(line, line.Add(n1.Rot90CCW()), c0, R0)
	} else {
		i0, i1, ok = intersectionCircleCircle(c0, R0, c1, R1)
	}
	if !ok {
		// no intersection
		return j.gapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
	}

	// find the closest intersection when following the arc (using either arc r0 or r1 with center c0 or c1 respectively)
	c, rcw := c0, r0 < 0.0
	if math.IsNaN(r0) {
		c, rcw = c1, r1 >= 0.0
	}
	thetaPivot := pivot.Sub(c).Angle()
	dtheta0 := i0.Sub(c).Angle() - thetaPivot
	dtheta1 := i1.Sub(c).Angle() - thetaPivot
	if rcw { // r runs clockwise, so look the other way around
		dtheta0 = -dtheta0
		dtheta1 = -dtheta1
	}
	mid := i0
	if angleNorm(dtheta1) < angleNorm(dtheta0) {
		mid = i1
	}

	if !math.IsNaN(limit) && mid.Sub(pivot).Length() > limit {
		return j.gapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
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
	return true
}

type pathState struct {
	cmd    float64
	size   int     // number of commands it ends up being (ie. cubic bezier becomes many linears)
	p0, p1 Point   // position of start and end
	n0, n1 Point   // normal of start and end
	r0, r1 float64 // radius of start and end

	cp1, cp2                    Point   // Béziers
	rx, ry, rot, theta0, theta1 float64 // arcs
	largeArc, sweep             bool    // arcs
}

var NOINTER bool

func strokeIntersect(p *Path, prev, next pathState, nPrev, nNext int, closer bool) {
	if NOINTER {
		return
	}

	prevCmd := prev.cmd
	nextCmd := next.cmd
	if prevCmd == CubeToCmd {
		prevCmd = LineToCmd
	}
	if nextCmd == CubeToCmd {
		nextCmd = LineToCmd
	}

	n := len(p.d)
	var iNext, iPrev, iPrevStart int
	if !closer {
		iNext = n - nNext
		iPrevStart = iNext - 3 - nPrev // inside bend has extra linear segment
	} else {
		iNext = 3                  // after first MoveTo
		iPrevStart = n - 3 - nPrev // inside bend has extra linear segment
	}
	iPrev = iPrevStart + nPrev - cmdLen(prevCmd) // if previous path consists of multiple (line) commands, start with the last

	startPrev := Point{p.d[iPrev-2], p.d[iPrev-1]}
	endPrev := Point{p.d[iPrev+1], p.d[iPrev+2]}
	startNext := Point{p.d[iNext-2], p.d[iNext-1]}
	endNext := Point{p.d[iNext+1], p.d[iNext+2]}

	success := false
	if prevCmd == LineToCmd && nextCmd == LineToCmd {
		if i, ok := intersectionLineLine(startPrev, endPrev, startNext, endNext); ok {
			p.d[iPrev+1] = i.X
			p.d[iPrev+2] = i.Y
			success = true
		}
	}

	if success {
		p.d = append(p.d[:iPrev+cmdLen(prevCmd)], p.d[iPrev+cmdLen(prevCmd)+3:]...) // remove extra linear segment
		if closer {
			p.d[1] = p.d[iPrev+1] // set first MoveTo to the last coordinates
			p.d[2] = p.d[iPrev+2]
		}
	}
}

// offsetSegment returns the rhs and lhs paths from offsetting a path segment
// it closes rhs and lhs when p is closed as well
func offsetSegment(p *Path, halfWidth float64, cr Capper, jr Joiner) (*Path, *Path) {
	p = p.Copy().Replace(nil, nil, flattenEllipse) // TODO: remove when path intersection supports elliptic arcs

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
				cp := Point{p.d[i+1], p.d[i+2]}
				end = Point{p.d[i+3], p.d[i+4]}
				cp1, cp2 = quadraticToCubicBezier(start, cp, end)
			} else {
				cp1 = Point{p.d[i+1], p.d[i+2]}
				cp2 = Point{p.d[i+3], p.d[i+4]}
				end = Point{p.d[i+5], p.d[i+6]}
			}
			n0 := cubicBezierNormal(start, cp1, cp2, end, 0.0, halfWidth)
			n1 := cubicBezierNormal(start, cp1, cp2, end, 1.0, halfWidth)
			r0 := cubicBezierCurvatureRadius(start, cp1, cp2, end, 0.0)
			r1 := cubicBezierCurvatureRadius(start, cp1, cp2, end, 1.0)
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
			n0 := ellipseNormal(rx, ry, phi, sweep, theta0, halfWidth)
			n1 := ellipseNormal(rx, ry, phi, sweep, theta1, halfWidth)
			r0 := ellipseCurvatureRadius(rx, ry, phi, sweep, theta0)
			r1 := ellipseCurvatureRadius(rx, ry, phi, sweep, theta1)
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
					cmd: LineToCmd,
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
	joined := false
	nRHS, nLHS, nRHSPrev, nLHSPrev, nRHSFirst, nLHSFirst := 0, 0, 0, 0, 0, 0
	var prev pathState
	for i, cur := range states {
		nRHS, nLHS = len(rhs.d), len(lhs.d)
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
		}
		nRHS, nLHS = len(rhs.d)-nRHS, len(lhs.d)-nLHS

		// join the prev and cur path segments on the inside of the bend
		if 0 < i && joined {
			ccw := prev.n1.Rot90CCW().Dot(cur.n0) > 0.0
			if ccw {
				strokeIntersect(lhs, prev, cur, nLHSPrev, nLHS, false)
			} else {
				strokeIntersect(rhs, prev, cur, nRHSPrev, nRHS, false)
			}
		}

		// join the cur and next path segments on the outside of the bend
		if i+1 < len(states) || closed {
			var next pathState
			if i+1 < len(states) {
				next = states[i+1]
			} else {
				next = states[0]
			}
			joined = jr.Join(rhs, lhs, halfWidth, cur.p1, cur.n1, next.n0, cur.r1, next.r0)
		}

		if i == 0 {
			nRHSFirst, nLHSFirst = nRHS, nLHS
		}
		nRHSPrev, nLHSPrev = nRHS, nLHS
		prev = cur
	}
	if closed && joined {
		cur := states[0]
		ccw := prev.n1.Rot90CCW().Dot(states[0].n0) > 0.0
		if ccw {
			strokeIntersect(lhs, prev, cur, nLHSPrev, nLHSFirst, true)
		} else {
			strokeIntersect(rhs, prev, cur, nRHSPrev, nRHSFirst, true)
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

// Offset offsets the path to expand by w. If w is negative it will contract (TODO: buggy).
func (p *Path) Offset(w float64) *Path {
	if w == 0.0 {
		return p
	}

	q := &Path{}
	//expand := w > 0.0
	for _, ps := range p.Split() {
		if !ps.Closed() {
			continue
		}
		// TODO: fix choosing rhs or lhs as we don't know if the path inverts an earlier one
		_, lhs := offsetSegment(ps, w, ButtCapper, RoundJoiner)
		if lhs != nil { // lhs is also nil, as path is closed
			q.Append(lhs)
		}
	}
	return q
}

// Stroke converts a path into a stroke of width w. It uses cr to cap the start and end of the path, and jr to
// join all path elemtents. If the path closes itself, it will use a join between the start and end instead of capping them.
// The tolerance is the maximum deviation from the original path when flattening Béziers and optimizing the stroke.
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
