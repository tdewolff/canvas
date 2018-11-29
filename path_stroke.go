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
	p.ArcTo(halfWidth, halfWidth, 0, false, true, end.X, end.Y)
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
	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)

	right := n0.Rot90CW().Dot(n1) >= 0
	largeAngle := n0.Dot(n1) < 0
	if right { // bend to the right
		rhs.LineTo(rEnd.X, rEnd.Y)
		lhs.ArcTo(halfWidth, halfWidth, 0, largeAngle, false, lEnd.X, lEnd.Y) // CW
	} else { // bend to the left
		rhs.ArcTo(halfWidth, halfWidth, 0, largeAngle, true, rEnd.X, rEnd.Y) // CCW
		lhs.LineTo(lEnd.X, lEnd.Y)
	}
}

var BevelJoiner Joiner = JoinerFunc(bevelJoiner)

func bevelJoiner(lhs, rhs *Path, halfWidth float64, pivot, n0, n1 Point) {
	rEnd := pivot.Add(n1)
	lEnd := pivot.Sub(n1)
	rhs.LineTo(rEnd.X, rEnd.Y)
	lhs.LineTo(lEnd.X, lEnd.Y)
}

////////////////

func (pMain *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	sp := &Path{}
	halfWidth := w / 2.0
	for _, p := range pMain.Split() {
		ret := &Path{}
		first := true
		closed := false

		i := 0
		var start0, start, end Point
		var nFirst, nPrev, n Point
		for _, cmd := range p.cmds {
			switch cmd {
			case MoveToCmd:
				end = Point{p.d[i+0], p.d[i+1]}
				start0 = end
				i += 2
			case LineToCmd:
				end = Point{p.d[i+0], p.d[i+1]}
				n = end.Sub(start).Norm(halfWidth).Rot90CW()

				if first {
					rStart := start.Add(n)
					lStart := start.Sub(n)
					sp.MoveTo(rStart.X, rStart.Y)
					ret.MoveTo(lStart.X, lStart.Y)
					nFirst = n
					first = false
				} else {
					jr.Join(sp, ret, halfWidth, start, nPrev, n)
				}

				rEnd := end.Add(n)
				lEnd := end.Sub(n)
				sp.LineTo(rEnd.X, rEnd.Y)
				ret.LineTo(lEnd.X, lEnd.Y)
				i += 2
			case QuadToCmd:
				panic("not implemented")
				i += 4
			case CubeToCmd:
				panic("not implemented")
				i += 6
			case ArcToCmd:
				panic("not implemented")
				i += 7
			case CloseCmd:
				// end = Point{p.d[i+0], p.d[i+1]}
				closed = true
				i += 2
			}
			start = end
			nPrev = n
		}
		if !closed {
			cr.Cap(sp, halfWidth, start, nPrev)
		} else {
			// handle
		}
		sp.Append(ret.Invert())
		if !closed {
			cr.Cap(sp, halfWidth, start0, nFirst.Neg())
		}
	}
	return sp
}
