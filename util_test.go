package canvas

import (
	"image/color"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestAngleNorm(t *testing.T) {
	test.Float(t, angleNorm(0.0), 0.0)
	test.Float(t, angleNorm(1.0*math.Pi), 1.0*math.Pi)
	test.Float(t, angleNorm(2.0*math.Pi), 0.0)
	test.Float(t, angleNorm(3.0*math.Pi), 1.0*math.Pi)
	test.Float(t, angleNorm(-1.0*math.Pi), 1.0*math.Pi)
	test.Float(t, angleNorm(-2.0*math.Pi), 0.0)
}

func TestAngleBetween(t *testing.T) {
	test.T(t, angleBetween(0.0, 0.0, 1.0), false)
	test.T(t, angleBetween(1.0, 0.0, 1.0), false)
	test.T(t, angleBetween(0.5, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5+2.0*math.Pi, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5, 0.0+2.0*math.Pi, 1.0+2.0*math.Pi), true)
	test.T(t, angleBetween(0.5, 1.0+2.0*math.Pi, 0.0+2.0*math.Pi), true)
	test.T(t, angleBetween(0.5-2.0*math.Pi, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5, 0.0-2.0*math.Pi, 1.0-2.0*math.Pi), true)
	test.T(t, angleBetween(0.5, 1.0-2.0*math.Pi, 0.0-2.0*math.Pi), true)
}

func TestCSSColor(t *testing.T) {
	test.String(t, toCSSColor(Cyan), "#0ff")
	test.String(t, toCSSColor(Aliceblue), "#f0f8ff")
	test.String(t, toCSSColor(color.RGBA{255, 255, 255, 0}), "rgba(0,0,0,0)")
	test.String(t, toCSSColor(color.RGBA{85, 85, 17, 85}), "rgba(255,255,51,0.33333)")
}

func TestPoint(t *testing.T) {
	Epsilon = 0.01
	p := Point{3, 4}
	test.T(t, p.Mul(2.0), Point{6, 8})
	test.T(t, p.Div(3.0), Point{1, 1.33})
	test.T(t, p.Rot90CW(), Point{4, -3})
	test.T(t, p.Rot90CCW(), Point{-4, 3})
	test.T(t, p.Rot(90*math.Pi/180.0, Point{}), p.Rot90CCW())
	test.T(t, p.Rot(90*math.Pi/180.0, p), p)
	test.Float(t, p.Dot(Point{3, 0}), 9.0)
	test.Float(t, p.PerpDot(Point{3, 0}), p.Rot90CCW().Dot(Point{3, 0}))
	test.Float(t, p.Length(), 5.0)
	test.Float(t, p.Slope(), 1.333333)
	test.Float(t, p.Angle(), 53.130095*math.Pi/180.0)
	test.Float(t, p.AngleBetween(p.Rot90CCW()), 90.0*math.Pi/180.0)
	test.T(t, p.Norm(3.0), Point{1.8, 2.4})
	test.T(t, p.Norm(0.0), Point{0.0, 0.0})
	test.T(t, Point{}.Norm(1.0), Point{0.0, 0.0})
	test.T(t, Point{}.Interpolate(p, 0.5), Point{1.5, 2.0})
	test.String(t, p.String(), "(3,4)")
}

func TestRect(t *testing.T) {
	Epsilon = 0.01
	r := Rect{0, 0, 5, 5}
	test.T(t, r.Move(Point{3, 3}), Rect{3, 3, 5, 5})
	test.T(t, r.Add(Rect{5, 5, 5, 5}), Rect{0, 0, 10, 10})
	test.T(t, r.Add(Rect{5, 5, 0, 5}), r)
	test.T(t, Rect{5, 5, 0, 5}.Add(r), r)
	test.T(t, r.Transform(Identity.Rotate(90)), Rect{-5, 0, 5, 5})
	test.T(t, r.Transform(Identity.Rotate(45)), Rect{-3.53, 0.0, 7.07, 7.07})
	test.T(t, r.ToPath(), MustParseSVG("M0,0H5V5H0z"))
	test.String(t, r.String(), "(0,0)-(5,5)")
}

func TestMatrix(t *testing.T) {
	Epsilon = 0.01
	p := Point{3, 4}
	test.T(t, Identity.Translate(2.0, 2.0).Dot(p), Point{5.0, 6.0})
	test.T(t, Identity.Scale(2.0, 2.0).Dot(p), Point{6.0, 8.0})
	test.T(t, Identity.Scale(1.0, -1.0).Dot(p), Point{3.0, -4.0})
	test.T(t, Identity.Shear(1.0, 0.0).Dot(p), Point{7.0, 4.0})
	test.T(t, Identity.Rotate(90.0).Dot(p), p.Rot90CCW())
	test.T(t, Identity.RotateAt(90.0, 5.0, 5.0).Dot(p), p.Rot(90.0*math.Pi/180.0, Point{5.0, 5.0}))
	test.T(t, Identity.ReflectX().Dot(p), Point{-3.0, 4.0})
	test.T(t, Identity.ReflectY().Dot(p), Point{3.0, -4.0})
	test.T(t, Identity.ReflectXAt(1.5).Dot(p), Point{0.0, 4.0})
	test.T(t, Identity.ReflectYAt(2.0).Dot(p), Point{3.0, 0.0})
	test.T(t, Identity.Rotate(90.0).T().Dot(p), p.Rot90CW())
	test.T(t, Identity.Scale(2.0, 4.0).Inv(), Identity.Scale(0.5, 0.25))
	test.T(t, Identity.Rotate(90.0).Inv(), Identity.Rotate(-90.0))
	test.T(t, Identity.Rotate(90.0).Scale(2.0, 1.0), Identity.Scale(1.0, 2.0).Rotate(90.0))

	lambda1, lambda2, v1, v2 := Identity.Rotate(-90.0).Scale(2.0, 1.0).Rotate(90.0).Eigen()
	test.Float(t, lambda1, 1.0)
	test.Float(t, lambda2, 2.0)
	test.T(t, v1, Point{1.0, 0.0})
	test.T(t, v2, Point{0.0, 1.0})

	lambda1, lambda2, v1, v2 = Identity.Shear(1.0, 1.0).Eigen()
	test.Float(t, lambda1, 0.0)
	test.Float(t, lambda2, 2.0)
	test.T(t, v1, Point{-0.707, 0.707})
	test.T(t, v2, Point{0.707, 0.707})

	lambda1, lambda2, v1, v2 = Identity.Shear(1.0, 0.0).Eigen()
	test.Float(t, lambda1, 1.0)
	test.Float(t, lambda2, 1.0)
	test.T(t, v1, Point{1.0, 0.0})
	test.T(t, v2, Point{1.0, 0.0})

	lambda1, lambda2, v1, v2 = Identity.Scale(math.NaN(), math.NaN()).Eigen()
	test.Float(t, lambda1, math.NaN())
	test.Float(t, lambda2, math.NaN())
	test.T(t, v1, Point{0.0, 0.0})
	test.T(t, v2, Point{0.0, 0.0})

	tx, ty, theta, sx, sy, phi := Identity.Rotate(-90.0).Scale(2.0, 1.0).Rotate(90.0).Translate(0.0, 10.0).Decompose()
	test.Float(t, tx, 0.0)
	test.Float(t, ty, 20.0)
	test.Float(t, theta, 90.0)
	test.Float(t, sx, 2.0)
	test.Float(t, sy, 1.0)
	test.Float(t, phi, -90.0)

	test.T(t, Identity.Translate(1.0, 1.0).IsRigid(), true)
	test.T(t, Identity.Rotate(90.0).IsRigid(), true)
	test.T(t, Identity.Scale(2.0, 1.0).IsRigid(), false)
	test.T(t, Identity.Scale(-1.0, 1.0).IsRigid(), false)
	test.T(t, Identity.Shear(2.0, -1.0).IsRigid(), false)
	test.T(t, Identity.Translate(1.0, 1.0).IsTranslation(), true)
	test.T(t, Identity.Rotate(90.0).IsTranslation(), false)

	x, y := Identity.Translate(p.X, p.Y).Pos()
	test.Float(t, x, p.X)
	test.Float(t, y, p.Y)

	test.String(t, Identity.Shear(2.0, 3.0).String(), "(1 2; 3 1) + (0,0)")

	test.T(t, Identity.Shear(1.0, 1.0), Identity.Rotate(45).Scale(2.0, 0.0).Rotate(-45))
	test.String(t, Identity.Shear(1.0, 1.0).ToSVG(10.0), "rotate(-45) scale(2,0) rotate(45)")
	test.String(t, Identity.Rotate(45).Scale(2.0, 0.0).Rotate(-45).ToSVG(10.0), "rotate(-45) scale(2,0) rotate(45)")
}

func TestSolveQuadraticFormula(t *testing.T) {
	x1, x2 := solveQuadraticFormula(0.0, 0.0, 0.0)
	test.Float(t, x1, 0.0)
	test.Float(t, x2, math.NaN())

	x1, x2 = solveQuadraticFormula(0.0, 0.0, 1.0)
	test.Float(t, x1, math.NaN())
	test.Float(t, x2, math.NaN())

	x1, x2 = solveQuadraticFormula(0.0, 1.0, 1.0)
	test.Float(t, x1, -1.0)
	test.Float(t, x2, math.NaN())

	x1, x2 = solveQuadraticFormula(1.0, 1.0, 0.0)
	test.Float(t, x1, 0.0)
	test.Float(t, x2, -1.0)

	x1, x2 = solveQuadraticFormula(1.0, 1.0, 1.0) // discriminant negative
	test.Float(t, x1, math.NaN())
	test.Float(t, x2, math.NaN())

	x1, x2 = solveQuadraticFormula(1.0, 1.0, 0.25) // discriminant zero
	test.Float(t, x1, -0.5)
	test.Float(t, x2, math.NaN())

	x1, x2 = solveQuadraticFormula(2.0, -5.0, 2.0) // negative b, flip x1 and x2
	test.Float(t, x1, 0.5)
	test.Float(t, x2, 2.0)
}

func TestGaussLegendre(t *testing.T) {
	test.Float(t, gaussLegendre3(math.Log, 0.0, 1.0), -0.947672)
	test.Float(t, gaussLegendre5(math.Log, 0.0, 1.0), -0.979001)
	test.Float(t, gaussLegendre7(math.Log, 0.0, 1.0), -0.988738)
}
