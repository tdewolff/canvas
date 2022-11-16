package canvas

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestEllipse(t *testing.T) {
	test.T(t, EllipsePos(2.0, 1.0, math.Pi/2.0, 1.0, 0.5, 0.0), Point{1.0, 2.5})
	test.T(t, ellipseDeriv(2.0, 1.0, math.Pi/2.0, true, 0.0), Point{-1.0, 0.0})
	test.T(t, ellipseDeriv(2.0, 1.0, math.Pi/2.0, false, 0.0), Point{1.0, 0.0})
	test.T(t, ellipseDeriv2(2.0, 1.0, math.Pi/2.0, 0.0), Point{0.0, -2.0})
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
	mid, large0, large1, ok := ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, math.Pi, 0.0, math.Pi/2.0)
	test.That(t, ok)
	test.T(t, mid, Point{0.0, 1.0})
	test.That(t, !large0)
	test.That(t, !large1)

	_, _, _, ok = ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, math.Pi, 0.0, -math.Pi/2.0)
	test.That(t, !ok)

	mid, large0, large1, ok = ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, 0.0, math.Pi*7.0/4.0, math.Pi/2.0)
	test.That(t, ok)
	test.T(t, mid, Point{0.0, 1.0})
	test.That(t, !large0)
	test.That(t, large1)

	mid, large0, large1, ok = ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, 0.0, math.Pi*7.0/4.0, math.Pi*3.0/2.0)
	test.That(t, ok)
	test.T(t, mid, Point{0.0, -1.0})
	test.That(t, large0)
	test.That(t, !large1)
}

func TestArcToQuad(t *testing.T) {
	test.T(t, arcToQuad(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}), MustParseSVG("M0 0Q0 100 100 100Q200 100 200 0"))
}

func TestArcToCube(t *testing.T) {
	defer setEpsilon(1e-3)()

	test.T(t, arcToCube(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}), MustParseSVG("M0 0C0 54.858 45.142 100 100 100C154.858 100 200 54.858 200 0"))
}

func TestFlattenEllipse(t *testing.T) {
	defer setEpsilon(1e-3)()
	defer setTolerance(1.0)()

	test.T(t, flattenEllipticArc(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}), MustParseSVG("M0 0L3.8202 27.243L15.092 52.545L33.225 74.179L56.889 90.115L84.082 98.716L100 100L127.243 96.18L152.545 84.908L174.179 66.775L190.115 43.111L198.716 15.918L200 0"))
}

func TestQuadraticBezier(t *testing.T) {
	defer setEpsilon(1e-6)()

	p1, p2 := quadraticToCubicBezier(Point{0.0, 0.0}, Point{1.5, 0.0}, Point{3.0, 0.0})
	test.T(t, p1, Point{1.0, 0.0})
	test.T(t, p2, Point{2.0, 0.0})

	p1, p2 = quadraticToCubicBezier(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0})
	test.T(t, p1, Point{0.666667, 0.0})
	test.T(t, p2, Point{1.0, 0.333333})

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

	p0, p1, p2, q0, q1, q2 := quadraticBezierSplit(Point{0.0, 0.0}, Point{1.0, 0.0}, Point{1.0, 1.0}, 0.5)
	test.T(t, p0, Point{0.0, 0.0})
	test.T(t, p1, Point{0.5, 0.0})
	test.T(t, p2, Point{0.75, 0.25})
	test.T(t, q0, Point{0.75, 0.25})
	test.T(t, q1, Point{1.0, 0.5})
	test.T(t, q2, Point{1.0, 1.0})
}

func TestQuadraticBezierDistance(t *testing.T) {
	var tests = []struct {
		p0, p1, p2 Point
		q          Point
		d          float64
	}{
		{Point{0.0, 0.0}, Point{4.0, 6.0}, Point{8.0, 0.0}, Point{9.0, 0.5}, math.Sqrt(1.25)},
		{Point{0.0, 0.0}, Point{1.0, 1.0}, Point{2.0, 0.0}, Point{0.0, 0.0}, 0.0},
		{Point{0.0, 0.0}, Point{1.0, 1.0}, Point{2.0, 0.0}, Point{1.0, 1.0}, 0.5},
		{Point{0.0, 0.0}, Point{1.0, 1.0}, Point{2.0, 0.0}, Point{2.0, 0.0}, 0.0},
		{Point{0.0, 0.0}, Point{1.0, 1.0}, Point{2.0, 0.0}, Point{1.0, 0.0}, 0.5},
		{Point{0.0, 0.0}, Point{1.0, 1.0}, Point{2.0, 0.0}, Point{-1.0, 0.0}, 1.0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.q), func(t *testing.T) {
			d := quadraticBezierDistance(tt.p0, tt.p1, tt.p2, tt.q)
			test.Float(t, d, tt.d)
		})
	}
}

func TestCubicBezier(t *testing.T) {
	defer setEpsilon(1e-5)()

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
	test.Float(t, cubicBezierLength(p0, p1, p2, p3), 1.623225)

	p0, p1, p2, p3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, 0.5)
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
	defer setEpsilon(1e-6)()

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
}

func TestCubicBezierStrokeFlatten(t *testing.T) {
	defer setEpsilon(1e-6)()

	tests := []struct {
		path      string
		d         float64
		tolerance float64
		expected  string
	}{
		{"C0.666667 0 1 0.333333 1 1", 0.5, 0.5, "L1.5 1"},
		{"C0.666667 0 1 0.333333 1 1", 0.5, 0.125, "L1.376154 0.308659L1.5 1"},
		{"C1 0 2 1 3 2", 0.0, 0.1, "L1.095445 0.351314L2.579154 1.581915L3 2"},
		{"C0 0 1 0 2 2", 0.0, 0.1, "L1.22865 0.8L2 2"},       // p0 == p1
		{"C1 1 2 2 3 5", 0.0, 0.1, "L2.481111 3.612482L3 5"}, // s2 == 0
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			path := MustParseSVG(tt.path)
			p0 := Point{path.d[1], path.d[2]}
			p1 := Point{path.d[5], path.d[6]}
			p2 := Point{path.d[7], path.d[8]}
			p3 := Point{path.d[9], path.d[10]}

			p := &Path{}
			flattenSmoothCubicBezier(p, p0, p1, p2, p3, tt.d, tt.tolerance)
			test.T(t, p, MustParseSVG(tt.expected))
		})
	}
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

func TestCubicBezierStroke(t *testing.T) {
	tests := []struct {
		p []Point
	}{
		// see "Analysis of Inflection Points for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
		// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
		{[]Point{{16, 467}, {185, 95}, {673, 545}, {810, 17}}},
		{[]Point{{859, 676}, {13, 422}, {781, 12}, {266, 425}}},
		{[]Point{{872, 686}, {11, 423}, {779, 13}, {220, 376}}},
		{[]Point{{819, 566}, {43, 18}, {826, 18}, {25, 533}}},
		{[]Point{{884, 574}, {135, 14}, {678, 14}, {14, 566}}},

		// be aware that we offset the bezier by 0.1
		// single inflection point, ranges outside t=[0,1]
		{[]Point{{0, 0}, {1, 1}, {0, 1}, {1, 0}}},

		// two inflection points, ranges outside t=[0,1]
		{[]Point{{0, 0}, {0.9, 1}, {0.1, 1}, {1, 0}}},

		// one inflection point, max range outside t=[0,1]
		{[]Point{{0, 0}, {80, 100}, {80, -100}, {100, 0}}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v %v %v %v", tt.p[0], tt.p[1], tt.p[2], tt.p[3]), func(t *testing.T) {
			length := cubicBezierLength(tt.p[0], tt.p[1], tt.p[2], tt.p[3])
			flatLength := strokeCubicBezier(tt.p[0], tt.p[1], tt.p[2], tt.p[3], 0.0, 0.001).Length()
			test.FloatDiff(t, flatLength, length, 0.25)
		})
	}

	defer setEpsilon(1e-6)()

	test.T(t, strokeCubicBezier(Point{0, 0}, Point{30, 0}, Point{30, 10}, Point{25, 10}, 5.0, 0.01).Bounds(), Rect{0.0, -5.0, 32.478752, 20.0})
}
