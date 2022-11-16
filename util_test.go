package canvas

import (
	"fmt"
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
	test.T(t, angleBetween(0.0, 0.0, 1.0), true)
	test.T(t, angleBetween(1.0, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5+2.0*math.Pi, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5, 0.0+2.0*math.Pi, 1.0+2.0*math.Pi), true)
	test.T(t, angleBetween(0.5, 1.0+2.0*math.Pi, 0.0+2.0*math.Pi), true)
	test.T(t, angleBetween(0.5-2.0*math.Pi, 0.0, 1.0), true)
	test.T(t, angleBetween(0.5, 0.0-2.0*math.Pi, 1.0-2.0*math.Pi), true)
	test.T(t, angleBetween(0.5, 1.0-2.0*math.Pi, 0.0-2.0*math.Pi), true)
	test.T(t, angleBetween(-0.1, 0.0, 1.0), false)
	test.T(t, angleBetween(1.1, 0.0, 1.0), false)
}

func TestCSSColor(t *testing.T) {
	test.String(t, CSSColor(Cyan).String(), "#0ff")
	test.String(t, CSSColor(Aliceblue).String(), "#f0f8ff")
	test.String(t, CSSColor(color.RGBA{255, 255, 255, 0}).String(), "rgba(0,0,0,0)")
	test.String(t, CSSColor(color.RGBA{85, 85, 17, 85}).String(), "rgba(255,255,51,.33333333)")
}

func TestToFromFixed(t *testing.T) {
	test.T(t, fromP26_6(toP26_6(Point{3.0, 5.0})), Point{3.0, 5.0})
	test.Float(t, fromI26_6(toI26_6(7.0)), 7.0)
}

func TestPoint(t *testing.T) {
	defer setEpsilon(1e-6)()

	p := Point{3, 4}
	test.T(t, p.Mul(2.0), Point{6, 8})
	test.T(t, p.Div(3.0), Point{1, 1.333333})
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
	test.Float(t, Point{0, 0}.AngleBetween(p), 0.0)
	test.Float(t, p.AngleBetween(Point{0, 0}), 0.0)
	test.Float(t, p.AngleBetween(p), 0.0)
	test.T(t, p.Norm(3.0), Point{1.8, 2.4})
	test.T(t, p.Norm(0.0), Point{0.0, 0.0})
	test.T(t, Point{}.Norm(1.0), Point{0.0, 0.0})
	test.T(t, Point{}.Interpolate(p, 0.5), Point{1.5, 2.0})
	test.String(t, p.String(), "(3,4)")
}

func TestRect(t *testing.T) {
	defer setEpsilon(1e-6)()

	r := Rect{0, 0, 5, 5}
	test.T(t, r.Move(Point{3, 3}), Rect{3, 3, 5, 5})
	test.T(t, r.Add(Rect{5, 5, 5, 5}), Rect{0, 0, 10, 10})
	test.T(t, r.Add(Rect{5, 5, 0, 5}), r)
	test.T(t, Rect{5, 5, 0, 5}.Add(r), r)
	test.T(t, r.AddPoint(Point{10, 10}), Rect{0, 0, 10, 10})
	test.T(t, r.AddPoint(Point{-10, -10}), Rect{-10, -10, 15, 15})
	test.T(t, r.Transform(Identity.Rotate(90)), Rect{-5, 0, 5, 5})
	test.T(t, r.Transform(Identity.Rotate(45)), Rect{-3.535533, 0.0, 7.071068, 7.071068})
	test.T(t, r.Contains(Point{1, 1}), true)
	test.T(t, r.Contains(Point{6, 6}), false)
	test.T(t, r.Contains(Point{-1, 1}), false)
	test.T(t, r.Overlaps(Rect{0, 5, 5, 5}), false)
	test.T(t, r.Overlaps(Rect{0, -5, 5, 5}), false)
	test.T(t, r.Overlaps(Rect{-5, 0, 5, 5}), false)
	test.T(t, r.Overlaps(Rect{5, 0, 5, 5}), false)
	test.T(t, r.Overlaps(Rect{4, 0, 5, 5}), true)
	test.T(t, r.Overlaps(Rect{0, 0, 5, 5}), true)
	test.T(t, r.Overlaps(Rect{-1, -1, 7, 7}), true)
	test.T(t, r.Overlaps(Rect{1, 1, 3, 3}), true)
	test.T(t, r.ToPath(), MustParseSVG("M0,0H5V5H0z"))
	test.String(t, r.String(), "(0,0)-(5,5)")
}

func TestMatrix(t *testing.T) {
	defer setEpsilon(1e-6)()

	p := Point{3, 4}
	test.T(t, Identity.Translate(2.0, 2.0).Dot(p), Point{5.0, 6.0})
	test.T(t, Identity.Scale(2.0, 2.0).Dot(p), Point{6.0, 8.0})
	test.T(t, Identity.Scale(1.0, -1.0).Dot(p), Point{3.0, -4.0})
	test.T(t, Identity.ScaleAbout(2.0, -1.0, 2.0, 2.0).Dot(p), Point{4.0, 0.0})
	test.T(t, Identity.Shear(1.0, 0.0).Dot(p), Point{7.0, 4.0})
	test.T(t, Identity.ShearAbout(1.0, 0.0, 2.0, 2.0).Dot(p), Point{5.0, 4.0})
	test.T(t, Identity.Rotate(90.0).Dot(p), p.Rot90CCW())
	test.T(t, Identity.RotateAbout(90.0, 5.0, 5.0).Dot(p), p.Rot(90.0*math.Pi/180.0, Point{5.0, 5.0}))
	test.T(t, Identity.ReflectX().Dot(p), Point{-3.0, 4.0})
	test.T(t, Identity.ReflectY().Dot(p), Point{3.0, -4.0})
	test.T(t, Identity.ReflectXAbout(1.5).Dot(p), Point{0.0, 4.0})
	test.T(t, Identity.ReflectYAbout(2.0).Dot(p), Point{3.0, 0.0})
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
	test.T(t, v1, Point{-0.707107, 0.707107})
	test.T(t, v2, Point{0.707107, 0.707107})

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

	test.T(t, Identity.Translate(1.0, 1.0).IsTranslation(), true)
	test.T(t, Identity.Rotate(90.0).IsTranslation(), false)
	test.T(t, Identity.Scale(-1.0, 1.0).IsTranslation(), false)
	test.T(t, Identity.Scale(2.0, 2.0).IsTranslation(), false)
	test.T(t, Identity.Scale(2.0, 1.0).IsTranslation(), false)
	test.T(t, Identity.Shear(2.0, -1.0).IsTranslation(), false)

	test.T(t, Identity.Translate(1.0, 1.0).IsRigid(), true)
	test.T(t, Identity.Rotate(90.0).IsRigid(), true)
	test.T(t, Identity.Scale(-1.0, 1.0).IsRigid(), true)
	test.T(t, Identity.Scale(2.0, 2.0).IsRigid(), false)
	test.T(t, Identity.Scale(2.0, 1.0).IsRigid(), false)
	test.T(t, Identity.Shear(2.0, -1.0).IsRigid(), false)

	test.T(t, Identity.Translate(1.0, 1.0).IsSimilarity(), true)
	test.T(t, Identity.Rotate(90.0).IsSimilarity(), true)
	test.T(t, Identity.Scale(-1.0, 1.0).IsSimilarity(), true)
	test.T(t, Identity.Scale(2.0, 2.0).IsSimilarity(), true)
	test.T(t, Identity.Scale(2.0, 1.0).IsSimilarity(), false)
	test.T(t, Identity.Shear(2.0, -1.0).IsSimilarity(), false)

	x, y := Identity.Translate(p.X, p.Y).Pos()
	test.Float(t, x, p.X)
	test.Float(t, y, p.Y)

	test.String(t, Identity.Shear(2.0, 3.0).String(), "(1 2; 3 1) + (0,0)")

	test.T(t, Identity.Shear(1.0, 1.0), Identity.Rotate(45).Scale(2.0, 0.0).Rotate(-45))
	test.String(t, Identity.ToSVG(10.0), "")
	test.String(t, Identity.Translate(3.0, 4.0).ToSVG(10.0), "translate(3,6)")
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

func TestSolveCubicFormula(t *testing.T) {
	var tests = []struct {
		a, b, c, d float64
		x1, x2, x3 float64
	}{
		{0.0, 1.0, 1.0, 0.25, -0.5, math.NaN(), math.NaN()},     // is quadratic formula
		{1.0, -15.0, 75.0, -125.0, 5.0, math.NaN(), math.NaN()}, // c0 == 0, c1 == 0
		{1.0, -3.0, -6.0, 8.0, -2.0, 1.0, 4.0},                  // c0 == 0, c1 < 0
		{1.0, -15.0, 75.0, -124.0, 4.0, math.NaN(), math.NaN()}, // c1 == 0, 0 < c0
		{1.0, -15.0, 75.0, -126.0, 6.0, math.NaN(), math.NaN()}, // c1 == 0, c0 < 0
		{1.0, 0.0, -7.0, 6.0, -3.0, 1.0, 2.0},                   // 0 < delta
		{1.0, -3.0, -9.0, -5.0, -1.0, 5.0, math.NaN()},          // delta == 0
		{1.0, -4.0, 2.0, -8.0, 4.0, math.NaN(), math.NaN()},     // delta < 0, 0 < tmp
		{1.0, -4.0, 2.0, 7.0, -1.0, math.NaN(), math.NaN()},     // delta < 0, tmp < 0
		{16.0, -24.0, 24.0, -8.0, 0.5, math.NaN(), math.NaN()},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("(%v %v %v %v)", tt.a, tt.b, tt.c, tt.d), func(t *testing.T) {
			x1, x2, x3 := solveCubicFormula(tt.a, tt.b, tt.c, tt.d)
			test.Float(t, x1, tt.x1)
			test.Float(t, x2, tt.x2)
			test.Float(t, x3, tt.x3)
		})
	}
}

func TestGaussLegendre(t *testing.T) {
	test.Float(t, gaussLegendre3(math.Log, 0.0, 1.0), -0.947672)
	test.Float(t, gaussLegendre5(math.Log, 0.0, 1.0), -0.979001)
	test.Float(t, gaussLegendre7(math.Log, 0.0, 1.0), -0.988738)
}

func TestPolynomialChebyshevApprox(t *testing.T) {
	f := func(x float64) float64 {
		return x * x
	}

	g := polynomialChebyshevApprox(3, f, 0.0, 11.0, 0.0, 100.0)
	test.Float(t, g(0.0), 0.0)
	test.Float(t, g(5.0), 25.0)
	test.Float(t, g(10.0), 100.0)
	test.Float(t, g(11.0), 100.0)
}

func TestInvSpeedPolynomialApprox(t *testing.T) {
	fp := func(t float64) float64 {
		xp := math.Cos(t)
		yp := 2 * t
		return math.Sqrt(xp*xp + yp*yp)
	}

	// https://www.wolframalpha.com/input/?i=arclength+x%28t%29%3Dsin+t%2C+y%28t%29%3Dt*t+for+t%3D0+to+2pi
	f, L := invSpeedPolynomialChebyshevApprox(15, gaussLegendre7, fp, 0.0, 2.0*math.Pi)
	test.Float(t, L, 40.051641)
	test.Float(t, f(0.0), 0.0)
	test.That(t, math.Abs(f(40.051641)-2.0*math.Pi) < 0.01)
	test.That(t, math.Abs(f(10.3539)-math.Pi) < 0.01)

	//f, L = invPolynomialApprox3(gaussLegendre7, fp, 0.0, 2.0*math.Pi)
	//test.Float(t, L, 40.051641)
	//test.Float(t, f(0.0), 0.0)
	//test.That(t, math.Abs(f(40.051641)-2.0*math.Pi) < 0.01)
	//test.That(t, math.Abs(f(10.3539)-math.Pi) < 1.0)
}
