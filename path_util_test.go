package canvas

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestIntersectionLineLine(t *testing.T) {
	var tts = []struct {
		a0, a1 Point
		b0, b1 Point
		p      Point
	}{
		{Point{2.0, 0.0}, Point{2.0, 3.0}, Point{1.0, 2.0}, Point{3.0, 2.0}, Point{2.0, 2.0}},
		{Point{2.0, 0.0}, Point{2.0, 1.0}, Point{0.0, 2.0}, Point{1.0, 2.0}, Point{}},
		{Point{2.0, 0.0}, Point{2.0, 1.0}, Point{3.0, 0.0}, Point{3.0, 1.0}, Point{}},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			p, _ := intersectionLineLine(tt.a0, tt.a1, tt.b0, tt.b1)
			test.T(t, p, tt.p)
		})
	}
}

func TestIntersectionRayCircle(t *testing.T) {
	var tts = []struct {
		l0, l1 Point
		c      Point
		r      float64
		p0, p1 Point
	}{
		{Point{0.0, 0.0}, Point{0.0, 1.0}, Point{0.0, 0.0}, 2.0, Point{0.0, 2.0}, Point{0.0, -2.0}},
		{Point{2.0, 0.0}, Point{2.0, 1.0}, Point{0.0, 0.0}, 2.0, Point{2.0, 0.0}, Point{2.0, 0.0}},
		{Point{0.0, 2.0}, Point{1.0, 2.0}, Point{0.0, 0.0}, 2.0, Point{0.0, 2.0}, Point{0.0, 2.0}},
		{Point{0.0, 3.0}, Point{1.0, 3.0}, Point{0.0, 0.0}, 2.0, Point{}, Point{}},
		{Point{0.0, 1.0}, Point{0.0, 0.0}, Point{0.0, 0.0}, 2.0, Point{0.0, 2.0}, Point{0.0, -2.0}},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			p0, p1, _ := intersectionRayCircle(tt.l0, tt.l1, tt.c, tt.r)
			test.T(t, p0, tt.p0)
			test.T(t, p1, tt.p1)
		})
	}
}

func TestIntersectionCircleCircle(t *testing.T) {
	var tts = []struct {
		c0     Point
		r0     float64
		c1     Point
		r1     float64
		p0, p1 Point
	}{
		{Point{0.0, 0.0}, 1.0, Point{2.0, 0.0}, 1.0, Point{1.0, 0.0}, Point{1.0, 0.0}},
		{Point{0.0, 0.0}, 1.0, Point{1.0, 1.0}, 1.0, Point{1.0, 0.0}, Point{0.0, 1.0}},
		{Point{0.0, 0.0}, 1.0, Point{3.0, 0.0}, 1.0, Point{}, Point{}},
		{Point{0.0, 0.0}, 1.0, Point{0.0, 0.0}, 1.0, Point{}, Point{}},
		{Point{0.0, 0.0}, 1.0, Point{0.5, 0.0}, 2.0, Point{}, Point{}},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			p0, p1, _ := intersectionCircleCircle(tt.c0, tt.r0, tt.c1, tt.r1)
			test.T(t, p0, tt.p0)
			test.T(t, p1, tt.p1)
		})
	}
}

func TestEllipse(t *testing.T) {
	test.T(t, ellipsePos(2.0, 1.0, math.Pi/2.0, 1.0, 0.5, 0.0), Point{1.0, 2.5})
	test.T(t, ellipseDeriv(2.0, 1.0, math.Pi/2.0, true, 0.0), Point{-1.0, 0.0})
	test.T(t, ellipseDeriv(2.0, 1.0, math.Pi/2.0, false, 0.0), Point{1.0, 0.0})
	test.T(t, ellipseDeriv2(2.0, 1.0, math.Pi/2.0, false, 0.0), Point{0.0, -2.0})
	test.T(t, ellipseCurvatureRadius(2.0, 1.0, true, 0.0), 0.5)
	test.T(t, ellipseCurvatureRadius(2.0, 1.0, false, 0.0), -0.5)
	test.T(t, ellipseCurvatureRadius(2.0, 1.0, true, math.Pi/2.0), 4.0)
	if !math.IsNaN(ellipseCurvatureRadius(2.0, 0.0, true, 0.0)) {
		test.Fail(t)
	}
	test.T(t, ellipseNormal(2.0, 1.0, math.Pi/2.0, true, 0.0, 1.0), Point{0.0, 1.0})
	test.T(t, ellipseNormal(2.0, 1.0, math.Pi/2.0, false, 0.0, 1.0), Point{0.0, -1.0})

	// https://www.wolframalpha.com/input/?i=arclength+x%28t%29%3D2*cos+t%2C+y%28t%29%3Dsin+t+for+t%3D0+to+0.5pi
	test.Float(t, ellipseLength(2.0, 1.0, 0.0, math.Pi/2.0), 2.422110)

	test.Float(t, ellipseRadiiCorrection(Point{0.0, 0.0}, 0.1, 0.1, 0.0, Point{1.0, 0.0}), 5.0)
}

func TestEllipseToCenter(t *testing.T) {
	cx, cy, theta0, theta1 := ellipseToCenter(0.0, 0.0, 2.0, 2.0, 0.0, false, false, 2.0, 2.0)
	test.Float(t, cx, 2.0)
	test.Float(t, cy, 0.0)
	test.Float(t, theta0, math.Pi)
	test.Float(t, theta1, math.Pi/2.0)

	cx, cy, theta0, theta1 = ellipseToCenter(0.0, 0.0, 2.0, 2.0, 0.0, true, false, 2.0, 2.0)
	test.Float(t, cx, 0.0)
	test.Float(t, cy, 2.0)
	test.Float(t, theta0, math.Pi*3.0/2.0)
	test.Float(t, theta1, 0.0)

	cx, cy, theta0, theta1 = ellipseToCenter(0.0, 0.0, 2.0, 2.0, 0.0, true, true, 2.0, 2.0)
	test.Float(t, cx, 2.0)
	test.Float(t, cy, 0.0)
	test.Float(t, theta0, math.Pi)
	test.Float(t, theta1, math.Pi*5.0/2.0)

	cx, cy, theta0, theta1 = ellipseToCenter(0.0, 0.0, 2.0, 1.0, math.Pi/2.0, false, false, 1.0, 2.0)
	test.Float(t, cx, 1.0)
	test.Float(t, cy, 0.0)
	test.Float(t, theta0, math.Pi/2.0)
	test.Float(t, theta1, 0.0)

	cx, cy, theta0, theta1 = ellipseToCenter(0.0, 0.0, 0.1, 0.1, 0.0, false, false, 1.0, 0.0)
	test.Float(t, cx, 0.5)
	test.Float(t, cy, 0.0)
	test.Float(t, theta0, math.Pi)
	test.Float(t, theta1, 0.0)

	cx, cy, theta0, theta1 = ellipseToCenter(0.0, 0.0, 1.0, 1.0, 0.0, false, false, 0.0, 0.0)
	test.Float(t, cx, 0.0)
	test.Float(t, cy, 0.0)
	test.Float(t, theta0, 0.0)
	test.Float(t, theta1, 0.0)
}

func TestEllipseSplit(t *testing.T) {
	mid, largeArc0, largeArc1, ok := splitEllipse(2.0, 1.0, 0.0, 0.0, 0.0, math.Pi, 0.0, math.Pi/2.0)
	test.That(t, ok)
	test.T(t, mid, Point{0.0, 1.0})
	test.That(t, !largeArc0)
	test.That(t, !largeArc1)

	_, _, _, ok = splitEllipse(2.0, 1.0, 0.0, 0.0, 0.0, math.Pi, 0.0, -math.Pi/2.0)
	test.That(t, !ok)

	mid, largeArc0, largeArc1, ok = splitEllipse(2.0, 1.0, 0.0, 0.0, 0.0, 0.0, math.Pi*7.0/4.0, math.Pi/2.0)
	test.That(t, ok)
	test.T(t, mid, Point{0.0, 1.0})
	test.That(t, !largeArc0)
	test.That(t, largeArc1)

	mid, largeArc0, largeArc1, ok = splitEllipse(2.0, 1.0, 0.0, 0.0, 0.0, 0.0, math.Pi*7.0/4.0, math.Pi*3.0/2.0)
	test.That(t, ok)
	test.T(t, mid, Point{0.0, -1.0})
	test.That(t, largeArc0)
	test.That(t, !largeArc1)
}

func TestEllipseToBeziers(t *testing.T) {
	Epsilon = 1e-2
	test.T(t, ellipseToBeziers(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}), MustParseSVG("M0 0C6.7182e-15 54.858 45.142 100 100 100C154.86 100 200 54.858 200 0"))
}

func TestFlattenEllipse(t *testing.T) {
	Epsilon = 1e-2
	Tolerance = 1.0
	test.T(t, flattenEllipse(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}), MustParseSVG("M0 0L3.8202 27.243L15.092 52.545L33.225 74.179L56.889 90.115L84.082 98.716L100 100L127.24 96.18L152.55 84.908L174.18 66.775L190.12 43.111L198.72 15.918L200 0"))
}

func TestQuadraticBezier(t *testing.T) {
	Epsilon = 1e-3

	p1, p2 := quadraticToCubicBezier(Point{0.0, 0.0}, Point{1.5, 0.0}, Point{3.0, 0.0})
	test.T(t, p1, Point{1.0, 0.0})
	test.T(t, p2, Point{2.0, 0.0})

	p1, p2 = quadraticToCubicBezier(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0})
	test.T(t, p1, Point{0.667, 0.0})
	test.T(t, p2, Point{1.0, 0.333})

	test.T(t, quadraticBezierPos(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 0.0), Point{0.0, 0.0})
	test.T(t, quadraticBezierPos(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 0.5), Point{0.75, 0.25})
	test.T(t, quadraticBezierPos(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 1.0), Point{1.0, 1.0})
	test.T(t, quadraticBezierDeriv(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 0.0), Point{2.0, 0.0})
	test.T(t, quadraticBezierDeriv(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 0.5), Point{1.0, 1.0})
	test.T(t, quadraticBezierDeriv(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 1.0), Point{0.0, 2.0})
	test.Float(t, quadraticBezierLength(Point{0.0, 0.0}, Point{0.5, 0.0}, Point{2.0, 0.0}), 2.0)
	test.Float(t, quadraticBezierLength(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{2.0, 0.0}), 2.0)

	// https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D2*%281-t%29*t*1.00+%2B+t%5E2*1.00%2C+y%3Dt%5E2*1.00%7D+from+0+to+1
	test.Float(t, quadraticBezierLength(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}), 1.623225)

	p0, p1, p2, q0, q1, q2 := splitQuadraticBezier(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 0.5)
	test.T(t, p0, Point{0.0, 0.0})
	test.T(t, p1, Point{0.5, 0.0})
	test.T(t, p2, Point{0.75, 0.25})
	test.T(t, q0, Point{0.75, 0.25})
	test.T(t, q1, Point{1.0, 0.5})
	test.T(t, q2, Point{1.0, 1.0})
}

func TestCubicBezier(t *testing.T) {
	p0, p1, p2, p3 := Point{0.0, 0.0}, Point{0.666667, 0.0}, Point{1.0, 0.333333}, Point{1.0, 1.0}
	test.T(t, cubicBezierPos(p0, p1, p2, p3, 0.0), Point{0.0, 0.0})
	test.T(t, cubicBezierPos(p0, p1, p2, p3, 0.5), Point{0.75, 0.25})
	test.T(t, cubicBezierPos(p0, p1, p2, p3, 1.0), Point{1.0, 1.0})
	test.T(t, cubicBezierDeriv(p0, p1, p2, p3, 0.0), Point{2.0, 0.0})
	test.T(t, cubicBezierDeriv(p0, p1, p2, p3, 0.5), Point{1.0, 1.0})
	test.T(t, cubicBezierDeriv(p0, p1, p2, p3, 1.0), Point{0.0, 2.0})
	test.T(t, cubicBezierDeriv2(p0, p1, p2, p3, 0.0), Point{-2.0, 2.0})
	test.T(t, cubicBezierDeriv2(p0, p1, p2, p3, 0.5), Point{-2.0, 2.0})
	test.T(t, cubicBezierDeriv2(p0, p1, p2, p3, 1.0), Point{-2.0, 2.0})
	test.Float(t, cubicBezierCurvatureRadius(p0, p1, p2, p3, 0.0), 2.000004)
	test.Float(t, cubicBezierCurvatureRadius(p0, p1, p2, p3, 0.5), 0.707107)
	test.Float(t, cubicBezierCurvatureRadius(p0, p1, p2, p3, 1.0), 2.000004)
	test.Float(t, cubicBezierCurvatureRadius(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{2.0, 0.0}, Point{3.0, 0.0}, 0.0), math.NaN())
	test.T(t, cubicBezierNormal(p0, p1, p2, p3, 0.0, 1.0), Point{0.0, -1.0})
	test.T(t, cubicBezierNormal(p0, p0, p1, p3, 0.0, 1.0), Point{0.0, -1.0})
	test.T(t, cubicBezierNormal(p0, p0, p0, p1, 0.0, 1.0), Point{0.0, -1.0})
	test.T(t, cubicBezierNormal(p0, p0, p0, p0, 0.0, 1.0), Point{})
	test.T(t, cubicBezierNormal(p0, p1, p2, p3, 1.0, 1.0), Point{1.0, 0.0})
	test.T(t, cubicBezierNormal(p0, p2, p3, p3, 1.0, 1.0), Point{1.0, 0.0})
	test.T(t, cubicBezierNormal(p2, p3, p3, p3, 1.0, 1.0), Point{1.0, 0.0})
	test.T(t, cubicBezierNormal(p3, p3, p3, p3, 1.0, 1.0), Point{})

	// https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D3*%281-t%29%5E2*t*0.666667+%2B+3*%281-t%29*t%5E2*1.00+%2B+t%5E3*1.00%2C+y%3D3*%281-t%29*t%5E2*0.333333+%2B+t%5E3*1.00%7D+from+0+to+1
	test.Float(t, cubicBezierLength(p0, p1, p2, p3), 1.623214)

	p0, p1, p2, p3, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, 0.5)
	test.T(t, p0, Point{0.0, 0.0})
	test.T(t, p1, Point{0.333333, 0.0})
	test.T(t, p2, Point{0.583333, 0.083333})
	test.T(t, p3, Point{0.75, 0.25})
	test.T(t, q0, Point{0.75, 0.25})
	test.T(t, q1, Point{0.916667, 0.416667})
	test.T(t, q2, Point{1.0, 0.666667})
	test.T(t, q3, Point{1.0, 1.0})
}

func TestCubicBezierStrokeHelpers(t *testing.T) {
	p0, p1, p2, p3 := Point{0.0, 0.0}, Point{0.666667, 0.0}, Point{1.0, 0.333333}, Point{1.0, 1.0}

	p := &Path{}
	addCubicBezierLine(p, p0, p1, p0, p0, 0.0, 0.5)
	test.That(t, p.Empty())

	p = &Path{}
	addCubicBezierLine(p, p0, p1, p2, p3, 0.0, 0.5)
	test.T(t, p, MustParseSVG("L0 -0.5"))

	p = &Path{}
	addCubicBezierLine(p, p0, p1, p2, p3, 1.0, 0.5)
	test.T(t, p, MustParseSVG("L1.5 1"))

	p = &Path{}
	flattenSmoothCubicBezier(p, p0, p1, p2, p3, 0.5, 0.5)
	test.T(t, p, MustParseSVG("L1.5 1"))

	p = &Path{}
	flattenSmoothCubicBezier(p, p0, p1, p2, p3, 0.5, 0.125)
	test.T(t, p, MustParseSVG("L1.4542 0.55703L1.5 1"))

	p = &Path{}
	flattenSmoothCubicBezier(p, p0, p0, p2, p3, 0.5, 0.125) // denom == 0
	test.T(t, p, MustParseSVG("L1.5 1"))
}

func TestCubicBezierInflectionPoints(t *testing.T) {
	x1, x2 := findInflectionPointsCubicBezier(Point{0.0, 0.0}, Point{0.0, 1.0}, Point{1.0, 1.0}, Point{1.0, 0.0})
	test.Float(t, x1, math.NaN())
	test.Float(t, x2, math.NaN())

	x1, x2 = findInflectionPointsCubicBezier(Point{0.0, 0.0}, Point{1.0, 1.0}, Point{0.0, 1.0}, Point{1.0, 0.0})
	test.Float(t, x1, 0.5)
	test.Float(t, x2, math.NaN())

	// see "Analysis of Inflection Points for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
	// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
	x1, x2 = findInflectionPointsCubicBezier(Point{16, 467}, Point{185, 95}, Point{673, 545}, Point{810, 17})
	test.Float(t, x1, 0.456590)
	test.Float(t, x2, math.NaN())

	x1, x2 = findInflectionPointsCubicBezier(Point{859, 676}, Point{13, 422}, Point{781, 12}, Point{266, 425})
	test.Float(t, x1, 0.681076)
	test.Float(t, x2, 0.705299)

	x1, x2 = findInflectionPointsCubicBezier(Point{872, 686}, Point{11, 423}, Point{779, 13}, Point{220, 376})
	test.Float(t, x1, 0.588071)
	test.Float(t, x2, 0.886863)

	x1, x2 = findInflectionPointsCubicBezier(Point{819, 566}, Point{43, 18}, Point{826, 18}, Point{25, 533})
	test.Float(t, x1, 0.476169)
	test.Float(t, x2, 0.539295)

	x1, x2 = findInflectionPointsCubicBezier(Point{884, 574}, Point{135, 14}, Point{678, 14}, Point{14, 566})
	test.Float(t, x1, 0.320836)
	test.Float(t, x2, 0.682291)
}

func TestCubicBezierInflectionPointRange(t *testing.T) {
	x1, x2 := findInflectionPointRangeCubicBezier(Point{0.0, 0.0}, Point{1.0, 1.0}, Point{0.0, 1.0}, Point{1.0, 0.0}, math.NaN(), 0.25)
	test.That(t, math.IsInf(x1, 1.0))
	test.That(t, math.IsInf(x2, 1.0))

	// p0==p1==p2
	x1, x2 = findInflectionPointRangeCubicBezier(Point{0.0, 0.0}, Point{0.0, 0.0}, Point{0.0, 0.0}, Point{1.0, 0.0}, 0.0, 0.25)
	test.Float(t, x1, 0.0)
	test.Float(t, x2, 1.0)

	// p0==p1, s3==0
	x1, x2 = findInflectionPointRangeCubicBezier(Point{0.0, 0.0}, Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 0.0}, 0.0, 0.25)
	test.Float(t, x1, 0.0)
	test.Float(t, x2, 1.0)

	// all within tolerance
	x1, x2 = findInflectionPointRangeCubicBezier(Point{0.0, 0.0}, Point{0.0, 1.0}, Point{1.0, 1.0}, Point{1.0, 0.0}, 0.5, 1.0)
	test.That(t, x1 <= 0.0)
	test.That(t, x2 >= 1.0)

	x1, x2 = findInflectionPointRangeCubicBezier(Point{0.0, 0.0}, Point{0.0, 1.0}, Point{1.0, 1.0}, Point{1.0, 0.0}, 0.5, 0.000000001)
	test.Float(t, x1, 0.499449)
	test.Float(t, x2, 0.500550)
}

func TestCubicBezierSplitAtInflections(t *testing.T) {
	Epsilon = 1.0

	beziers := splitCubicBezierAtInflections(Point{0, 0}, Point{0, 1}, Point{1, 1}, Point{1, 1})
	test.T(t, len(beziers), 1)

	// see "Analysis of Inflection Points for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
	// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
	beziers = splitCubicBezierAtInflections(Point{16, 467}, Point{185, 95}, Point{673, 545}, Point{810, 17})
	test.T(t, len(beziers), 2)
	test.T(t, beziers[1][0], Point{383, 300})

	beziers = splitCubicBezierAtInflections(Point{884, 574}, Point{135, 14}, Point{678, 14}, Point{14, 566})
	test.T(t, len(beziers), 3)
	test.T(t, beziers[1][0], Point{479, 207})
	test.T(t, beziers[2][0], Point{361, 207})
}

func TestCubicBezierStroke(t *testing.T) {
	Epsilon = 1e-2

	// see "Analysis of Inflection Points for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
	// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
	// stroke results tested in browser
	test.T(t, strokeCubicBezier(Point{16, 467}, Point{185, 95}, Point{673, 545}, Point{810, 17}, 0.1, 0.1), MustParseSVG("L23.486 451.24L31.409 436.61L39.674 423L48.278 410.35L57.225 398.59L66.521 387.67L76.18 377.54L86.218 368.16L96.658 359.48L107.53 351.46L118.87 344.08L130.72 337.31L143.12 331.13L156.15 325.52L169.87 320.48L184.38 315.98L199.77 312.03L216.19 308.62L233.81 305.76L252.9 303.45L273.8 301.71L297.12 300.52L323.93 299.91L346.57 299.82L420 300.39L462.27 299.94L490.69 298.71L514.65 296.81L535.91 294.29L555.25 291.17L573.09 287.45L589.72 283.15L605.34 278.27L620.07 272.82L634.04 266.77L647.33 260.14L660.03 252.89L672.17 245.03L683.83 236.51L695.03 227.32L705.81 217.42L716.2 206.78L726.22 195.35L735.89 183.09L745.21 169.95L754.21 155.87L762.87 140.8L771.2 124.67L779.19 107.43L786.83 88.987L794.12 69.28L801.04 48.23L807.57 25.753L809.9 16.975"))

	test.T(t, strokeCubicBezier(Point{859, 676}, Point{13, 422}, Point{781, 12}, Point{266, 425}, 0.1, 0.1), MustParseSVG("L822.6 664.76L788.4 653.29L756.29 641.69L726.2 630L698.04 618.21L671.75 606.36L647.24 594.46L624.44 582.53L603.28 570.57L583.69 558.6L565.6 546.64L548.94 534.68L533.64 522.75L519.65 510.85L506.9 498.99L495.34 487.17L484.9 475.38L475.54 463.65L467.21 451.95L459.87 440.3L453.46 428.68L447.96 417.1L443.34 405.55L439.55 394.02L436.59 382.51L434.43 371.02L433.06 359.54L432.46 348.07L432.63 336.63L433.57 325.22L435.26 313.88L437.69 302.64L440.84 291.58L444.67 280.81L449.07 270.53L453.85 261.06L458.51 253.12L458.74 252.76L460.57 250.15L457.86 253.37L442.84 269.71L423.51 289.04L399.37 311.8L370.39 337.85L336.5 367.1L297.62 399.5L266.06 425.08"))

	test.T(t, strokeCubicBezier(Point{872, 686}, Point{11, 423}, Point{779, 13}, Point{220, 376}, 0.1, 0.1), MustParseSVG("L835.03 674.4L800.28 662.58L767.63 650.64L737.01 638.62L708.34 626.52L681.53 614.37L656.52 602.16L633.22 589.93L611.56 577.67L591.46 565.41L572.87 553.15L555.7 540.9L539.9 528.66L525.39 516.44L512.13 504.25L500.04 492.09L489.07 479.95L479.18 467.84L470.3 455.74L462.41 443.67L455.44 431.6L449.37 419.53L444.16 407.45L439.79 395.34L436.22 383.2L433.44 371L431.44 358.74L430.19 346.38L429.71 333.91L429.97 321.3L430.99 308.52L432.77 295.5L435.32 282.13L438.24 269.81L444.63 246.04L446.04 238.02L445.78 234.96L444.93 233.53L443.67 232.89L441.75 232.85L438.65 233.63L433.65 235.8L425.54 240.25L412 248.65L399.45 256.86L312.89 314.95L228.25 370.76L220.05 376.08"))

	test.T(t, strokeCubicBezier(Point{819, 566}, Point{43, 18}, Point{826, 18}, Point{25, 533}, 0.1, 0.1), MustParseSVG("L780.31 538.32L744.15 511.38L710.39 485.27L678.97 459.99L649.81 435.54L622.83 411.91L597.98 389.12L575.17 367.17L554.33 346.05L535.4 325.78L518.31 306.36L502.99 287.8L489.36 270.11L477.36 253.31L466.93 237.4L457.99 222.42L450.47 208.4L444.32 195.39L439.46 183.45L435.83 172.68L433.35 163.27L432.16 157.2L431.31 150.88L429.55 157.23L425.34 168.86L419.91 180.74L412.97 193.62L404.46 207.51L394.31 222.38L382.46 238.18L368.85 254.9L353.4 272.52L336.06 291.01L316.73 310.36L295.37 330.56L271.89 351.61L246.22 373.49L218.29 396.2L188.03 419.74L155.36 444.09L120.22 469.25L82.511 495.22L42.176 522L25.054 533.08"))

	test.T(t, strokeCubicBezier(Point{884, 574}, Point{135, 14}, Point{678, 14}, Point{14, 566}, 0.1, 0.1), MustParseSVG("L840.01 540.75L798.43 508.24L759.13 476.54L722.06 445.66L687.13 415.57L654.27 386.26L623.37 357.69L594.32 329.82L566.97 302.56L541.09 275.74L516.26 249.01L501.7 232.87L461.39 187.33L445.49 170.49L436.91 162.65L430.6 157.96L425.51 155.15L421.15 153.64L417.21 153.1L413.46 153.39L409.69 154.48L405.71 156.47L401.28 159.58L396.14 164.2L389.84 170.95L381.55 181.01L376.52 187.48L343.32 231.72L315.63 267.25L291.69 296.37L267.3 324.63L241.76 352.87L214.78 381.44L186.16 410.5L155.79 440.14L123.56 470.41L89.392 501.35L53.214 532.98L14.957 565.33L14.064 566.08"))

	// be aware that we offset the bezier by 0.1
	// single inflection point, ranges outside t=[0,1]
	test.T(t, strokeCubicBezier(Point{0, 0}, Point{1, 1}, Point{0, 1}, Point{1, 0}, 0.1, 1.0), MustParseSVG("L0.92929 -0.070711"))

	// two inflection points, ranges outside t=[0,1]
	test.T(t, strokeCubicBezier(Point{0, 0}, Point{0.9, 1}, Point{0.1, 1}, Point{1, 0}, 0.1, 1.0), MustParseSVG("L0.92567 -0.066896"))

	// one inflection point, max range outside t=[0,1]
	test.T(t, strokeCubicBezier(Point{0, 0}, Point{80, 100}, Point{80, -100}, Point{100, 0}, 0.1, 50), MustParseSVG("L11.922 13.186L100.1 -0.019612"))
}
