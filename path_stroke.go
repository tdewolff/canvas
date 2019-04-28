package canvas

import "math"

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
	Join(*Path, *Path, float64, Point, Point, Point)
}

type JoinerFunc func(*Path, *Path, float64, Point, Point, Point)

func (f JoinerFunc) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	f(rhs, lhs, halfWidth, pivot, n0, n1)
}

// RoundJoiner connects two path elements by a round join.
var RoundJoiner Joiner = JoinerFunc(roundJoiner)

func roundJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	if equal(n0.X, n1.X) && equal(n0.Y, n1.Y) {
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

// BevelJoiner connects two path elements by a linear join.
var BevelJoiner Joiner = JoinerFunc(bevelJoiner)

func bevelJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	if equal(n0.X, n1.X) && equal(n0.Y, n1.Y) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

var MiterJoiner Joiner = JoinerFunc(bevelJoiner) // TODO
var ArcsJoiner Joiner = JoinerFunc(bevelJoiner)  // TODO

func strokeJoin(rhs, lhs *Path, jr Joiner, halfWidth float64, start, n1Prev, n0 Point, first *bool, n0First *Point) {
	if !*first {
		jr.Join(rhs, lhs, halfWidth, start, n1Prev, n0)
	} else {
		rStart := start.Add(n0)
		lStart := start.Sub(n0)
		rhs.MoveTo(rStart.X, rStart.Y)
		lhs.MoveTo(lStart.X, lStart.Y)
		*n0First = n0
		*first = false
	}
}

// Stroke converts a path into a stroke of width w. It uses cr to cap the start and end of the path, and jr to
// join all path elemtents. If the path closes itself, it will use a join between the start and end instead of capping them.
// The tolerance is the maximum deviation from the original path when flattening Beziers and optimizing the stroke.
func (p *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	sp := &Path{}
	halfWidth := w / 2.0
	for _, p = range p.Split() {
		ret := &Path{}
		first := true
		closed := false

		// n0 is the 'normal' at the beginning of a path command
		// n1 is the 'normal' at the end of a path command
		// Join and Cap are performed as we process the next path command
		//   Join joins from n1Prev to n0
		//   Cap caps from n1Prev

		var startFirst, start, end Point
		var n0First, n1Prev, n0, n1 Point
		for i := 0; i < len(p.d); {
			cmd := p.d[i]
			switch cmd {
			case MoveToCmd:
				end = Point{p.d[i+1], p.d[i+2]}
				startFirst = end
			case LineToCmd:
				end = Point{p.d[i+1], p.d[i+2]}
				n0 = end.Sub(start).Rot90CW().Norm(halfWidth)
				n1 = n0

				strokeJoin(sp, ret, jr, halfWidth, start, n1Prev, n0, &first, &n0First)

				rEnd := end.Add(n1)
				lEnd := end.Sub(n1)
				sp.LineTo(rEnd.X, rEnd.Y)
				ret.LineTo(lEnd.X, lEnd.Y)
			case QuadToCmd:
				c := Point{p.d[i+1], p.d[i+2]}
				end = Point{p.d[i+3], p.d[i+4]}
				c1 := start.Interpolate(c, 2.0/3.0)
				c2 := end.Interpolate(c, 2.0/3.0)
				n0 = cubicBezierNormal(start, c1, c2, end, 0.0).Norm(halfWidth)
				n1 = cubicBezierNormal(start, c1, c2, end, 1.0).Norm(halfWidth)

				strokeJoin(sp, ret, jr, halfWidth, start, n1Prev, n0, &first, &n0First)

				rhs := strokeCubicBezier(start, c1, c2, end, halfWidth, Tolerance)
				lhs := strokeCubicBezier(start, c1, c2, end, -halfWidth, Tolerance)
				sp.Join(rhs)
				ret.Join(lhs)
			case CubeToCmd:
				c1 := Point{p.d[i+1], p.d[i+2]}
				c2 := Point{p.d[i+3], p.d[i+4]}
				end = Point{p.d[i+5], p.d[i+6]}
				n0 = cubicBezierNormal(start, c1, c2, end, 0.0).Norm(halfWidth)
				n1 = cubicBezierNormal(start, c1, c2, end, 1.0).Norm(halfWidth)

				strokeJoin(sp, ret, jr, halfWidth, start, n1Prev, n0, &first, &n0First)

				rhs := strokeCubicBezier(start, c1, c2, end, halfWidth, Tolerance)
				lhs := strokeCubicBezier(start, c1, c2, end, -halfWidth, Tolerance)
				sp.Join(rhs)
				ret.Join(lhs)
			case ArcToCmd:
				rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
				largeAngle, sweep := fromArcFlags(p.d[i+4])
				end = Point{p.d[i+5], p.d[i+6]}
				_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, largeAngle, sweep, end.X, end.Y)
				n0 = ellipseNormal(theta0, phi).Norm(halfWidth)
				n1 = ellipseNormal(theta1, phi).Norm(halfWidth)
				if !sweep { // CW
					n0 = n0.Neg()
					n1 = n1.Neg()
				}

				strokeJoin(sp, ret, jr, halfWidth, start, n1Prev, n0, &first, &n0First)

				rEnd := end.Add(n1)
				lEnd := end.Sub(n1)
				if !sweep { // bend to the right, ie. CW
					sp.ArcTo(rx-halfWidth, ry-halfWidth, phi*180.0/math.Pi, largeAngle, sweep, rEnd.X, rEnd.Y)
					ret.ArcTo(rx+halfWidth, ry+halfWidth, phi*180.0/math.Pi, largeAngle, sweep, lEnd.X, lEnd.Y)
				} else { // bend to the left, ie. CCW
					sp.ArcTo(rx+halfWidth, ry+halfWidth, phi*180.0/math.Pi, largeAngle, sweep, rEnd.X, rEnd.Y)
					ret.ArcTo(rx-halfWidth, ry-halfWidth, phi*180.0/math.Pi, largeAngle, sweep, lEnd.X, lEnd.Y)
				}
			case CloseCmd:
				end = Point{p.d[i+1], p.d[i+2]}
				if !equal(start.X, end.X) || !equal(start.Y, end.Y) {
					n1 = end.Sub(start).Rot90CW().Norm(halfWidth)
					if !first {
						jr.Join(sp, ret, halfWidth, start, n1Prev, n1)
						rEnd := end.Add(n1)
						lEnd := end.Sub(n1)
						sp.LineTo(rEnd.X, rEnd.Y)
						ret.LineTo(lEnd.X, lEnd.Y)
					}

					rEnd := end.Add(n1)
					lEnd := end.Sub(n1)
					sp.LineTo(rEnd.X, rEnd.Y)
					ret.LineTo(lEnd.X, lEnd.Y)
				}
				closed = true
			}
			start = end
			n1Prev = n1
			i += cmdLen(cmd)
		}
		if first {
			continue
		}

		if !closed {
			cr.Cap(sp, halfWidth, start, n1Prev)
		} else {
			jr.Join(sp, ret, halfWidth, start, n1Prev, n0First)
			// close path and move to inverse path (which runs the other way around to negate the other)
			invStart := start.Sub(n0First)
			sp.Close()
			sp.MoveTo(invStart.X, invStart.Y)
		}
		sp.Join(ret.Reverse())
		if !closed {
			cr.Cap(sp, halfWidth, startFirst, n0First.Neg())
		}
		sp.Close()
	}
	return sp
}
