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
			{Point{2.0, 2.0}, 0, 0, 2.0 / 3.0, 0.5, 0.5 * math.Pi, 0.0, AintoB, NoParallel, false},
		}},

		// tangent
		{"M2 0L2 3", "M2 2L3 2", Intersections{
			{Point{2.0, 2.0}, 0, 0, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 0.0, AintoB, NoParallel, true},
		}},
		{"M2 0L2 2", "M2 2L3 2", Intersections{
			{Point{2.0, 2.0}, 0, 0, 1.0, 0.0, 0.5 * math.Pi, 0.0, AintoB, NoParallel, true},
		}},
		{"L2 2", "M0 4L2 2", Intersections{
			{Point{2.0, 2.0}, 0, 0, 1.0, 1.0, 0.25 * math.Pi, 1.75 * math.Pi, AintoB, NoParallel, true},
		}},
		{"L10 5", "M0 10L10 5", Intersections{
			{Point{10.0, 5.0}, 0, 0, 1.0, 1.0, Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle(), AintoB, NoParallel, true},
		}},
		{"M10 5L20 10", "M10 5L20 0", Intersections{
			{Point{10.0, 5.0}, 0, 0, 0.0, 0.0, Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle(), AintoB, NoParallel, true},
		}},

		// parallel
		{"L2 2", "M3 3L5 5", Intersections{}},
		{"L2 2", "M-1 1L1 3", Intersections{}},
		{"L2 2", "M2 2L4 4", Intersections{
			{Point{2.0, 2.0}, 0, 0, 1.0, 0.0, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
		}},
		{"L2 2", "M-2 -2L0 0", Intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 1.0, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
		}},
		{"L2 2", "L2 2", Intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
			{Point{2.0, 2.0}, 0, 0, 1.0, 1.0, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
		}},
		{"L4 4", "M2 2L6 6", Intersections{
			{Point{2.0, 2.0}, 0, 0, 0.5, 0.0, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
			{Point{4.0, 4.0}, 0, 0, 1.0, 0.5, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
		}},
		{"L4 4", "M-2 -2L2 2", Intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.5, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
			{Point{2.0, 2.0}, 0, 0, 0.5, 1.0, 0.25 * math.Pi, 0.25 * math.Pi, 0, Parallel, true},
		}},

		// none
		{"M2 0L2 1", "M3 0L3 1", Intersections{}},
		{"M2 0L2 1", "M0 2L1 2", Intersections{}},

		// bugs
		{"M21.590990257669734 18.40900974233027L22.651650429449557 17.348349570550447", "M21.23743686707646 18.762563132923542L21.590990257669738 18.409009742330266", Intersections{
			{Point{21.590990257669734, 18.40900974233027}, 0, 0, 0.0, 1.0, 1.75 * math.Pi, 1.75 * math.Pi, 0, Parallel, true},
		}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line1, "x", tt.line2), func(t *testing.T) {
			line1 := MustParseSVG(tt.line1).ReverseScanner()
			line2 := MustParseSVG(tt.line2).ReverseScanner()
			line1.Scan()
			line2.Scan()

			zs := Intersections{}
			zs = zs.LineLine(line1.Start(), line1.End(), line2.Start(), line2.End())
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
			{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},

		// tangent
		{"L0 10", "Q10 5 0 10", Intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.5 * math.Pi, Point{2.0, 1.0}.Angle(), AintoB, NoParallel, true},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, 0.5 * math.Pi, Point{-2.0, 1.0}.Angle(), BintoA, NoParallel, true},
		}},
		{"M5 0L5 10", "Q10 5 0 10", Intersections{
			{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, 0, Parallel, true},
		}},

		// none
		{"M-1 0L-1 10", "Q10 5 0 10", Intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.quad), func(t *testing.T) {
			line := MustParseSVG(tt.line).ReverseScanner()
			quad := MustParseSVG(tt.quad).ReverseScanner()
			line.Scan()
			quad.Scan()

			zs := Intersections{}
			zs = zs.LineQuad(line.Start(), line.End(), quad.Start(), quad.CP1(), quad.End())
			test.T(t, len(zs), len(tt.zs))
			reset := setEpsilon(3.0 * Epsilon)
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
			reset()
		})
	}
}

func TestIntersectionLineCube(t *testing.T) {
	var tts = []struct {
		line, cube string
		zs         Intersections
	}{
		// secant
		{"M0 5L10 5", "C8 0 8 10 0 10", Intersections{
			{Point{6.0, 5.0}, 0, 0, 0.6, 0.5, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"M0 1L1 1", "C0 2 1 0 1 2", Intersections{ // parallel at intersection
			{Point{0.5, 1.0}, 0, 0, 0.5, 0.5, 0.0, math.Atan(2.0), BintoA, NoParallel, false}, // direction is incorrect on purpose
		}},
		{"M0 1L1 1", "C0 3 1 -1 1 2", Intersections{ // three intersections
			{Point{0.0791512117, 1.0}, 0, 0, 0.0791512117, 0.1726731646, 0.0, 74.05460410 / 180.0 * math.Pi, BintoA, NoParallel, false},
			{Point{0.5, 1.0}, 0, 0, 0.5, 0.5, 0.0, 315 / 180.0 * math.Pi, AintoB, NoParallel, false},
			{Point{0.9208487883, 1.0}, 0, 0, 0.9208487883, 0.8273268354, 0.0, 74.05460410 / 180.0 * math.Pi, BintoA, NoParallel, false},
		}},

		// tangent
		{"L0 10", "C8 0 8 10 0 10", Intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.5 * math.Pi, 0.0, AintoB, NoParallel, true},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, 0.5 * math.Pi, math.Pi, BintoA, NoParallel, true},
		}},
		{"M6 0L6 10", "C8 0 8 10 0 10", Intersections{
			{Point{6.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, 0, Parallel, true},
		}},

		// none
		{"M-1 0L-1 10", "C8 0 8 10 0 10", Intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.cube), func(t *testing.T) {
			line := MustParseSVG(tt.line).ReverseScanner()
			cube := MustParseSVG(tt.cube).ReverseScanner()
			line.Scan()
			cube.Scan()

			zs := Intersections{}
			zs = zs.LineCube(line.Start(), line.End(), cube.Start(), cube.CP1(), cube.CP2(), cube.End())
			test.T(t, len(zs), len(tt.zs))
			reset := setEpsilon(3.0 * Epsilon)
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
			reset()
		})
	}
}

func TestIntersectionLineEllipse(t *testing.T) {
	var tts = []struct {
		line, arc string
		zs        Intersections
	}{
		// secant
		{"M0 5L10 5", "A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"M0 5L10 5", "A5 5 0 1 1 0 10", Intersections{
			{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"M0 5L-10 5", "A5 5 0 0 0 0 10", Intersections{
			{Point{-5.0, 5.0}, 0, 0, 0.5, 0.5, math.Pi, 0.5 * math.Pi, AintoB, NoParallel, false},
		}},
		{"M-5 0L-5 -10", "A5 5 0 0 0 -10 0", Intersections{
			{Point{-5.0, -5.0}, 0, 0, 0.5, 0.5, 1.5 * math.Pi, math.Pi, AintoB, NoParallel, false},
		}},
		{"M0 10L10 10", "A10 5 90 0 1 0 20", Intersections{
			{Point{5.0, 10.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},

		// tangent
		{"M-5 0L-15 0", "A5 5 0 0 0 -10 0", Intersections{
			{Point{-10.0, 0.0}, 0, 0, 0.5, 1.0, math.Pi, 0.5 * math.Pi, AintoB, NoParallel, true},
		}},
		{"M-5 0L-15 0", "A5 5 0 0 1 -10 0", Intersections{
			{Point{-10.0, 0.0}, 0, 0, 0.5, 1.0, math.Pi, 1.5 * math.Pi, BintoA, NoParallel, true},
		}},
		{"L0 10", "A10 5 0 0 1 0 10", Intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.5 * math.Pi, 0.0, AintoB, NoParallel, true},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, 0.5 * math.Pi, math.Pi, BintoA, NoParallel, true},
		}},
		{"M5 0L5 10", "A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, 0, Parallel, true},
		}},
		{"M-5 0L-5 10", "A5 5 0 0 0 0 10", Intersections{
			{Point{-5.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, 0, Parallel, true},
		}},
		{"M5 0L5 20", "A10 5 90 0 1 0 20", Intersections{
			{Point{5.0, 10.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, 0, Parallel, true},
		}},
		{"M4 3L0 3", "M2 3A1 1 0 0 0 4 3", Intersections{
			{Point{2.0, 3.0}, 0, 0, 0.5, 0.0, math.Pi, 0.5 * math.Pi, AintoB, NoParallel, true},
			{Point{4.0, 3.0}, 0, 0, 0.0, 1.0, math.Pi, 1.5 * math.Pi, BintoA, NoParallel, true},
		}},

		// see #200, at intersection the arc angle is deviated towards positive angle
		{"M0 -0.7L1 -0.7", "M-0.7 0A0.7 0.7 0 0 1 0.7 0", Intersections{
			{Point{0.0, -0.7}, 0, 0, 0.0, 0.5, 0.0, 0.0, BintoA, NoParallel, true},
		}},

		// none
		{"M6 0L6 10", "A5 5 0 0 1 0 10", Intersections{}},
		{"M10 5L15 5", "A5 5 0 0 1 0 10", Intersections{}},
		{"M6 0L6 20", "A10 5 90 0 1 0 20", Intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.arc), func(t *testing.T) {
			line := MustParseSVG(tt.line).ReverseScanner()
			arc := MustParseSVG(tt.arc).ReverseScanner()
			line.Scan()
			arc.Scan()

			rx, ry, rot, large, sweep := arc.Arc()
			phi := rot * math.Pi / 180.0
			cx, cy, theta0, theta1 := ellipseToCenter(arc.Start().X, arc.Start().Y, rx, ry, phi, large, sweep, arc.End().X, arc.End().Y)

			zs := Intersections{}
			zs = zs.LineEllipse(line.Start(), line.End(), Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			test.T(t, len(zs), len(tt.zs))
			reset := setEpsilon(3.0 * Epsilon)
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
			reset()
		})
	}
}

func TestIntersections(t *testing.T) {
	var tts = []struct {
		p, q string
		zs   Intersections
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", Intersections{
			{Point{7.5, 5.0}, 2, 1, 0.5, 0.75, Point{-1.0, 2.0}.Angle(), 0.0, AintoB, NoParallel, false},
			{Point{2.5, 5.0}, 3, 1, 0.5, 0.25, Point{-1.0, -2.0}.Angle(), 0.0, BintoA, NoParallel, false},
		}},
		{"L10 0L5 10z", "M0 -5L10 -5A5 5 0 0 1 0 -5", Intersections{}},
		{"M5 5L0 0", "M-5 0A5 5 0 0 0 5 0", Intersections{
			{Point{5.0 / math.Sqrt(2.0), 5.0 / math.Sqrt(2.0)}, 1, 1, 0.292893219, 0.75, 1.25 * math.Pi, 1.75 * math.Pi, BintoA, NoParallel, false},
		}},

		// intersection on one segment endpoint
		{"L0 15", "M5 0L0 5L5 5", Intersections{}},
		{"L0 15", "M5 0L0 5L-5 5", Intersections{
			{Point{0.0, 5.0}, 1, 2, 1.0 / 3.0, 0.0, 0.5 * math.Pi, math.Pi, BintoA, NoParallel, false},
		}},
		{"L0 15", "M5 5L0 5L5 0", Intersections{}},
		{"L0 15", "M-5 5L0 5L5 0", Intersections{
			{Point{0.0, 5.0}, 1, 2, 1.0 / 3.0, 0.0, 0.5 * math.Pi, 1.75 * math.Pi, AintoB, NoParallel, false},
		}},
		{"M5 0L0 5L5 5", "L0 15", Intersections{}},
		{"M5 0L0 5L-5 5", "L0 15", Intersections{
			{Point{0.0, 5.0}, 2, 1, 0.0, 1.0 / 3.0, math.Pi, 0.5 * math.Pi, AintoB, NoParallel, false},
		}},
		{"M5 5L0 5L5 0", "L0 15", Intersections{}},
		{"M-5 5L0 5L5 0", "L0 15", Intersections{
			{Point{0.0, 5.0}, 2, 1, 0.0, 1.0 / 3.0, 1.75 * math.Pi, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"L0 10", "M5 0A5 5 0 0 0 0 5A5 5 0 0 0 5 10", Intersections{}},
		{"L0 10", "M5 10A5 5 0 0 1 0 5A5 5 0 0 1 5 0", Intersections{}},
		{"L0 5L5 5", "M5 0A5 5 0 0 0 5 10", Intersections{
			{Point{0.0, 5.0}, 2, 1, 0.0, 0.5, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"L0 5L5 5", "M5 10A5 5 0 0 1 5 0", Intersections{
			{Point{0.0, 5.0}, 2, 1, 0.0, 0.5, 0.0, 1.5 * math.Pi, AintoB, NoParallel, false},
		}},

		// intersection on two segment endpoint
		{"L10 6L20 0", "M0 10L10 6L20 10", Intersections{}},
		{"L10 6L20 0", "M20 10L10 6L0 10", Intersections{}},
		{"M20 0L10 6L0 0", "M0 10L10 6L20 10", Intersections{}},
		{"M20 0L10 6L0 0", "M20 10L10 6L0 10", Intersections{}},
		{"L10 6L20 10", "M0 10L10 6L20 0", Intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{10.0, 4.0}.Angle(), Point{10.0, -6.0}.Angle(), AintoB, NoParallel, false},
		}},
		{"L10 6L20 10", "M20 0L10 6L0 10", Intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{10.0, 4.0}.Angle(), Point{-10.0, 4.0}.Angle(), BintoA, NoParallel, false},
		}},
		{"M20 10L10 6L0 0", "M0 10L10 6L20 0", Intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{-10.0, -6.0}.Angle(), Point{10.0, -6.0}.Angle(), BintoA, NoParallel, false},
		}},
		{"M20 10L10 6L0 0", "M20 0L10 6L0 10", Intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{-10.0, -6.0}.Angle(), Point{-10.0, 4.0}.Angle(), AintoB, NoParallel, false},
		}},
		{"M4 1L4 3L0 3", "M3 4L4 3L3 2", Intersections{
			{Point{4.0, 3.0}, 2, 2, 0.0, 0.0, math.Pi, 1.25 * math.Pi, BintoA, NoParallel, false},
		}},
		{"M0 1L4 1L4 3L0 3z", MustParseSVG("M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z").Flatten(Tolerance).ToSVG(), Intersections{
			{Point{4.0, 3.0}, 3, 1, 0.0, 0.0, math.Pi, 262.01783160 * math.Pi / 180.0, BintoA, NoParallel, false},
			{Point{2.0, 3.0}, 3, 13, 0.5, 0.0, math.Pi, 82.01783160 * math.Pi / 180.0, AintoB, NoParallel, false},
		}},
		{"M5 1L9 1L9 5L5 5z", MustParseSVG("M9 5A4 4 0 0 1 1 5A4 4 0 0 1 9 5z").Flatten(Tolerance).ToSVG(), Intersections{
			{Point{5.0, 1.0}, 1, 37, 0.0, 0.0, 0.0, 4.02145240 * math.Pi / 180.0, BintoA, NoParallel, false},
			{Point{9.0, 5.0}, 3, 1, 0.0, 0.0, math.Pi, 94.02145240 * math.Pi / 180.0, AintoB, NoParallel, false},
		}},

		// touches / parallel
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", Intersections{
			{Point{2.0, 0.0}, 2, 1, 0.0, 0.0, 0.5 * math.Pi, 0.0, AintoB, AParallel, true},
			{Point{2.0, 2.0}, 3, 4, 0.0, 0.0, math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"L2 0L2 2L0 2z", "M2 0L2 2L4 2L4 0z", Intersections{
			{Point{2.0, 0.0}, 2, 1, 0.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, true},
			{Point{2.0, 2.0}, 3, 2, 0.0, 0.0, math.Pi, 0.0, AintoB, NoParallel, true},
		}},
		{"M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z", Intersections{
			{Point{2.0, 0.0}, 1, 2, 0.0, 0.0, 0.0, 0.5 * math.Pi, BintoA, BParallel, true},
			{Point{2.0, 2.0}, 4, 3, 0.0, 0.0, 1.5 * math.Pi, math.Pi, AintoB, AParallel, true},
		}},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", Intersections{
			{Point{2.0, 1.0}, 2, 1, 0.5, 0.0, 0.5 * math.Pi, 0.0, AintoB, AParallel, true},
			{Point{2.0, 2.0}, 3, 4, 0.0, 0.5, math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"L2 0L2 2L0 2z", "M2 -1L4 -1L4 1L2 1z", Intersections{
			{Point{2.0, 0.0}, 2, 4, 0.0, 0.5, 0.5 * math.Pi, 1.5 * math.Pi, AintoB, AParallel, true},
			{Point{2.0, 1.0}, 2, 4, 0.5, 0.0, 0.5 * math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"L2 0L2 2L0 2z", "M2 -1L4 -1L4 3L2 3z", Intersections{
			{Point{2.0, 0.0}, 2, 4, 0.0, 0.75, 0.5 * math.Pi, 1.5 * math.Pi, AintoB, AParallel, true},
			{Point{2.0, 2.0}, 3, 4, 0.0, 0.25, math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"M0 -1L2 -1L2 3L0 3z", "M2 0L4 0L4 2L2 2z", Intersections{
			{Point{2.0, 0.0}, 2, 1, 0.25, 0.0, 0.5 * math.Pi, 0.0, AintoB, AParallel, true},
			{Point{2.0, 2.0}, 2, 4, 0.75, 0.0, 0.5 * math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"L1 0L1 1zM2 0L1.9 1L1.9 -1z", "L1 0L1 -1zM2 0L1.9 1L1.9 -1z", Intersections{
			{Point{0.0, 0.0}, 1, 1, 0.0, 0.0, 0.0, 0.0, BintoA, Parallel, true},
			{Point{1.0, 0.0}, 2, 2, 0.0, 0.0, 0.5 * math.Pi, 1.5 * math.Pi, AintoB, NoParallel, true},
		}},

		// head-on collisions
		{"M2 0L2 2L0 2", "M4 2L2 2L2 4", Intersections{}},
		{"M0 2Q2 4 2 2Q4 2 2 4", "M2 4L2 2L4 2", Intersections{
			{Point{2.0, 2.0}, 2, 2, 0.0, 0.0, 0.0, 0.0, AintoB, NoParallel, false},
		}},
		{"M0 2C0 4 2 4 2 2C4 2 4 4 2 4", "M2 4L2 2L4 2", Intersections{
			{Point{2.0, 2.0}, 2, 2, 0.0, 0.0, 0.0, 0.0, AintoB, NoParallel, false},
		}},
		{"M0 2A1 1 0 0 0 2 2A1 1 0 0 1 2 4", "M2 4L2 2L4 2", Intersections{
			{Point{2.0, 2.0}, 2, 2, 0.0, 0.0, 0.0, 0.0, AintoB, NoParallel, false},
		}},
		{"M0 2A1 1 0 0 1 2 2A1 1 0 0 1 2 4", "M2 4L2 2L4 2", Intersections{
			{Point{2.0, 2.0}, 2, 2, 0.0, 0.0, 0.0, 0.0, AintoB, NoParallel, false},
		}},
		{"M0 2A1 1 0 0 1 2 2A1 1 0 0 1 2 4", "M2 0L2 2L0 2", Intersections{
			{Point{2.0, 2.0}, 2, 2, 0.0, 0.0, 0.0, math.Pi, BintoA, NoParallel, false},
		}},
		{"M0 1L4 1L4 3L0 3z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", Intersections{
			{Point{4.0, 3.0}, 3, 1, 0.0, 0.0, math.Pi, 1.5 * math.Pi, BintoA, NoParallel, false},
			{Point{2.0, 3.0}, 3, 2, 0.5, 0.0, math.Pi, 0.5 * math.Pi, AintoB, NoParallel, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", Intersections{
			{Point{3.0, 2.0}, 2, 1, 0.5, 0.5, 0.5 * math.Pi, math.Pi, BintoA, NoParallel, false},
			{Point{3.0, 4.0}, 3, 2, 0.0, 0.5, math.Pi, 0.0, AintoB, NoParallel, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M3 0A1 1 0 0 0 1 0A1 1 0 0 0 3 0z", Intersections{
			{Point{1.0, 0.0}, 1, 2, 0.0, 0.0, 0.0, 0.5 * math.Pi, BintoA, NoParallel, false},
			{Point{3.0, 0.0}, 2, 1, 0.0, 0.0, 0.5 * math.Pi, 1.5 * math.Pi, AintoB, NoParallel, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0A1 1 0 0 0 -1 0A1 1 0 0 0 1 0z", Intersections{}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L1 -1L0 0z", Intersections{}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L0 0L1 -1z", Intersections{}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L2 0L1 1z", Intersections{
			{Point{2.0, 0.0}, 1, 2, 0.5, 0.0, 0.0, 0.75 * math.Pi, BintoA, NoParallel, true},
			{Point{1.0, 1.0}, 4, 3, 0.75, 0.0, 1.5 * math.Pi, 1.5 * math.Pi, AintoB, Parallel, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L1 1L2 0z", Intersections{
			{Point{2.0, 0.0}, 1, 3, 0.5, 0.0, 0.0, math.Pi, AintoB, BParallel, true},
			{Point{1.0, 1.0}, 4, 2, 0.75, 0.0, 1.5 * math.Pi, 1.75 * math.Pi, BintoA, AParallel, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L2 1L0 1z", Intersections{
			{Point{1.0, 0.0}, 1, 1, 0.0, 0.0, 0.0, 0.25 * math.Pi, BintoA, NoParallel, false},
			{Point{1.0, 1.0}, 4, 2, 0.75, 0.5, 1.5 * math.Pi, math.Pi, AintoB, NoParallel, false},
		}},

		// intersection with parallel lines
		{"L0 15", "M5 0L0 5L0 10L5 15", Intersections{
			{Point{0.0, 5.0}, 1, 2, 1.0 / 3.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, true},
			{Point{0.0, 10.0}, 1, 3, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 0.25 * math.Pi, AintoB, NoParallel, true},
		}},
		{"L0 15", "M5 0L0 5L0 10L-5 15", Intersections{
			{Point{0.0, 5.0}, 1, 2, 1.0 / 3.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, false},
			{Point{0.0, 10.0}, 1, 3, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 0.75 * math.Pi, BintoA, NoParallel, false},
		}},
		{"L0 15", "M5 15L0 10L0 5L5 0", Intersections{
			{Point{0.0, 5.0}, 1, 3, 1.0 / 3.0, 0.0, 0.5 * math.Pi, 1.75 * math.Pi, AintoB, AParallel, true},
			{Point{0.0, 10.0}, 1, 2, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"L0 15", "M5 15L0 10L0 5L-5 0", Intersections{
			{Point{0.0, 5.0}, 1, 3, 1.0 / 3.0, 0.0, 0.5 * math.Pi, 1.25 * math.Pi, BintoA, AParallel, false},
			{Point{0.0, 10.0}, 1, 2, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 1.5 * math.Pi, BintoA, BParallel, false},
		}},
		{"L0 10L-5 15", "M5 0L0 5L0 15", Intersections{
			{Point{0.0, 5.0}, 1, 2, 0.5, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, true},
			{Point{0.0, 10.0}, 2, 2, 0.0, 0.5, 0.75 * math.Pi, 0.5 * math.Pi, AintoB, NoParallel, true},
		}},
		{"L0 10L5 15", "M5 0L0 5L0 15", Intersections{
			{Point{0.0, 5.0}, 1, 2, 0.5, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, false},
			{Point{0.0, 10.0}, 2, 2, 0.0, 0.5, 0.25 * math.Pi, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"L0 10L-5 15", "M0 15L0 5L5 0", Intersections{
			{Point{0.0, 5.0}, 1, 2, 0.5, 0.0, 0.5 * math.Pi, 1.75 * math.Pi, AintoB, AParallel, true},
			{Point{0.0, 10.0}, 2, 1, 0.0, 0.5, 0.75 * math.Pi, 1.5 * math.Pi, BintoA, BParallel, true},
		}},
		{"L0 10L5 15", "M0 15L0 5L5 0", Intersections{
			{Point{0.0, 5.0}, 1, 2, 0.5, 0.0, 0.5 * math.Pi, 1.75 * math.Pi, AintoB, AParallel, false},
			{Point{0.0, 10.0}, 2, 1, 0.0, 0.5, 0.25 * math.Pi, 1.5 * math.Pi, AintoB, BParallel, false},
		}},
		{"L5 5L5 10L0 15", "M10 0L5 5L5 15", Intersections{
			{Point{5.0, 5.0}, 2, 2, 0.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, true},
			{Point{5.0, 10.0}, 3, 2, 0.0, 0.5, 0.75 * math.Pi, 0.5 * math.Pi, AintoB, NoParallel, true},
		}},
		{"L5 5L5 10L10 15", "M10 0L5 5L5 15", Intersections{
			{Point{5.0, 5.0}, 2, 2, 0.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, false},
			{Point{5.0, 10.0}, 3, 2, 0.0, 0.5, 0.25 * math.Pi, 0.5 * math.Pi, BintoA, NoParallel, false},
		}},
		{"L5 5L5 10L0 15", "M10 0L5 5L5 10L10 15", Intersections{
			{Point{5.0, 5.0}, 2, 2, 0.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, true},
			{Point{5.0, 10.0}, 3, 3, 0.0, 0.0, 0.75 * math.Pi, 0.25 * math.Pi, AintoB, NoParallel, true},
		}},
		{"L5 5L5 10L10 15", "M10 0L5 5L5 10L0 15", Intersections{
			{Point{5.0, 5.0}, 2, 2, 0.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, false},
			{Point{5.0, 10.0}, 3, 3, 0.0, 0.0, 0.25 * math.Pi, 0.75 * math.Pi, BintoA, NoParallel, false},
		}},
		{"L5 5L5 10L10 15L5 20", "M10 0L5 5L5 10L10 15L10 20", Intersections{
			{Point{5.0, 5.0}, 2, 2, 0.0, 0.0, 0.5 * math.Pi, 0.5 * math.Pi, BintoA, Parallel, true},
			{Point{10.0, 15.0}, 4, 4, 0.0, 0.0, 0.75 * math.Pi, 0.5 * math.Pi, AintoB, NoParallel, true},
		}},
		{"L5 5L5 10L10 15L5 20", "M10 20L10 15L5 10L5 5L10 0", Intersections{
			{Point{5.0, 5.0}, 2, 4, 0.0, 0.0, 0.5 * math.Pi, 1.75 * math.Pi, AintoB, AParallel, true},
			{Point{10.0, 15.0}, 4, 2, 0.0, 0.0, 0.75 * math.Pi, 1.25 * math.Pi, BintoA, BParallel, true},
		}},
		{"L2 0L2 1L0 1z", "M1 0L3 0L3 1L1 1z", Intersections{
			{Point{1.0, 0.0}, 1, 1, 0.5, 0.0, 0.0, 0.0, AintoB, Parallel, false},
			{Point{2.0, 0.0}, 2, 1, 0.0, 0.5, 0.5 * math.Pi, 0.0, AintoB, NoParallel, false},
			{Point{2.0, 1.0}, 3, 3, 0.0, 0.5, math.Pi, math.Pi, BintoA, Parallel, false},
			{Point{1.0, 1.0}, 3, 4, 0.5, 0.0, math.Pi, 1.5 * math.Pi, BintoA, NoParallel, false},
		}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)

			zs := p.Intersections(q)
			test.T(t, len(zs), len(tt.zs))
			reset := setEpsilon(3.0 * Epsilon)
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
			reset()
		})
	}
}

func TestSelfIntersections(t *testing.T) {
	var tts = []struct {
		p  string
		zs Intersections
	}{
		{"L10 10L10 0L0 10z", Intersections{
			{Point{5.0, 5.0}, 1, 3, 0.5, 0.5, 0.25 * math.Pi, 0.75 * math.Pi, BintoA, NoParallel, false},
		}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			zs := p.SelfIntersections()
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
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)

			rs := p.Cut(q)
			test.T(t, len(rs), len(tt.rs))
			for i := range rs {
				test.T(t, rs[i], MustParseSVG(tt.rs[i]))
			}
		})
	}
}

func TestPathSettle(t *testing.T) {
	var tts = []struct {
		p string
		r string
	}{
		{"L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "M10 5L15 5L15 15L5 15L5 10L0 10L0 0L10 0z"},
		{"L10 0L10 10L0 10zM5 5L5 15L15 15L15 5z", "M10 5L5 5L5 10L0 10L0 0L10 0zM10 5L15 5L15 15L5 15L5 10L10 10z"},
		{"M0 1L4 1L4 3L0 3zM4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 0 2 3L0 3L0 1L4 1zM4 3A1 1 0 0 1 2 3z"},
		{"L0 1L1 1L1 0z", "L1 0L1 1L0 1z"}, // to CCW
		{"L2 0L2 2L0 2zM1 1L1 3L3 3L3 1z", "M2 1L1 1L1 2L0 2L0 0L2 0zM2 1L3 1L3 3L1 3L1 2L2 2z"}, // to CCW
		{"L0 2L2 2L2 0zM1 1L1 3L3 3L3 1z", "M1 2L0 2L0 0L2 0L2 1L3 1L3 3L1 3z"},                  // to CCW
		{"L0 2L2 2L2 0zM1 1L3 1L3 3L1 3z", "M1 2L0 2L0 0L2 0L2 1L1 1zM1 2L2 2L2 1L3 1L3 3L1 3z"}, // to CCW

		{"L3 0L3 1L0 1zM1 -0.1L1 1.1L2 1.1L2 -0.1z", "M1 0L1 1L0 1L0 0zM1 0L1 -0.1L2 -0.1L2 0zM2 0L3 0L3 1L2 1zM2 1L2 1.1L1 1.1L1 1z"},
		{"L3 0L3 1L0 1zM1 0L1 1L2 1L2 0z", "M1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"}, // containing with parallel touches

		{"L10 10L10 0L0 10z", "M5 5L0 10L0 0L5 5L10 0L10 10z"},                                                                                          // self-intersections
		{"L10 10L20 0L20 10L10 0L0 10z", "M5 5L0 10L0 0L5 5L10 0L15 5L20 0L20 10L15 5L10 10z"},                                                          // self-intersections
		{"L10 0L10 5L0 10L0 5L10 10L10 15L0 15z", "M5 7.5L0 5L0 0L10 0L10 5zM5 7.5L10 10L10 15L0 15L0 10z"},                                             // self-intersections
		{"L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", "M7.5 7.5L5 5L2.5 7.5L5 10zM0 10L0 0L10 0L10 5L7.5 7.5L10 10L10 15L0 15zM2.5 7.5L0 5L0 10z"}, // self-intersections
		{"L10 0L5 2L5 8L0 10L10 10L5 8L5 2z", "M5 8L5 2L0 0L10 0L5 2L5 8L10 10L0 10z"},                                                                  // self-intersections
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			test.T(t, p.Settle(), MustParseSVG(tt.r))
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
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L1 1L0 1L0 0z"},
		{"L2 0L2 1L0 1z", "L0 1L1 1L1 0z", "M1 0L1 1L0 1L0 0z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L1 1L0 1L0 0z"},
		{"L3 0L3 1L0 1z", "M1 -0.1L2 -0.1L2 1.1L1 1.1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M2 0L2 1L1 1L1 0z"},
		{"L3 0L3 1L0 1z", "M1 0L1 1L2 1L2 0z", "M2 0L2 1L1 1L1 0z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L1 1L0 1L0 0z"},
		{"L1 0L1 1L0 1z", "L2 0L2 2L0 2z", "M1 0L1 1L0 1L0 0z"},

		// subpaths on A cross at the same point on B
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1.1L1.6 0.5L1 -0.1L1 -1L2 -1L2 2L1 2z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L1.5 0.5L1 0L1 -1L2 -1L2 2L1 2z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L1 0.9L1.4 0.5L1 0.1L1 -1L2 -1L2 2L1 2z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 -0.1L0.4 0.5L1 1.1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M1 -0.1L0.4 0.5L1 1.1L1 2L0 2L0 -1L1 -1z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0L0.5 0.5L1 1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M1 0L0.5 0.5L1 1L1 2L0 2L0 -1L1 -1z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M1 0L1 0.1L0.6 0.5L1 0.9L1 2L0 2L0 -1L1 -1z"},
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
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L5 0L5 5"},                 // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "M1 1L2 0M8 0L9 1"}, // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", ""},               // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                            // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L5 0L5 1L6 0"},        // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M6 0L7 1"},            // touch with parallel

		// bugs
		{"M23 15L24 15L24 16L23 16zM23.4 14L24.4 14L24.4 15L23.4 15z", "M15 16A1 1 0 0 1 16 15L24 15A1 1 0 0 1 25 16L25 24A1 1 0 0 1 24 25L16 25A1 1 0 0 1 15 24z", "M23 15L24 15L24 16L23 16z"},

		{"M23 15L24 15L24 16L23 16zM24 15.4L25 15.4L25 16.4L24 16.4z", "M14 14L24 14L24 24L14 24z", "M23 15L24 15L24 16L23 16z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)
			r := p.And(q)
			test.T(t, r, MustParseSVG(tt.r))
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
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "M2 0L4 0L4 2L0 2L0 0z"},
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
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L9 3L9 7L1 7L1 3z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M6 3L9 3L9 7L1 7L1 3z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L0 1L0 0z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L2 0L2 1L0 1L0 0z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M2 0L3 0L3 1L0 1L0 0z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L2 0L2 2L0 2L0 0z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L4 1L4 2L3 2L3 3L4 3L4 2L5 2L5 5L2 5L2 4L1 4L1 3L0 3L0 1L1 1L1 0L3 0z"}, // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M2 0L3 0L3 1L0 1L0 0zM2 2L3 2L3 3L0 3L0 2z"},                   // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M2 0L3 0L3 1L0 1L0 0zM0 2L2 2L2 3L0 3z"},                       // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M2 0L3 0L3 1L0 1L0 0zM0 2L2 2L2 3L0 3z"},       // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M2 0L3 0L3 1L0 1L0 0zM2 2L4 2L4 3L0 3L0 2z"},                   // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z"},                                                                 // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "L7 0L7 4L0 4z"},                                                                 // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 1L5 9"},                     // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM15 1L15 9"},                 // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 5L5 10M5 10L5 15"},         // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L10 10"},                     // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L5 0L5 5"},                 // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 1L2 0M8 0L9 1"},     // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0M8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L10 0"},                       // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zL5 0L5 1L6 0M6 0L7 -1"},   // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zL5 0L5 -1L6 0M6 0L7 1"},   // touch with parallel
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)
			r := p.Or(q)
			test.T(t, r, MustParseSVG(tt.r))
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
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 4L4 7L1 7L1 3L4 3zM4 3L9 3L9 7L6 7L6 6L4 4z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L4 7L1 7L1 3zM6 3L9 3L9 7L4 7L4 6L6 4z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M1 0L1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L2 0L2 2L0 2L0 1L1 1z"},
		{"L2 0L0 2z", "L2 2L0 2z", "L2 0L1 1zM0 2L1 1L2 2z"},
		//{"L2 0L2 2L4 2L4 4L2 2L0 2z", "L2 0L2 2L4 4L2 4L2 2L0 2z", "M2 2L4 2L4 4L2 4z"}, // TODO

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
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0M8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM6 0L7 -1"},               // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zL5 0L5 -1L6 0"},           // touch with parallel
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)
			r := p.Xor(q)
			test.T(t, r, MustParseSVG(tt.r))
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
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 4L4 7L1 7L1 3L4 3z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L4 7L1 7L1 3z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", ""},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M1 0L1 0L1 1L0 1L0 0zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M1 0L2 0L2 2L0 2L0 1L1 1z"},

		// subpaths on A cross at the same point on B
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L0 1L0 0L1 0z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 1L0 1L0 0L1 0z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 0L1 1L0 1L0 0z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M1 1.1L1 -0.1L1.6 0.5z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M1 1L1 0L1.5 0.5z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M1 0.1L1.4 0.5L1 0.9z"},

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
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "M1 -1L2 0M8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                                // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "M6 0L7 -1"},               // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L5 0L5 -1L6 0"},           // touch with parallel
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)
			r := p.Not(q)
			test.T(t, r, MustParseSVG(tt.r))
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
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)
			test.T(t, p.DivideBy(q), MustParseSVG(tt.r))
		})
	}
}
