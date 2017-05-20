package canvas

import (
	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

func toF32Vec(x, y float64) f32.Vec2 {
	return f32.Vec2{float32(x), float32(y)}
}

func toP26_6(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{toI26_6(x), toI26_6(y)}
}

func toI26_6(f float64) fixed.Int26_6 {
	return fixed.Int26_6(f * 64)
}

func fromI26_6(f fixed.Int26_6) float64 {
	return float64(f) / 64
}
