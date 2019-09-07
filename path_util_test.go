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

	mid, largeArc0, largeArc1, ok = splitEllipse(2.0, 1.0, 0.0, 0.0, 0.0, math.Pi, 0.0, -math.Pi/2.0)
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
	test.T(t, ellipseToBeziers(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}).String(), "M0 0C6.7182e-15 54.858 45.142 100 100 100C154.86 100 200 54.858 200 0")
}

func TestFlattenEllipse(t *testing.T) {
	Tolerance = 1.0
	test.T(t, flattenEllipse(Point{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, Point{200.0, 0.0}).String(), "M0 0L3.8202 27.243L15.092 52.545L33.225 74.179L56.889 90.115L84.082 98.716L100 100L127.24 96.18L152.55 84.908L174.18 66.775L190.12 43.111L198.72 15.918L200 0")
}
