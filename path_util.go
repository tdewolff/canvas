package canvas

type Point struct {
	X, Y float64
}

func (p Point) Add(a Point) Point {
	return Point{p.X + a.X, p.Y + a.Y}
}

func (p Point) Rot90CW() Point {
	return Point{-p.Y, p.X}
}

func (p Point) Rot90CCW() Point {
	return Point{p.Y, -p.X}
}
