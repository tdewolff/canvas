package canvas

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

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

func TestIntersectionLineLine(t *testing.T) {
	var tts = []struct {
		line1, line2 string
		zs           Intersections
	}{
		// secant
		{"M2 0L2 3", "M1 2L3 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{2.0 / 3.0, 0.5}, [2]float64{0.5 * math.Pi, 0.0}, false},
		}},

		// tangent
		{"M2 0L2 3", "M2 2L3 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{2.0 / 3.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true},
		}},
		{"M2 0L2 2", "M2 2L3 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{1.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true},
		}},
		{"L2 2", "M0 4L2 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{1.0, 1.0}, [2]float64{0.25 * math.Pi, 1.75 * math.Pi}, true},
		}},
		{"L10 5", "M0 10L10 5", Intersections{
			{Point{10.0, 5.0}, [2]float64{1.0, 1.0}, [2]float64{Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle()}, true},
		}},
		{"M10 5L20 10", "M10 5L20 0", Intersections{
			{Point{10.0, 5.0}, [2]float64{0.0, 0.0}, [2]float64{Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle()}, true},
		}},

		// parallel
		{"L2 2", "L2 2", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
			{Point{2.0, 2.0}, [2]float64{1.0, 1.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
		}},
		{"L2 2", "M2 2L0 0", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 1.0}, [2]float64{0.25 * math.Pi, 1.25 * math.Pi}, true},
			{Point{2.0, 2.0}, [2]float64{1.0, 0.0}, [2]float64{0.25 * math.Pi, 1.25 * math.Pi}, true},
		}},
		{"L2 2", "M3 3L5 5", Intersections{}},
		{"L2 2", "M-1 1L1 3", Intersections{}},
		{"L2 2", "M2 2L4 4", Intersections{
			{Point{2.0, 2.0}, [2]float64{1.0, 0.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
		}},
		{"L2 2", "M-2 -2L0 0", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 1.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
		}},
		{"L4 4", "M2 2L6 6", Intersections{
			{Point{2.0, 2.0}, [2]float64{0.5, 0.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
			{Point{4.0, 4.0}, [2]float64{1.0, 0.5}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
		}},
		{"L4 4", "M-2 -2L2 2", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.5}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
			{Point{2.0, 2.0}, [2]float64{0.5, 1.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true},
		}},

		// none
		{"M2 0L2 1", "M3 0L3 1", Intersections{}},
		{"M2 0L2 1", "M0 2L1 2", Intersections{}},

		// bugs
		{"M21.590990257669734 18.40900974233027L22.651650429449557 17.348349570550447", "M21.23743686707646 18.762563132923542L21.590990257669738 18.409009742330266", Intersections{
			{Point{21.590990257669734, 18.40900974233027}, [2]float64{0.0, 1.0}, [2]float64{1.75 * math.Pi, 1.75 * math.Pi}, true},
		}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line1, "x", tt.line2), func(t *testing.T) {
			line1 := MustParseSVGPath(tt.line1).ReverseScanner()
			line2 := MustParseSVGPath(tt.line2).ReverseScanner()
			line1.Scan()
			line2.Scan()

			zs := intersectionLineLine(nil, line1.Start(), line1.End(), line2.Start(), line2.End())
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}

func TestIntersectionLineQuad(t *testing.T) {
	var tts = []struct {
		line, quad string
		zs         Intersections
	}{
		// secant
		{"M0 5L10 5", "Q10 5 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false},
		}},

		// tangent
		{"L0 10", "Q10 5 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.5 * math.Pi, Point{2.0, 1.0}.Angle()}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{0.5 * math.Pi, Point{-2.0, 1.0}.Angle()}, true},
		}},
		{"M5 0L5 10", "Q10 5 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},

		// none
		{"M-1 0L-1 10", "Q10 5 0 10", Intersections{}},
	}
	origEpsilon := Epsilon
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.quad), func(t *testing.T) {
			Epsilon = origEpsilon
			line := MustParseSVGPath(tt.line).ReverseScanner()
			quad := MustParseSVGPath(tt.quad).ReverseScanner()
			line.Scan()
			quad.Scan()

			zs := intersectionLineQuad(nil, line.Start(), line.End(), quad.Start(), quad.CP1(), quad.End())
			Epsilon = 3.0 * origEpsilon
			test.T(t, zs, tt.zs)
		})
	}
	Epsilon = origEpsilon
}

func TestIntersectionLineCube(t *testing.T) {
	var tts = []struct {
		line, cube string
		zs         Intersections
	}{
		// secant
		{"M0 5L10 5", "C8 0 8 10 0 10", Intersections{
			{Point{6.0, 5.0}, [2]float64{0.6, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false},
		}},
		{"M0 1L1 1", "C0 2 1 0 1 2", Intersections{ // parallel at intersection
			{Point{0.5, 1.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.0}, false},
		}},
		{"M0 1L1 1", "M0 2C0 0 1 2 1 0", Intersections{ // parallel at intersection
			{Point{0.5, 1.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.0}, false},
		}},
		{"M0 1L1 1", "C0 3 1 -1 1 2", Intersections{ // three intersections
			{Point{0.0791512117, 1.0}, [2]float64{0.0791512117, 0.1726731646}, [2]float64{0.0, 74.05460410 / 180.0 * math.Pi}, false},
			{Point{0.5, 1.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 315 / 180.0 * math.Pi}, false},
			{Point{0.9208487883, 1.0}, [2]float64{0.9208487883, 0.8273268354}, [2]float64{0.0, 74.05460410 / 180.0 * math.Pi}, false},
		}},

		// tangent
		{"L0 10", "C8 0 8 10 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{0.5 * math.Pi, math.Pi}, true},
		}},
		{"M6 0L6 10", "C8 0 8 10 0 10", Intersections{
			{Point{6.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},

		// none
		{"M-1 0L-1 10", "C8 0 8 10 0 10", Intersections{}},
	}
	origEpsilon := Epsilon
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.cube), func(t *testing.T) {
			Epsilon = origEpsilon
			line := MustParseSVGPath(tt.line).ReverseScanner()
			cube := MustParseSVGPath(tt.cube).ReverseScanner()
			line.Scan()
			cube.Scan()

			zs := intersectionLineCube(nil, line.Start(), line.End(), cube.Start(), cube.CP1(), cube.CP2(), cube.End())
			Epsilon = 3.0 * origEpsilon
			test.T(t, zs, tt.zs)
		})
	}
	Epsilon = origEpsilon
}

func TestIntersectionLineEllipse(t *testing.T) {
	var tts = []struct {
		line, arc string
		zs        Intersections
	}{
		// secant
		{"M0 5L10 5", "A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false},
		}},
		{"M0 5L10 5", "A5 5 0 1 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false},
		}},
		{"M0 5L-10 5", "A5 5 0 0 0 0 10", Intersections{
			{Point{-5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{math.Pi, 0.5 * math.Pi}, false},
		}},
		{"M-5 0L-5 -10", "A5 5 0 0 0 -10 0", Intersections{
			{Point{-5.0, -5.0}, [2]float64{0.5, 0.5}, [2]float64{1.5 * math.Pi, math.Pi}, false},
		}},
		{"M0 10L10 10", "A10 5 90 0 1 0 20", Intersections{
			{Point{5.0, 10.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false},
		}},

		// tangent
		{"M-5 0L-15 0", "A5 5 0 0 0 -10 0", Intersections{
			{Point{-10.0, 0.0}, [2]float64{0.5, 1.0}, [2]float64{math.Pi, 0.5 * math.Pi}, true},
		}},
		{"M-5 0L-15 0", "A5 5 0 0 1 -10 0", Intersections{
			{Point{-10.0, 0.0}, [2]float64{0.5, 1.0}, [2]float64{math.Pi, 1.5 * math.Pi}, true},
		}},
		{"L0 10", "A10 5 0 0 1 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{0.5 * math.Pi, math.Pi}, true},
		}},
		{"M5 0L5 10", "A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},
		{"M-5 0L-5 10", "A5 5 0 0 0 0 10", Intersections{
			{Point{-5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},
		{"M5 0L5 20", "A10 5 90 0 1 0 20", Intersections{
			{Point{5.0, 10.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},
		{"M4 3L0 3", "M2 3A1 1 0 0 0 4 3", Intersections{
			{Point{2.0, 3.0}, [2]float64{0.5, 0.0}, [2]float64{math.Pi, 0.5 * math.Pi}, true},
			{Point{4.0, 3.0}, [2]float64{0.0, 1.0}, [2]float64{math.Pi, 1.5 * math.Pi}, true},
		}},

		// see #200, at intersection the arc angle is deviated towards positive angle
		{"M0 -0.7L1 -0.7", "M-0.7 0A0.7 0.7 0 0 1 0.7 0", Intersections{
			{Point{0.0, -0.7}, [2]float64{0.0, 0.5}, [2]float64{0.0, 0.0}, true},
		}},

		// none
		{"M6 0L6 10", "A5 5 0 0 1 0 10", Intersections{}},
		{"M10 5L15 5", "A5 5 0 0 1 0 10", Intersections{}},
		{"M6 0L6 20", "A10 5 90 0 1 0 20", Intersections{}},
	}
	origEpsilon := Epsilon
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.arc), func(t *testing.T) {
			Epsilon = origEpsilon
			line := MustParseSVGPath(tt.line).ReverseScanner()
			arc := MustParseSVGPath(tt.arc).ReverseScanner()
			line.Scan()
			arc.Scan()

			rx, ry, rot, large, sweep := arc.Arc()
			phi := rot * math.Pi / 180.0
			cx, cy, theta0, theta1 := ellipseToCenter(arc.Start().X, arc.Start().Y, rx, ry, phi, large, sweep, arc.End().X, arc.End().Y)

			zs := intersectionLineEllipse(nil, line.Start(), line.End(), Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			Epsilon = 3.0 * origEpsilon
			test.T(t, zs, tt.zs)
		})
	}
	Epsilon = origEpsilon
}

func TestIntersectionEllipseEllipse(t *testing.T) {
	var tts = []struct {
		arc, arc2 string
		zs        Intersections
	}{
		// secant
		{"M5 0A5 5 0 0 1 -5 0", "M-10 -5A10 10 0 0 1 -10 15", Intersections{
			{Point{0.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{math.Pi, 0.5 * math.Pi}, false},
		}},

		// tangent
		{"A5 5 0 0 1 0 10", "M10 0A5 5 0 0 0 10 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},

		// fully parallel
		{"A5 5 0 0 1 0 10", "A5 5 0 0 1 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.0, 0.0}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{math.Pi, math.Pi}, true},
		}},
		{"A5 5 0 0 1 0 10", "M0 10A5 5 0 0 0 0 0", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 1.0}, [2]float64{0.0, math.Pi}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 0.0}, [2]float64{math.Pi, 0.0}, true},
		}},

		// partly parallel
		{"A5 5 0 0 1 0 10", "A5 5 0 0 1 5 5", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.0, 0.0}, true},
			{Point{5.0, 5.0}, [2]float64{0.5, 1.0}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
		}},
		{"A5 5 0 0 1 0 10", "M5 5A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.0}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{math.Pi, math.Pi}, true},
		}},
		{"A5 5 0 0 1 0 10", "M5 5A5 5 0 0 1 -5 5", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.0}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 0.5}, [2]float64{math.Pi, math.Pi}, true},
		}},
	}
	origEpsilon := Epsilon
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.arc, "x", tt.arc2), func(t *testing.T) {
			Epsilon = origEpsilon
			arc := MustParseSVGPath(tt.arc).ReverseScanner()
			arc2 := MustParseSVGPath(tt.arc2).ReverseScanner()
			arc.Scan()
			arc2.Scan()

			rx, ry, rot, large, sweep := arc.Arc()
			phi := rot * math.Pi / 180.0
			cx, cy, theta0, theta1 := ellipseToCenter(arc.Start().X, arc.Start().Y, rx, ry, phi, large, sweep, arc.End().X, arc.End().Y)

			rx2, ry2, rot2, large2, sweep2 := arc2.Arc()
			phi2 := rot2 * math.Pi / 180.0
			cx2, cy2, theta20, theta21 := ellipseToCenter(arc2.Start().X, arc2.Start().Y, rx2, ry2, phi2, large2, sweep2, arc2.End().X, arc2.End().Y)

			zs := intersectionEllipseEllipse(nil, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1, Point{cx2, cy2}, Point{rx2, ry2}, phi2, theta20, theta21)
			Epsilon = 3.0 * origEpsilon
			test.T(t, zs, tt.zs)
		})
	}
	Epsilon = origEpsilon
}

func TestIntersections(t *testing.T) {
	var tts = []struct {
		p, q   string
		zp, zq []PathIntersection
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", []PathIntersection{
			{Point{7.5, 5.0}, 2, 0.5, Point{-1.0, 2.0}.Angle(), true, false, false},
			{Point{2.5, 5.0}, 3, 0.5, Point{-1.0, -2.0}.Angle(), false, false, false},
		}, []PathIntersection{
			{Point{7.5, 5.0}, 1, 0.75, 0.0, false, false, false},
			{Point{2.5, 5.0}, 1, 0.25, 0.0, true, false, false},
		}},
		{"L10 0L5 10z", "M0 -5L10 -5A5 5 0 0 1 0 -5", []PathIntersection{
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, false, true},
		}, []PathIntersection{
			{Point{5.0, 0.0}, 2, 0.5, math.Pi, false, false, true},
		}},
		{"M5 5L0 0", "M-5 0A5 5 0 0 0 5 0", []PathIntersection{
			{Point{5.0 / math.Sqrt(2.0), 5.0 / math.Sqrt(2.0)}, 1, 0.292893219, 1.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{5.0 / math.Sqrt(2.0), 5.0 / math.Sqrt(2.0)}, 1, 0.75, 1.75 * math.Pi, true, false, false},
		}},

		// intersection on one segment endpoint
		{"L0 15", "M5 0L0 5L5 5", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.0, false, false, true},
		}},
		{"L0 15", "M5 0L0 5L-5 5", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, math.Pi, true, false, false},
		}},
		{"L0 15", "M5 5L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, true},
		}},
		{"L0 15", "M-5 5L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, false},
		}},
		{"M5 0L0 5L5 5", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.0, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, true},
		}},
		{"M5 0L0 5L-5 5", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}},
		{"M5 5L0 5L5 0", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, true},
		}},
		{"M-5 5L0 5L5 0", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, false, false},
		}},
		{"L0 10", "M5 0A5 5 0 0 0 0 5A5 5 0 0 0 5 10", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, false, true},
		}},
		{"L0 10", "M5 10A5 5 0 0 1 0 5A5 5 0 0 1 5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"L0 5L5 5", "M5 0A5 5 0 0 0 5 10", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.0, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, true, false, false},
		}},
		{"L0 5L5 5", "M5 10A5 5 0 0 1 5 0", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.0, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 1.5 * math.Pi, false, false, false},
		}},

		// intersection on two segment endpoint
		{"L10 6L20 0", "M0 10L10 6L20 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, -6.0}.Angle(), false, false, true},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, 4.0}.Angle(), false, false, true},
		}},
		{"L10 6L20 0", "M20 10L10 6L0 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, -6.0}.Angle(), false, false, true},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, 4.0}.Angle(), false, false, true},
		}},
		{"M20 0L10 6L0 0", "M0 10L10 6L20 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, -6.0}.Angle(), false, false, true},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, 4.0}.Angle(), false, false, true},
		}},
		{"M20 0L10 6L0 0", "M20 10L10 6L0 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, -6.0}.Angle(), false, false, true},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, 4.0}.Angle(), false, false, true},
		}},
		{"L10 6L20 10", "M0 10L10 6L20 0", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, 4.0}.Angle(), true, false, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, -6.0}.Angle(), false, false, false},
		}},
		{"L10 6L20 10", "M20 0L10 6L0 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, 4.0}.Angle(), false, false, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, 4.0}.Angle(), true, false, false},
		}},
		{"M20 10L10 6L0 0", "M0 10L10 6L20 0", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, -6.0}.Angle(), false, false, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, -6.0}.Angle(), true, false, false},
		}},
		{"M20 10L10 6L0 0", "M20 0L10 6L0 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, -6.0}.Angle(), true, false, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, 4.0}.Angle(), false, false, false},
		}},
		{"M4 1L4 3L0 3", "M3 4L4 3L3 2", []PathIntersection{
			{Point{4.0, 3.0}, 2, 0.0, math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{4.0, 3.0}, 2, 0.0, 1.25 * math.Pi, true, false, false},
		}},
		{"M0 1L4 1L4 3L0 3z", MustParseSVGPath("M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z").Flatten(Tolerance).ToSVG(), []PathIntersection{
			{Point{4.0, 3.0}, 3, 0.0, math.Pi, false, false, false},
			{Point{2.0, 3.0}, 3, 0.5, math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{4.0, 3.0}, 1, 0.0, 262.83296263 * math.Pi / 180.0, true, false, false},
			{Point{2.0, 3.0}, 10, 0.0, 82.83296263 * math.Pi / 180.0, false, false, false},
		}},
		{"M5 1L9 1L9 5L5 5z", MustParseSVGPath("M9 5A4 4 0 0 1 1 5A4 4 0 0 1 9 5z").Flatten(Tolerance).ToSVG(), []PathIntersection{
			{Point{9.0, 5.0}, 3, 0.0, math.Pi, true, false, false},
			{Point{5.0, 1.00828530}, 4, 0.997928675, 1.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{9.0, 5.0}, 1, 0.0, 93.76219714 * math.Pi / 180.0, false, false, false},
			{Point{5.0, 1.00828530}, 26, 0.5, 0.0, true, false, false},
		}},

		// touches / parallel
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.0, false, false, true},
			{Point{2.0, 2.0}, 4, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"L2 0L2 2L0 2z", "M2 0L2 2L4 2L4 0z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, true},
		}},
		{"M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z", []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.0, false, false, true},
			{Point{2.0, 2.0}, 4, 0.0, 1.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, false, true},
		}},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", []PathIntersection{
			{Point{2.0, 1.0}, 2, 0.5, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 1.0}, 1, 0.0, 0.0, false, false, true},
			{Point{2.0, 2.0}, 4, 0.5, 1.5 * math.Pi, false, true, false},
		}},
		{"L2 0L2 2L0 2z", "M2 -1L4 -1L4 1L2 1z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 1.0}, 2, 0.5, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 4, 0.5, 1.5 * math.Pi, false, false, true},
			{Point{2.0, 1.0}, 4, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"L2 0L2 2L0 2z", "M2 -1L4 -1L4 3L2 3z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 4, 0.75, 1.5 * math.Pi, false, false, true},
			{Point{2.0, 2.0}, 4, 0.25, 1.5 * math.Pi, false, true, false},
		}},
		{"M0 -1L2 -1L2 3L0 3z", "M2 0L4 0L4 2L2 2z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.25, 0.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 2, 0.75, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.0, false, false, true},
			{Point{2.0, 2.0}, 4, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"L1 0L1 1zM2 0L1.9 1L1.9 -1z", "L1 0L1 -1zM2 0L1.9 1L1.9 -1z", []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.0, 0.0, false, true, false},
			{Point{1.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.0, 0.0, false, true, false},
			{Point{1.0, 0.0}, 2, 0.0, 1.5 * math.Pi, false, false, true},
		}},

		// head-on collisions
		{"M2 0L2 2L0 2", "M4 2L2 2L2 4", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.5 * math.Pi, false, false, true},
		}},
		{"M0 2Q2 4 2 2Q4 2 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"M0 2C0 4 2 4 2 2C4 2 4 4 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"M0 2A1 1 0 0 0 2 2A1 1 0 0 1 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"M0 2A1 1 0 0 1 2 2A1 1 0 0 1 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"M0 2A1 1 0 0 1 2 2A1 1 0 0 1 2 4", "M2 0L2 2L0 2", []PathIntersection{
			{Point{0.0, 2.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 2.0}, 2, 1.0, math.Pi, false, false, true},
			{Point{2.0, 2.0}, 2, 0.0, math.Pi, true, false, false},
		}},
		{"M0 1L4 1L4 3L0 3z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", []PathIntersection{
			{Point{4.0, 3.0}, 3, 0.0, math.Pi, false, false, false},
			{Point{2.0, 3.0}, 3, 0.5, math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{4.0, 3.0}, 1, 0.0, 1.5 * math.Pi, true, false, false},
			{Point{2.0, 3.0}, 2, 0.0, 0.5 * math.Pi, false, false, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", []PathIntersection{
			{Point{3.0, 2.0}, 2, 0.5, 0.5 * math.Pi, false, false, false},
			{Point{3.0, 4.0}, 3, 0.0, math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{3.0, 2.0}, 1, 0.5, math.Pi, true, false, false},
			{Point{3.0, 4.0}, 2, 0.5, 0.0, false, false, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M3 0A1 1 0 0 0 1 0A1 1 0 0 0 3 0z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, false},
			{Point{3.0, 0.0}, 2, 0.0, 0.5 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 2, 0.0, 0.5 * math.Pi, true, false, false},
			{Point{3.0, 0.0}, 1, 0.0, 1.5 * math.Pi, false, false, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0A1 1 0 0 0 -1 0A1 1 0 0 0 1 0z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, true},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L1 -1L0 0z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, true},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 1.5 * math.Pi, false, false, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L0 0L1 -1z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, true},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, math.Pi, false, false, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L2 0L1 1z", []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.5, 0.0, false, false, true},
			{Point{1.0, 1.0}, 4, 0.75, 1.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.75 * math.Pi, false, false, true},
			{Point{1.0, 1.0}, 3, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L1 1L2 0z", []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.5, 0.0, false, false, true},
			{Point{1.0, 1.0}, 4, 0.75, 1.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 3, 0.0, math.Pi, false, true, false},
			{Point{1.0, 1.0}, 2, 0.0, 1.75 * math.Pi, false, false, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L2 1L0 1z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, false},
			{Point{1.0, 1.0}, 4, 0.75, 1.5 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.25 * math.Pi, true, false, false},
			{Point{1.0, 1.0}, 2, 0.5, math.Pi, false, false, false},
		}},

		// intersection with parallel lines
		{"L0 15", "M5 0L0 5L0 10L5 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, false, true},
		}},
		{"L0 15", "M5 0L0 5L0 10L-5 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 3, 0.0, 0.75 * math.Pi, true, false, false},
		}},
		{"L0 15", "M5 15L0 10L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 3, 0.0, 1.75 * math.Pi, false, false, true},
			{Point{0.0, 10.0}, 2, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"L0 15", "M5 15L0 10L0 5L-5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 3, 0.0, 1.25 * math.Pi, true, false, false},
			{Point{0.0, 10.0}, 2, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"L0 10L-5 15", "M5 0L0 5L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.5, 0.5 * math.Pi, false, false, true},
		}},
		{"L0 10L5 15", "M5 0L0 5L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.0, 0.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.5, 0.5 * math.Pi, true, false, false},
		}},
		{"L0 10L-5 15", "M0 15L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, true},
			{Point{0.0, 10.0}, 1, 0.5, 1.5 * math.Pi, false, true, false},
		}},
		{"L0 10L5 15", "M0 15L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.0, 0.25 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, false},
			{Point{0.0, 10.0}, 1, 0.5, 1.5 * math.Pi, false, true, false},
		}},
		{"L5 5L5 10L0 15", "M10 0L5 5L5 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 3, 0.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 2, 0.5, 0.5 * math.Pi, false, false, true},
		}},
		{"L5 5L5 10L10 15", "M10 0L5 5L5 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 2, 0.5, 0.5 * math.Pi, true, false, false},
		}},
		{"L5 5L5 10L0 15", "M10 0L5 5L5 10L10 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 3, 0.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, false, true},
		}},
		{"L5 5L5 10L10 15", "M10 0L5 5L5 10L0 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{5.0, 10.0}, 3, 0.0, 0.75 * math.Pi, true, false, false},
		}},
		{"L5 5L5 10L10 15L5 20", "M10 0L5 5L5 10L10 15L10 20", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{10.0, 15.0}, 4, 0.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{10.0, 15.0}, 4, 0.0, 0.5 * math.Pi, false, false, true},
		}},
		{"L5 5L5 10L10 15L5 20", "M10 20L10 15L5 10L5 5L10 0", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
			{Point{10.0, 15.0}, 4, 0.0, 0.75 * math.Pi, false, false, true},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 4, 0.0, 1.75 * math.Pi, false, false, true},
			{Point{10.0, 15.0}, 2, 0.0, 1.25 * math.Pi, false, true, false},
		}},
		{"L2 0L2 1L0 1z", "M1 0L3 0L3 1L1 1z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.5, 0.0, false, true, false},
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, true, false, false},
			{Point{2.0, 1.0}, 3, 0.0, math.Pi, false, true, false},
			{Point{1.0, 1.0}, 3, 0.5, math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, true, false},
			{Point{2.0, 0.0}, 1, 0.5, 0.0, false, false, false},
			{Point{2.0, 1.0}, 3, 0.5, math.Pi, false, true, false},
			{Point{1.0, 1.0}, 4, 0.0, 1.5 * math.Pi, true, false, false},
		}},

		// bugs
		{"M67.89174682452696 63.79390646055095L67.89174682452696 63.91890646055095L59.89174682452683 50.06250000000001", "M68.10825317547533 63.79390646055193L67.89174682452919 63.91890646055186M67.89174682452672 63.918906460550865L59.891746824526074 50.06250000000021", []PathIntersection{
			{Point{67.89174682452696, 63.918906460551284}, 1, 1.0, 90.0 * math.Pi / 180.0, false, false, true},
			{Point{67.89174682452696, 63.918906460553146}, 1, 1.0, 90.0 * math.Pi / 180.0, false, false, true},
			{Point{67.89174682452696, 63.91890646055095}, 2, 0.0, 240.0 * math.Pi / 180.0, false, false, true},
			{Point{67.89174682452793, 63.918906460552606}, 2, 0.0, 240.0 * math.Pi / 180.0, false, false, true},
			{Point{59.89174682452683, 50.06250000000001}, 2, 1.0, 240.0 * math.Pi / 180.0, false, false, true},
		}, []PathIntersection{
			{Point{67.89174682452696, 63.918906460553146}, 1, 1.0, 150.0 * math.Pi / 180.0, false, false, true},
			{Point{67.89174682452696, 63.918906460551284}, 3, 0.0, 240.0 * math.Pi / 180.0, false, false, true},
			{Point{67.89174682452793, 63.918906460552606}, 1, 1.0, 150.0 * math.Pi / 180.0, false, false, true},
			{Point{67.89174682452696, 63.91890646055095}, 3, 0.0, 240.0 * math.Pi / 180.0, false, false, true},
			{Point{59.89174682452683, 50.06250000000001}, 3, 1.0, 240.0 * math.Pi / 180.0, false, false, true},
		}},
	}
	origEpsilon := Epsilon
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			Epsilon = origEpsilon
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)

			zp, zq := p.Collisions(q)

			Epsilon = 3.0 * origEpsilon
			test.T(t, zp, tt.zp)
			test.T(t, zq, tt.zq)
		})
	}
	Epsilon = origEpsilon
}

func TestSelfIntersections(t *testing.T) {
	var tts = []struct {
		p  string
		zs []PathIntersection
	}{
		//{"L10 10L10 0L0 10z", Intersections{ {Point{5.0, 5.0}, 1, 3, 0.5, 0.5, 0.25 * math.Pi, 0.75 * math.Pi, BintoA, NoParallel, false},
		//}},

		/// bugs
		{"M3.512162397982181 1.239754268684486L3.3827323986701674 1.1467946944092953L3.522449858001167 1.2493787337129587A0.21166666666666667 0.21166666666666667 0 0 1 3.5121623979821806 1.2397542686844856z", []PathIntersection{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			zs, _ := pathIntersections(p, nil, true, true)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}

func TestPathCut(t *testing.T) {
	var tts = []struct {
		p, q string
		rs   []string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z",
			[]string{"M7.5 5L5 10L2.5 5", "M2.5 5L0 0L10 0L7.5 5"},
		},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L5 1L5 3L1 3z",
			[]string{"M2 1L2 2L1 2", "M1 2L0 2L0 0L2 0L2 1", "M5 2L4 2L4 1", "M4 1L4 0L6 0L6 2L5 2"},
		},
		{"L2 0M2 1L4 1L4 3L2 3zM0 4L2 4", "M1 -1L1 5",
			[]string{"L1 0", "M1 0L2 0M2 1L4 1L4 3L2 3zM0 4L1 4", "M1 4L2 4"},
		},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)

			rs := p.Cut(q)
			test.T(t, len(rs), len(tt.rs))
			for i := range rs {
				test.T(t, rs[i], MustParseSVGPath(tt.rs[i]))
			}
		})
	}
}

func TestPathSettle(t *testing.T) {
	var tts = []struct {
		fillRule FillRule
		p        string
		r        string
	}{
		// non-intersecting
		{NonZero, "L2 0L2 2L0 2z", "L2 0L2 2L0 2z"}, //  ccw
		{NonZero, "L0 2L2 2L2 0z", "L2 0L2 2L0 2z"}, // !ccw

		// self-intersections
		{NonZero, "L10 10L10 0L0 10z", "M5 5L0 10L0 0zM5 5L10 0L10 10z"},
		{NonZero, "L10 10L0 10L10 0z", "M5 5L0 0L10 0zM5 5L10 10L0 10z"},
		{NonZero, "L10 10L20 0L20 10L10 0L0 10z", "M5 5L0 10L0 0zM5 5L10 0L15 5L10 10zM15 5L20 0L20 10z"},

		// single path with inner part doubly winded
		{NonZero, "M0 2L6 2L4 4L1 1L5 1L2 4z", "M2 2L1 1L5 1L4 2L6 2L4 4L3 3L2 4L0 2z"}, //  ccw
		{NonZero, "M0 2L2 4L5 1L1 1L4 4L6 2z", "M3 3L2 4L0 2L2 2L1 1L5 1L4 2L6 2L4 4z"}, // !ccw

		// two paths with overlapping part either zero or doubly winded
		{NonZero, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "M10 5L15 5L15 15L5 15L5 10L0 10L0 0L10 0z"},
		{NonZero, "L4 0L4 5L6 5L6 10L0 10zM2 2L8 2L8 8L2 8z", "M4 2L8 2L8 8L6 8L6 10L0 10L0 0L4 0z"},                          //  ccwA  ccwB
		{NonZero, "L4 0L4 5L6 5L6 10L0 10zM2 2L2 8L8 8L8 2z", "M4 2L2 2L2 8L6 8L6 10L0 10L0 0L4 0zM4 2L8 2L8 8L6 8L6 5L4 5z"}, //  ccwA !ccwB
		{NonZero, "L0 10L6 10L6 5L4 5L4 0zM2 2L8 2L8 8L2 8z", "M6 8L6 10L0 10L0 0L4 0L4 2L2 2L2 8zM6 8L6 5L4 5L4 2L8 2L8 8z"}, // !ccwA  ccwB
		{NonZero, "L0 10L6 10L6 5L4 5L4 0zM2 2L2 8L8 8L8 2z", "M6 8L6 10L0 10L0 0L4 0L4 2L8 2L8 8z"},                          // !ccwA !ccwB

		// same but flipped on Y (different starting vertex)
		{NonZero, "L6 0L6 5L4 5L4 10L0 10zM2 2L8 2L8 8L2 8z", "M6 2L8 2L8 8L4 8L4 10L0 10L0 0L6 0z"},                          //  ccwA  ccwB
		{NonZero, "L6 0L6 5L4 5L4 10L0 10zM2 2L2 8L8 8L8 2z", "M6 2L2 2L2 8L4 8L4 10L0 10L0 0L6 0zM6 2L8 2L8 8L4 8L4 5L6 5z"}, //  ccwA !ccwB
		{NonZero, "L0 10L4 10L4 5L6 5L6 0zM2 2L8 2L8 8L2 8z", "M4 8L4 10L0 10L0 0L6 0L6 2L2 2L2 8zM4 8L4 5L6 5L6 2L8 2L8 8z"}, // !ccwA  ccwB
		{NonZero, "L0 10L4 10L4 5L6 5L6 0zM2 2L2 8L8 8L8 2z", "M4 8L4 10L0 10L0 0L6 0L6 2L8 2L8 8z"},                          // !ccwA !ccwB

		// multiple paths
		{NonZero, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "M10 5L15 5L15 15L5 15L5 10L0 10L0 0L10 0z"},
		{EvenOdd, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "M10 5L15 5L15 15L5 15L5 10L0 10L0 0L10 0zM10 5L5 5L5 10L10 10z"},
		{NonZero, "L4 0L4 4L0 4zM-1 1L1 1L1 3L-1 3zM3 1L5 1L5 3L3 3zM4.5 1.5L5.5 1.5L5.5 2.5L4.5 2.5z", "M0 1L0 0L4 0L4 1L5 1L5 1.5L5.5 1.5L5.5 2.5L5 2.5L5 3L4 3L4 4L0 4L0 3L-1 3L-1 1L0 1z"},
		{EvenOdd, "L4 0L4 4L0 4zM-1 1L1 1L1 3L-1 3zM3 1L5 1L5 3L3 3zM4.5 1.5L5.5 1.5L5.5 2.5L4.5 2.5z", "M0 1L0 0L4 0L4 1L5 1L5 1.5L5.5 1.5L5.5 2.5L5 2.5L5 3L4 3L4 4L0 4L0 3L-1 3L-1 1L0 1zM0 1L0 3L1 3L1 1zM4 1L3 1L3 3L4 3zM5 1.5L4.5 1.5L4.5 2.5L5 2.5z"},

		// tangent
		//{NonZero, "L5 5L10 0L10 10L5 5L0 10z", "M5 5L0 10L0 0zM5 5L10 0L10 10z"},
		{NonZero, "L2 2L3 0zM1 0L2 2L4 0L4 -1L1 -1z", "M2 2L0 0L1 0L1 -1L4 -1L4 0z"},
		//{NonZero, "L2 2L3 0zM1 0L2 2L4 0L4 3L1 3z", "M1 1L0 0L1 0zM1 1L2 2L4 0L4 3L1 3zM2 2L1 0L3 0z"},

		// parallel segments
		//{NonZero, "L1 0L1 1L0 1zM1 0L2 0L2 1L1 1z", ""},
		//{NonZero, "L1 0L1 1L0 1zM1 0L1 1L2 1L2 0z", ""},
		//{NonZero, "L10 0L5 2L5 8L0 10L10 10L5 8L5 2z", "M5 8L5 2L0 0L10 0L5 2L5 8L10 10L0 10z"},
		//{NonZero, "L10 0L10 10L5 7.5L10 5L10 15L0 15z", ""},
		//{EvenOdd, "L10 0L10 10L5 7.5L10 5L10 15L0 15z", ""},
		//{NonZero, "L10 0L10 5L0 10L0 5L10 10L10 15L0 15z", "M5 7.5L0 5L0 0L10 0L10 5zM5 7.5L10 10L10 15L0 15L0 10z"},
		//{NonZero, "L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", "M7.5 7.5L5 5L2.5 7.5L5 10zM0 15L0 0L10 0L10 5L7.5 7.5L10 10L10 15zM2.5 7.5L0 5L0 10z"},
		//{EvenOdd, "L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", ""},
		//{"L3 0L3 1L0 1zM1 0L1 1L2 1L2 0z", "M1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"},

		// non flat
		//{"M0 1L4 1L4 3L0 3zM4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 0 2 3L0 3L0 1L4 1zM4 3A1 1 0 0 1 2 3z"},

		// special cases
		{NonZero, "M1 4L0 2L1 0L2 0L2 4zM1 3L0 2L1 1z", "M1 4L0 2L1 0L2 0L2 4z"},              // tangent left-most endpoint
		{NonZero, "M0 2L1 0L2 0L2 4L1 4zM0 2L1 1L1 3z", "M0 2L1 0L2 0L2 4L1 4z"},              // tangent left-most endpoint
		{NonZero, "M0 2L1 0L2 0L2 4L1 4zM0 2L1 3L1 1z", "M0 2L1 0L2 0L2 4L1 4zM0 2L1 3L1 1z"}, // tangent left-most endpoint
		{NonZero, "M0 2L1 0L2 1L1 3zM0 2L1 1L2 3L1 4z", "M0 2L1 0L2 1L1.5 2L2 3L1 4z"},        // secant left-most endpoint
		//{NonZero, "M0 2L1 0L2 1L1 3zM0 2L1 4L2 3L1 1z", "M0 2L1 3L1.5 2L2 3L1 4zM0 2L1 0L2 1L1.5 2L1 1z"}, // secant left-most endpoint
		//{NonZero, "L2 0L2 2L0 2L0 1L-1 2L-2 2L-1 2L0 1z", ""}, // parallel left-most endpoint
		//{NonZero, "L0 1L-1 2L0 1z", ""},                       // all parallel

		// example from Subramaniam's thesis
		{NonZero, "L5.5 10L4.5 10L9 1.5L1 8L9 8L1.6 2L3 1L5 7L8 0z", "M3.349892008639309 6.090712742980562L0 0L8 0L6.479452054794521 3.547945205479452L9 1.5L6.592324805339266 6.047830923248053L9 8L5.5588235294117645 8L4.990463215258855 9.073569482288828L4.3999999999999995 8L1 8L3.349892008639309 6.090712742980561zM3.9753086419753085 3.9259259259259265L3 1L1.6 2L3.975308641975309 3.925925925925927zM4.990463215258855 9.073569482288828L5.5 10L4.5 10L4.990463215258855 9.073569482288828z"},
		{EvenOdd, "L5.5 10L4.5 10L9 1.5L1 8L9 8L1.6 2L3 1L5 7L8 0z", "M3.349892008639309 6.090712742980562L0 0L8 0L6.479452054794521 3.547945205479452L9 1.5L6.592324805339266 6.047830923248053L9 8L5.5588235294117645 8L4.990463215258855 9.073569482288828L4.3999999999999995 8L1 8L3.349892008639309 6.090712742980561zM3.349892008639309 6.090712742980562L4.4 8L5.5588235294117645 8L6.592324805339265 6.047830923248053L5.713467048710601 5.335243553008596L6.47945205479452 3.5479452054794525L4.995837669094692 4.753381893860562L3.975308641975309 3.925925925925927L4.409836065573771 5.229508196721312L3.349892008639309 6.090712742980561zM5.713467048710601 5.335243553008596L5 7L4.409836065573771 5.22950819672131L4.9958376690946915 4.753381893860562L5.713467048710601 5.335243553008596zM3.9753086419753085 3.9259259259259265L3 1L1.6 2L3.975308641975309 3.925925925925927zM4.990463215258855 9.073569482288828L5.5 10L4.5 10L4.990463215258855 9.073569482288828z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			test.T(t, p.Settle(tt.fillRule), MustParseSVGPath(tt.r))
		})
	}
}

func TestPathAnd(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M7.5 5L5 10L2.5 5z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "M7.5 5L5 10L2.5 5z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M7.5 5L5 10L2.5 5z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M7.5 5L5 10L2.5 5z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", ""},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", ""},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", ""},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "M2 2L8 2L5 8z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M2 2L8 2L5 8z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L1 1L0 1L0 0z"},
		{"L2 0L2 1L0 1z", "L0 1L1 1L1 0z", "M1 0L1 1L0 1L0 0z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L1 1L0 1L0 0z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M2 0L2 1L1 1L1 0z"},
		{"L3 0L3 1L0 1z", "M1 0L1 1L2 1L2 0z", "M2 0L2 1L1 1L1 0z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L1 1L0 1L0 0z"},
		{"L1 0L1 1L0 1z", "L2 0L2 2L0 2z", "M1 0L1 1L0 1L0 0z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 0L5 10z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 -10L20 0L10 10z"},
		{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "Q10 0 10 -10Q10 0 20 0Q10 0 10 10Q10 0 0 0z"},

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 4L6 6L6 7L4 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M6 3L6 4L4 6L4 3z"},
		{"L3 0L3 1L0 1z", "M1 -0.1L2 -0.1L2 1.1L1 1.1z", "M1 0L2 0L2 1L1 1z"},

		// fully parallel
		{"L10 0L10 5L7.5 7.5L5 5L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z", "M7.5 7.5L5 10L2.5 7.5L5 5z", ""},

		// subpaths on A cross at the same point on B
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1.1L1.6 0.5L1 -0.1L1 -1L2 -1L2 2L1 2z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L1.5 0.5L1 0L1 -1L2 -1L2 2L1 2z"},
		//{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 -0.1L0.4 0.5L1 1.1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M1 -0.1L0.4 0.5L1 1.1L1 2L0 2L0 -1L1 -1z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0L0.5 0.5L1 1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M1 0L0.5 0.5L1 1L1 2L0 2L0 -1L1 -1z"},
		//{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z"},
		{"L1 0L1.1 0.5L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 0L1.1 0.5L1 1zM1 1L1.5 0.5L1 0L1 -1L2 -1L2 2L1 2z"},
		{"L1 0L0.9 0.5L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L1.5 0.5L1 0L1 -1L2 -1L2 2L1 2z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L3 2L2 2L2 3L1 3L1 1zM3 3L3 4L2 4L2 3z"},                                      // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M2 0L2 1L1 1L1 0zM2 2L2 3L1 3L1 2z"},                                 // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M2 0L2 1L1 1L1 0zM0 2L2 2L2 3L0 3z"},                                 // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M2 0L2 1L1 1L1 0zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M2 0L2 1L1 1L1 0z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z"},                                                  // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z"},                                                  // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "M5 1L5 9"},                 // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", ""},                       // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "M5 5L5 10"},               // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 10"},                     // touch
		{"M5 0L10 0L10 5", "L10 0L10 10L0 10z", ""},                   // boundary
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L5 0L5 5"},                 // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "M1 1L2 0L8 0L9 1"}, // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", ""},               // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                            // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L5 0L5 1L6 0"},        // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M6 0L7 1"},            // touch with parallel

		// bugs
		//{"M23 15L24 15L24 16L23 16zM23.4 14L24.4 14L24.4 15L23.4 15z", "M15 16A1 1 0 0 1 16 15L24 15A1 1 0 0 1 25 16L25 24A1 1 0 0 1 24 25L16 25A1 1 0 0 1 15 24z", "M23 15L24 15L24 16L23 16z"},
		//{"M23 15L24 15L24 16L23 16zM24 15.4L25 15.4L25 16.4L24 16.4z", "M14 14L24 14L24 24L14 24z", "M23 15L24 15L24 16L23 16z"},
		{"M0 1L2 1L2 2L0 2zM3 1L5 1L5 2L3 2z", "M1 0L4 0L4 3L1 3z", "M1 1L2 1L2 2L1 2zM4 1L4 2L3 2L3 1z"},
		{"L5 0L5 5L0 5zM1 1L1 4L4 4L4 1z", "M3 2L6 2L6 3L3 3z", "M5 2L5 3L4 3L4 2z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.And(q)
			test.T(t, r, MustParseSVGPath(tt.r))
		})
	}
}

func TestPathOr(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M7.5 5L10 5L5 15L0 5L2.5 5L0 0L10 0z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "M7.5 5L10 5L5 15L0 5L2.5 5L0 0L10 0z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M7.5 5L10 5L5 15L0 5L2.5 5L0 0L10 0z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M7.5 5L10 5L5 15L0 5L2.5 5L0 0L10 0z"},
		{"M0 1L4 1L4 3L0 3z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 1 2 3L0 3L0 1L4 1z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "M4 0L4 2L0 2L0 0z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "M2 1L4 1L4 3L2 3L2 2L0 2L0 0L2 0z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "L10 0L5 10z"},
		{"M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z", "M10 0L5 5L0 0L5 -5z", "M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "A10 10 0 0 1 20 0A10 10 0 0 1 0 0z"},
		{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10z"},

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M9 3L9 7L1 7L1 3z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M9 3L9 7L1 7L1 3z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M2 0L2 1L0 1L0 0z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M2 0L2 1L0 1L0 0z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M3 0L3 1L0 1L0 0z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M2 0L2 2L0 2L0 0z"},

		// fully parallel
		{"L10 0L10 5L7.5 7.5L5 5L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z", "M7.5 7.5L5 10L2.5 7.5L5 5z", "M7.5 7.5L10 10L10 15L0 15L0 0L10 0L10 5z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L4 1L4 2L3 2L3 3L4 3L4 2L5 2L5 5L2 5L2 4L1 4L1 3L0 3L0 1L1 1L1 0L3 0z"}, // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M3 0L3 1L0 1L0 0zM3 2L3 3L0 3L0 2z"},                           // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M3 0L3 1L0 1L0 0zM0 2L2 2L2 3L0 3z"},                           // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M3 0L3 1L0 1L0 0zM0 2L2 2L2 3L0 3z"},           // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M3 0L3 1L0 1L0 0zM4 2L4 3L0 3L0 2z"},                           // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z"},                                                                 // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "L7 0L7 4L0 4z"},                                                                 // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 1L5 9"},                     // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM15 1L15 9"},                 // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 5L5 10M5 10L5 15"},         // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L10 10"},                     // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L5 0L5 5"},                 // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 1L2 0L8 0L9 1"},     // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0L8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L10 0"},                       // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zL5 0L5 1L6 0M6 0L7 -1"},   // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zL5 0L5 -1L6 0M6 0L7 1"},   // touch with parallel
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.Or(q)
			test.T(t, r, MustParseSVGPath(tt.r))
		})
	}
}

func TestPathXor(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M7.5 5L2.5 5L0 0L10 0zM7.5 5L10 5L5 15L0 5L2.5 5L5 10z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "M7.5 5L2.5 5L0 0L10 0zM7.5 5L10 5L5 15L0 5L2.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M7.5 5L2.5 5L0 0L10 0zM7.5 5L10 5L5 15L0 5L2.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M7.5 5L2.5 5L0 0L10 0zM7.5 5L10 5L5 15L0 5L2.5 5L5 10z"},
		{"M0 1L4 1L4 3L0 3z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 0 2 3L0 3L0 1L4 1zM4 3A1 1 0 0 1 2 3z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "M2 0L4 0L4 2L2 2zM2 2L0 2L0 0L2 0z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "M2 1L4 1L4 3L2 3zM2 2L0 2L0 0L2 0z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M2 2L5 8L8 2zM0 0L10 0L5 10z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", ""},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 10L20 0L10 -10zA10 10 0 0 1 20 0A10 10 0 0 1 0 0z"},
		{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10zQ10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z"},

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 7L1 7L1 3L4 3zM4 3L9 3L9 7L6 7L6 6L4 4z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L4 7L1 7L1 3zM6 3L9 3L9 7L4 7L4 6L6 4z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M1 0L1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L2 0L2 2L0 2L0 1L1 1z"},
		{"L2 0L0 2z", "L2 2L0 2z", "L2 0L1 1zM0 2L1 1L2 2z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L1 1L1 0L3 0zM3 1L4 1L4 2L3 2zM3 2L3 3L2 3L2 4L1 4L1 3L2 3L2 2zM3 3L4 3L4 2L5 2L5 5L2 5L2 4L3 4zM1 3L0 3L0 1L1 1z"}, // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1zM1 2L1 3L0 3L0 2zM2 2L3 2L3 3L2 3z"},                                     // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"},                                                                       // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1zM0 2L2 2L2 3L0 3zM0.1 2.1L0.1 2.9L1.9 2.9L1.9 2.1z"},     // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1zM2 2L4 2L4 3L2 3zM2 3L0 3L0 2L2 2z"},                                     // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                                           // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "M1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1zM0 0L7 0L7 4L0 4z"},                                                                       // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                             // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM15 1L15 9"},                 // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 10L5 15"},                  // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                               // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                             // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                     // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0L8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM6 0L7 -1"},               // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zL5 0L5 -1L6 0"},           // touch with parallel

		// multiple intersections in one point
		{"L2 0L2 2L4 2L4 4L2 2L0 2z", "L2 0L2 2L4 4L2 4L2 2L0 2z", "M2 2L4 2L4 4zM2 2L4 4L2 4z"},
		{"L2 0L2 1.9L4 1.9L4 4L2 2L0 2z", "L2 0L2 2L4 4L1.9 4L1.9 2L0 2z", "M2 1.9L4 1.9L4 4L2 2zM1.9 2L2 2L4 4L1.9 4z"}, // simple version of above
		{"L2 0L2 1L3 1L2 0L4 0L4 2L0 2z", "L2 0L2 2L0 2z", "M2 1L3 1L2 0L4 0L4 2L2 2z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.Xor(q)
			test.T(t, r, MustParseSVGPath(tt.r))
		})
	}
}

func TestPathNot(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M7.5 5L2.5 5L0 0L10 0z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "M7.5 5L2.5 5L0 0L10 0z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M7.5 5L2.5 5L0 0L10 0z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M7.5 5L2.5 5L0 0L10 0z"},

		{"M0 5L10 5L5 15z", "L10 0L5 10z", "M2.5 5L5 10L7.5 5L10 5L5 15L0 5z"},
		{"M0 5L10 5L5 15z", "L5 10L10 0z", "M2.5 5L5 10L7.5 5L10 5L5 15L0 5z"},
		{"M0 5L5 15L10 5z", "L10 0L5 10z", "M2.5 5L5 10L7.5 5L10 5L5 15L0 5z"},
		{"M0 5L5 15L10 5z", "L5 10L10 0z", "M2.5 5L5 10L7.5 5L10 5L5 15L0 5z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "M2 2L0 2L0 0L2 0z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "M2 2L0 2L0 0L2 0z"},
		{"M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z"},
		{"M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10z"},
		{"M0 10L10 10L5 20z", "L10 0L5 10z", "M0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", ""},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", ""},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", ""},
		{"A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 -10L20 0L10 10z", "A10 10 0 0 1 20 0A10 10 0 0 1 0 0zL10 10L20 0L10 -10z"},
		{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10zQ10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z"},
		{"Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10z", ""},

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 7L1 7L1 3L4 3z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L4 7L1 7L1 3z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", ""},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M1 0L1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L2 0L2 2L0 2L0 1L1 1z"},

		// subpaths on A cross at the same point on B
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L0 1L0 0L1 0z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L0 1L0 0L1 0z"},
		//{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M1 1.1L1 -0.1L1.6 0.5z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M1 1L1 0L1.5 0.5z"},
		//{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M1 0.1L1.4 0.5L1 0.9z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L1 1L1 0L3 0zM3 2L3 3L2 3L2 4L1 4L1 3L2 3L2 2z"},                                               // different winding
		{"M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 0L3 0L3 4L1 4z", "M3 2L3 1L4 1L4 2zM1 3L0 3L0 1L1 1zM2 4L3 4L3 3L4 3L4 2L5 2L5 5L2 5z"},                              // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M1 0L1 1L0 1L0 0zM1 2L1 3L0 3L0 2z"},                                                  // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M1 0L1 1L0 1L0 0z"},                                                                   // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M1 0L1 1L0 1L0 0zM0 2L2 2L2 3L0 3zM0.1 2.1L0.1 2.9L1.9 2.9L1.9 2.1z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M1 0L1 1L0 1L0 0zM2 3L0 3L0 2L2 2z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                      // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", ""},                                                                                                     // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", ""},                             // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "M15 1L15 9"},                 // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "M5 10L5 15"},                  // cross
		{"L10 10", "L10 0L10 10L0 10z", ""},                               // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", ""},                             // touch with parallel
		{"M1 1L2 0L8 0L9 9", "L10 0L10 10L0 10z", ""},                     // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "M1 -1L2 0L8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                                // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "M6 0L7 -1"},               // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L5 0L5 -1L6 0"},           // touch with parallel
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.Not(q)
			test.T(t, r, MustParseSVGPath(tt.r))
		})
	}
}

func TestPathDivideBy(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M7.5 5L2.5 5L0 0L10 0zM7.5 5L5 10L2.5 5z"},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L5 1L5 3L1 3z", "M2 1L1 1L1 2L0 2L0 0L2 0zM2 1L2 2L1 2L1 1zM5 2L5 1L4 1L4 0L6 0L6 2zM5 2L4 2L4 1L5 1z"},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L1 3L5 3L5 1z", "M2 1L1 1L1 2L0 2L0 0L2 0zM2 1L2 2L1 2L1 1zM5 2L5 1L4 1L4 0L6 0L6 2zM5 2L4 2L4 1L5 1z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			test.T(t, p.DivideBy(q), MustParseSVGPath(tt.r))
		})
	}
}
