package canvas

import (
	"math"

	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

const Epsilon = 1e-10

// equal returns true if a and b are equal with tolerance Epsilon.
func equal(a, b float64) bool {
	return math.Abs(a-b) < Epsilon
}

// angleNorm returns the angle theta in the range [0,2PI).
func angleNorm(theta float64) float64 {
	theta = math.Mod(theta, 2.0*math.Pi)
	if theta < 0.0 {
		theta += 2.0 * math.Pi
	}
	return theta
}

// angleBetween is true when theta is in range (lower,upper) excluding the end points. Angles can be outside the [0,2PI) range.
func angleBetween(theta, lower, upper float64) bool {
	sweep := lower <= upper // true for CCW, ie along a positive angle
	theta = angleNorm(theta - lower)
	upper = angleNorm(upper - lower)
	if theta != 0.0 && (sweep && theta < upper || !sweep && theta > upper) {
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

// Point is a coordinate in 2D space. OP refers to the line that goes through the origin (0,0) and this point (x,y).
type Point struct {
	X, Y float64
}

// IsZero returns true if P is exactly zero.
func (p Point) IsZero() bool {
	return p.X == 0.0 && p.Y == 0.0
}

// Equals returns true if P and Q are equal with tolerance Epsilon.
func (p Point) Equals(q Point) bool {
	return equal(p.X, q.X) && equal(p.Y, q.Y)
}

// Neg negates x and y.
func (p Point) Neg() Point {
	return Point{-p.X, -p.Y}
}

// Add adds Q to P.
func (p Point) Add(q Point) Point {
	return Point{p.X + q.X, p.Y + q.Y}
}

// Sub subtracts Q from P.
func (p Point) Sub(q Point) Point {
	return Point{p.X - q.X, p.Y - q.Y}
}

// Mul multiplies x and y by f.
func (p Point) Mul(f float64) Point {
	return Point{f * p.X, f * p.Y}
}

// Div divides x and y by f.
func (p Point) Div(f float64) Point {
	return Point{p.X / f, p.Y / f}
}

// Rot90CW rotates the line OP by 90 degrees CW.
func (p Point) Rot90CW() Point {
	return Point{p.Y, -p.X}
}

// Rot90CCW rotates the line OP by 90 degrees CCW.
func (p Point) Rot90CCW() Point {
	return Point{-p.Y, p.X}
}

// Rot rotates the line OP by rot degrees CCW.
func (p Point) Rot(rot float64, p0 Point) Point {
	sinphi, cosphi := math.Sincos(rot * math.Pi / 180.0)
	return Point{
		p0.X + cosphi*(p.X-p0.X) - sinphi*(p.Y-p0.Y),
		p0.Y + sinphi*(p.X-p0.X) + cosphi*(p.Y-p0.Y),
	}
}

// Dot returns the dot product between OP and OQ, ie. zero if perpendicular and |OP|*|OQ| if aligned.
func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

// PerpDot returns the perp dot product between OP and OQ, ie. zero if aligned and |OP|*|OQ| if perpendicular.
func (p Point) PerpDot(q Point) float64 {
	return p.X*q.Y - p.Y*q.X
}

// Length returns the length of OP.
func (p Point) Length() float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}

// Slope returns the slope between OP, ie. y/x.
func (p Point) Slope() float64 {
	return p.Y / p.X
}

// Angle returns the angle between the x-axis and OP.
func (p Point) Angle() float64 {
	return math.Atan2(p.Y, p.X)
}

// AngleBetween returns the angle between OP and OQ.
func (p Point) AngleBetween(q Point) float64 {
	return math.Atan2(p.PerpDot(q), p.Dot(q))
}

// Norm normalized OP to be of certain length.
func (p Point) Norm(length float64) Point {
	d := p.Length()
	if equal(d, 0.0) {
		return Point{}
	}
	return Point{p.X / d * length, p.Y / d * length}
}

// Interpolate returns a point on PQ that is linearly interpolated by t, ie. t=0 returns P and t=1 returns Q.
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
