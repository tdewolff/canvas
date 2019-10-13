package canvas

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/tdewolff/test"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func TestPathEmpty(t *testing.T) {
	p := &Path{}
	test.That(t, p.Empty())

	p.MoveTo(5, 2)
	test.That(t, p.Empty())

	p.LineTo(6, 2)
	test.That(t, !p.Empty())
}

func TestPathEquals(t *testing.T) {
	test.That(t, !MustParseSVG("M5 0L5 10").Equals(MustParseSVG("M5 0")))
	test.That(t, !MustParseSVG("M5 0L5 10").Equals(MustParseSVG("M5 0M5 10")))
	test.That(t, !MustParseSVG("M5 0L5 10").Equals(MustParseSVG("M5 0L5 9")))
	test.That(t, MustParseSVG("M5 0L5 10").Equals(MustParseSVG("M5 0L5 10")))
}

func TestPathClosed(t *testing.T) {
	test.That(t, !MustParseSVG("M5 0L5 10").Closed())
	test.That(t, MustParseSVG("M5 0L5 10z").Closed())
	test.That(t, !MustParseSVG("M5 0L5 10zM5 10").Closed())
	test.That(t, MustParseSVG("M5 0L5 10zM5 10z").Closed())
}

func TestPathAppend(t *testing.T) {
	test.T(t, MustParseSVG("M5 0L5 10").Append(nil), MustParseSVG("M5 0L5 10"))
	test.T(t, (&Path{}).Append(MustParseSVG("M5 0L5 10")), MustParseSVG("M5 0L5 10"))

	p := MustParseSVG("M5 0L5 10").Append(MustParseSVG("M5 15L10 15"))
	test.T(t, p, MustParseSVG("M5 0L5 10M5 15L10 15"))

	p = MustParseSVG("M5 0L5 10").Append(MustParseSVG("L10 15M20 15L25 15"))
	test.T(t, p, MustParseSVG("M5 0L5 10M0 0L10 15M20 15L25 15"))
}

func TestPathJoin(t *testing.T) {
	test.T(t, MustParseSVG("M5 0L5 10").Join(nil), MustParseSVG("M5 0L5 10"))
	test.T(t, (&Path{}).Join(MustParseSVG("M5 0L5 10")), MustParseSVG("M5 0L5 10"))

	p := MustParseSVG("M5 0L5 10").Join(MustParseSVG("L10 15"))
	test.T(t, p, MustParseSVG("M5 0L5 10M0 0L10 15"))

	p = MustParseSVG("M5 0L5 10").Join(MustParseSVG("M5 10L10 15"))
	test.T(t, p, MustParseSVG("M5 0L5 10L10 15"))

	p = MustParseSVG("M5 0L5 10").Join(MustParseSVG("L10 15M20 15L25 15"))
	test.T(t, p, MustParseSVG("M5 0L5 10M0 0L10 15M20 15L25 15"))

	p = MustParseSVG("M5 0L5 10").Join(MustParseSVG("M5 10L10 15M20 15L25 15"))
	test.T(t, p, MustParseSVG("M5 0L5 10L10 15M20 15L25 15"))

	p = MustParseSVG("M5 0L10 5").Join(MustParseSVG("M10 5L15 10"))
	test.T(t, p, MustParseSVG("M5 0L15 10"))

	p = MustParseSVG("M5 0L10 5").Join(MustParseSVG("L5 5z"))
	test.T(t, p, MustParseSVG("M5 0L10 5M0 0L5 5z"))
}

func TestPathCoords(t *testing.T) {
	coords := MustParseSVG("L5 10").Coords()
	test.T(t, len(coords), 2)
	test.T(t, coords[0], Point{0.0, 0.0})
	test.T(t, coords[1], Point{5.0, 10.0})

	coords = MustParseSVG("L5 10C2.5 10 0 5 0 0z").Coords()
	test.T(t, len(coords), 3)
	test.T(t, coords[0], Point{0.0, 0.0})
	test.T(t, coords[1], Point{5.0, 10.0})
	test.T(t, coords[2], Point{0.0, 0.0})
}

func TestPathCommands(t *testing.T) {
	var tts = []struct {
		p *Path
		s string
	}{
		{(&Path{}).MoveTo(3, 4), "M3 4"},
		{(&Path{}).MoveTo(3, 4).QuadTo(3, 4, 3, 4), "M3 4"},
		{(&Path{}).MoveTo(3, 4).CubeTo(3, 4, 3, 4, 3, 4), "M3 4"},
		{(&Path{}).MoveTo(3, 4).ArcTo(2, 2, 0, false, false, 3, 4), "M3 4"},

		{(&Path{}).LineTo(3, 4), "M0 0L3 4"},
		{(&Path{}).QuadTo(3, 4, 3, 4), "M0 0L3 4"},
		{(&Path{}).QuadTo(1, 2, 3, 4), "M0 0Q1 2 3 4"},
		{(&Path{}).QuadTo(0, 0, 0, 0), ""},
		{(&Path{}).QuadTo(3, 4, 0, 0), "M0 0Q3 4 0 0"},
		{(&Path{}).QuadTo(1.5, 2, 3, 4), "M0 0L3 4"},
		{(&Path{}).CubeTo(0, 0, 3, 4, 3, 4), "M0 0L3 4"},
		{(&Path{}).CubeTo(1, 1, 2, 2, 3, 4), "M0 0C1 1 2 2 3 4"},
		{(&Path{}).CubeTo(1, 1, 2, 2, 0, 0), "M0 0C1 1 2 2 0 0"},
		{(&Path{}).CubeTo(0, 0, 0, 0, 0, 0), ""},
		{(&Path{}).CubeTo(1, 1, 2, 2, 3, 3), "M0 0L3 3"},
		{(&Path{}).ArcTo(0, 0, 0, false, false, 4, 0), "M0 0L4 0"},
		{(&Path{}).ArcTo(2, 1, 0, false, false, 4, 0), "M0 0A2 1 0 0 0 4 0"},
		{(&Path{}).ArcTo(1, 2, 0, true, true, 4, 0), "M0 0A4 2 90 1 1 4 0"},
		{(&Path{}).ArcTo(1, 2, 90, false, false, 4, 0), "M0 0A2 1 0 0 0 4 0"},
		{(&Path{}).Close(), ""},

		{(&Path{}).LineTo(5, 0).Close().LineTo(6, 3), "M0 0L5 0zM0 0L6 3"},
		{(&Path{}).LineTo(5, 0).Close().QuadTo(5, 3, 6, 3), "M0 0L5 0zM0 0Q5 3 6 3"},
		{(&Path{}).LineTo(5, 0).Close().CubeTo(5, 1, 5, 3, 6, 3), "M0 0L5 0zM0 0C5 1 5 3 6 3"},
		{(&Path{}).LineTo(5, 0).Close().ArcTo(5, 5, 0, false, false, 10, 0), "M0 0L5 0zM0 0A5 5 0 0 0 10 0"},

		{(&Path{}).MoveTo(3, 4).MoveTo(5, 3), "M5 3"},
		{(&Path{}).MoveTo(3, 4).Close(), ""},
		{(&Path{}).LineTo(3, 4).LineTo(0, 0).Close(), "M0 0L3 4z"},
		{(&Path{}).LineTo(3, 4).LineTo(4, 0).LineTo(2, 0).Close(), "M0 0L3 4L4 0z"},
		{(&Path{}).LineTo(3, 4).Close().Close(), "M0 0L3 4z"},
		{(&Path{}).MoveTo(2, 1).LineTo(3, 4).LineTo(5, 0).Close().LineTo(6, 3), "M2 1L3 4L5 0zM2 1L6 3"},
		{(&Path{}).MoveTo(2, 1).LineTo(3, 4).LineTo(5, 0).Close().MoveTo(2, 1).LineTo(6, 3), "M2 1L3 4L5 0zM2 1L6 3"},
	}
	for _, tt := range tts {
		t.Run(tt.s, func(t *testing.T) {
			test.String(t, tt.p.String(), tt.s)
		})
	}

	test.T(t, (&Path{}).Arc(2, 1, 0, 180, 0), MustParseSVG("A2 1 0 0 0 4 0"))
	test.T(t, (&Path{}).Arc(2, 1, 0, 0, 180), MustParseSVG("A2 1 0 0 1 -4 0"))
	test.T(t, (&Path{}).Arc(2, 1, 0, 540, 0), MustParseSVG("A2 1 0 0 0 4 0A2 1 0 0 0 0 0A2 1 0 0 0 4 0"))
	test.T(t, (&Path{}).Arc(2, 1, 0, 180, -180), MustParseSVG("A2 1 0 0 0 4 0A2 1 0 0 0 0 0"))
}

func TestPathCCW(t *testing.T) {
	test.That(t, MustParseSVG("L10 0L10 10z").CCW())
	test.That(t, !MustParseSVG("L10 0L10 -10z").CCW())
	test.That(t, MustParseSVG("L10 0").CCW())
	test.That(t, MustParseSVG("M10 0").CCW())
}

func TestPathFilling(t *testing.T) {
	fillings := MustParseSVG("M0 0").Filling()
	test.T(t, len(fillings), 0)

	fillings = MustParseSVG("L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z").Filling() // outer CCW, inner CCW
	test.T(t, len(fillings), 2)
	test.T(t, fillings[0], true)
	test.T(t, fillings[1], true)

	fillings = MustParseSVG("L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z").Filling() // outer CCW, inner CW
	test.T(t, fillings[0], true)
	test.T(t, fillings[1], false)

	FillRule = EvenOdd
	fillings = MustParseSVG("L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z").Filling() // outer CCW, inner CCW
	test.T(t, fillings[0], true)
	test.T(t, fillings[1], false)

	fillings = MustParseSVG("L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z").Filling() // outer CCW, inner CW
	test.T(t, fillings[0], true)
	test.T(t, fillings[1], false)
	FillRule = NonZero

	fillings = MustParseSVG("L10 10z").Filling()
	test.T(t, fillings[0], false)

	fillings = MustParseSVG("C5 0 10 5 10 10z").Filling()
	test.T(t, fillings[0], true)

	fillings = MustParseSVG("C0 5 5 10 10 10z").Filling()
	test.T(t, fillings[0], true)

	fillings = MustParseSVG("Q10 0 10 10z").Filling()
	test.T(t, fillings[0], true)

	fillings = MustParseSVG("Q0 10 10 10z").Filling()
	test.T(t, fillings[0], true)

	fillings = MustParseSVG("A10 10 0 0 1 10 10z").Filling()
	test.T(t, fillings[0], true)

	fillings = MustParseSVG("A10 10 0 0 0 10 10z").Filling()
	test.T(t, fillings[0], true)
}

func TestPathInterior(t *testing.T) {
	test.That(t, MustParseSVG("L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z").Interior(1, 1))
	test.That(t, MustParseSVG("L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z").Interior(3, 3))
	test.That(t, MustParseSVG("L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z").Interior(1, 1))
	test.That(t, !MustParseSVG("L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z").Interior(3, 3))

	FillRule = EvenOdd
	test.That(t, MustParseSVG("L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z").Interior(1, 1))
	test.That(t, !MustParseSVG("L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z").Interior(3, 3))
	test.That(t, MustParseSVG("L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z").Interior(1, 1))
	test.That(t, !MustParseSVG("L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z").Interior(3, 3))
	FillRule = NonZero
}

func TestPathBounds(t *testing.T) {
	Epsilon = 1e-6
	var tts = []struct {
		orig   string
		bounds Rect
	}{
		{"", Rect{}},
		{"Q50 100 100 0", Rect{0, 0, 100, 50}},
		{"Q100 50 0 100", Rect{0, 0, 50, 100}},
		{"Q0 0 100 0", Rect{0, 0, 100, 0}},
		{"Q100 0 100 0", Rect{0, 0, 100, 0}},
		{"Q100 0 100 100", Rect{0, 0, 100, 100}},
		{"C0 0 100 0 100 0", Rect{0, 0, 100, 0}},
		{"C0 100 100 100 100 0", Rect{0, 0, 100, 75}},
		{"C0 0 100 90 100 0", Rect{0, 0, 100, 40}},
		{"C0 90 100 0 100 0", Rect{0, 0, 100, 40}},
		{"C100 100 0 100 100 0", Rect{0, 0, 100, 75}},
		{"C66.667 0 100 33.333 100 100", Rect{0, 0, 100, 100}},
		{"M3.1125 1.7812C3.4406 1.7812 3.5562 1.5938 3.4578 1.2656", Rect{3.1125, 1.2656, 0.379252, 0.515599}},
		{"A100 100 0 0 0 100 100", Rect{0, 0, 100, 100}},
		{"A50 100 90 0 0 200 0", Rect{0, 0, 200, 50}},
		{"A100 100 0 1 0 -100 100", Rect{-200, -100, 200, 200}}, // hit xmin, ymin
		{"A100 100 0 1 1 -100 100", Rect{-100, 0, 200, 200}},    // hit xmax, ymax
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			test.T(t, MustParseSVG(tt.orig).Bounds(), tt.bounds)
		})
	}
}

// for quadratic Bézier use https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D2*(1-t)*t*50.00+%2B+t%5E2*100.00,+y%3D2*(1-t)*t*66.67+%2B+t%5E2*0.00%7D+from+0+to+1
// for cubic Bézier use https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D3*(1-t)%5E2*t*0.00+%2B+3*(1-t)*t%5E2*100.00+%2B+t%5E3*100.00,+y%3D3*(1-t)%5E2*t*66.67+%2B+3*(1-t)*t%5E2*66.67+%2B+t%5E3*0.00%7D+from+0+to+1
// for ellipse use https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D10.00*cos(t),+y%3D20.0*sin(t)%7D+from+0+to+pi
func TestPathLength(t *testing.T) {
	var tts = []struct {
		orig   string
		length float64
	}{
		{"M10 0z", 0.0},
		{"Q50 66.67 100 0", 124.533},
		{"Q100 0 100 0", 100.0000},
		{"C0 66.67 100 66.67 100 0", 158.5864},
		{"C0 0 100 66.67 100 0", 125.746},
		{"C0 0 100 0 100 0", 100.0000},
		{"C100 66.67 0 66.67 100 0", 143.9746},
		{"A10 20 0 0 0 20 0", 48.4422},
		{"A10 20 0 0 1 20 0", 48.4422},
		{"A10 20 0 1 0 20 0", 48.4422},
		{"A10 20 0 1 1 20 0", 48.4422},
		{"A10 20 30 0 0 20 0", 31.4622},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			length := MustParseSVG(tt.orig).Length()
			if math.Abs(tt.length-length)/length > 0.01 {
				test.Fail(t, length, "!=", tt.length, "±1%")
			}
		})
	}
}

func TestPathTransform(t *testing.T) {
	Epsilon = 1e-3
	var tts = []struct {
		orig string
		m    Matrix
		res  string
	}{
		{"M0 0L10 0Q15 10 20 0C23 10 27 10 30 0z", Identity.Translate(0, 100), "M0 100L10 100Q15 110 20 100C23 110 27 110 30 100z"},
		{"A10 10 0 0 0 20 0", Identity.Translate(0, 10), "M0 10A10 10 0 0 0 20 10"},
		{"A10 10 0 0 0 20 0", Identity.Scale(1, -1), "M0 0A10 10 0 0 1 20 0"},
		{"A10 5 0 0 0 20 0", Identity.Rotate(270), "M0 0A10 5 90 0 0 0 -20"},
		{"A10 10 0 0 0 20 0", Identity.Rotate(120).Scale(1, -2), "M0 0A20 10 30 0 1 -10 17.321"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			test.T(t, MustParseSVG(tt.orig).Transform(tt.m), MustParseSVG(tt.res))
		})
	}
}

func TestPathReplace(t *testing.T) {
	line := func(p0, p1 Point) *Path {
		return (&Path{}).MoveTo(p0.X, p0.Y).LineTo(p1.X, p1.Y-5.0)
	}
	bezier := func(p0, p1, p2, p3 Point) *Path {
		return (&Path{}).MoveTo(p0.X, p0.Y).LineTo(p3.X, p3.Y)
	}
	arc := func(p0 Point, rx, ry, phi float64, largeArc, sweep bool, p1 Point) *Path {
		return (&Path{}).MoveTo(p0.X, p0.Y).ArcTo(rx, ry, phi, !largeArc, sweep, p1.X, p1.Y)
	}

	var tts = []struct {
		orig   string
		res    string
		line   func(Point, Point) *Path
		bezier func(Point, Point, Point, Point) *Path
		arc    func(Point, float64, float64, float64, bool, bool, Point) *Path
	}{
		{"C0 10 10 10 10 0L30 0", "L30 0", nil, bezier, nil},
		{"M20 0L30 0C0 10 10 10 10 0", "M20 0L30 0L10 0", nil, bezier, nil},
		{"M10 0L20 0Q25 10 20 10A5 5 0 0 0 30 10z", "M10 0L20 -5L20 10A5 5 0 1 0 30 10L10 -5z", line, bezier, arc},
		{"L10 0L0 5z", "L10 -5L10 0L0 0L0 5L0 -5z", line, nil, nil},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := MustParseSVG(tt.orig)
			test.T(t, p.replace(tt.line, tt.bezier, tt.arc), MustParseSVG(tt.res))
		})
	}
}

func TestPathMarkers(t *testing.T) {
	start := MustParseSVG("L1 0L0 1z")
	mid := MustParseSVG("M-1 0A1 1 0 0 0 1 0z")
	end := MustParseSVG("L-1 0L0 1z")

	var tts = []struct {
		orig    string
		markers []string
	}{
		{"M10 0", []string{}},
		{"M10 0L20 10", []string{"M10 0L11 0L10 1z", "M20 10L19 10L20 11z"}},
		{"L10 0L20 10", []string{"M0 0L1 0L0 1z", "M9 0A1 1 0 0 0 11 0z", "M20 10L19 10L20 11z"}},
		{"L10 0L20 10z", []string{"M9 0A1 1 0 0 0 11 0z", "M19 10A1 1 0 0 0 21 10z", "M-1 0A1 1 0 0 0 1 0z"}},
		{"M10 0L20 10M30 0L40 10", []string{"M10 0L11 0L10 1z", "M20 10L19 10L20 11z", "M30 0L31 0L30 1z", "M40 10L39 10L40 11z"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := MustParseSVG(tt.orig)
			ps := p.Markers(start, mid, end, false)
			if len(ps) != len(tt.markers) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.markers, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p, MustParseSVG(tt.markers[i]))
				}
			}
		})
	}
}

func TestPathMarkersAligned(t *testing.T) {
	start := MustParseSVG("L1 0L0 1z")
	mid := MustParseSVG("M-1 0A1 1 0 0 0 1 0z")
	end := MustParseSVG("L-1 0L0 1z")

	var tts = []struct {
		orig    string
		markers []string
	}{
		{"M10 0z", []string{}},
		{"M10 0L20 10", []string{"M10 0L10.707 0.707L9.293 0.707z", "M20 10L19.293 9.293L19.293 10.707z"}},
		{"L10 0L20 10", []string{"M0 0L1 0L0 1z", "M9.076 -0.383A1 1 0 0 0 10.924 0.383z", "M20 10L19.293 9.293L19.293 10.707z"}},
		{"L10 0L20 10z", []string{"M9.076 -0.383A1 1 0 0 0 10.924 0.383z", "M20.585 9.189A1 1 0 0 0 19.415 10.811z", "M-0.230 0.973A1 1 0 0 0 0.230 -0.973z"}},
		{"M10 0L20 10M30 0L40 10", []string{"M10 0L10.707 0.707L9.293 0.707z", "M20 10L19.293 9.293L19.293 10.707z", "M30 0L30.707 0.707L29.293 0.707z", "M40 10L39.293 9.293L39.293 10.707z"}},
		{"Q0 10 10 10Q20 10 20 0", []string{"L0 1L-1 0z", "M9 10A1 1 0 0 0 11 10z", "M20 0L20 1L21 0z"}},
		{"C0 6.66667 3.33333 10 10 10C16.66667 10 20 6.66667 20 0", []string{"L0 1L-1 0z", "M9 10A1 1 0 0 0 11 10z", "M20 0L20 1L21 0z"}},
		{"A10 10 0 0 0 10 10A10 10 0 0 0 20 0", []string{"L0 1L-1 0z", "M9 10A1 1 0 0 0 11 10z", "M20 0L20 1L21 0z"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := MustParseSVG(tt.orig)
			ps := p.Markers(start, mid, end, true)
			if len(ps) != len(tt.markers) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.markers, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p, MustParseSVG(tt.markers[i]))
				}
			}
		})
	}
}

func TestPathSplit(t *testing.T) {
	var tts = []struct {
		orig  string
		split []string
	}{
		{"M5 5L6 6z", []string{"M5 5L6 6z"}},
		{"L5 5M10 10L20 20z", []string{"L5 5", "M10 10L20 20z"}},
		{"L5 5zL10 10", []string{"L5 5z", "L10 10"}},
		{"M5 5L15 5zL10 10zL20 20", []string{"M5 5L15 5z", "M5 5L10 10z", "M5 5L20 20"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := MustParseSVG(tt.orig)
			ps := p.Split()
			if len(ps) != len(tt.split) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.split, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p, MustParseSVG(tt.split[i]))
				}
			}
		})
	}

	ps := (&Path{[]float64{moveToCmd, 5.0, 5.0, moveToCmd, moveToCmd, 10.0, 10.0, moveToCmd, closeCmd, 10.0, 10.0, closeCmd}}).Split()
	test.T(t, ps[0].String(), "M5 5")
	test.T(t, ps[1].String(), "M10 10z")
}

func TestPathSplitAt(t *testing.T) {
	var tts = []struct {
		orig  string
		d     []float64
		split []string
	}{
		{"L4 3L8 0z", []float64{}, []string{"L4 3L8 0z"}},
		{"M2 0L4 3Q10 10 20 0C20 10 30 10 30 0A10 10 0 0 0 50 0z", []float64{0.0}, []string{"M2 0L4 3Q10 10 20 0C20 10 30 10 30 0A10 10 0 0 0 50 0L2 0"}},
		{"L4 3L8 0z", []float64{0.0, 5.0, 10.0, 18.0}, []string{"L4 3", "M4 3L8 0", "M8 0L0 0"}},
		{"L4 3L8 0z", []float64{5.0, 20.0}, []string{"L4 3", "M4 3L8 0L0 0"}},
		{"L4 3L8 0z", []float64{2.5, 7.5, 14.0}, []string{"L2 1.5", "M2 1.5L4 3L6 1.5", "M6 1.5L8 0L4 0", "M4 0L0 0"}},
		{"Q10 10 20 0", []float64{11.477858}, []string{"Q5 5 10 5", "M10 5Q15 5 20 0"}},
		{"C0 10 20 10 20 0", []float64{13.947108}, []string{"C0 5 5 7.5 10 7.5", "M10 7.5C15 7.5 20 5 20 0"}},
		{"A10 10 0 0 1 -20 0", []float64{15.707963}, []string{"A10 10 0 0 1 -10 10", "M-10 10A10 10 0 0 1 -20 0"}},
		{"A10 10 0 0 0 20 0", []float64{15.707963}, []string{"A10 10 0 0 0 10 10", "M10 10A10 10 0 0 0 20 0"}},
		{"A10 10 0 1 0 2.9289 -7.0711", []float64{15.707963}, []string{"A10 10 0 0 0 10.024 9.9999", "M10.024 9.9999A10 10 0 1 0 2.9289 -7.0711"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := MustParseSVG(tt.orig)
			ps := p.SplitAt(tt.d...)
			if len(ps) != len(tt.split) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.split, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p, MustParseSVG(tt.split[i]))
				}
			}
		})
	}
}

func TestDashCanonical(t *testing.T) {
	var tts = []struct {
		origOffset float64
		origDashes []float64
		offset     float64
		dashes     []float64
	}{
		{0.0, []float64{0.0}, 0.0, []float64{0.0}},
		{0.0, []float64{-1.0}, 0.0, []float64{0.0}},
		{0.0, []float64{2.0, 0.0}, 0.0, []float64{}},
		{0.0, []float64{0.0, 2.0}, 0.0, []float64{0.0}},
		{0.0, []float64{0.0, 2.0, 0.0}, -2.0, []float64{2.0}},
		{0.0, []float64{0.0, 2.0, 3.0, 0.0}, -2.0, []float64{3.0, 2.0}},
		{0.0, []float64{0.0, 2.0, 3.0, 1.0, 0.0}, -2.0, []float64{3.0, 1.0, 2.0}},
		{0.0, []float64{0.0, 1.0, 2.0}, -1.0, []float64{3.0}},
		{0.0, []float64{0.0, 1.0, 2.0, 4.0}, -1.0, []float64{2.0, 5.0}},
		{0.0, []float64{2.0, 1.0, 0.0}, 1.0, []float64{3.0}},
		{0.0, []float64{4.0, 2.0, 1.0, 0.0}, 1.0, []float64{5.0, 2.0}},

		{0.0, []float64{1.0, 0.0, 2.0}, 0.0, []float64{3.0}},
		{0.0, []float64{1.0, 0.0, 2.0, 2.0, 0.0, 1.0}, 0.0, []float64{3.0}},
		{0.0, []float64{2.0, 0.0, -1.0}, 0.0, []float64{1.0}},
		{0.0, []float64{1.0, 0.0, 2.0, 0.0, 3.0, 0.0}, 0.0, []float64{}},
		{0.0, []float64{0.0, 1.0, 0.0, 2.0, 0.0, 3.0}, 0.0, []float64{0.0}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprintf("%v +%v", tt.origDashes, tt.origOffset), func(t *testing.T) {
			offset, dashes := dashCanonical(tt.origOffset, tt.origDashes)

			diff := offset != tt.offset || len(dashes) != len(tt.dashes)
			if !diff {
				for i := 0; i < len(tt.dashes); i++ {
					if dashes[i] != tt.dashes[i] {
						diff = true
						break
					}
				}
			}
			if diff {
				test.Fail(t, fmt.Sprintf("%v +%v != %v +%v", dashes, offset, tt.dashes, tt.offset))
			}
		})
	}
}

func TestPathDash(t *testing.T) {
	var tts = []struct {
		orig   string
		offset float64
		d      []float64
		dashes string
	}{
		{"", 0.0, []float64{0.0}, ""},
		{"L10 0", 0.0, []float64{}, "L10 0"},
		{"L10 0", 0.0, []float64{2.0}, "L2 0M4 0L6 0M8 0L10 0"},
		{"L10 0", 0.0, []float64{2.0, 1.0}, "L2 0M3 0L5 0M6 0L8 0M9 0L10 0"},
		{"L10 0", 1.0, []float64{2.0, 1.0}, "L1 0M2 0L4 0M5 0L7 0M8 0L10 0"},
		{"L10 0", -1.0, []float64{2.0, 1.0}, "M1 0L3 0M4 0L6 0M7 0L9 0"},
		{"L10 0", 2.0, []float64{2.0, 1.0}, "M1 0L3 0M4 0L6 0M7 0L9 0"},
		{"L10 0", 5.0, []float64{2.0, 1.0}, "M1 0L3 0M4 0L6 0M7 0L9 0"},
		{"L10 0L20 0", 0.0, []float64{15.0}, "L10 0L15 0"},
		{"L10 0L20 0", 15.0, []float64{15.0}, "M15 0L20 0"},
		{"L10 0L10 10L0 10z", 0.0, []float64{10.0}, "L10 0M10 10L0 10"},
		{"L10 0L10 10L0 10z", 0.0, []float64{15.0}, "M0 10L0 0L10 0L10 5"},
		{"M10 0L20 0L20 10L10 10z", 0.0, []float64{15.0}, "M10 10L10 0L20 0L20 5"},
		{"L10 0M0 10L10 10", 0.0, []float64{8.0}, "L8 0M0 10L8 10"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			test.T(t, MustParseSVG(tt.orig).Dash(tt.offset, tt.d...), MustParseSVG(tt.dashes))
		})
	}
}

func TestPathReverse(t *testing.T) {
	var tts = []struct {
		orig string
		inv  string
	}{
		{"", ""},
		{"M5 5", "M5 5"},
		{"M5 5z", "M5 5z"},
		{"M5 5L5 10L10 5", "M10 5L5 10L5 5"},
		{"M5 5L5 10L10 5z", "M5 5L10 5L5 10z"},
		{"M5 5L5 10L10 5M10 10L10 20L20 10z", "M10 10L20 10L10 20zM10 5L5 10L5 5"},
		{"M5 5L5 10L10 5zM10 10L10 20L20 10z", "M10 10L20 10L10 20zM5 5L10 5L5 10z"},
		{"M5 5Q10 10 15 5", "M15 5Q10 10 5 5"},
		{"M5 5Q10 10 15 5z", "M5 5L15 5Q10 10 5 5z"},
		{"M5 5C5 10 10 10 10 5", "M10 5C10 10 5 10 5 5"},
		{"M5 5C5 10 10 10 10 5z", "M5 5L10 5C10 10 5 10 5 5z"},
		{"M5 5A2.5 5 0 0 0 10 5", "M10 5A5 2.5 90 0 1 5 5"}, // bottom-half of ellipse along y
		{"M5 5A2.5 5 0 0 1 10 5", "M10 5A5 2.5 90 0 0 5 5"},
		{"M5 5A2.5 5 0 1 0 10 5", "M10 5A5 2.5 90 1 1 5 5"},
		{"M5 5A2.5 5 0 1 1 10 5", "M10 5A5 2.5 90 1 0 5 5"},
		{"M5 5A5 2.5 90 0 0 10 5", "M10 5A5 2.5 90 0 1 5 5"}, // same shape
		{"M5 5A2.5 5 0 0 0 10 5z", "M5 5L10 5A5 2.5 90 0 1 5 5z"},
		{"L0 5L5 5", "M5 5L0 5L0 0"},
		{"L-1 5L5 5z", "L5 5L-1 5z"},
		{"Q0 5 5 5", "M5 5Q0 5 0 0"},
		{"Q0 5 5 5z", "L5 5Q0 5 0 0z"},
		{"C0 5 5 5 5 0", "M5 0C5 5 0 5 0 0"},
		{"C0 5 5 5 5 0z", "L5 0C5 5 0 5 0 0z"},
		{"A2.5 5 0 0 0 5 0", "M5 0A5 2.5 90 0 1 0 0"},
		{"A2.5 5 0 0 0 5 0z", "L5 0A5 2.5 90 0 1 0 0z"},
		{"M5 5L10 10zL15 10", "M15 10L5 5M5 5L10 10z"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			test.T(t, MustParseSVG(tt.orig).Reverse(), MustParseSVG(tt.inv))
		})
	}
}

func TestPathParseSVG(t *testing.T) {
	var tts = []struct {
		orig string
		res  string
	}{
		{"M10 0L20 0H30V10C40 10 50 10 50 0Q55 10 60 0A5 5 0 0 0 70 0Z", "M10 0L20 0L30 0L30 10C40 10 50 10 50 0Q55 10 60 0A5 5 0 0 0 70 0z"},
		{"m10 0l10 0h10v10c10 0 20 0 20 -10q5 10 10 0a5 5 0 0 0 10 0z", "M10 0L20 0L30 0L30 10C40 10 50 10 50 0Q55 10 60 0A5 5 0 0 0 70 0z"},
		{"C0 10 10 10 10 0S20 -10 20 0", "C0 10 10 10 10 0C10 -10 20 -10 20 0"},
		{"c0 10 10 10 10 0s10 -10 10 0", "C0 10 10 10 10 0C10 -10 20 -10 20 0"},
		{"Q5 10 10 0T20 0", "Q5 10 10 0Q15 -10 20 0"},
		{"q5 10 10 0t10 0", "Q5 10 10 0Q15 -10 20 0"},
		{"A10 10 0 0 0 40 0", "A20 20 0 0 0 40 0"},  // scale ellipse
		{"A10 5 90 0 0 40 0", "A40 20 90 0 0 40 0"}, // scale ellipse

		// go-fuzz
		{"V0 ", ""},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)
			test.T(t, p, MustParseSVG(tt.res))
		})
	}
}

func TestPathParseSVGErrors(t *testing.T) {
	var tts = []struct {
		orig string
		err  string
	}{
		{"5", "bad path: path should start with command"},
		{"MM", "bad path: 2 numbers should follow command 'M' at position 1"},

		// go-fuzz
		{"V4-z\n0ìGßIzØ", "bad path: unknown command '0' at position 6"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			_, err := ParseSVG(tt.orig)
			test.That(t, err != nil)
			test.T(t, err.Error(), tt.err)
		})
	}
}

func TestPathToSVG(t *testing.T) {
	var tts = []struct {
		orig string
		ps   string
	}{
		{"", ""},
		{"L10 0Q15 10 20 0M20 10C20 20 30 20 30 10z", "M0 0H10Q15 10 20 0M20 10C20 20 30 20 30 10z"},
		{"L10 0M20 0L30 0", "M0 0H10M20 0H30"},
		{"L0 0L0 10L20 20", "M0 0V10L20 20"},
		{"A5 5 0 0 1 10 0", "M0 0A5 5 0 0 1 10 0"},
		{"A10 5 90 0 0 10 0", "M0 0A5 10 0 0 0 10 0"},
		{"A10 5 90 1 0 10 0", "M0 0A5 10 0 1 0 10 0"},
		{"M20 0L20 0", ""},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := MustParseSVG(tt.orig)
			test.T(t, p.ToSVG(), tt.ps)
		})
	}
}

func TestPathToPS(t *testing.T) {
	var tts = []struct {
		orig string
		ps   string
	}{
		{"", ""},
		{"L10 0Q15 10 20 0M20 10C20 20 30 20 30 10z", "0 0 moveto 10 0 lineto 13.333333 6.6666667 16.666667 6.6666667 20 0 curveto 20 10 moveto 20 20 30 20 30 10 curveto closepath"},
		{"L10 0M20 0L30 0", "0 0 moveto 10 0 lineto 20 0 moveto 30 0 lineto"},
		{"A5 5 0 0 1 10 0", "0 0 moveto 5 0 5 5 180 360 0 ellipse"},
		{"A10 5 90 0 0 10 0", "0 0 moveto 5 0 10 5 90 -90 90 ellipsen"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			test.T(t, MustParseSVG(tt.orig).ToPS(), tt.ps)
		})
	}
}

func TestPathToPDF(t *testing.T) {
	var tts = []struct {
		orig string
		ps   string
	}{
		{"", ""},
		{"L10 0Q15 10 20 0M20 10C20 20 30 20 30 10z", "0 0 m 10 0 l 13.333333 6.6666667 16.666667 6.6666667 20 0 c 20 10 m 20 20 30 20 30 10 c h"},
		{"L10 0M20 0L30 0", "0 0 m 10 0 l 20 0 m 30 0 l"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			test.T(t, MustParseSVG(tt.orig).ToPDF(), tt.ps)
		})
	}
}

func plotPathLengthParametrization(filename string, N int, speed, length func(float64) float64, tmin, tmax float64) {
	T3, _ := invPolynomialApprox3(gaussLegendre7, speed, tmin, tmax)
	Tc, totalLength := invSpeedPolynomialChebyshevApprox(N, gaussLegendre7, speed, tmin, tmax)

	anchor1Data := make(plotter.XYs, 2)
	anchor1Data[0].X = totalLength * 1.0 / 3.0
	anchor1Data[0].Y = T3(totalLength * 1.0 / 3.0)
	anchor1Data[1].X = totalLength * 2.0 / 3.0
	anchor1Data[1].Y = T3(totalLength * 2.0 / 3.0)

	n := 100
	realData := make(plotter.XYs, n+1)
	model1Data := make(plotter.XYs, n+1)
	model2Data := make(plotter.XYs, n+1)
	for i := 0; i < n+1; i++ {
		t := tmin + (tmax-tmin)*float64(i)/float64(n)
		l := totalLength * float64(i) / float64(n)
		realData[i].X = length(t)
		realData[i].Y = t
		model1Data[i].X = l
		model1Data[i].Y = T3(l)
		model2Data[i].X = l
		model2Data[i].Y = Tc(l)
	}

	scatter, err := plotter.NewScatter(realData)
	if err != nil {
		panic(err)
	}
	scatter.Shape = draw.CircleGlyph{}

	anchors1, err := plotter.NewScatter(anchor1Data)
	if err != nil {
		panic(err)
	}
	anchors1.GlyphStyle.Shape = draw.CircleGlyph{}
	anchors1.GlyphStyle.Color = Steelblue
	anchors1.GlyphStyle.Radius = 5.0

	line1, err := plotter.NewLine(model1Data)
	if err != nil {
		panic(err)
	}
	line1.LineStyle.Color = Steelblue
	line1.LineStyle.Width = 2.0

	line2, err := plotter.NewLine(model2Data)
	if err != nil {
		panic(err)
	}
	line2.LineStyle.Color = Orangered
	line2.LineStyle.Width = 1.0

	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.X.Label.Text = "L"
	p.Y.Label.Text = "t"
	p.Add(scatter, line1, line2, anchors1)

	p.Legend.Add("real", scatter)
	p.Legend.Add("Polynomial", line1)
	p.Legend.Add(fmt.Sprintf("Chebyshev N=%v", N), line2)

	if err := p.Save(7*vg.Inch, 4*vg.Inch, filename); err != nil {
		panic(err)
	}
}

func TestPathLengthParametrization(t *testing.T) {
	if !testing.Verbose() {
		t.SkipNow()
		return
	}
	_ = os.Mkdir("test", 0755)

	start := Point{0.0, 0.0}
	cp := Point{1000.0, 0.0}
	end := Point{10.0, 10.0}
	speed := func(t float64) float64 {
		return quadraticBezierDeriv(start, cp, end, t).Length()
	}
	length := func(t float64) float64 {
		p0, p1, p2, _, _, _ := splitQuadraticBezier(start, cp, end, t)
		return quadraticBezierLength(p0, p1, p2)
	}
	plotPathLengthParametrization("test/len_param_quad.png", 20, speed, length, 0.0, 1.0)

	plotCube := func(name string, start, cp1, cp2, end Point) {
		N := 20 + 20*cubicBezierNumInflections(start, cp1, cp2, end)
		speed := func(t float64) float64 {
			return cubicBezierDeriv(start, cp1, cp2, end, t).Length()
		}
		length := func(t float64) float64 {
			p0, p1, p2, p3, _, _, _, _ := splitCubicBezier(start, cp1, cp2, end, t)
			return cubicBezierLength(p0, p1, p2, p3)
		}
		plotPathLengthParametrization(name, N, speed, length, 0.0, 1.0)
	}

	plotCube("test/len_param_cube.png", Point{0.0, 0.0}, Point{10.0, 0.0}, Point{10.0, 2.0}, Point{8.0, 2.0})

	// see "Analysis of Inflection Points for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
	// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
	plotCube("test/len_param_cube1.png", Point{16, 467}, Point{185, 95}, Point{673, 545}, Point{810, 17})
	plotCube("test/len_param_cube2.png", Point{859, 676}, Point{13, 422}, Point{781, 12}, Point{266, 425})
	plotCube("test/len_param_cube3.png", Point{872, 686}, Point{11, 423}, Point{779, 13}, Point{220, 376})
	plotCube("test/len_param_cube4.png", Point{819, 566}, Point{43, 18}, Point{826, 18}, Point{25, 533})
	plotCube("test/len_param_cube5.png", Point{884, 574}, Point{135, 14}, Point{678, 14}, Point{14, 566})

	rx, ry := 10000.0, 10.0
	phi := 0.0
	sweep := false
	end = Point{-100.0, 10.0}
	theta1, theta2 := 0.0, 0.5*math.Pi
	speed = func(theta float64) float64 {
		return ellipseDeriv(rx, ry, phi, sweep, theta).Length()
	}
	length = func(theta float64) float64 {
		return ellipseLength(rx, ry, theta1, theta)
	}
	plotPathLengthParametrization("test/len_param_ellipse.png", 20, speed, length, theta1, theta2)
}
