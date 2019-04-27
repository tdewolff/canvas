package canvas

import (
	"math"

	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

const epsilon = 1e-10

func equal(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func angleNorm(theta float64) float64 {
	theta = math.Mod(theta, 2.0*math.Pi)
	if theta < 0.0 {
		theta += 2.0 * math.Pi
	}
	return theta
}

func angleBetween(theta, lower, upper float64) bool {
	theta = angleNorm(theta)
	if lower > upper {
		lower, upper = upper, lower
	}
	// TODO: optimze
	if lower < theta && theta < upper ||
		lower < theta-2.0*math.Pi && theta-2.0*math.Pi < upper ||
		lower < theta+2.0*math.Pi && theta+2.0*math.Pi < upper {
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////

func toF32Vec(x, y float64) f32.Vec2 {
	return f32.Vec2{float32(x), float32(y)}
}

func toP26_6(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{toI26_6(x), toI26_6(y)}
}

func fromP26_6(f fixed.Point26_6) Point {
	return Point{float64(f.X) / 64.0, float64(f.Y) / 64.0}
}

func toI26_6(f float64) fixed.Int26_6 {
	return fixed.Int26_6(f * 64.0)
}

func fromI26_6(f fixed.Int26_6) float64 {
	return float64(f) / 64.0
}

////////////////////////////////////////////////////////////////

type Point struct {
	X, Y float64
}

func (p Point) IsZero() bool {
	return equal(p.X, 0.0) && equal(p.Y, 0.0) // TODO: need Equal, or just compare?, and rename to Zero()
}

func (p Point) Equals(q Point) bool {
	return equal(p.X, q.X) && equal(p.Y, q.Y)
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

func (p Point) Mul(f float64) Point {
	return Point{f * p.X, f * p.Y}
}

func (p Point) Rot90CW() Point {
	return Point{p.Y, -p.X}
}

func (p Point) Rot90CCW() Point {
	return Point{-p.Y, p.X}
}

func (p Point) Rot(rot float64, p0 Point) Point {
	sinphi, cosphi := math.Sincos(rot * math.Pi / 180.0)
	return Point{
		p0.X + cosphi*(p.X-p0.X) - sinphi*(p.Y-p0.Y),
		p0.Y + sinphi*(p.X-p0.X) + cosphi*(p.Y-p0.Y),
	}
}

func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

func (p Point) PerpDot(q Point) float64 {
	return p.X*q.Y - p.Y*q.X
}

func (p Point) Length() float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}

func (p Point) Angle(q Point) float64 {
	return math.Atan2(p.PerpDot(q), p.Dot(q))
}

func (p Point) Norm(length float64) Point {
	d := p.Length()
	if equal(d, 0.0) {
		return Point{}
	}
	return Point{p.X / d * length, p.Y / d * length}
}

func (p Point) Interpolate(q Point, t float64) Point {
	return Point{(1-t)*p.X + t*q.X, (1-t)*p.Y + t*q.Y}
}

////////////////////////////////////////////////////////////////

type Rect struct {
	X, Y, W, H float64
}

func (rect Rect) ToPath() *Path {
	return Rectangle(rect.X, rect.Y, rect.W, rect.H)
}
