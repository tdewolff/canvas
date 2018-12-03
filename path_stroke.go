package canvas

import "fmt"

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
	p.ArcTo(halfWidth, halfWidth, 0, false, false, end.X, end.Y)
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
	if Equal(n0.X, n1.X) && Equal(n0.Y, n1.Y) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)

	cw := n0.Rot90CW().Dot(n1) >= 0
	if cw { // bend to the right, ie. CW
		rhs.LineTo(rEnd.X, rEnd.Y)
		lhs.ArcTo(halfWidth, halfWidth, 0, false, true, lEnd.X, lEnd.Y)
	} else { // bend to the left, ie. CCW
		rhs.ArcTo(halfWidth, halfWidth, 0, false, false, rEnd.X, rEnd.Y)
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

// BevelJoiner connects two path elements by a linear join.
var BevelJoiner Joiner = JoinerFunc(bevelJoiner)

func bevelJoiner(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	if Equal(n0.X, n1.X) && Equal(n0.Y, n1.Y) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

// Stroke will convert a path into a stroke of width w. It uses cr to cap the start and end of the path, and jr to
// join all path elemtents. If the path closes itself, it will use a join between the start and end instead of capping them.
// The tolerance is the maximum deviation from the original path when flattening Beziers and optimizing the stroke.
func (pWhole *Path) Stroke(w float64, cr Capper, jr Joiner, tolerance float64) *Path {
	sp := &Path{}
	halfWidth := w / 2.0
	for _, p := range pWhole.Split() {
		ret := &Path{}
		first := true
		closed := false

		// n0 is the 'normal' at the beginning of a path command
		// n1 is the 'normal' at the end of a path command
		// Join and Cap are performed as we process the next path command
		//   Join joins from n1Prev to n0
		//   Cap caps from n1Prev

		i := 0
		var startFirst, start, end Point
		var n0First, n1Prev, n0, n1 Point
		for _, cmd := range p.cmds {
			switch cmd {
			case MoveToCmd:
				end = Point{p.d[i+0], p.d[i+1]}
				startFirst = end
				i += 2
			case LineToCmd:
				end = Point{p.d[i+0], p.d[i+1]}
				n0 = end.Sub(start).Rot90CW().Norm(halfWidth)
				n1 = n0

				if !first {
					jr.Join(sp, ret, halfWidth, start, n1Prev, n0)
				} else {
					rStart := start.Add(n0)
					lStart := start.Sub(n0)
					sp.MoveTo(rStart.X, rStart.Y)
					ret.MoveTo(lStart.X, lStart.Y)
					n0First = n0
					first = false
				}

				rEnd := end.Add(n1)
				lEnd := end.Sub(n1)
				sp.LineTo(rEnd.X, rEnd.Y)
				ret.LineTo(lEnd.X, lEnd.Y)
				i += 2
			case ArcToCmd:
				rx, ry := p.d[i+0], p.d[i+1]
				rot, largeAngle, sweep := p.d[i+2], p.d[i+3] == 1.0, p.d[i+4] == 1.0
				end = Point{p.d[i+5], p.d[i+6]}
				_, _, angle0, angle1 := arcToCenter(start.X, start.Y, rx, ry, rot, largeAngle, sweep, end.X, end.Y)
				n0 = angleToNormal(angle0).Norm(halfWidth)
				n1 = angleToNormal(angle1).Norm(halfWidth)
				if sweep { // CW
					n0 = n0.Neg()
					n1 = n1.Neg()
				}

				if !first {
					jr.Join(sp, ret, halfWidth, start, n1Prev, n0)
				} else {
					rStart := start.Add(n0)
					lStart := start.Sub(n0)
					sp.MoveTo(rStart.X, rStart.Y)
					ret.MoveTo(lStart.X, lStart.Y)
					n0First = n0
					first = false
				}

				rEnd := end.Add(n1)
				lEnd := end.Sub(n1)
				if sweep { // bend to the right, ie. CW
					sp.ArcTo(rx-halfWidth, ry-halfWidth, rot, largeAngle, sweep, rEnd.X, rEnd.Y)
					ret.ArcTo(rx+halfWidth, ry+halfWidth, rot, largeAngle, sweep, lEnd.X, lEnd.Y)
				} else { // bend to the left, ie. CCW
					sp.ArcTo(rx+halfWidth, ry+halfWidth, rot, largeAngle, sweep, rEnd.X, rEnd.Y)
					ret.ArcTo(rx-halfWidth, ry-halfWidth, rot, largeAngle, sweep, lEnd.X, lEnd.Y)
				}
				i += 7
			case QuadToCmd:
				c := Point{p.d[i+0], p.d[i+1]}
				end := Point{p.d[i+2], p.d[i+3]}
				c1 := start.Interpolate(c, 2.0/3.0)
				c2 := end.Interpolate(c, 2.0/3.0)
				n0 := cubicBezierNormal(start, c1, c2, end, 0.0).Norm(halfWidth)

				if !first {
					jr.Join(sp, ret, halfWidth, start, n1Prev, n0)
				} else {
					rStart := start.Add(n0)
					lStart := start.Sub(n0)
					sp.MoveTo(rStart.X, rStart.Y)
					ret.MoveTo(lStart.X, lStart.Y)
					n0First = n0
					first = false
				}

				qr := flattenCubicBezier(start, c1, c2, end, halfWidth, tolerance)
				ql := flattenCubicBezier(start, c1, c2, end, -halfWidth, tolerance)
				sp.Append(qr)
				ret.Append(ql)
				i += 4
			case CubeToCmd:
				c1 := Point{p.d[i+0], p.d[i+1]}
				c2 := Point{p.d[i+2], p.d[i+3]}
				end := Point{p.d[i+4], p.d[i+5]}
				n0 := cubicBezierNormal(start, c1, c2, end, 0.0).Norm(halfWidth)

				if !first {
					jr.Join(sp, ret, halfWidth, start, n1Prev, n0)
				} else {
					rStart := start.Add(n0)
					lStart := start.Sub(n0)
					sp.MoveTo(rStart.X, rStart.Y)
					ret.MoveTo(lStart.X, lStart.Y)
					n0First = n0
					first = false
				}

				qr := flattenCubicBezier(start, c1, c2, end, halfWidth, tolerance)
				ql := flattenCubicBezier(start, c1, c2, end, -halfWidth, tolerance)
				sp.Append(qr)
				ret.Append(ql)
				fmt.Println(qr, ql)
				i += 6
			case CloseCmd:
				end = Point{p.d[i+0], p.d[i+1]}
				if !Equal(start.X, end.X) || !Equal(start.Y, end.Y) {
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
				i += 2
			}
			start = end
			n1Prev = n1
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
		sp.Append(ret.Reverse())
		if !closed {
			cr.Cap(sp, halfWidth, startFirst, n0First.Neg())
		}
		sp.Close()
	}
	return sp
}
