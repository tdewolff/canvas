package canvas

import "math"

// NOTE: implementation mostly taken from github.com/golang/freetype/raster/stroke.go

type Point struct {
	X, Y float64
}

func (p Point) Neg() Point {
	return Point{-p.X, -p.Y}
}

func (p Point) Add(a Point) Point {
	return Point{p.X + a.X, p.Y + a.Y}
}

func (p Point) Sub(a Point) Point {
	return Point{p.X - a.X, p.Y - a.Y}
}

func (p Point) Rot90CW() Point {
	return Point{-p.Y, p.X}
}

func (p Point) Rot90CCW() Point {
	return Point{p.Y, -p.X}
}

func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

func (p Point) Norm(length float64) Point {
	d := math.Sqrt(p.X*p.X + p.Y*p.Y)
	if Equal(d, 0.0) {
		return Point{}
	}
	return Point{p.X / d * length, p.Y / d * length}
}

////////////////////////////////////////////////////////////////

// Capper implements Cap, with rhs the path to append to, pivot the pivot point around which to construct a cap,
// and n = (start-pivot) with start the start of the cap. The length of n is the half width of the stroke.
type Capper interface {
	Cap(*Path, float64, Point, Point)
}

type CapperFunc func(*Path, float64, Point, Point)

func (f CapperFunc) Cap(p *Path, halfWidth float64, pivot, n0 Point) {
	f(p, halfWidth, pivot, n0)
}

var RoundCapper Capper = CapperFunc(roundCapper)

func roundCapper(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.ArcTo(halfWidth, halfWidth, 0, false, false, end.X, end.Y)
}

var ButtCapper Capper = CapperFunc(buttCapper)

func buttCapper(p *Path, halfWidth float64, pivot, n0 Point) {
	end := pivot.Sub(n0)
	p.LineTo(end.X, end.Y)
}

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

type Joiner interface {
	Join(*Path, *Path, float64, Point, Point, Point)
}

type JoinerFunc func(*Path, *Path, float64, Point, Point, Point)

func (f JoinerFunc) Join(rhs, lhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	f(rhs, lhs, halfWidth, pivot, n0, n1)
}

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

var BevelJoiner Joiner = JoinerFunc(bevelJoiner)

func bevelJoiner(lhs, rhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	if Equal(n0.X, n1.X) && Equal(n0.Y, n1.Y) {
		return
	}

	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

////////////////

func strokeQuad(rhs, lhs *Path, halfWidth float64, p0, p1, p2, n0, n2 Point) {
	panic("not implemented")
	return
}

func strokeCube(rhs, lhs *Path, halfWidth float64, p0, p1, p2, p3, n0, n2 Point) {
	panic("not implemented")
	return
}

func (pMain *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	sp := &Path{}
	halfWidth := w / 2.0
	for _, p := range pMain.Split() {
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
				n0 = end.Sub(start).Norm(halfWidth).Rot90CW()
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
			case QuadToCmd:
				end = Point{p.d[i+2], p.d[i+3]}
				// p0 and p2 are the end points, p1 is the control point
				// a quadratic bezier curve will follow: B(t) = (1-t)^2*p0 + 2(1-t)t*p1 + t^2*p2
				// and its derivative: B'(t) = 2(1-t)(p1-p0) + 2t(p2-p1), from which we can derive the normals
				p0, p1, p2 := start, Point{p.d[i+0], p.d[i+1]}, end
				n0 = p1.Sub(p0).Rot90CW().Norm(halfWidth) // as we normalize, the factor 2 is irrelevant
				n1 = p2.Sub(p1).Rot90CW().Norm(halfWidth) // as we normalize, the factor 2 is irrelevant

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

				strokeQuad(sp, ret, halfWidth, p0, p1, p2, n0, n1)
				i += 4
			case CubeToCmd:
				end = Point{p.d[i+2], p.d[i+3]}
				// p0 and p2 are the end points, p1 is the control point
				// a quadratic bezier curve will follow: B(t) = (1-t)^2*p0 + 2(1-t)t*p1 + t^2*p2
				// and its derivative: B'(t) = 2(1-t)(p1-p0) + 2t(p2-p1), from which we can derive the normals
				p0, p1, p2, p3 := start, Point{p.d[i+0], p.d[i+1]}, Point{p.d[i+2], p.d[i+3]}, end
				n0 = p1.Sub(p0).Rot90CW().Norm(halfWidth) // as we normalize, the factor 2 is irrelevant
				n1 = p3.Sub(p2).Rot90CW().Norm(halfWidth) // as we normalize, the factor 2 is irrelevant

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

				strokeCube(sp, ret, halfWidth, p0, p1, p2, p3, n0, n1)
				i += 6
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
			case CloseCmd:
				end = Point{p.d[i+0], p.d[i+1]}
				if !Equal(start.X, end.X) || !Equal(start.Y, end.Y) {
					n1 = end.Sub(start).Norm(halfWidth).Rot90CW()
					if !first {
						jr.Join(sp, ret, halfWidth, start, n1Prev, n1)
						rEnd := end.Add(n1)
						lEnd := end.Sub(n1)
						sp.LineTo(rEnd.X, rEnd.Y)
						ret.LineTo(lEnd.X, lEnd.Y)
					}
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
			// butt cap close the stroke to each other
			invStart := start.Sub(n0First)
			sp.LineTo(invStart.X, invStart.Y)
		}
		sp.Append(ret.Invert())
		if !closed {
			cr.Cap(sp, halfWidth, startFirst, n0First.Neg())
		}
		sp.Close()
	}
	return sp
}
