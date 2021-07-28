package canvas

import (
	"fmt"
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

func TestIntersectionLineQuad(t *testing.T) {
	var tts = []struct {
		line, quad string
		zs         intersections
	}{
		{"M0 5L10 5", "Q10 5 0 10", intersections{{Point{5.0, 5.0}, 0.5, 0.5, true}}},
		{"M5 0L5 10", "Q10 5 0 10", intersections{{Point{5.0, 5.0}, 0.5, 0.5, false}}},
		{"M0 0L0 10", "Q10 5 0 10", intersections{
			{Point{0.0, 0.0}, 0.0, 0.0, true},
			{Point{0.0, 10.0}, 1.0, 1.0, true},
		}},
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
		{"M0 5L10 5", "C8 0 8 10 0 10", intersections{{Point{6.0, 5.0}, 0.6, 0.5, true}}},
		{"M6 0L6 10", "C8 0 8 10 0 10", intersections{{Point{6.0, 5.0}, 0.5, 0.5, false}}},
		{"M0 0L0 10", "C8 0 8 10 0 10", intersections{
			{Point{0.0, 0.0}, 0.0, 0.0, true},
			{Point{0.0, 10.0}, 1.0, 1.0, true},
		}},
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
		{"M0 5L10 5", "A5 5 0 0 1 0 10", intersections{{Point{5.0, 5.0}, 0.5, 0.0, true}}},
		{"M0 0L0 10", "A10 5 0 0 1 0 10", intersections{{Point{6.0, 5.0}, 1.0, 0.0, true}}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.arc), func(t *testing.T) {
			lineSegs := MustParseSVG(tt.line).Segments()
			arcSegs := MustParseSVG(tt.arc).Segments()
			line := lineSegs[len(lineSegs)-1]
			arc := arcSegs[len(arcSegs)-1]

			rx, ry, rot, large, sweep := arc.Arc()
			cx, cy, theta0, theta1 := ellipseToCenter(arc.Start.X, arc.Start.Y, rx, ry, rot, large, sweep, arc.End.X, arc.End.Y)

			zs := intersectionLineEllipse(line.Start, line.End, Point{cx, cy}, Point{rx, ry}, rot, theta0, theta1)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}
