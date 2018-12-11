package canvas

import (
	"math"
	"strconv"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

const epsilon = 1e-10

func Equal(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func ftos(f float64) string {
	return strconv.FormatFloat(f, 'g', 5, 64)
}

////////////////////////////////////////////////////////////////

func toF32Vec(x, y float64) f32.Vec2 {
	return f32.Vec2{float32(x), float32(y)}
}

func toP26_6(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{toI26_6(x), toI26_6(y)}
}

func toI26_6(f float64) fixed.Int26_6 {
	return fixed.Int26_6(f * 64.0)
}

func fromI26_6(f fixed.Int26_6) float64 {
	return float64(f) / 64.0
}

func fromTTPoint(p truetype.Point) (Point, bool) {
	return Point{fromI26_6(p.X), -fromI26_6(p.Y)}, p.Flags&0x01 != 0
}

////////////////////////////////////////////////////////////////

type Point struct {
	X, Y float64
}

func (p Point) IsZero() bool {
	return Equal(p.X, 0.0) && Equal(p.Y, 0.0) // TODO: need Equal, or just compare?
}

func (p Point) Equals(q Point) bool {
	return Equal(p.X, q.X) && Equal(p.Y, q.Y)
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

func (p Point) Length() float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}

func (p Point) Norm(length float64) Point {
	d := p.Length()
	if Equal(d, 0.0) {
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
