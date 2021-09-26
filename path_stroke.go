package canvas

import (
	"math"
)

// NOTE: implementation inspired from github.com/golang/freetype/raster/stroke.go

// Capper implements Cap, with rhs the path to append to, halfWidth the half width of the stroke, pivot the pivot point around which to construct a cap, and n0 the normal at the start of the path. The length of n0 is equal to the halfWidth.
type Capper interface {
	Cap(*Path, float64, Point, Point)
}

// RoundCap caps the start or end of a path by a round cap.
var RoundCap Capper = RoundCapper{}

// RoundCapper is a round capper.
type RoundCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth, at a pivot point and initial normal direction of n0.
func (RoundCapper) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.ArcTo(halfWidth, halfWidth, 0, false, true, end.X, end.Y)
}

func (RoundCapper) String() string {
	return "Round"
}

// ButtCap caps the start or end of a path by a butt cap.
var ButtCap Capper = ButtCapper{}

// ButtCapper is a butt capper.
type ButtCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth, at a pivot point and initial normal direction of n0.
func (ButtCapper) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.LineTo(end.X, end.Y)
}

func (ButtCapper) String() string {
	return "Butt"
}

// SquareCap caps the start or end of a path by a square cap.
var SquareCap Capper = SquareCapper{}

// SquareCapper is a square capper.
type SquareCapper struct{}

// Cap adds a cap to path p of width 2*halfWidth, at a pivot point and initial normal direction of n0.
func (SquareCapper) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	e := n0.Rot90CCW()
	corner1 := pivot.Add(e).Add(n0)
	corner2 := pivot.Add(e).Sub(n0)
	end := pivot.Sub(n0)
	p.LineTo(corner1.X, corner1.Y)
	p.LineTo(corner2.X, corner2.Y)
	p.LineTo(end.X, end.Y)
}

func (SquareCapper) String() string {
	return "Square"
}

////////////////

// Joiner implements Join, with rhs the right path and lhs the left path to append to, pivot the intersection of both path elements, n0 and n1 the normals at the start and end of the path respectively. The length of n0 and n1 are equal to the halfWidth.
type Joiner interface {
	Join(*Path, *Path, float64, Point, Point, Point, float64, float64)
}

// BevelJoin connects two path elements by a linear join.
var BevelJoin Joiner = BevelJoiner{}

// BevelJoiner is a bevel joiner.
type BevelJoiner struct{}

// Join adds a join to a right-hand-side and left-hand-side path, of width 2*halfWidth, around a pivot point with starting and ending normals of n0 and n1, and radius of curvatures of the previous and next segments.
func (BevelJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

func (BevelJoiner) String() string {
	return "Bevel"
}

// RoundJoin connects two path elements by a round join.
var RoundJoin Joiner = RoundJoiner{}

// RoundJoiner is a round joiner.
type RoundJoiner struct{}

// Join adds a join to a right-hand-side and left-hand-side path, of width 2*halfWidth, around a pivot point with starting and ending normals of n0 and n1, and radius of curvatures of the previous and next segments.
func (RoundJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	cw := n0.Rot90CW().Dot(n1) >= 0.0
	if cw { // bend to the right, ie. CW (or 180 degree turn)
		rhs.LineTo(rEnd.X, rEnd.Y)
		lhs.ArcTo(halfWidth, halfWidth, 0.0, false, false, lEnd.X, lEnd.Y)
	} else { // bend to the left, ie. CCW
		rhs.ArcTo(halfWidth, halfWidth, 0.0, false, true, rEnd.X, rEnd.Y)
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

func (RoundJoiner) String() string {
	return "Round"
}

// MiterJoin connects two path elements by extending the ends of the paths as lines until they meet. If this point is further than 2 mm * (strokeWidth / 2.0) away, this will result in a bevel join.
var MiterJoin Joiner = MiterJoiner{BevelJoin, 2.0}

// MiterClipJoin returns a MiterJoiner with given limit*strokeWidth/2.0 in mm upon which the gapJoiner function will be used. Limit can be NaN so that the gapJoiner is never used.
func MiterClipJoin(gapJoiner Joiner, limit float64) Joiner {
	return MiterJoiner{gapJoiner, limit}
}

// MiterJoiner is a miter joiner.
type MiterJoiner struct {
	GapJoiner Joiner
	Limit     float64
}

// Join adds a join to a right-hand-side and left-hand-side path, of width 2*halfWidth, around a pivot point with starting and ending normals of n0 and n1, and radius of curvatures of the previous and next segments.
func (j MiterJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1.Neg()) {
		BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}
	limit := math.Max(j.Limit, 1.001) // otherwise nearly linear joins will also get clipped

	cw := n0.Rot90CW().Dot(n1) >= 0.0
	hw := halfWidth
	if cw {
		hw = -hw // used to calculate |R|, when running CW then n0 and n1 point the other way, so the sign of r0 and r1 is negated
	}

	theta := n0.AngleBetween(n1) / 2.0
	d := hw / math.Cos(theta)
	if !math.IsNaN(limit) && limit*halfWidth < math.Abs(d) {
		j.GapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
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

func (j MiterJoiner) String() string {
	if math.IsNaN(j.Limit) {
		return "Miter"
	}
	return "MiterClip"
}

// ArcsJoin connects two path elements by extending the ends of the paths as circle arcs until they meet. If this point is further than 10 mm * (strokeWidth / 2.0) away, this will result in a bevel join.
var ArcsJoin Joiner = ArcsJoiner{BevelJoin, 10.0}

// ArcsClipJoin returns an ArcsJoiner with given limit in mm*strokeWidth/2.0 upon which the gapJoiner function will be used. Limit can be NaN so that the gapJoiner is never used.
func ArcsClipJoin(gapJoiner Joiner, limit float64) Joiner {
	return ArcsJoiner{gapJoiner, limit}
}

// ArcsJoiner is an arcs joiner.
type ArcsJoiner struct {
	GapJoiner Joiner
	Limit     float64
}

// Join adds a join to a right-hand-side and left-hand-side path, of width 2*halfWidth, around a pivot point with starting and ending normals of n0 and n1, and radius of curvatures of the previous and next segments.
func (j ArcsJoiner) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point, r0, r1 float64) {
	if n0.Equals(n1.Neg()) {
		BevelJoin.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	} else if math.IsNaN(r0) && math.IsNaN(r1) {
		MiterJoiner(j).Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
		return
	}
	limit := math.Max(j.Limit, 1.001) // 1.001 so that nearly linear joins will not get clipped

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
		j.GapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
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

	if !math.IsNaN(limit) && limit*halfWidth < mid.Sub(pivot).Length() {
		j.GapJoiner.Join(rhs, lhs, halfWidth, pivot, n0, n1, r0, r1)
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

func (j ArcsJoiner) String() string {
	if math.IsNaN(j.Limit) {
		return "Arcs"
	}
	return "ArcsClip"
}

type pathStrokeState struct {
	cmd    float64
	p0, p1 Point   // position of start and end
	n0, n1 Point   // normal of start and end
	r0, r1 float64 // radius of start and end

	cp1, cp2                    Point   // Béziers
	rx, ry, rot, theta0, theta1 float64 // arcs
	large, sweep                bool    // arcs
}

// offsetSegment returns the rhs and lhs paths from offsetting a path segment. It closes rhs and lhs when p is closed as well.
func offsetSegment(p *Path, halfWidth float64, cr Capper, jr Joiner) (*Path, *Path) {
	// only non-empty paths are evaluated
	closed := false
	states := []pathStrokeState{}
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
		case LineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			n := end.Sub(start).Rot90CW().Norm(halfWidth)
			states = append(states, pathStrokeState{
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
			states = append(states, pathStrokeState{
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
			large, sweep := toArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			n0 := ellipseNormal(rx, ry, phi, sweep, theta0, halfWidth)
			n1 := ellipseNormal(rx, ry, phi, sweep, theta1, halfWidth)
			r0 := ellipseCurvatureRadius(rx, ry, sweep, theta0)
			r1 := ellipseCurvatureRadius(rx, ry, sweep, theta1)
			states = append(states, pathStrokeState{
				cmd:    ArcToCmd,
				p0:     start,
				p1:     end,
				n0:     n0,
				n1:     n1,
				r0:     r0,
				r1:     r1,
				rx:     rx,
				ry:     ry,
				rot:    phi * 180.0 / math.Pi,
				theta0: theta0,
				theta1: theta1,
				large:  large,
				sweep:  sweep,
			})
		case CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			if !Equal(start.X, end.X) || !Equal(start.Y, end.Y) {
				n := end.Sub(start).Rot90CW().Norm(halfWidth)
				states = append(states, pathStrokeState{
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

	rhs, lhs := &Path{}, &Path{}
	rStart := states[0].p0.Add(states[0].n0)
	lStart := states[0].p0.Sub(states[0].n0)
	rhs.MoveTo(rStart.X, rStart.Y)
	lhs.MoveTo(lStart.X, lStart.Y)

	rhsInnerBends := []int{}
	lhsInnerBends := []int{}
	for i, cur := range states {
		switch cur.cmd {
		case LineToCmd:
			rEnd := cur.p1.Add(cur.n1)
			lEnd := cur.p1.Sub(cur.n1)
			rhs.LineTo(rEnd.X, rEnd.Y)
			lhs.LineTo(lEnd.X, lEnd.Y)
		case CubeToCmd:
			rhs = rhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, halfWidth, Tolerance))
			lhs = lhs.Join(strokeCubicBezier(cur.p0, cur.cp1, cur.cp2, cur.p1, -halfWidth, Tolerance))
		case ArcToCmd:
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
			rhs.ArcTo(rLambda*(cur.rx+dr), rLambda*(cur.ry+dr), cur.rot, cur.large, cur.sweep, rEnd.X, rEnd.Y)
			lhs.ArcTo(lLambda*(cur.rx-dr), lLambda*(cur.ry-dr), cur.rot, cur.large, cur.sweep, lEnd.X, lEnd.Y)
		}

		// join the cur and next path segments
		if i+1 < len(states) || closed {
			var next pathStrokeState
			if i+1 < len(states) {
				next = states[i+1]
			} else {
				next = states[0]
			}

			if !cur.n1.Equals(next.n0) {
				jr.Join(rhs, lhs, halfWidth, cur.p1, cur.n1, next.n0, cur.r1, next.r0)

				if !cur.n1.Equals(next.n0.Neg()) {
					// all turns except 0 degrees and 180 degrees are added
					cw := cur.n1.Rot90CW().Dot(next.n0) >= 0.0
					if cw {
						rhsInnerBends = append(rhsInnerBends, len(rhs.d)-cmdLen(LineToCmd))
					} else {
						lhsInnerBends = append(lhsInnerBends, len(lhs.d)-cmdLen(LineToCmd))
					}
				}
			}
		}
	}

	closeInnerBends(rhs, rhsInnerBends, closed)
	closeInnerBends(lhs, lhsInnerBends, closed)

	if closed {
		rhs.Close()
		lhs.Close()
		optimizeMoveTo(rhs)
		optimizeMoveTo(lhs)
		return rhs, lhs
	}

	// default to CCW direction
	lhs = lhs.Reverse()
	cr.Cap(rhs, halfWidth, states[len(states)-1].p1, states[len(states)-1].n1)
	rhs = rhs.Join(lhs)
	cr.Cap(rhs, halfWidth, states[0].p0, states[0].n0.Neg())
	rhs.Close()
	optimizeMoveTo(rhs)
	return rhs, nil
}

func closeInnerBends(p *Path, indices []int, closed bool) {
	// closed paths end with a LineTo to the original MoveTo but are not (yet) closed
	di := 0
	for _, i := range indices {
		i -= di
		cmd := p.d[i]
		iPrev := i - cmdLen(p.d[i-1])
		iNext := i + cmdLen(cmd)
		if closed && iNext == len(p.d) {
			iNext = cmdLen(MoveToCmd)
		}
		if 0 < iPrev && iNext < len(p.d) {
			// TODO: (stroke) implement inner bend optimization for all combinations
			// TODO: (stroke) if segments do not cross keep looking, what if while looking we pass another index in indices? Remove all?
			prevStart := Point{p.d[iPrev-3], p.d[iPrev-2]}
			prevEnd := Point{p.d[i-3], p.d[i-2]}
			nextStart := Point{p.d[i+1], p.d[i+2]}
			nextEnd := Point{p.d[iNext+1], p.d[iNext+2]}

			if p.d[iPrev] == LineToCmd && p.d[iNext] == LineToCmd {
				zs := intersectionLineLine(prevStart, prevEnd, nextStart, nextEnd)
				if zs.HasSecant() {
					p.d[i-3] = zs[0].X
					p.d[i-2] = zs[0].Y
					p.d = append(p.d[:i:i], p.d[i+cmdLen(cmd):]...)
					di += cmdLen(cmd)
				}
			} else if p.d[iPrev] == LineToCmd && p.d[iNext] == ArcToCmd {
			} else if p.d[iPrev] == ArcToCmd && p.d[iNext] == LineToCmd {
			} else if p.d[iPrev] == ArcToCmd && p.d[iNext] == ArcToCmd {
			}
		}
	}

	if closed {
		// update MoveTo to match the last LineTo (which will be a Close)
		p.d[1] = p.d[len(p.d)-3]
		p.d[2] = p.d[len(p.d)-2]
	}
}

func optimizeMoveTo(p *Path) {
	// move MoveTo to the initial position of the Close if they are colinear
	if p.d[cmdLen(MoveToCmd)] == LineToCmd && p.d[len(p.d)-cmdLen(CloseCmd)-1] == LineToCmd {
		start := Point{p.d[len(p.d)-cmdLen(CloseCmd)-3], p.d[len(p.d)-cmdLen(CloseCmd)-2]}
		mid := Point{p.d[1], p.d[2]}
		end := Point{p.d[cmdLen(MoveToCmd)+1], p.d[cmdLen(MoveToCmd)+2]}
		if Equal(end.Sub(mid).AngleBetween(mid.Sub(start)), 0.0) {
			p.d[1] = p.d[len(p.d)-cmdLen(CloseCmd)-3]
			p.d[2] = p.d[len(p.d)-cmdLen(CloseCmd)-2]
			p.d[len(p.d)-cmdLen(CloseCmd)-4] = CloseCmd
			p.d[len(p.d)-cmdLen(CloseCmd)-1] = CloseCmd
			p.d = p.d[:len(p.d)-cmdLen(CloseCmd)]
		}
	}
}

// Offset offsets the path to expand by w and returns a new path. If w is negative it will contract. Path must be closed.
func (p *Path) Offset(w float64, fillRule FillRule) *Path {
	if Equal(w, 0.0) {
		return p
	}

	q := &Path{}
	filling := p.Filling(fillRule)
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
		if filling[i] {
			useRHS = !useRHS
		}

		rhs, lhs := offsetSegment(ps, math.Abs(w), ButtCap, RoundJoin)
		if useRHS {
			q = q.Append(rhs)
		} else {
			q = q.Append(lhs)
		}
	}
	return q
}

// Stroke converts a path into a stroke of width w and returns a new path. It uses cr to cap the start and end of the path, and jr to join all path elements. If the path closes itself, it will use a join between the start and end instead of capping them. The tolerance is the maximum deviation from the original path when flattening Béziers and optimizing the stroke.
func (p *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	// TODO: start first point at intersection between last and first segment. This allows a rectangle to have a stroke with twice 1xM, 3xL and one z command, just like a rectangle itself.
	q := &Path{}
	halfWidth := w / 2.0
	for _, ps := range p.Split() {
		rhs, lhs := offsetSegment(ps, halfWidth, cr, jr)
		if lhs != nil { // closed path
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
		} else {
			q = q.Append(rhs)
		}
	}
	return q
}
