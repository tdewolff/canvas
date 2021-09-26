package canvas

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestIntersectionLineLine(t *testing.T) {
	var tts = []struct {
		line1, line2 string
		zs           intersections
	}{
		// secant
		{"M2 0L2 3", "M1 2L3 2", intersections{{Point{2.0, 2.0}, 0, 0, 2.0 / 3.0, 0.5, false}}},
		{"M2 0L2 3", "M2 2L3 2", intersections{{Point{2.0, 2.0}, 0, 0, 2.0 / 3.0, 0.0, false}}},
		{"M2 0L2 2", "M2 2L3 2", intersections{{Point{2.0, 2.0}, 0, 0, 1.0, 0.0, false}}},

		// parallel
		{"M2 0L2 2", "M2 0L2 2", intersections{}},
		{"M2 0L2 2", "M2 1L2 3", intersections{}},
		{"M2 0L2 2", "M2 2L2 4", intersections{}},

		// none
		{"M2 0L2 1", "M3 0L3 1", intersections{}},
		{"M2 0L2 1", "M0 2L1 2", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line1, "x", tt.line2), func(t *testing.T) {
			line1Segs := MustParseSVG(tt.line1).Segments()
			line2Segs := MustParseSVG(tt.line2).Segments()
			line1 := line1Segs[len(line1Segs)-1]
			line2 := line2Segs[len(line2Segs)-1]

			zs := intersectionLineLine(line1.Start, line1.End, line2.Start, line2.End)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
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

func TestIntersectionLineQuad(t *testing.T) {
	var tts = []struct {
		line, quad string
		zs         intersections
	}{
		// secant
		{"M0 5L10 5", "Q10 5 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, false}}},
		{"M0 0L0 10", "Q10 5 0 10", intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, false},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, false},
		}},

		// tangent
		{"M5 0L5 10", "Q10 5 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, true}}},

		// none
		{"M-1 0L-1 10", "Q10 5 0 10", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.quad), func(t *testing.T) {
			lineSegs := MustParseSVG(tt.line).Segments()
			quadSegs := MustParseSVG(tt.quad).Segments()
			line := lineSegs[len(lineSegs)-1]
			quad := quadSegs[len(quadSegs)-1]

			zs := intersectionLineQuad(line.Start, line.End, quad.Start, quad.CP1(), quad.End)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}

func TestIntersectionLineCube(t *testing.T) {
	var tts = []struct {
		line, cube string
		zs         intersections
	}{
		// secant
		{"M0 5L10 5", "C8 0 8 10 0 10", intersections{{Point{6.0, 5.0}, 0, 0, 0.6, 0.5, false}}},
		{"M0 0L0 10", "C8 0 8 10 0 10", intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, false},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, false},
		}},

		// tangent
		{"M6 0L6 10", "C8 0 8 10 0 10", intersections{{Point{6.0, 5.0}, 0, 0, 0.5, 0.5, true}}},

		// none
		{"M-1 0L-1 10", "C8 0 8 10 0 10", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.cube), func(t *testing.T) {
			lineSegs := MustParseSVG(tt.line).Segments()
			cubeSegs := MustParseSVG(tt.cube).Segments()
			line := lineSegs[len(lineSegs)-1]
			cube := cubeSegs[len(cubeSegs)-1]

			zs := intersectionLineCube(line.Start, line.End, cube.Start, cube.CP1(), cube.CP2(), cube.End)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}

func TestIntersectionLineEllipse(t *testing.T) {
	var tts = []struct {
		line, arc string
		zs        intersections
	}{
		// secant
		{"M0 5L10 5", "A5 5 0 0 1 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, false}}},
		{"M0 5L10 5", "A5 5 0 1 1 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, false}}},
		{"M0 5L-10 5", "A5 5 0 0 0 0 10", intersections{{Point{-5.0, 5.0}, 0, 0, 0.5, 0.5, false}}},
		{"M-5 0L-5 -10", "A5 5 0 0 0 -10 0", intersections{{Point{-5.0, -5.0}, 0, 0, 0.5, 0.5, false}}},
		{"M-5 0L-15 0", "A5 5 0 0 0 -10 0", intersections{{Point{-10.0, 0.0}, 0, 0, 0.5, 1.0, false}}},
		{"M0 0L0 10", "A10 5 0 0 1 0 10", intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, false},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, false},
		}},
		{"M0 10L10 10", "A10 5 90 0 1 0 20", intersections{{Point{5.0, 10.0}, 0, 0, 0.5, 0.5, false}}},

		// tangent
		{"M5 0L5 10", "A5 5 0 0 1 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, true}}},
		{"M-5 0L-5 10", "A5 5 0 0 0 0 10", intersections{{Point{-5.0, 5.0}, 0, 0, 0.5, 0.5, true}}},
		{"M5 0L5 20", "A10 5 90 0 1 0 20", intersections{{Point{5.0, 10.0}, 0, 0, 0.5, 0.5, true}}},

		// none
		{"M6 0L6 10", "A5 5 0 0 1 0 10", intersections{}},
		{"M10 5L15 5", "A5 5 0 0 1 0 10", intersections{}},
		{"M6 0L6 20", "A10 5 90 0 1 0 20", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.arc), func(t *testing.T) {
			lineSegs := MustParseSVG(tt.line).Segments()
			arcSegs := MustParseSVG(tt.arc).Segments()
			line := lineSegs[len(lineSegs)-1]
			arc := arcSegs[len(arcSegs)-1]

			rx, ry, rot, large, sweep := arc.Arc()
			phi := rot * math.Pi / 180.0
			cx, cy, theta0, theta1 := ellipseToCenter(arc.Start.X, arc.Start.Y, rx, ry, phi, large, sweep, arc.End.X, arc.End.Y)

			zs := intersectionLineEllipse(line.Start, line.End, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}

func TestIntersect(t *testing.T) {
	var tts = []struct {
		p, q string
		zs   intersections
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", intersections{
			{Point{7.5, 5.0}, 2, 1, 0.5, 0.75, false},
			{Point{2.5, 5.0}, 3, 1, 0.5, 0.25, false},
		}},
		{"L10 0L5 10z", "M0 -5L10 -5A5 5 0 0 1 0 -5", intersections{
			{Point{5.0, 0.0}, 1, 2, 0.5, 0.5, true},
		}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)

			zs := p.Intersections(q)
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
		r    []string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", []string{"M2.5 5L0 0L10 0L7.5 5", "M7.5 5L5 10L2.5 5"}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)

			rs := p.Cut(q)
			test.T(t, len(rs), len(tt.r))
			for i := range tt.r {
				test.T(t, rs[i], MustParseSVG(tt.r[i]))
			}
		})
	}
}
