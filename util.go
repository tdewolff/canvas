package canvas

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

func ToF32Vec(x, y float64) f32.Vec2 {
	return f32.Vec2{float32(x), float32(y)}
}

func ToP26_6(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{ToI26_6(x), ToI26_6(y)}
}

func ToI26_6(f float64) fixed.Int26_6 {
	return fixed.Int26_6(f * 64.0)
}

func FromI26_6(f fixed.Int26_6) float64 {
	return float64(f) / 64.0
}

func FromTTPoint(p truetype.Point) (Point, bool) {
	return Point{FromI26_6(p.X), -FromI26_6(p.Y)}, p.Flags&0x01 != 0
}
