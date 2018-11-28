package canvas

// NOTE: implementation taken from github.com/golang/freetype/raster/stroke.go

type Capper interface {
	Cap(p *Path, halfWidth float64, pivot, normal Point)
}

type CapperFunc func(p *Path, halfWidth float64, pivot, normal Point)

func (f CapperFunc) Cap(p *Path, halfWidth float64, pivot, normal Point) {
	f(p, halfWidth, pivot, normal)
}

var RoundCapper Capper = CapperFunc(roundCapper)

func roundCapper(p *Path, halfWidth float64, pivot, normal Point) {
	end := pivot.Add(normal.Rot90CCW())
	p.ArcTo(halfWidth, halfWidth, 0, false, true, end.X, end.Y)
}

var ButtCapper Capper = CapperFunc(buttCapper)

func buttCapper(p *Path, halfWidth float64, pivot, normal Point) {
	end := pivot.Add(normal.Rot90CCW())
	p.LineTo(end.X, end.Y)
}

var SquareCapper Capper = CapperFunc(squareCapper)

func squareCapper(p *Path, halfWidth float64, pivot, normal Point) {
	corner1 := pivot.Add(normal).Add(normal.Rot90CW())
	corner2 := pivot.Add(normal).Add(normal.Rot90CCW())
	end := pivot.Add(normal.Rot90CCW())
	p.LineTo(corner1.X, corner1.Y)
	p.LineTo(corner2.X, corner2.Y)
	p.LineTo(end.X, end.Y)
}

////////////////

type Joiner interface {
	Join(lhs, rhs *Path, halfWidth float64, pivot, normal0, normal1 Point)
}

type JoinerFunc func(lhs, rhs *Path, halfWidth float64, pivot, normal0, normal1 Point)

func (f JoinerFunc) Join(lhs, rhs *Path, halfWidth float64, pivot, normal0, normal1 Point) {
	f(lhs, rhs, halfWidth, pivot, normal0, normal1)
}

var RoundJoiner Joiner = JoinerFunc(roundJoiner)

func roundJoiner(lhs, rhs *Path, halfWidth float64, pivot, normal0, normal1 Point) {
	dot := normal0.Rot90CW().Dot(normal1)
	if dot >= 0 { // bend to the right
		// lhs.ArcTo()
		r := pivot.Add(normal1.Rot90CCW())
		rhs.LineTo(r.X, r.Y)
	} else { // bend to the left
		r := pivot.Add(normal1.Rot90CCW())
		rhs.LineTo(r.X, r.Y)
	}
}

////////////////

func (p *Path) Stroke(w float64, cr Capper, jr Joiner) *Path {
	return p
}
