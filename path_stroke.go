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

// RoundCapper caps the start or end of a path by a round cap.
var RoundCapper Capper = roundCapper{}

type roundCapper struct{}

func (roundCapper) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.ArcTo(halfWidth, halfWidth, 0, false, true, end.X, end.Y)
}

func (roundCapper) String() string {
	return "Round"
}

// ButtCapper caps the start or end of a path by a butt cap.
var ButtCapper Capper = buttCapper{}

type buttCapper struct{}

func (buttCapper) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.LineTo(end.X, end.Y)
}

func (buttCapper) String() string {
	return "Butt"
}

// SquareCapper caps the start or end of a path by a square cap.
var SquareCapper Capper = squareCapper{}

type squareCapper struct{}

func (squareCapper) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	e := n0.Rot90CCW()
	corner1 := pivot.Add(e).Add(n0)
	corner2 := pivot.Add(e).Sub(n0)
	end := pivot.Sub(n0)
	p.LineTo(corner1.X, corner1.Y)
	p.LineTo(corner2.X, corner2.Y)
	p.LineTo(end.X, end.Y)
}

func (squareCapper) String() string {
	return "Square"
}

////////////////

// Joiner implements Join, with rhs the right path and lhs the left path to append to, pivot the intersection of both
// path elements, n0 and n1 the normals at the start and end of the path respectively.
// The length of n0 and n1 are equal to the halfWidth.
type Joiner interface {
	Join(*Path, *Path, float64, Point, Point, Point, float64, float64)
}

// BevelJoiner connects two path elements by a linear join.
var BevelJoiner Joiner = bevelJoiner{}

type bevelJoiner struct{}

func (bevelJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

func (bevelJoiner) String() string {
	return "Bevel"
}

// RoundJoiner connects two path elements by a round join.
var RoundJoiner Joiner = roundJoiner{}

type roundJoiner struct{}

func (roundJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
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
}

func (roundJoiner) String() string {
	return "Round"
}

// MiterJoiner connects two path elements by extending the ends of the paths as lines until they meet. If this point is further than 4 mm away, this will result in a bevel join.
var MiterJoiner Joiner = miterJoiner{BevelJoiner, 4.0} // TODO: use miter-clip from SVG, ie. relative to stroke width. make sure SVG and PDF implementations use the right limits.

// MiterClipJoiner returns a MiterJoiner with given limit in mm upon which the gapJoiner function will be used. Limit can be NaN so that the gapJoiner is never used.
func MiterClipJoiner(gapJoiner Joiner, limit float64) Joiner {
	return miterJoiner{gapJoiner, limit}
}

type miterJoiner struct {
	gapJoiner Joiner
	limit     float64
}

func (j miterJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	} else if n0.Equals(n1.Neg()) {
		BevelJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}
	limit := math.Max(j.limit/2.0, halfWidth*1.001) // otherwise nearly linear joins will also get clipped

	cw := n0.Rot90CW().Dot(n1) >= 0.0
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	theta := n0.AngleBetween(n1) / 2.0
	d := hw / math.Cos(theta)
	if !math.IsNaN(limit) && math.Abs(d) > limit {
		j.gapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
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
}

func (j miterJoiner) String() string {
	if math.IsNaN(j.limit) {
		return "Miter"
	}
	return "MiterClip"
}

// ArcsJoiner connects two path elements by extending the ends of the paths as circle arcs until they meet. If this point is further than 4 mm away, this will result in a bevel join.
var ArcsJoiner Joiner = arcsJoiner{BevelJoiner, 4.0}

// ArcsClipJoiner returns an ArcsJoiner with given limit in mm upon which the gapJoiner function will be used. Limit can be NaN so that the gapJoiner is never used.
func ArcsClipJoiner(gapJoiner Joiner, limit float64) Joiner {
	return arcsJoiner{gapJoiner, limit}
}

type arcsJoiner struct {
	gapJoiner Joiner
	limit     float64
}

func (j arcsJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1) {
		return
	} else if n0.Equals(n1.Neg()) {
		BevelJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	} else if math.IsNaN(r0) && math.IsNaN(r1) {
		miterJoiner{j.gapJoiner, j.limit}.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}
	limit := math.Max(j.limit/2.0, halfWidth*1.001) // 1.001 so that nearly linear joins will not get clipped

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
		j.gapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
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
		j.gapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
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

func (j arcsJoiner) String() string {
	if math.IsNaN(j.limit) {
		return "Arcs"
	}
	return "ArcsClip"
}

type pathStrokeState struct {
	cmd    float64
	size   int     // number of commands it ends up being (ie. cubic bezier becomes many linears)
	p0, p1 Point   // position of start and end
	n0, n1 Point   // normal of start and end
	r0, r1 float64 // radius of start and end

	cp1, cp2                    Point   // Béziers
	rx, ry, rot, theta0, theta1 float64 // arcs
	largeArc, sweep             bool    // arcs
}

// offsetSegment returns the rhs and lhs paths from offsetting a path segment.
// It closes rhs and lhs when p is closed as well.
func offsetSegment(p *Path, halfWidth float64, cr Capper, jr Joiner) (*Path, *Path) {
	closed := false
	states := []pathStrokeState{}
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case moveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
		case lineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			n := end.Sub(start).Rot90CW().Norm(halfWidth)
			states = append(states, pathStrokeState{
				cmd: lineToCmd,
				p0:  start,
				p1:  end,
				n0:  n,
				n1:  n,
				r0:  math.NaN(),
				r1:  math.NaN(),
			})
		case quadToCmd, cubeToCmd:
			var cp1, cp2 Point
			if cmd == quadToCmd {
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
			states = append(states, pathStrokeState{
				cmd: cubeToCmd,
				p0:  start,
				p1:  end,
				n0:  n0,
				n1:  n1,
				r0:  r0,
				r1:  r1,
				cp1: cp1,
				cp2: cp2,
			})
		case arcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, largeArc, sweep, end.X, end.Y)
			n0 := ellipseNormal(rx, ry, phi, sweep, theta0, halfWidth)
			n1 := ellipseNormal(rx, ry, phi, sweep, theta1, halfWidth)
			r0 := ellipseCurvatureRadius(rx, ry, sweep, theta0)
			r1 := ellipseCurvatureRadius(rx, ry, sweep, theta1)
			states = append(states, pathStrokeState{
				cmd:      arcToCmd,
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
		case closeCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			if !equal(start.X, end.X) || !equal(start.Y, end.Y) {
				n := end.Sub(start).Rot90CW().Norm(halfWidth)
				states = append(states, pathStrokeState{
					cmd: lineToCmd,
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
	if len(states) == 0 || len(states) == 1 && states[0].cmd == closeCmd {
		return nil, nil
	}

	rhs, lhs := &Path{}, &Path{}
	rStart := states[0].p0.Add(states[0].n0)
	lStart := states[0].p0.Sub(states[0].n0)
	rhs.MoveTo(rStart.X, rStart.Y)
	lhs.MoveTo(lStart.X, lStart.Y)

	for i, cur := range states {
		switch cur.cmd {
		case lineToCmd:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			rhs.LineTo(rEnd.X, rEnd.Y)
			lhs.LineTo(lEnd.X, lEnd.Y)
		case cubeToCmd:
			rhs = rhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, halfWidth, Tolerance))
			lhs = lhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, -halfWidth, Tolerance))
		case arcToCmd:
			rStart := cur.p0.Add(cur.n0)
			lStart := cur.p0.Sub(cur.n0)
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			dr := halfWidth
			if !cur.sweep { // bend to the right, ie. CW
				dr = -dr
			}

			rLambda := ellipseRadiiCorrection(rStart, cur.rx+dr, cur.ry+dr, cur.rot*math.Pi/180.0, rEnd)
			lLambda := ellipseRadiiCorrection(lStart, cur.rx-dr, cur.ry-dr, cur.rot*math.Pi/180.0, lEnd)
			if rLambda <= 1.0 && lLambda <= 1.0 {
				rLambda, lLambda = 1.0, 1.0
			}
			rhs.ArcTo(rLambda*(cur.rx+dr), rLambda*(cur.ry+dr), cur.rot, cur.largeArc, cur.sweep, rEnd.X, rEnd.Y)
			lhs.ArcTo(lLambda*(cur.rx-dr), lLambda*(cur.ry-dr), cur.rot, cur.largeArc, cur.sweep, lEnd.X, lEnd.Y)
		}

		// join the cur and next path segments
		if i+1 < len(states) || closed {
			var next pathStrokeState
			if i+1 < len(states) {
				next = states[i+1]
			} else {
				next = states[0]
			}
			jr.Join(rhs, lhs, halfWidth, cur.p1, cur.n1, next.n0, cur.r1, next.r0)
		}
	}

	// TODO: split at intersections and filter out overlapped parts

	if closed {
		rhs.Close()
		lhs.Close()
		return rhs, lhs
	}

	// default to CCW direction
	lhs = lhs.Reverse()
	cr.Cap(rhs, halfWidth, states[len(states)-1].p1, states[len(states)-1].n1)
	rhs = rhs.Join(lhs)
	cr.Cap(rhs, halfWidth, states[0].p0, states[0].n0.Neg())
	rhs.Close()
	return rhs, nil
}

// Offset offsets the path to expand by w and returns a new path. If w is negative it will contract. Path must be closed.
func (p *Path) Offset(w float64) *Path {
	if equal(w, 0.0) {
		return p
	}

	q := &Path{}
	fillings := p.Filling()
	for i, ps := range p.Split() {
		if !ps.Closed() {
			continue
		}

		useRHS := false
		if ps.CCW() {
			useRHS = !useRHS
		}
		if w > 0.0 {
			useRHS = !useRHS
		}
		if fillings[i] {
			useRHS = !useRHS
		}

		rhs, lhs := offsetSegment(ps, math.Abs(w), ButtCapper, RoundJoiner)
		if useRHS {
			if rhs != nil {
				q = q.Append(rhs)
			}
		} else {
			if lhs != nil {
				q = q.Append(lhs)
			}
		}
	}
	return q
}

// Stroke converts a path into a stroke of width w and returns a new path. It uses cr to cap the start and end of the path, and
// jr to join all path elemtents. If the path closes itself, it will use a join between the start and end instead of capping them.
// The tolerance is the maximum deviation from the original path when flattening Béziers and optimizing the stroke.
func (p *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	q := &Path{}
	halfWidth := w / 2.0
	for _, ps := range p.Split() {
		rhs, lhs := offsetSegment(ps, halfWidth, cr, jr)
		if rhs != nil && lhs != nil { // closed path
			// inner path should go opposite direction to cancel the outer path
			if ps.CCW() {
				lhs = lhs.Reverse()
				q = q.Append(rhs)
				q = q.Append(lhs)
			} else {
				rhs = rhs.Reverse()
				q = q.Append(lhs)
				q = q.Append(rhs)
			}
		} else if rhs != nil {
			q = q.Append(rhs)
		} else {
			q = q.Append(lhs)
		}
	}
	return q
}
