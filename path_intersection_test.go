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
		zs           intersections
	}{
		// secant
		{"M2 0L2 3", "M1 2L3 2", intersections{{Point{2.0, 2.0}, 0, 0, 2.0 / 3.0, 0.5, 0.5 * math.Pi, 0.0, false}}},

		// tangent
		{"M2 0L2 3", "M2 2L3 2", intersections{{Point{2.0, 2.0}, 0, 0, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 0.0, true}}},
		{"M2 0L2 2", "M2 2L3 2", intersections{{Point{2.0, 2.0}, 0, 0, 1.0, 0.0, 0.5 * math.Pi, 0.0, true}}},
		{"L2 2", "M0 4L2 2", intersections{{Point{2.0, 2.0}, 0, 0, 1.0, 1.0, 0.25 * math.Pi, 1.75 * math.Pi, true}}},
		{"L10 5", "M0 10L10 5", intersections{{Point{10.0, 5.0}, 0, 0, 1.0, 1.0, Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle(), true}}},
		{"M10 5L20 10", "M10 5L20 0", intersections{{Point{10.0, 5.0}, 0, 0, 0.0, 0.0, Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle(), true}}},

		// parallel
		{"L2 2", "M3 3L5 5", intersections{}},
		{"L2 2", "M-1 1L1 3", intersections{}},
		{"L2 2", "M2 2L4 4", intersections{}},
		{"L2 2", "M-2 -2L0 0", intersections{}},
		{"L2 2", "L2 2", intersections{}},
		{"L4 4", "M2 2L6 6", intersections{}},
		{"L4 4", "M-2 -2L2 2", intersections{}},

		// none
		{"M2 0L2 1", "M3 0L3 1", intersections{}},
		{"M2 0L2 1", "M0 2L1 2", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line1, "x", tt.line2), func(t *testing.T) {
			line1 := MustParseSVG(tt.line1).ReverseScanner()
			line2 := MustParseSVG(tt.line2).ReverseScanner()
			line1.Scan()
			line2.Scan()

			zs := intersections{}
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
		zs         intersections
	}{
		// secant
		{"M0 5L10 5", "Q10 5 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, false}}},

		// tangent
		{"L0 10", "Q10 5 0 10", intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.5 * math.Pi, Point{2.0, 1.0}.Angle(), true},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, 0.5 * math.Pi, Point{-2.0, 1.0}.Angle(), true},
		}},
		{"M5 0L5 10", "Q10 5 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, true}}},

		// none
		{"M-1 0L-1 10", "Q10 5 0 10", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.quad), func(t *testing.T) {
			line := MustParseSVG(tt.line).ReverseScanner()
			quad := MustParseSVG(tt.quad).ReverseScanner()
			line.Scan()
			quad.Scan()

			zs := intersections{}
			zs = zs.LineQuad(line.Start(), line.End(), quad.Start(), quad.CP1(), quad.End())
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
		{"M0 5L10 5", "C8 0 8 10 0 10", intersections{{Point{6.0, 5.0}, 0, 0, 0.6, 0.5, 0.0, 0.5 * math.Pi, false}}},

		// tangent
		{"L0 10", "C8 0 8 10 0 10", intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.5 * math.Pi, 0.0, true},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, 0.5 * math.Pi, math.Pi, true},
		}},
		{"M6 0L6 10", "C8 0 8 10 0 10", intersections{{Point{6.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, true}}},

		// none
		{"M-1 0L-1 10", "C8 0 8 10 0 10", intersections{}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line, "x", tt.cube), func(t *testing.T) {
			line := MustParseSVG(tt.line).ReverseScanner()
			cube := MustParseSVG(tt.cube).ReverseScanner()
			line.Scan()
			cube.Scan()

			zs := intersections{}
			zs = zs.LineCube(line.Start(), line.End(), cube.Start(), cube.CP1(), cube.CP2(), cube.End())
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
		{"M0 5L10 5", "A5 5 0 0 1 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, false}}},
		{"M0 5L10 5", "A5 5 0 1 1 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, false}}},
		{"M0 5L-10 5", "A5 5 0 0 0 0 10", intersections{{Point{-5.0, 5.0}, 0, 0, 0.5, 0.5, math.Pi, 0.5 * math.Pi, false}}},
		{"M-5 0L-5 -10", "A5 5 0 0 0 -10 0", intersections{{Point{-5.0, -5.0}, 0, 0, 0.5, 0.5, 1.5 * math.Pi, math.Pi, false}}},
		{"M0 10L10 10", "A10 5 90 0 1 0 20", intersections{{Point{5.0, 10.0}, 0, 0, 0.5, 0.5, 0.0, 0.5 * math.Pi, false}}},

		// tangent
		{"M-5 0L-15 0", "A5 5 0 0 0 -10 0", intersections{{Point{-10.0, 0.0}, 0, 0, 0.5, 1.0, math.Pi, 0.5 * math.Pi, true}}},
		{"M-5 0L-15 0", "A5 5 0 0 1 -10 0", intersections{{Point{-10.0, 0.0}, 0, 0, 0.5, 1.0, math.Pi, 1.5 * math.Pi, true}}},
		{"L0 10", "A10 5 0 0 1 0 10", intersections{
			{Point{0.0, 0.0}, 0, 0, 0.0, 0.0, 0.5 * math.Pi, 0.0, true},
			{Point{0.0, 10.0}, 0, 0, 1.0, 1.0, 0.5 * math.Pi, math.Pi, true},
		}},
		{"M5 0L5 10", "A5 5 0 0 1 0 10", intersections{{Point{5.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, true}}},
		{"M-5 0L-5 10", "A5 5 0 0 0 0 10", intersections{{Point{-5.0, 5.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, true}}},
		{"M5 0L5 20", "A10 5 90 0 1 0 20", intersections{{Point{5.0, 10.0}, 0, 0, 0.5, 0.5, 0.5 * math.Pi, 0.5 * math.Pi, true}}},

		// none
		{"M6 0L6 10", "A5 5 0 0 1 0 10", intersections{}},
		{"M10 5L15 5", "A5 5 0 0 1 0 10", intersections{}},
		{"M6 0L6 20", "A10 5 90 0 1 0 20", intersections{}},
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

			zs := intersections{}
			zs = zs.LineEllipse(line.Start(), line.End(), Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			test.T(t, len(zs), len(tt.zs))
			for i := range zs {
				test.T(t, zs[i], tt.zs[i])
			}
		})
	}
}

func TestIntersections(t *testing.T) {
	var tts = []struct {
		p, q string
		zs   intersections
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", intersections{
			{Point{7.5, 5.0}, 2, 1, 0.5, 0.75, Point{-1.0, 2.0}.Angle(), 0.0, false},
			{Point{2.5, 5.0}, 3, 1, 0.5, 0.25, Point{-1.0, -2.0}.Angle(), 0.0, false},
		}},
		{"L10 0L5 10z", "M0 -5L10 -5A5 5 0 0 1 0 -5", intersections{}},

		// intersection on one segment endpoint
		{"L0 15", "M5 0L0 5L5 5", intersections{}},
		{"L0 15", "M5 0L0 5L-5 5", intersections{
			{Point{0.0, 5.0}, 1, 2, 1.0 / 3.0, 0.0, 0.5 * math.Pi, math.Pi, false},
		}},
		{"L0 15", "M5 5L0 5L5 0", intersections{}},
		{"L0 15", "M-5 5L0 5L5 0", intersections{
			{Point{0.0, 5.0}, 1, 2, 1.0 / 3.0, 0.0, 0.5 * math.Pi, 1.75 * math.Pi, false},
		}},
		{"M5 0L0 5L5 5", "L0 15", intersections{}},
		{"M5 0L0 5L-5 5", "L0 15", intersections{
			{Point{0.0, 5.0}, 2, 1, 0.0, 1.0 / 3.0, math.Pi, 0.5 * math.Pi, false},
		}},
		{"M5 5L0 5L5 0", "L0 15", intersections{}},
		{"M-5 5L0 5L5 0", "L0 15", intersections{
			{Point{0.0, 5.0}, 2, 1, 0.0, 1.0 / 3.0, 1.75 * math.Pi, 0.5 * math.Pi, false},
		}},

		// intersection on two segment endpoint
		{"L10 6L20 0", "M0 10L10 6L20 10", intersections{}},
		{"L10 6L20 0", "M20 10L10 6L0 10", intersections{}},
		{"M20 0L10 6L0 0", "M0 10L10 6L20 10", intersections{}},
		{"M20 0L10 6L0 0", "M20 10L10 6L0 10", intersections{}},
		{"L10 6L20 10", "M0 10L10 6L20 0", intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{10.0, 4.0}.Angle(), Point{10.0, -6.0}.Angle(), false},
		}},
		{"L10 6L20 10", "M20 0L10 6L0 10", intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{10.0, 4.0}.Angle(), Point{-10.0, 4.0}.Angle(), false},
		}},
		{"M20 10L10 6L0 0", "M0 10L10 6L20 0", intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{-10.0, -6.0}.Angle(), Point{10.0, -6.0}.Angle(), false},
		}},
		{"M20 10L10 6L0 0", "M20 0L10 6L0 10", intersections{
			{Point{10.0, 6.0}, 2, 2, 0.0, 0.0, Point{-10.0, -6.0}.Angle(), Point{-10.0, 4.0}.Angle(), false},
		}},

		// intersection with parallel lines
		{"L0 15", "M5 0L0 5L0 10L5 15", intersections{}},
		{"L0 15", "M5 0L0 5L0 10L-5 15", intersections{
			{Point{0.0, 10.0}, 1, 3, 2.0 / 3.0, 0.0, 0.5 * math.Pi, 0.75 * math.Pi, false},
		}},
		{"L0 15", "M5 15L0 10L0 5L5 0", intersections{}},
		{"L0 15", "M5 15L0 10L0 5L-5 0", intersections{
			{Point{0.0, 10.0}, 1, 1, 2.0 / 3.0, 1.0, 0.5 * math.Pi, 1.25 * math.Pi, false},
		}},
		{"L0 10L-5 15", "M5 0L0 5L0 15", intersections{}},
		{"L0 10L5 15", "M5 0L0 5L0 15", intersections{
			{Point{0.0, 10.0}, 2, 2, 0.0, 0.5, 0.25 * math.Pi, 0.5 * math.Pi, false},
		}},
		{"L0 10L-5 15", "M0 15L0 5L5 0", intersections{}},
		{"L0 10L5 15", "M0 15L0 5L5 0", intersections{
			{Point{0.0, 10.0}, 2, 1, 0.0, 0.5, 0.25 * math.Pi, 1.5 * math.Pi, false},
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

func TestPathIntersections(t *testing.T) {
	var tts = []struct {
		p, q     string
		zsP, zsQ []Point
	}{
		{"V50H10V0z", "M30 10V40H-10V30H20V20H-10V10z",
			[]Point{{0, 10}, {0, 20}, {0, 30}, {0, 40}, {10, 40}, {10, 30}, {10, 20}, {10, 10}},
			[]Point{{0, 10}, {10, 10}, {10, 40}, {0, 40}, {0, 30}, {10, 30}, {10, 20}, {0, 20}},
		},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)
			zs := pathIntersections(p, q)

			i := 0
			visited := map[int]bool{}
			for _, z0 := range zs {
				if !visited[z0.i] {
					for z := z0; ; {
						visited[z.i] = true
						test.T(t, z.Point, tt.zsP[i])
						i++
						z = z.nextA
						if z.i == z0.i {
							break
						}
					}
				}
			}
			i = 0
			visited = map[int]bool{}
			for _, z0 := range zs {
				if !visited[z0.i] {
					for z := z0; ; {
						visited[z.i] = true
						test.T(t, z.Point, tt.zsQ[i])
						i++
						z = z.nextB
						if z.i == z0.i {
							break
						}
					}
				}
			}
		})
	}
}

func TestPathCut(t *testing.T) {
	var tts = []struct {
		p, q   string
		ps, qs []string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z",
			[]string{"M7.5 5L5 10L2.5 5", "M2.5 5L0 0L10 0L7.5 5"},
			[]string{"M7.5 5L10 5L5 15L0 5L2.5 5", "M2.5 5L7.5 5"},
		},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L5 1L5 3L1 3z",
			[]string{"M2 1L2 2L1 2", "M1 2L0 2L0 0L2 0L2 1", "M5 2L4 2L4 1", "M4 1L4 0L6 0L6 2L5 2"},
			[]string{"M2 1L4 1", "M4 1L5 1L5 2", "M5 2L5 3L1 3L1 2", "M1 2L1 1L2 1"},
		},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			q := MustParseSVG(tt.q)

			ps, qs := p.Cut(q)
			test.T(t, len(ps), len(tt.ps))
			test.T(t, len(qs), len(tt.qs))
			for i := range tt.ps {
				test.T(t, ps[i], MustParseSVG(tt.ps[i]))
				test.T(t, qs[i], MustParseSVG(tt.qs[i]))
			}
		})
	}
}

func TestPathCombine(t *testing.T) {
	var tts = []struct {
		p string
		r string
	}{
		{"L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "M10 5L15 5L15 15L5 15L5 10L0 10L0 0L10 0z"},
		{"L10 0L10 10L0 10zM5 5L5 15L15 15L15 5z", "M10 5L5 5L5 10L0 10L0 0L10 0zM10 5L15 5L15 15L5 15L5 10L10 10z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			p := MustParseSVG(tt.p)
			test.T(t, p.Combine(), MustParseSVG(tt.r))
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
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M2.5 5L5 10L7.5 5z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M2.5 5L5 10L7.5 5z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", ""},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", ""},

		// subpath winding
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L3 2L2 2L2 3L1 3L1 1zM3 3L3 4L2 4L2 3z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", ""},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "M2 2L8 2L5 8z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M2 2L8 2L5 8z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 0L5 10z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L10 0L5 10z", "L5 10L10 0z"},
		{"L5 10L10 0z", "L5 10L10 0z", "L5 10L10 0z"},
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
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M2.5 5L0 5L5 15L10 5L7.5 5L10 0L0 0z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M2.5 5L0 5L5 15L10 5L7.5 5L10 0L0 0z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2zM2 0L4 0L4 2L2 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2zM2 1L4 1L4 3L2 3z"},

		// subpath winding
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L4 1L4 2L3 2L3 3L4 3L4 2L5 2L5 5L2 5L2 4L1 4L1 3L0 3L0 1L1 1L1 0L3 0z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "L10 0L5 10z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
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
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M2.5 5L7.5 5L10 0L0 0zM2.5 5L0 5L5 15L10 5L7.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M2.5 5L7.5 5L10 0L0 0zM2.5 5L0 5L5 15L10 5L7.5 5L5 10z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2zM2 0L4 0L4 2L2 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2zM2 1L4 1L4 3L2 3z"},

		// subpath winding
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L1 1L1 0L3 0zM3 1L4 1L4 2L3 2zM3 2L3 3L2 3L2 4L1 4L1 3L2 3L2 2zM3 3L4 3L4 2L5 2L5 5L2 5L2 4L3 4zM1 3L0 3L0 1L1 1z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "L10 0L5 10zM2 2L5 8L8 2z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", ""},
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
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M2.5 5L7.5 5L10 0L0 0z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M2.5 5L7.5 5L10 0L0 0z"},

		{"M0 5L10 5L5 15z", "L10 0L5 10z", "M2.5 5L5 10L7.5 5L10 5L5 15L0 5z"},
		{"M0 5L10 5L5 15z", "L5 10L10 0z", "M2.5 5L5 10L7.5 5L10 5L5 15L0 5z"},
		{"M0 5L5 15L10 5z", "L10 0L5 10z", "M7.5 5L5 10L2.5 5L0 5L5 15L10 5z"},
		{"M0 5L5 15L10 5z", "L5 10L10 0z", "M7.5 5L5 10L2.5 5L0 5L5 15L10 5z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2z"},
		{"M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z"},
		{"M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z"},

		// subpath winding
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M3 1L1 1L1 0L3 0zM3 2L3 3L2 3L2 4L1 4L1 3L2 3L2 2z"},
		{"M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 0L3 0L3 4L1 4z", "M3 2L3 1L4 1L4 2zM1 3L0 3L0 1L1 1zM2 4L3 4L3 3L4 3L4 2L5 2L5 5L2 5z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10z"},
		{"M0 10L10 10L5 20z", "L10 0L5 10z", "M0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", ""},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", ""},
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
