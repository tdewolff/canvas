package canvas

import (
	"fmt"
	"math"
	"testing"
	"time"

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
			{Point{2.0, 2.0}, [2]float64{2.0 / 3.0, 0.5}, [2]float64{0.5 * math.Pi, 0.0}, false, false},
		}},

		// tangent
		{"M2 0L2 3", "M2 2L3 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{2.0 / 3.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true, false},
		}},
		{"M2 0L2 2", "M2 2L3 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{1.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true, false},
		}},
		{"L2 2", "M0 4L2 2", Intersections{
			{Point{2.0, 2.0}, [2]float64{1.0, 1.0}, [2]float64{0.25 * math.Pi, 1.75 * math.Pi}, true, false},
		}},
		{"L10 5", "M0 10L10 5", Intersections{
			{Point{10.0, 5.0}, [2]float64{1.0, 1.0}, [2]float64{Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle()}, true, false},
		}},
		{"M10 5L20 10", "M10 5L20 0", Intersections{
			{Point{10.0, 5.0}, [2]float64{0.0, 0.0}, [2]float64{Point{2.0, 1.0}.Angle(), Point{2.0, -1.0}.Angle()}, true, false},
		}},

		// parallel
		{"L2 2", "L2 2", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, true},
			{Point{2.0, 2.0}, [2]float64{1.0, 1.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, true},
		}},
		{"L2 2", "M2 2L0 0", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 1.0}, [2]float64{0.25 * math.Pi, 1.25 * math.Pi}, true, true},
			{Point{2.0, 2.0}, [2]float64{1.0, 0.0}, [2]float64{0.25 * math.Pi, 1.25 * math.Pi}, true, true},
		}},
		{"L2 2", "M3 3L5 5", Intersections{}},
		{"L2 2", "M-1 1L1 3", Intersections{}},
		{"L2 2", "M2 2L4 4", Intersections{
			{Point{2.0, 2.0}, [2]float64{1.0, 0.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, false},
		}},
		{"L2 2", "M-2 -2L0 0", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 1.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, false},
		}},
		{"L4 4", "M2 2L6 6", Intersections{
			{Point{2.0, 2.0}, [2]float64{0.5, 0.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, true},
			{Point{4.0, 4.0}, [2]float64{1.0, 0.5}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, true},
		}},
		{"L4 4", "M-2 -2L2 2", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.5}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, true},
			{Point{2.0, 2.0}, [2]float64{0.5, 1.0}, [2]float64{0.25 * math.Pi, 0.25 * math.Pi}, true, true},
		}},

		// none
		{"M2 0L2 1", "M3 0L3 1", Intersections{}},
		{"M2 0L2 1", "M0 2L1 2", Intersections{}},

		// bugs
		{"M21.590990257669734 18.40900974233027L22.651650429449557 17.348349570550447", "M21.23743686707646 18.762563132923542L21.590990257669738 18.409009742330266", Intersections{
			{Point{21.590990257669734, 18.40900974233027}, [2]float64{0.0, 1.0}, [2]float64{1.75 * math.Pi, 1.75 * math.Pi}, true, false},
		}},
		{"M-0.1997406229376793 296.9999999925494L-0.1997406229376793 158.88740153238177", "M-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834", Intersections{
			{Point{-0.1997406229376793, 158.88740153238177}, [2]float64{1.0, 0.0}, [2]float64{1.5 * math.Pi, 89.9999462 * math.Pi / 180.0}, true, false},
			//{Point{-0.1997406229376793, 158.8874019079834}, [2]float64{0.9999999973, 1.0}, [2]float64{1.5 * math.Pi, 89.9999462 * math.Pi / 180.0}, true, true},
		}}, // #287
		{"M-0.1997406229376793 296.9999999925494L-0.1997406229376793 158.88740153238177", "M-0.19974062293732664 158.8874019079834L-0.19999999999964735 20.77454766193328", Intersections{
			{Point{-0.1997406229376793, 158.88740172019808}, [2]float64{0.9999999986, 1.359651238e-09}, [2]float64{270.0 * math.Pi / 180.0, 269.9998924 * math.Pi / 180.0}, false, false},
		}}, // #287
		{"M-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834", "M-0.19974062293732664 158.8874019079834L-0.19999999999964735 20.77454766193328", Intersections{
			//{Point{-0.19974062293732664, 158.88740153238177}, [2]float64{0.0, 2.72e-9}, [2]float64{89.9999462 * math.Pi / 180.0, 269.9998924 * math.Pi / 180.0}, true, true},
			{Point{-0.19974062293732664, 158.8874019079834}, [2]float64{1.0, 0.0}, [2]float64{89.9999462 * math.Pi / 180.0, 269.9998924 * math.Pi / 180.0}, true, false},
		}}, // #287
		{"M162.43449681368278 -9.999996185876771L162.43449681368278 -9.99998551284069", "M162.43449681368278 -9.999985512840682L162.2344968136828 -9.99998551284069", Intersections{
			{Point{162.43449681368278, -9.99998551284069}, [2]float64{1.0, 0.0}, [2]float64{0.5 * math.Pi, math.Pi}, true, false},
		}}, // #287
		{"M0.7814861805182336,0.39875653588924026L0.7814851602550552,0.3987574923859699", "M0.7814852358775772,0.39875773815916654L0.7814861805182336,0.39875653588924026", Intersections{
			{Point{0.7814861805182336, 0.39875653588924026}, [2]float64{0.0, 1.0}, [2]float64{136.84761026848332 * math.Pi / 180.0, 308.15722658736905 * math.Pi / 180.0}, true, false},
			//{Point{0.7814852358775772, 0.39875773815916654}, [2]float64{0.07, 0.0}, [2]float64{math.Pi, 0.0}, true, true},
		}},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line1, "x", tt.line2), func(t *testing.T) {
			line1 := MustParseSVGPath(tt.line1).ReverseScanner()
			line2 := MustParseSVGPath(tt.line2).ReverseScanner()
			line1.Scan()
			line2.Scan()

			zs := intersectionLineLine(nil, line1.Start(), line1.End(), line2.Start(), line2.End())
			test.T(t, zs, tt.zs)
		})
	}
}

func isIncreasing(a, b Point) bool {
	if b.X < a.X {
		return false
	} else if a.X == b.X && b.Y < a.Y {
		return false
	}
	return true
}

func TestIntersectionLineLineBentleyOttmann(t *testing.T) {
	var tts = []struct {
		line1, line2 string
		zs           []Point
	}{
		// secant
		{"M2 0L2 3", "M1 2L3 2", []Point{{2.0, 2.0}}},

		// tangent
		{"M2 0L2 3", "M2 2L3 2", []Point{{2.0, 2.0}}},
		{"M2 0L2 2", "M2 2L3 2", nil},
		{"L2 2", "M0 4L2 2", nil},
		{"L10 5", "M0 10L10 5", nil},
		{"M10 5L20 10", "M10 5L20 0", nil},

		// parallel
		{"L2 2", "L2 2", nil},
		{"L2 2", "M3 3L5 5", nil},
		{"L2 2", "M-3 -3L-1 -1", nil},
		{"L2 2", "M-1 1L1 3", nil},
		{"L2 2", "M2 2L4 4", nil},
		{"L2 2", "M-2 -2L0 0", nil},
		{"L4 4", "M2 2L4 4", []Point{{2.0, 2.0}}},
		{"L4 4", "M2 2L6 6", []Point{{2.0, 2.0}, {4.0, 4.0}}},
		{"L4 4", "M0 0L2 2", []Point{{2.0, 2.0}}},
		{"L4 4", "M0 0L6 6", []Point{{4.0, 4.0}}},
		{"L4 4", "M-2 -2L2 2", []Point{{0.0, 0.0}, {2.0, 2.0}}},
		{"L4 4", "M-2 -2L4 4", []Point{{0.0, 0.0}}},
		{"L4 4", "M1 1L3 3", []Point{{1.0, 1.0}, {3.0, 3.0}}},
		{"L4 4", "M-1 -1L5 5", []Point{{0.0, 0.0}, {4.0, 4.0}}},

		// none
		{"M2 0L2 1", "M3 0L3 1", nil},
		{"M2 0L2 1", "M0 2L1 2", nil},

		// almost vertical
		{"L2 0", "M1 -1L1.0000000000000002 2", []Point{{1.0000000000000002, 0.0}}},
		{"L2 0", "M1 1L1.0000000000000002 -2", []Point{{1.0000000000000002, 0.0}}},
		{"M1 -1L1.0000000000000002 2", "L2 0", []Point{{1.0000000000000002, 0.0}}},
		{"M1 1L1.0000000000000002 -2", "L2 0", []Point{{1.0000000000000002, 0.0}}},

		// bugs
		{"M21.590990257669734 18.40900974233027L22.651650429449557 17.348349570550447", "M21.23743686707646 18.762563132923542L21.590990257669738 18.409009742330266",
			nil, //[]Point{{21.590990257669738, 18.40900974233027}},
		}, // almost colinear
		{"M-0.1997406229376793 158.88740153238177L-0.1997406229376793 296.9999999925494", "M-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834",
			nil,
		}, // #287
		{"M-0.1997406229376793 158.88740153238177L-0.1997406229376793 296.9999999925494", "M-0.19999999999964735 20.77454766193328L-0.19974062293732664 158.8874019079834",
			[]Point{{-0.1997406229376793, 158.88740172019808}},
		}, // #287
		{"M-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834", "M-0.19999999999964735 20.77454766193328L-0.19974062293732664 158.8874019079834",
			nil, //[]Point{{-0.19974062293732664, 158.8874019079834}},
		}, // almost colinear, #287
		{"M162.43449681368278 -9.999996185876771L162.43449681368278 -9.99998551284069", "M162.2344968136828 -9.99998551284069L162.43449681368278 -9.999985512840682",
			nil,
		}, // almost colinear, #287
		{"M0.7814851602550552,0.3987574923859699L0.7814861805182336,0.39875653588924026", "M0.7814852358775772,0.39875773815916654L0.7814861805182336,0.39875653588924026",
			nil, //[]Point{{0.7814861805182336, 0.39875653588924026}},
		}, // almost colinear
		{"M0.6187340865555582,7.030136875251485L0.6203785454666688,7.0296030922171955", "M0.6187340865555582,7.030136875251485L0.6189178552813777,7.03007722485377",
			nil,
		}, // colinear
		{"M3.937495009359532,6.968745009359532L3.937495009359532,6.968745009361169", "M3.93359874,6.96874501L3.937504990640468,6.968745009359532",
			[]Point{{3.937495009359532, 6.968745009359532}},
		},
		{"M0.7099618031467739,7.781251248018358L0.7099618032015841,7.781251247660366", "M0.70995969,7.78125125L0.709962185160117,7.781251247660117",
			[]Point{{0.7099618031467739, 7.781251248018357}}, // weird!
		},
		{"M1.0025552034069727,1.0004403403618525L1.0025552034069727,1.0004403403686006", "M1.0025552034026004,1.0004403403635602L1.0050346154413319,0.9994720107590638",
			[]Point{{1.002555203406973, 1.0004403403618525}},
		},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.line1, "x", tt.line2), func(t *testing.T) {
			line1 := MustParseSVGPath(tt.line1).ReverseScanner()
			line2 := MustParseSVGPath(tt.line2).ReverseScanner()
			line1.Scan()
			line2.Scan()

			if !isIncreasing(line1.Start(), line1.End()) || !isIncreasing(line2.Start(), line2.End()) {
				t.Fatal("bad test: lines not increasing")
			}

			zs := intersectionLineLineBentleyOttmann(nil, line1.Start(), line1.End(), line2.Start(), line2.End())
			test.T(t, zs, tt.zs)
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
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false, false},
		}},

		// tangent
		{"L0 10", "Q10 5 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.5 * math.Pi, Point{2.0, 1.0}.Angle()}, true, false},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{0.5 * math.Pi, Point{-2.0, 1.0}.Angle()}, true, false},
		}},
		{"M5 0L5 10", "Q10 5 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, false},
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
			{Point{6.0, 5.0}, [2]float64{0.6, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false, false},
		}},
		{"M0 1L1 1", "C0 2 1 0 1 2", Intersections{ // parallel at intersection
			{Point{0.5, 1.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.0}, false, false},
		}},
		{"M0 1L1 1", "M0 2C0 0 1 2 1 0", Intersections{ // parallel at intersection
			{Point{0.5, 1.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.0}, false, false},
		}},
		{"M0 1L1 1", "C0 3 1 -1 1 2", Intersections{ // three intersections
			{Point{0.0791512117, 1.0}, [2]float64{0.0791512117, 0.1726731646}, [2]float64{0.0, 74.05460410 / 180.0 * math.Pi}, false, false},
			{Point{0.5, 1.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 315 / 180.0 * math.Pi}, false, false},
			{Point{0.9208487883, 1.0}, [2]float64{0.9208487883, 0.8273268354}, [2]float64{0.0, 74.05460410 / 180.0 * math.Pi}, false, false},
		}},

		// tangent
		{"L0 10", "C8 0 8 10 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true, false},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{0.5 * math.Pi, math.Pi}, true, false},
		}},
		{"M6 0L6 10", "C8 0 8 10 0 10", Intersections{
			{Point{6.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, false},
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
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false, false},
		}},
		{"M0 5L10 5", "A5 5 0 1 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false, false},
		}},
		{"M0 5L-10 5", "A5 5 0 0 0 0 10", Intersections{
			{Point{-5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{math.Pi, 0.5 * math.Pi}, false, false},
		}},
		{"M-5 0L-5 -10", "A5 5 0 0 0 -10 0", Intersections{
			{Point{-5.0, -5.0}, [2]float64{0.5, 0.5}, [2]float64{1.5 * math.Pi, math.Pi}, false, false},
		}},
		{"M0 10L10 10", "A10 5 90 0 1 0 20", Intersections{
			{Point{5.0, 10.0}, [2]float64{0.5, 0.5}, [2]float64{0.0, 0.5 * math.Pi}, false, false},
		}},

		// tangent
		{"M-5 0L-15 0", "A5 5 0 0 0 -10 0", Intersections{
			{Point{-10.0, 0.0}, [2]float64{0.5, 1.0}, [2]float64{math.Pi, 0.5 * math.Pi}, true, false},
		}},
		{"M-5 0L-15 0", "A5 5 0 0 1 -10 0", Intersections{
			{Point{-10.0, 0.0}, [2]float64{0.5, 1.0}, [2]float64{math.Pi, 1.5 * math.Pi}, true, false},
		}},
		{"L0 10", "A10 5 0 0 1 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.5 * math.Pi, 0.0}, true, false},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{0.5 * math.Pi, math.Pi}, true, false},
		}},
		{"M5 0L5 10", "A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, false},
		}},
		{"M-5 0L-5 10", "A5 5 0 0 0 0 10", Intersections{
			{Point{-5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, false},
		}},
		{"M5 0L5 20", "A10 5 90 0 1 0 20", Intersections{
			{Point{5.0, 10.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, false},
		}},
		{"M4 3L0 3", "M2 3A1 1 0 0 0 4 3", Intersections{
			{Point{4.0, 3.0}, [2]float64{0.0, 1.0}, [2]float64{math.Pi, 1.5 * math.Pi}, true, false},
			{Point{2.0, 3.0}, [2]float64{0.5, 0.0}, [2]float64{math.Pi, 0.5 * math.Pi}, true, false},
		}},

		// none
		{"M6 0L6 10", "A5 5 0 0 1 0 10", Intersections{}},
		{"M10 5L15 5", "A5 5 0 0 1 0 10", Intersections{}},
		{"M6 0L6 20", "A10 5 90 0 1 0 20", Intersections{}},

		// bugs
		{"M0 -0.7L1 -0.7", "M-0.7 0A0.7 0.7 0 0 1 0.7 0", Intersections{
			{Point{0.0, -0.7}, [2]float64{0.0, 0.5}, [2]float64{0.0, 0.0}, true, false},
		}}, // #200, at intersection the arc angle is deviated towards positive angle
		{"M30.23402723090112,37.620459766287226L30.170131507649785,37.66143576791836", "M30.242341004748596 37.609669236818846A0.8700000000000001 0.8700000000000001 0 0 1 28.82999999999447 36.9294", Intersections{
			{Point{30.170131507649785, 37.66143576791836}, [2]float64{1.0, 0.04553266140095003}, [2]float64{2.571361371137828, 2.570702752627385}, true, false},
		}}, // #280
		{"M30.23402723090112,37.620459766287226L30.170131507649785,37.66143576791836", "M30.170131507649785 37.66143576791836A0.8700000000002787 0.8700000000002787 0 0 1 28.82999999999447 36.92939999999941", Intersections{
			{Point{30.170131507649785, 37.66143576791836}, [2]float64{1.0, 0.0}, [2]float64{2.571361371137828, 2.570702753023132}, true, false},
		}}, // #280
		{"M18.28586369751671 1.9033410129748447L18.285524146153797 1.9012793871179001", "M18.285524146153797 1.9012793871179001A0.09877777777777778 0.09877777777777778 0 0 0 18.188037109374996 1.8184178602430556", Intersections{
			{Point{18.285524146153797, 1.9012793871179001}, [2]float64{1.0, 0}, [2]float64{4.549153676432458, 4.550551548951796}, true, false},
		}}, // in preview
		{"M32761.5,32383.691L32761.52,32383.691", "M31511.49999999751,33633.691 A1250 1250 0 0 1 32761.50000000074,32383.691", Intersections{
			{Point{32761.5, 32383.691}, [2]float64{0.0, 1.0}, [2]float64{0.0, 0.0}, true, false},
		}}, // #293
		{"M73643.30051730774,34889.01290159931L73639.44503270132,34889.13797316706", "M73639.44503270132,34889.13797316706A1250 1250 0 0 1 73599.5139303418 34889.76998615123", Intersections{
			{Point{73639.44503270132, 34889.13797316706}, [2]float64{1.0, 0.0}, [2]float64{3.109164117290831, 3.1097912676269237}, true, false},
		}}, // #293
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
			{Point{0.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{math.Pi, 0.5 * math.Pi}, false, false},
		}},

		// tangent
		{"A5 5 0 0 1 0 10", "M10 0A5 5 0 0 0 10 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.5}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, false},
		}},

		// fully same
		{"A5 5 0 0 1 0 10", "A5 5 0 0 1 0 10", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.0, 0.0}, true, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{math.Pi, math.Pi}, true, true},
		}},
		{"A5 5 0 0 1 0 10", "M0 10A5 5 0 0 0 0 0", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 1.0}, [2]float64{0.0, math.Pi}, true, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 0.0}, [2]float64{math.Pi, 0.0}, true, true},
		}},

		// partly same
		{"A5 5 0 0 1 0 10", "A5 5 0 0 1 5 5", Intersections{
			{Point{0.0, 0.0}, [2]float64{0.0, 0.0}, [2]float64{0.0, 0.0}, true, true},
			{Point{5.0, 5.0}, [2]float64{0.5, 1.0}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, true},
		}},
		{"A5 5 0 0 1 0 10", "M5 5A5 5 0 0 1 0 10", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.0}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 1.0}, [2]float64{math.Pi, math.Pi}, true, true},
		}},
		{"A5 5 0 0 1 0 10", "M5 5A5 5 0 0 1 -5 5", Intersections{
			{Point{5.0, 5.0}, [2]float64{0.5, 0.0}, [2]float64{0.5 * math.Pi, 0.5 * math.Pi}, true, true},
			{Point{0.0, 10.0}, [2]float64{1.0, 0.5}, [2]float64{math.Pi, math.Pi}, true, true},
		}},

		{"M30.170131507649785 37.66143576791836A0.8700000000002787 0.8700000000002787 0 0 1 28.82999999999447 36.92939999999941", "M30.242341004748596 37.609669236818846A0.8700000000000001 0.8700000000000001 0 0 1 28.82999999999447 36.9294", Intersections{
			{Point{30.170131507649785, 37.66143576791836}, [2]float64{0.0, 0.0455326614}, [2]float64{2.5707027528269983, 2.5707027528269983}, true, true},
			{Point{28.82999999999447, 36.92939999999941}, [2]float64{1.0, 1.0}, [2]float64{1.5 * math.Pi, 1.5 * math.Pi}, true, true},
		}}, // #280
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

/*func TestIntersections(t *testing.T) {
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
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, true, false},
		}, []PathIntersection{
			{Point{5.0, 0.0}, 2, 0.5, math.Pi, false, true, false},
		}},
		{"M5 5L0 0", "M-5 0A5 5 0 0 0 5 0", []PathIntersection{
			{Point{5.0 / math.Sqrt(2.0), 5.0 / math.Sqrt(2.0)}, 1, 0.292893219, 1.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{5.0 / math.Sqrt(2.0), 5.0 / math.Sqrt(2.0)}, 1, 0.75, 1.75 * math.Pi, true, false, false},
		}},

		// intersection on one segment endpoint
		{"L0 15", "M5 0L0 5L5 5", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.0, false, true, false},
		}},
		{"L0 15", "M5 0L0 5L-5 5", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, math.Pi, true, false, false},
		}},
		{"L0 15", "M5 5L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, true, false},
		}},
		{"L0 15", "M-5 5L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, false},
		}},
		{"M5 0L0 5L5 5", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.0, false, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, true, false},
		}},
		{"M5 0L0 5L-5 5", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}},
		{"M5 5L0 5L5 0", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, false},
		}},
		{"M-5 5L0 5L5 0", "L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, false, false},
		}},
		{"L0 10", "M5 0A5 5 0 0 0 0 5A5 5 0 0 0 5 10", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
		}},
		{"L0 10", "M5 10A5 5 0 0 1 0 5A5 5 0 0 1 5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.5 * math.Pi, false, true, false},
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
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, -6.0}.Angle(), false, true, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, 4.0}.Angle(), true, true, false},
		}},
		{"L10 6L20 0", "M20 10L10 6L0 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, -6.0}.Angle(), true, true, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, 4.0}.Angle(), true, true, false},
		}},
		{"M20 0L10 6L0 0", "M0 10L10 6L20 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, -6.0}.Angle(), false, true, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{10.0, 4.0}.Angle(), false, true, false},
		}},
		{"M20 0L10 6L0 0", "M20 10L10 6L0 10", []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, -6.0}.Angle(), true, true, false},
		}, []PathIntersection{
			{Point{10.0, 6.0}, 2, 0.0, Point{-10.0, 4.0}.Angle(), false, true, false},
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

		// touches / same
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.0, false, true, false},
			{Point{2.0, 2.0}, 4, 0.0, 1.5 * math.Pi, false, true, true},
		}},
		{"L2 0L2 2L0 2z", "M2 0L2 2L4 2L4 0z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, true, true, true},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, true, false},
		}},
		{"M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z", []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.0, false, true, false},
			{Point{2.0, 2.0}, 4, 0.0, 1.5 * math.Pi, false, true, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, true, false},
		}},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", []PathIntersection{
			{Point{2.0, 1.0}, 2, 0.5, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 1.0}, 1, 0.0, 0.0, false, true, false},
			{Point{2.0, 2.0}, 4, 0.5, 1.5 * math.Pi, false, true, true},
		}},
		{"L2 0L2 2L0 2z", "M2 -1L4 -1L4 1L2 1z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 1.0}, 2, 0.5, 0.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 4, 0.5, 1.5 * math.Pi, false, true, false},
			{Point{2.0, 1.0}, 4, 0.0, 1.5 * math.Pi, false, true, true},
		}},
		{"L2 0L2 2L0 2z", "M2 -1L4 -1L4 3L2 3z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 2.0}, 3, 0.0, math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 4, 0.75, 1.5 * math.Pi, false, true, false},
			{Point{2.0, 2.0}, 4, 0.25, 1.5 * math.Pi, false, true, true},
		}},
		{"M0 -1L2 -1L2 3L0 3z", "M2 0L4 0L4 2L2 2z", []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.25, 0.5 * math.Pi, false, true, true},
			{Point{2.0, 2.0}, 2, 0.75, 0.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.0, 0.0, false, true, false},
			{Point{2.0, 2.0}, 4, 0.0, 1.5 * math.Pi, false, true, true},
		}},
		{"L1 0L1 1zM2 0L1.9 1L1.9 -1z", "L1 0L1 -1zM2 0L1.9 1L1.9 -1z", []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.0, 0.0, true, true, true},
			{Point{1.0, 0.0}, 2, 0.0, 0.5 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.0, 0.0, false, true, true},
			{Point{1.0, 0.0}, 2, 0.0, 1.5 * math.Pi, false, true, false},
		}},

		// head-on collisions
		{"M2 0L2 2L0 2", "M4 2L2 2L2 4", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.5 * math.Pi, false, true, false},
		}},
		{"M0 2Q2 4 2 2Q4 2 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, 0.75 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M0 2C0 4 2 4 2 2C4 2 4 4 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M0 2A1 1 0 0 0 2 2A1 1 0 0 1 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M0 2A1 1 0 0 1 2 2A1 1 0 0 1 2 4", "M2 4L2 2L4 2", []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, true, false, false},
			{Point{2.0, 4.0}, 2, 1.0, math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
			{Point{2.0, 4.0}, 1, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M0 2A1 1 0 0 1 2 2A1 1 0 0 1 2 4", "M2 0L2 2L0 2", []PathIntersection{
			{Point{0.0, 2.0}, 1, 0.0, 1.5 * math.Pi, true, true, false},
			{Point{2.0, 2.0}, 2, 0.0, 0.0, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 2.0}, 2, 1.0, math.Pi, false, true, false},
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
			{Point{1.0, 0.0}, 1, 0.0, 0.0, true, true, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L1 -1L0 0z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, true, true, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 1.5 * math.Pi, false, true, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L0 0L1 -1z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, true, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, math.Pi, false, true, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L2 0L1 1z", []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.5, 0.0, false, true, false},
			{Point{1.0, 1.0}, 4, 0.75, 1.5 * math.Pi, false, true, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 2, 0.0, 0.75 * math.Pi, true, true, false},
			{Point{1.0, 1.0}, 3, 0.0, 1.5 * math.Pi, true, true, true},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L1 1L2 0z", []PathIntersection{
			{Point{2.0, 0.0}, 1, 0.5, 0.0, true, true, false},
			{Point{1.0, 1.0}, 4, 0.75, 1.5 * math.Pi, true, true, true},
		}, []PathIntersection{
			{Point{2.0, 0.0}, 3, 0.0, math.Pi, true, true, true},
			{Point{1.0, 1.0}, 2, 0.0, 1.75 * math.Pi, true, true, false},
		}},
		{"M1 0L3 0L3 4L1 4z", "M1 0L2 1L0 1z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, false},
			{Point{1.0, 1.0}, 4, 0.75, 1.5 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.25 * math.Pi, true, false, false},
			{Point{1.0, 1.0}, 2, 0.5, math.Pi, false, false, false},
		}},

		// intersection with overlapping lines
		{"L0 15", "M5 0L0 5L0 10L5 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, true, true, true},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{0.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, true, false},
		}},
		{"L0 15", "M5 0L0 5L0 10L-5 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, true},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, false, true},
			{Point{0.0, 10.0}, 3, 0.0, 0.75 * math.Pi, true, false, false},
		}},
		{"L0 15", "M5 15L0 10L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, true, true},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 3, 0.0, 1.75 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 2, 0.0, 1.5 * math.Pi, false, true, true},
		}},
		{"L0 15", "M5 15L0 10L0 5L-5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 1.0 / 3.0, 0.5 * math.Pi, false, false, true},
			{Point{0.0, 10.0}, 1, 2.0 / 3.0, 0.5 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 3, 0.0, 1.25 * math.Pi, true, false, false},
			{Point{0.0, 10.0}, 2, 0.0, 1.5 * math.Pi, true, false, true},
		}},
		{"L0 10L-5 15", "M5 0L0 5L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, true, true, true},
			{Point{0.0, 10.0}, 2, 0.0, 0.75 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{0.0, 10.0}, 2, 0.5, 0.5 * math.Pi, false, true, false},
		}},
		{"L0 10L5 15", "M5 0L0 5L0 15", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, false, true},
			{Point{0.0, 10.0}, 2, 0.0, 0.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, false, true},
			{Point{0.0, 10.0}, 2, 0.5, 0.5 * math.Pi, true, false, false},
		}},
		{"L0 10L-5 15", "M0 15L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, false, true, true},
			{Point{0.0, 10.0}, 2, 0.0, 0.75 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, true, false},
			{Point{0.0, 10.0}, 1, 0.5, 1.5 * math.Pi, false, true, true},
		}},
		{"L0 10L5 15", "M0 15L0 5L5 0", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.5 * math.Pi, true, false, true},
			{Point{0.0, 10.0}, 2, 0.0, 0.25 * math.Pi, true, false, false},
		}, []PathIntersection{
			{Point{0.0, 5.0}, 2, 0.0, 1.75 * math.Pi, false, false, false},
			{Point{0.0, 10.0}, 1, 0.5, 1.5 * math.Pi, false, false, true},
		}},
		{"L5 5L5 10L0 15", "M10 0L5 5L5 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, true, true},
			{Point{5.0, 10.0}, 3, 0.0, 0.75 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{5.0, 10.0}, 2, 0.5, 0.5 * math.Pi, false, true, false},
		}},
		{"L5 5L5 10L10 15", "M10 0L5 5L5 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, false, true},
			{Point{5.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, false, true},
			{Point{5.0, 10.0}, 2, 0.5, 0.5 * math.Pi, true, false, false},
		}},
		{"L5 5L5 10L0 15", "M10 0L5 5L5 10L10 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, true, true},
			{Point{5.0, 10.0}, 3, 0.0, 0.75 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{5.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, true, false},
		}},
		{"L5 5L5 10L10 15", "M10 0L5 5L5 10L0 15", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, false, true},
			{Point{5.0, 10.0}, 3, 0.0, 0.25 * math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, false, true},
			{Point{5.0, 10.0}, 3, 0.0, 0.75 * math.Pi, true, false, false},
		}},
		{"L5 5L5 10L10 15L5 20", "M10 0L5 5L5 10L10 15L10 20", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, true, true, true},
			{Point{10.0, 15.0}, 4, 0.0, 0.75 * math.Pi, true, true, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{10.0, 15.0}, 4, 0.0, 0.5 * math.Pi, false, true, false},
		}},
		{"L5 5L5 10L10 15L5 20", "M10 20L10 15L5 10L5 5L10 0", []PathIntersection{
			{Point{5.0, 5.0}, 2, 0.0, 0.5 * math.Pi, false, true, true},
			{Point{10.0, 15.0}, 4, 0.0, 0.75 * math.Pi, false, true, false},
		}, []PathIntersection{
			{Point{5.0, 5.0}, 4, 0.0, 1.75 * math.Pi, false, true, false},
			{Point{10.0, 15.0}, 2, 0.0, 1.25 * math.Pi, false, true, true},
		}},
		{"L2 0L2 1L0 1z", "M1 0L3 0L3 1L1 1z", []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.5, 0.0, true, false, true},
			{Point{2.0, 0.0}, 2, 0.0, 0.5 * math.Pi, true, false, false},
			{Point{2.0, 1.0}, 3, 0.0, math.Pi, false, false, true},
			{Point{1.0, 1.0}, 3, 0.5, math.Pi, false, false, false},
		}, []PathIntersection{
			{Point{1.0, 0.0}, 1, 0.0, 0.0, false, false, true},
			{Point{2.0, 0.0}, 1, 0.5, 0.0, false, false, false},
			{Point{2.0, 1.0}, 3, 0.5, math.Pi, true, false, true},
			{Point{1.0, 1.0}, 4, 0.0, 1.5 * math.Pi, true, false, false},
		}},

		// bugs
		{"M67.89174682452696 63.79390646055095L67.89174682452696 63.91890646055095L59.89174682452683 50.06250000000001", "M68.10825317547533 63.79390646055193L67.89174682452919 63.91890646055186M67.89174682452672 63.918906460550865L59.891746824526074 50.06250000000021", []PathIntersection{
			{Point{67.89174682452696, 63.91890646055095}, 2, 0.0, 240.0 * math.Pi / 180.0, false, true, false},
			{Point{67.89174682452696, 63.91890646055095}, 2, 0.0, 240.0 * math.Pi / 180.0, false, true, true},
			{Point{59.89174682452683, 50.06250000000001}, 2, 1.0, 240.0 * math.Pi / 180.0, false, true, false},
		}, []PathIntersection{
			{Point{67.89174682452919, 63.91890646055186}, 1, 1.0, 150.0 * math.Pi / 180.0, false, true, false},
			{Point{67.89174682452672, 63.918906460550865}, 3, 0.0, 240.0 * math.Pi / 180.0, false, true, true},
			{Point{59.891746824526074, 50.06250000000021}, 3, 1.0, 240.0 * math.Pi / 180.0, false, true, false},
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
		{"L10 10L10 0L0 10z", []PathIntersection{
			{Point{5.0, 5.0}, 1, 0.5, 0.25 * math.Pi, false, false, false},
			{Point{5.0, 5.0}, 3, 0.5, 0.75 * math.Pi, true, false, false},
		}},

		// intersection
		{"M2 1L0 0L0 2L2 1L1 0L1 2z", []PathIntersection{
			{Point{2.0, 1.0}, 1, 0.0, 206.5650511771 * math.Pi / 180.0, false, false, false},
			{Point{1.0, 0.5}, 1, 0.5, 206.5650511771 * math.Pi / 180.0, true, false, false},
			{Point{1.0, 1.5}, 3, 0.5, 333.4349488229 * math.Pi / 180.0, false, false, false},
			{Point{2.0, 1.0}, 4, 0.0, 1.25 * math.Pi, true, false, false},
			{Point{1.0, 0.5}, 5, 0.25, 0.5 * math.Pi, false, false, false},
			{Point{1.0, 1.5}, 5, 0.75, 0.5 * math.Pi, true, false, false},
		}},

		// parallel segment TODO
		{"L10 0L5 0L15 0", []PathIntersection{
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L10 0L0 0L15 0", []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.5, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L10 0L15 5L20 10L15 5L10 0L0 5", []PathIntersection{
			{Point{10.0, 0.0}, 1, 0.0, 0.0, true, true, true},
			{Point{10.0, 0.0}, 6, 0.0, 0.0, true, true, true},
		}},
		{"L15 0L15 10L5 10L5 0L10 0L10 5L0 5z", []PathIntersection{
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L15 0L15 10L0 10L0 0L10 0L10 5L-5 5L-5 0z", []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.5, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L15 0L15 10L0 10L0 0L10 0L10 5L0 5z", []PathIntersection{
			{Point{0.0, 5.0}, 1, 0.5, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L-5 0A5 5 0 0 1 5 0A5 5 0 0 1 -5 0z", []PathIntersection{
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L-5 0A5 5 0 0 1 5 0A5 5 0 0 1 -5 0L0 0L0 1L1 0L0 -1z", []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.0, math.Pi, false, true, true},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, true},
		}},
		{"L15 0L15 5L5 0L10 0L15 -5", []PathIntersection{
			{Point{5.0, 0.0}, 1, 1.0 / 3.0, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 2.0 / 3.0, 0.0, false, true, true},
		}},
		{"L15 0L15 5L10 0L5 0L0 5", []PathIntersection{
			{Point{5.0, 0.0}, 1, 1.0 / 3.0, 0.0, false, true, true},
			{Point{10.0, 0.0}, 3, 2.0 / 3.0, 0.0, false, true, true},
		}},

		// bugs
		{"M3.512162397982181 1.239754268684486L3.3827323986701674 1.1467946944092953L3.522449858001167 1.2493787337129587A0.21166666666666667 0.21166666666666667 0 0 1 3.5121623979821806 1.2397542686844856z", []PathIntersection{}}, // #277, very small circular arc at the end of the path to the start
		{"M-0.1997406229376793 296.9999999925494L-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834L-0.19999999999964735 20.77454766193328", []PathIntersection{
			{Point{-0.1997406229376793, 158.88740172019808}, 1, 0.9999999986401219, 270.0 * math.Pi / 180.0, true, false, false},
			{Point{-0.1997406229376793, 158.88740172019808}, 3, 1.359651237533596e-09, 269.9998923980606 * math.Pi / 180.0, false, false, false},
		}}, // #287
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			zs, _ := pathIntersections(p, nil, true, true)
			test.T(t, zs, tt.zs)
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
		ttrs := []*Path{}
		for _, rs := range tt.rs {
			ttrs = append(ttrs, MustParseSVGPath(rs))
		}
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			rs := p.Cut(q)
			test.T(t, rs, ttrs)
		})
	}
}*/

func boSP(a, b Point, clipping bool) *SweepPoint {
	vertical := Equal(a.X, b.X)
	increasing := a.X < b.X
	if vertical {
		increasing = a.Y < b.Y
	}
	selfWindings := 1
	if !increasing {
		selfWindings = -1
	}
	A := &SweepPoint{
		Point:        a,
		left:         increasing,
		selfWindings: selfWindings,
		vertical:     vertical,
		clipping:     clipping,
	}
	B := &SweepPoint{
		Point:        b,
		left:         !increasing,
		selfWindings: selfWindings,
		vertical:     vertical,
		clipping:     clipping,
	}
	A.other = B
	B.other = A
	return A
}

func TestBentleyOttmannSortH(t *testing.T) {
	var tts = []struct {
		a, b *SweepPoint
		cmp  int
	}{
		// horizontal
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{5, 0}, Point{15, 0}, false), -1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{-5, 0}, Point{15, 0}, false), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{-10, 0}, Point{0, 0}, false), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{10, -1}, false), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{10, 1}, false), -1},

		// horizontal left/right
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{-10, 0}, false), 1},

		// horizontal overlap
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{10, 0}, true), -1},
		{boSP(Point{10, 0}, Point{0, 0}, false), boSP(Point{10, 0}, Point{0, 0}, true), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{20, 0}, false), 0},
		{boSP(Point{20, 0}, Point{0, 0}, false), boSP(Point{20, 0}, Point{10, 0}, false), 0},

		// vertical
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 5}, Point{0, 15}, false), -1},
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, -5}, Point{0, 15}, false), 1},
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, -10}, Point{0, 0}, false), 1},
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{-1, 10}, false), 1},
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{1, 10}, false), 1},

		// vertical left/right
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{0, -10}, false), 1},

		// vertical overlap
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{0, 10}, true), -1},
		{boSP(Point{0, 10}, Point{0, 0}, false), boSP(Point{0, 10}, Point{0, 0}, true), 1},
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{0, 20}, false), 0},
		{boSP(Point{0, 20}, Point{0, 0}, false), boSP(Point{0, 20}, Point{0, 10}, false), 0},

		// CCW order for left and right endpoints
		{boSP(Point{0, 0}, Point{-1, 10}, false), boSP(Point{0, 0}, Point{-10, 0}, false), -1},
		{boSP(Point{0, 0}, Point{-10, 0}, false), boSP(Point{0, 0}, Point{0, -10}, false), -1},
		{boSP(Point{0, 0}, Point{0, -10}, false), boSP(Point{0, 0}, Point{1, -10}, false), -1},
		{boSP(Point{0, 0}, Point{1, -10}, false), boSP(Point{0, 0}, Point{10, 0}, false), -1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{0, 10}, false), -1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{-1, 10}, false), 1},

		{boSP(Point{0, 10}, Point{10, 10}, false), boSP(Point{0, 10}, Point{0, 0}, false), 1},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.a, "x", tt.b), func(t *testing.T) {
			x := tt.a.LessH(tt.b)
			y := tt.b.LessH(tt.a)
			cmp := 0
			if x != y {
				if x {
					cmp = -1
				} else {
					cmp = 1
				}
			}
			test.T(t, cmp, tt.cmp)

			//test.T(t, tt.a.CompareH(tt.b), tt.cmp)
		})
	}
}

func TestBentleyOttmannSortV(t *testing.T) {
	var tts = []struct {
		a, b *SweepPoint
		cmp  int
	}{
		// common
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{-5, -5}, Point{5, -5}, false), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{-5, 5}, Point{5, 5}, false), -1},

		// same left-endpoint X
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, -5}, Point{10, -5}, false), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 5}, Point{10, 5}, false), -1},

		// same left-endpoint X and Y
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{10, -1}, false), 1},
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{10, 1}, false), -1},

		// horizontal equal
		{boSP(Point{0, 0}, Point{10, 0}, false), boSP(Point{0, 0}, Point{0, 10}, true), -1},

		// vertical
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{10, 0}, false), 1},
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{0, 5}, true), -1},

		// vertical equal
		{boSP(Point{0, 0}, Point{0, 10}, false), boSP(Point{0, 0}, Point{0, 10}, true), -1},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.a, "x", tt.b), func(t *testing.T) {
			test.T(t, tt.a.CompareV(tt.b), tt.cmp)
			test.T(t, tt.b.CompareV(tt.a), -tt.cmp)
		})
	}
}

func TestBentleyOttmannPrecision(t *testing.T) {
	var tts = []struct {
		p  string
		op pathOp
		q  string
		r  string
	}{
		// intersection is moved upwards
		{"L4 1L2 2L1 1L4 0z", opSettle, "", "L4 0L2 1zM1 1L4 1L2 2z"},

		// intersection is moved downwards
		{"L4 1L2 2L-1 1L4 0z", opSettle, "", "M-1 1L2 0L4 1L2 2z"},

		// one intersection and one passing snap to (2,2)
		{"M-2 -1L5 3L-3 1L-2 0L5 4L6 4L6 -1z", opSettle, "", "M-3 1L-2 0L2 2zM-2 -1L6 -1L6 4L5 4L2 2z"},

		// two intersections snap to (2,2)
		{"M-2 -1L4 3L5 3L-3 1L-2 4L4 1z", opSettle, "", "M-3 1L2 2L-2 4zM-2 -1L4 1L2 2z"},
		{"M-2 -1L7 5L9 4L-3 1L-2 4L4 1z", opSettle, "", "M-3 1L2 2L-2 4zM-2 -1L4 1L2 2zM3 3L9 4L7 5z"},

		// one intersections in (2,1) and one in (2,2), first intersection causes second
		{"M0 1L4 1L4 2L0 2L0 3L3 3L1 0z", opSettle, "", "M0 1L1 0L2 1zM0 2L2 2L3 3L0 3zM2 1L4 1L4 2L2 2z"},

		// one intersections in (2,1) and one in (2,2), first intersection passes through (2,2)
		{"M0 1L4 1L4 -1L-1 -1L7 5L11 6L-15 -4zM-4 1L4 3L4 4L-4 4z", opSettle, "", "M-15 -4L0 1L2 1L2 2zM-4 1L2 2L4 3L4 4L-4 4zM-1 -1L4 -1L4 1L2 1zM4 3L11 6L7 5z"},

		// segments becomes vertical and overlapping
		{"M1 4L2.1 1L2 3L2 0L3 0L3 4z", opSettle, "", "M1 4L2 3L2 0L3 0L3 4z"},
		{"M1 4L2.4 1L2 3L2 0L3 0L3 4z", opSettle, "", "M1 4L2 3L2 0L3 0L3 4z"},
		{"M2.6 4L2.4 1L2 3L2 0L3 0L3 4z", opSettle, "", "M2 0L3 0L3 4L2 1z"},

		// collapse only bottom-left corner
		{"M0 2L2 2L1 3L1 1z", opSettle, "", "M1 2L2 2L1 3z"},
		{"M0 2L2 2L1 1L1 3z", opSettle, "", "M0 2L1 2L1 3zM1 1L2 2L1 2z"},

		// segment is almost vertical but downward-sloped
		{"M0 2L2 2L1 3L1.0000000000000002 0z", opSettle, "", "M0 2L1 0L1 2zM1 2L2 2L1 3z"},

		// order of overlapping segments
		{"M0 2L1 1L3 -3L4 -3L4 2z", opOR, "M0 2L1 1L2.1 -1L4 -1L4 2z", "M0 2L1 1L3 -3L4 -3L4 2z"},
		{"M1 2L4 1L3 2zM1 3L2.4 0L5 3z", opSettle, "", "M1 2L2 2L2 0L3 1L4 1L5 3L1 3z"},

		// breakup test
		{"M0 2L1 0L2 2zM0 0L2 -1L2 -3z", opSettle, "", "L2 -3L2 -1L1 0zM0 2L1 0L2 2z"},

		// segment crosses square in between
		{"L3 0L3 7L1.1 7zM1 1L2 2L2 1zM1 3L2 4L2 3zM1 5L2 6L2 5z", opSettle, "", "M0 0L3 0L3 7L1 7L1 5L2 6L2 5L1 5L1 3L2 4L2 3L1 3zM1 1L2 2L2 1z"},
		{"M1.2 0L3 0L3 10L0 10zM1 3L2 3L2 1zM1 6L2 6L2 4zM1 9L2 9L2 7z", opSettle, "", "M0 10L1 6L2 6L2 4L1 6L1 3L2 3L2 1L1 3L1 0L3 0L3 10zM1 9L2 9L2 7z"},

		{"M0 0L5 2L5 4L0 4L4 -2L4 -1L0 2z", opSettle, "", "M0 0L2 1L0 2zM0 4L2 1L5 2L5 4z"},
	}

	origEpsilon := BentleyOttmannEpsilon
	BentleyOttmannEpsilon = 1.0
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			ps := MustParseSVGPath(tt.p).Split()
			var qs []*Path
			if tt.q != "" {
				qs = MustParseSVGPath(tt.q).Split()
			}

			r := bentleyOttmann(ps, qs, tt.op, NonZero)
			test.T(t, r.Merge(), MustParseSVGPath(tt.r))

			if tt.q != "" {
				r = bentleyOttmann(qs, ps, tt.op, NonZero)
				test.T(t, r.Merge(), MustParseSVGPath(tt.r), "swapped arguments")
				//} else {
				//	r = bentleyOttmann(r.Split(), nil, tt.op, NonZero)
				//	test.T(t, r, MustParseSVGPath(tt.r), "idempotency")
			}
		})
	}
	BentleyOttmannEpsilon = origEpsilon
}

func TestBentleyOttmannPerformance(t *testing.T) {
	var tts = []struct {
		p  string
		op pathOp
		q  string
		d  time.Duration
		r  string
	}{
		// performance bug, takes >5s
		{"M0.05603022976765715 -0.0003059739935906691L0.01800512753993644 -0.0030528993247571634z M0.024894563646512324 -0.002555207919613167L0.12000309429959088 -0.002555207919613167z", opSettle, "", 10 * time.Millisecond, ""},
		{"M0.05603022976765715 -0.0003059739935906691L0.01800512753993644 -0.0030528993247571634z M0.05603022976765715 -0.0003059739935906691L0.024894563646512324 -0.002555207919613167L0.12000309429959088 -0.002555207919613167z", opSettle, "", 10 * time.Millisecond, "M0.02489459 -0.00255521L0.12000309 -0.00255521L0.05603023 -0.00030597z"},
	}

	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p), func(t *testing.T) {
			ps := MustParseSVGPath(tt.p).Split()
			var qs []*Path
			if tt.q != "" {
				qs = MustParseSVGPath(tt.q).Split()
			}

			t0 := time.Now()
			r := bentleyOttmann(ps, qs, tt.op, NonZero)
			test.T(t, r.Merge(), MustParseSVGPath(tt.r))
			if d := time.Since(t0); tt.d < d {
				test.Fail(t, fmt.Sprintf("takes too long: %v instead of <%v", d, tt.d))
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
		{NonZero, "L10 10L10 0L0 10z", "L5 5L0 10zM5 5L10 0L10 10z"},
		{NonZero, "L10 10L0 10L10 0z", "L10 0L5 5zM0 10L5 5L10 10z"},
		{NonZero, "L10 10L20 0L20 10L10 0L0 10z", "L5 5L0 10zM5 5L10 0L15 5L10 10zM15 5L20 0L20 10z"},

		// single path with inner part doubly winded
		{NonZero, "M0 2L6 2L4 4L1 1L5 1L2 4z", "M0 2L2 2L1 1L5 1L4 2L6 2L4 4L3 3L2 4z"},               //  ccw
		{NonZero, "M0 2L2 4L5 1L1 1L4 4L6 2z", "M0 2L2 2L1 1L5 1L4 2L6 2L4 4L3 3L2 4z"},               // !ccw
		{EvenOdd, "M0 2L6 2L4 4L1 1L5 1L2 4z", "M0 2L2 2L3 3L2 4zM1 1L5 1L4 2L2 2zM3 3L4 2L6 2L4 4z"}, //  ccw
		{EvenOdd, "M0 2L2 4L5 1L1 1L4 4L6 2z", "M0 2L2 2L3 3L2 4zM1 1L5 1L4 2L2 2zM3 3L4 2L6 2L4 4z"}, // !ccw

		// two paths with overlapping part either zero or doubly winded
		{NonZero, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "L10 0L10 5L15 5L15 15L5 15L5 10L0 10z"},
		{NonZero, "L4 0L4 5L6 5L6 10L0 10zM2 2L8 2L8 8L2 8z", "L4 0L4 2L8 2L8 8L6 8L6 10L0 10z"},                          //  ccwA  ccwB
		{NonZero, "L4 0L4 5L6 5L6 10L0 10zM2 2L2 8L8 8L8 2z", "L4 0L4 2L2 2L2 8L6 8L6 10L0 10zM4 2L8 2L8 8L6 8L6 5L4 5z"}, //  ccwA !ccwB
		{NonZero, "L0 10L6 10L6 5L4 5L4 0zM2 2L8 2L8 8L2 8z", "L4 0L4 2L2 2L2 8L6 8L6 10L0 10zM4 2L8 2L8 8L6 8L6 5L4 5z"}, // !ccwA  ccwB
		{NonZero, "L0 10L6 10L6 5L4 5L4 0zM2 2L2 8L8 8L8 2z", "L4 0L4 2L8 2L8 8L6 8L6 10L0 10z"},                          // !ccwA !ccwB

		// same but flipped on Y (different starting vertex)
		{NonZero, "L6 0L6 5L4 5L4 10L0 10zM2 2L8 2L8 8L2 8z", "L6 0L6 2L8 2L8 8L4 8L4 10L0 10z"},                          //  ccwA  ccwB
		{NonZero, "L6 0L6 5L4 5L4 10L0 10zM2 2L2 8L8 8L8 2z", "L6 0L6 2L2 2L2 8L4 8L4 10L0 10zM4 5L6 5L6 2L8 2L8 8L4 8z"}, //  ccwA !ccwB
		{NonZero, "L0 10L4 10L4 5L6 5L6 0zM2 2L8 2L8 8L2 8z", "L6 0L6 2L2 2L2 8L4 8L4 10L0 10zM4 5L6 5L6 2L8 2L8 8L4 8z"}, // !ccwA  ccwB
		{NonZero, "L0 10L4 10L4 5L6 5L6 0zM2 2L2 8L8 8L8 2z", "L6 0L6 2L8 2L8 8L4 8L4 10L0 10z"},                          // !ccwA !ccwB

		// multiple paths
		{NonZero, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "L10 0L10 5L15 5L15 15L5 15L5 10L0 10z"},
		{EvenOdd, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "L10 0L10 5L5 5L5 10L0 10zM5 10L10 10L10 5L15 5L15 15L5 15z"},
		{NonZero, "L4 0L4 4L0 4zM-1 1L1 1L1 3L-1 3zM3 1L5 1L5 3L3 3zM4.5 1.5L5.5 1.5L5.5 2.5L4.5 2.5z", "M-1 1L0 1L0 0L4 0L4 1L5 1L5 1.5L5.5 1.5L5.5 2.5L5 2.5L5 3L4 3L4 4L0 4L0 3L-1 3z"},
		{EvenOdd, "L4 0L4 4L0 4zM-1 1L1 1L1 3L-1 3zM3 1L5 1L5 3L3 3zM4.5 1.5L5.5 1.5L5.5 2.5L4.5 2.5z", "M-1 1L0 1L0 3L-1 3zM0 0L4 0L4 1L3 1L3 3L4 3L4 4L0 4L0 3L1 3L1 1L0 1zM4 1L5 1L5 1.5L4.5 1.5L4.5 2.5L5 2.5L5 3L4 3zM5 1.5L5.5 1.5L5.5 2.5L5 2.5z"},

		// tangent
		{NonZero, "L5 5L10 0L10 10L5 5L0 10z", "L5 5L0 10zM5 5L10 0L10 10z"},
		{NonZero, "L2 2L3 0zM1 0L2 2L4 0L4 -1L1 -1z", "M0 0L1 0L1 -1L4 -1L4 0L2 2z"},
		{NonZero, "L2 2L3 0zM1 0L2 2L4 0L4 3L1 3z", "L1 0L1 1zM1 0L3 0L2 2zM1 1L2 2L4 0L4 3L1 3z"},

		// parallel segments
		{NonZero, "L1 0L1 1L0 1zM1 0L1 1L2 1L2 0z", "L2 0L2 1L0 1z"},
		{NonZero, "L1 0L1 1L0 1zM1 0L2 0L2 1L1 1z", "L2 0L2 1L0 1z"},
		{NonZero, "L10 0L5 2L5 8L0 10L10 10L5 8L5 2z", "L10 0L5 2zM0 10L5 8L10 10z"},
		{NonZero, "L10 0L10 10L5 7.5L10 5L10 15L0 15z", "L10 0L10 15L0 15z"},
		{EvenOdd, "L10 0L10 10L5 7.5L10 5L10 15L0 15z", "L10 0L10 5L5 7.5L10 10L10 15L0 15z"},
		{NonZero, "L10 0L10 5L0 10L0 5L10 10L10 15L0 15z", "L10 0L10 5L5 7.5L10 10L10 15L0 15z"},
		{EvenOdd, "L10 0L10 5L0 10L0 5L10 10L10 15L0 15z", "L10 0L10 5L5 7.5L0 5zM0 10L5 7.5L10 10L10 15L0 15z"},
		{NonZero, "L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", "L10 0L10 5L7.5 7.5L5 5L2.5 7.5L0 5zM0 10L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z"},
		{EvenOdd, "L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", "L10 0L10 5L7.5 7.5L5 5L2.5 7.5L0 5zM0 10L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z"},
		{NonZero, "L3 0L3 1L0 1zM1 0L1 1L2 1L2 0z", "L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},

		// self-overlap
		{Positive, "L2 0L1 0L3 0L1.5 1z", "L3 0L1.5 1z"},
		{Positive, "L0 2L0 1L0 3L-1 1.5z", "M-1 1.5L0 0L0 3z"},
		{Positive, "L2 0L1 0L3 0z", ""},
		{Positive, "L0 2L0 1L0 3z", ""},

		// non flat
		//{"M0 1L4 1L4 3L0 3zM4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 0 2 3L0 3L0 1L4 1zM4 3A1 1 0 0 1 2 3z"}, // TODO

		// special cases
		{NonZero, "M1 4L0 2L1 0L2 0L2 4zM1 3L0 2L1 1z", "M0 2L1 0L2 0L2 4L1 4z"},                          // tangent left-most endpoint
		{NonZero, "M0 2L1 0L2 0L2 4L1 4zM0 2L1 1L1 3z", "M0 2L1 0L2 0L2 4L1 4z"},                          // tangent left-most endpoint
		{NonZero, "M0 2L1 0L2 0L2 4L1 4zM0 2L1 3L1 1z", "M0 2L1 0L2 0L2 4L1 4L0 2L1 3L1 1z"},              // tangent left-most endpoint
		{NonZero, "M0 2L1 0L2 1L1 3zM0 2L1 1L2 3L1 4z", "M0 2L1 0L2 1L1.5 2L2 3L1 4z"},                    // secant left-most endpoint
		{NonZero, "M0 2L1 0L2 1L1 3zM0 2L1 4L2 3L1 1z", "M0 2L1 0L2 1L1.5 2L1 1zM0 2L1 3L1.5 2L2 3L1 4z"}, // secant left-most endpoint
		{NonZero, "L2 0L2 2L0 2L0 1L-1 2L-2 2L-1 2L0 1z", "L2 0L2 2L0 2z"},                                // parallel left-most endpoint
		{NonZero, "L0 1L-1 2L0 1z", ""}, // all parallel
		{NonZero, "M1 0L1 2.1L1.1 2.1L1.1 2L0 2L0 3L2 3L2 1L5 1L5 2L5.1 2.1L5.1 1.9L5 2L5 3L6 3L6 0z", "M0 2L1 2L1 0L6 0L6 3L5 3L5 1L2 1L2 3L0 3z"}, // CCW open first leg for CW path
		//{Positive, "M4.428086304186892 0.375A0.375 0.375 0 0 1 4.428086304186892 -0.375L6.428086304186892 -0.375A0.375 0.375 0 0 1 6.428086304186892 0.375z", "M4.428086304186892 0.375A0.375 0.375 0 0 1 4.053086304186892 4.592425496802568e-17A0.375 0.375 0 0 1 4.428086304186892 -0.375L6.428086304186892 -0.375A0.375 0.375 0 0 1 6.803086304186892 -9.184850993605136e-17A0.375 0.375 0 0 1 6.428086304186892 0.375z"}, // two arcs as leftmost point TODO

		// example from Subramaniam's thesis
		{NonZero, "L5.5 10L4.5 10L9 1.5L1 8L9 8L1.6 2L3 1L5 7L8 0z", "M0 0L8 0L6.47945206 3.5479452L9 1.5L6.59232481 6.04783092L9 8L5.55882353 8L4.9904632200000005 9.07356948L4.4 8L1 8L3.34989201 6.09071274zM1.6 2L3.97530864 3.92592593L3 1zM4.5 10L4.9904632200000005 9.07356948L5.5 10z"},
		{EvenOdd, "L5.5 10L4.5 10L9 1.5L1 8L9 8L1.6 2L3 1L5 7L8 0z", "M0 0L8 0L6.47945206 3.5479452L4.99583767 4.75338189L3.97530864 3.92592593L3 1L1.6 2L3.97530864 3.92592593L4.40983607 5.2295082L3.34989201 6.09071274zM1 8L3.34989201 6.09071274L4.4 8zM4.4 8L5.55882353 8L4.9904632200000005 9.07356948zM4.40983607 5.2295082L4.99583767 4.75338189L5.71346705 5.33524355L5 7zM4.5 10L4.9904632200000005 9.07356948L5.5 10zM5.55882353 8L6.59232481 6.04783092L9 8zM5.71346705 5.33524355L6.47945206 3.5479452L9 1.5L6.59232481 6.04783092z"},

		// bugs
		{Negative, "M2 1L0 0L0 2L2 1L1 0L1 2z", "L1 0.5L1 0L2 1L1 2L1 1.5L0 2z"},
		{Positive, "M0 -1L10 -1L10 1L5 1L5 -1L10 -1L10 1L0 1z", "M0 -1L10 -1L10 1L0 1z"},
		{Positive, "M0.346107634210633 0.2871967618163768L0.3626348519907416 0.28892214962920265L0.3626062868506122 0.28891875151562996L0.3796118521641162 0.2911902442838161L0.3880491429729769 0.2909157121602171L0.38726252832823393 0.2905730077076176L0 1z", "M0 1L0.34610763 0.28719676L0.36262054 0.28892066L0.37961185000000003 0.29119024L0.38705785 0.29094797z"},
		{Positive, "M0.780676733347056 0.3997867413798298L0.784973608347056 0.3943179913798298L0.7810078125 0.3942734375000001L0.7845234375 0.39896093750000006L0.7848135846777966 0.39563709448964984L0.7785635846777966 0.40149646948964984L0.7826628850218049 0.40258509787790625L0.7811003850218049 0.3975069728779062z", "M0.78148617 0.39875654L0.78296169 0.39687861L0.78317951 0.39716904000000003z"},
		{Positive, "M0.4650156250267547 2.7484372686791643L0.4654062500267547 1.0597653936791642L0.46540625 1.059765625L0.46540625 0.7642365158052812L0.4676251520722006 0.7593549312464399L0.5078703048822648 0.7451507596664171L0.5085359451177351 0.7470367403335829L0.4686921951177352 0.761099240333583L0.46926974147746264 0.7605700529443012L0.46731661647746264 0.7648669279443012L0.46740625 0.7644531250000001L0.46740625 1.0597657406604195L0.4670156249732454 2.748437731320836z", "M0.46501563 2.74843727L0.46540625 1.05976563L0.46540625 0.76423652L0.46762515 0.75935493L0.5078703 0.74515076L0.50853595 0.74703674L0.4690936 0.76095757L0.46740625 0.76466974L0.46740625 1.05976574L0.46701562 2.74843773z"},
		{NonZero, "M3.99964905 7.68626398L4.0003194 7.68603185L4.0003194 7.68986693zM3.99996948 7.68618779L4.00137328 7.68556219L4.00234985 7.68550107z", "M3.99964905 7.6862639800000006L4.0003193900000005 7.68603185L4.0003194 7.68603185L4.00137328 7.68556219L4.00234985 7.68550107L4.0003194 7.68608684L4.0003194 7.68986693z"},
		{NonZero, "M6.04896518 2.0000894L6.04935631 2.0003194zM6.0509217500000005 2.0003194L6.05165396 2.00001907L6.05184385 2.0003194zM6.04927096 2.00019985L6.04952461 2.00055514L6.05051263 2.00086559L6.05099455 2.00022221L6.05165396 2.00001907L6.05181993 2.00015502L6.05185513 2.00093121L6.06848788 2.0390040000000003z", "M6.04927096 2.00019985L6.04935631 2.0003194L6.04952461 2.00055514L6.05051263 2.00086559L6.0509217500000005 2.0003194L6.05099455 2.00022221L6.05165396 2.00001907L6.05181993 2.0001550200000002L6.05182611 2.00029135L6.05184385 2.0003194L6.05182738 2.0003194L6.05185513 2.00093121L6.06848788 2.0390040000000003L6.04931942 2.00029771z"}, // SweepPoint pool reuse for collapsed segments causes conflict pointers in `handled`
		{NonZero, "M1.37118285 6.0003194L1.37137777 6.00027829L1.88426666 5.93378961zM1.37102221 6.00070481L1.37119999 6.00027829L1.9712163 5.9996806000000005z", "M1.37102221 6.00070481L1.37118285 6.0003194L1.37118286 6.0003194L1.37119999 6.00027829L1.37137915 6.00027811L1.88426666 5.93378961L1.37150223 6.00027799L1.9712163 5.9996806000000005z"},
		{NonZero, "M-0.052000000000000005 8.052000000000001L-0.052000000000000005 7.999511116117297L-0.017996398619230213 7.998320624626715L-0.052000000000000005 8.002319400000001L-0.052000000000000005 2.695363795711625z", "M-0.052000000000000005 7.99951112L-0.0179964 7.99832062L-0.052000000000000005 8.0023194z"},
		{NonZero, "M1.65138996 6.46265112L1.68750998 6.43749002L1.68750998 6.50000998zM1.75000998 6.43749002L1.68749002 6.50000998zM1.66805433 6.43750998L1.68750998 6.43750998zM1.75000998 6.43750998L1.68749002 6.43750998z", "M1.65138996 6.46265112L1.68748133 6.43750998L1.68750998 6.43749002L1.68750998 6.50000998L1.68750017 6.49999983z"},
		{NonZero, "M0.15938157957878576 7.958567843990149L0.18949002 7.946636108655732L0.18949002 7.94527026L0.18750958428124603 7.947270164307436L0.18790070591516111 7.947266338117543L0.18824865124077875 7.947128052154844L0.18550998 7.94526946L0.18550998 7.947290036168203L0.19643651485990857 7.947181440654685z", "M0.15938158 7.95856784L0.18789968 7.9472663500000005L0.18790071 7.9472663400000005L0.18790088 7.94726627L0.19643651 7.9471814400000005zM0.18550998 7.9452694600000004L0.1878908 7.9468852000000005L0.18949002 7.94527026L0.18949002 7.94663611L0.18824865000000002 7.94712805L0.18790088 7.94726627L0.18789984 7.94726628L0.18789968 7.9472663500000005L0.18750959 7.94727016L0.18750958 7.94727016L0.18550998 7.94729004z"},
		{NonZero, "M0.74999002 7.31249002L0.81250998 7.31249002L0.81250998 7.37500998L0.7621670700000001 7.37500998L0.7501199000000001 7.36803658z M0.76224438 7.37511984L0.81250998 7.37499002L0.81250998 7.38459986L0.7910117 7.3870006z M0.81249002 7.31249002L0.87500998 7.31249002L0.87500998 7.37500998L0.84717878 7.37500998L0.84754558 7.37387787L0.81249002 7.37500998z M0.81249002 7.37499002L0.84430604 7.37499002L0.84037723 7.37915786L0.82951334 7.38501444z", "M0.74999002 7.31249002L0.87500998 7.31249002L0.87500998 7.37500998L0.84717878 7.37500998L0.84754558 7.37387787L0.81310808 7.37499002L0.84430604 7.37499002L0.84037723 7.37915786L0.82951334 7.38501444L0.81252215 7.37500894L0.81250998 7.37500934L0.81250998 7.38459986L0.7910117 7.3870006L0.76224438 7.37511984L0.80478158 7.37500998L0.7621670700000001 7.37500998L0.7501199000000001 7.36803658z"},
		{NonZero, "M1.87588889 6.4393147L1.93750998 6.43749002L1.93750998 6.50000998L1.8857352 6.50000998L1.87852778 6.48022779z M1.88577778 6.50005704L1.93750998 6.49999002L1.93728625 6.50897677z", "M1.8758888900000001 6.4393147L1.93750998 6.43749002L1.93750998 6.50000998L1.9375094800000001 6.50000998L1.93728625 6.50897677L1.88577778 6.50005704L1.92210302 6.50000998L1.8857352 6.50000998L1.87852778 6.48022779z"},
		{NonZero, "M4.052 3.998L1.7328357441061668 6.75201996L1.7323368935370298 6.750676097014102L1.7323549266182294 6.750722871352709L1.7323455886569568 6.750699526449527L1.732130807990627 6.75012088513192L1.7340358225805368 6.751424694798573L1.6918801731330348 6.7520573659498595z", "M1.6918801700000001 6.75205737L4.0520000000000005 3.998L1.73359251 6.75112129L1.7340358200000001 6.75142469L1.73332807 6.75143531L1.73283574 6.75201996L1.73262265 6.7514459z"},
		{NonZero, "M1.999680599010034 -0.0003194009899516459L4.000319400989966 -0.0003194009899516459zM1.999680599010034 -2.0003194009899516L4.000319400989966 0.0003194009899516459zM3.999680599010048 -2.0003194009899516L3.999680599010048 0.0003194009899516459z", ""},
		{NonZero, "M3.013236014856396 2.0031317319105995L2.999840299505024 1.999840299505024zM2.999840299505024 2.000159700494976L3.0059914294253502 2.000159700494976L2.999840299505024 1.993265873649733zM3.000159700494976 1.999840299505024L2.360148227002341 1.999840299505024zM3.000159700494976 2.000159700494976L2.995741685965072 1.9901318032831057z", "M2.9998403000000002 1.99326587L3.00599143 2.0001597L2.9998403000000002 2.0001597z"},
		{NonZero, "M0.070313747660117 7.153464460137324L0.062498752339883 7.148436252339883zM0.062498752339883 7.148438747660117L0.070313747660117 7.148438747660117z", ""},
		{NonZero, "M1.828127495320234 7.578127495320234L1.828127495320234 7.562497504679766z M1.828122504679766 7.562502495320234L1.843752495320234 7.562502495320234z M1.841755539006334 7.57144770006846L1.841799370320234 7.562497504679766L1.826169379679766 7.562497504679766z M1.826169379679766 7.562502495320234L1.841799370320234 7.562502495320234z M1.841794379679766 7.562497504679766L1.841794379679766 7.567987843663488z M1.841794379679766 7.546872504679766L1.841794379679766 7.562502495320234L1.857424370320234 7.562502495320234z", "M1.82616938 7.5624975L1.84179438 7.5624975L1.84179438 7.5468725L1.85742437 7.5625025L1.84179935 7.5625025L1.84179438 7.5635166L1.84175554 7.5714477L1.8281275000000001 7.56362194L1.82617807 7.5625025z"},
		{NonZero, "M0.5791607706898958 -0.0017990842812309893L0.5791256283147277 0.0006388019799032918z M0.24732247524502785 -0.0006388019799032918L0.5791440451040444 -0.0006388019799032918L0.5790883795787067 0.0032228100758402434z", "M0.24732248 -0.0006388L0.57914405 -0.0006388L0.57914404 -0.0006387200000000001L0.5791256300000001 0.0006388L0.57908838 0.00322281z"},
		{NonZero, "M-1.0012776039598066 0.9987223960401934L0.5679470168256557 0.9987223960401934L0.5679320848125826 0.9991455162279408L0.5689697381460519 1.0137330234576891z M0.56848134703489 0.9835813767807052L0.5679320848125826 0.9991455162279408L0.5680837466356934 1.0012776039598066L-1.0012776039598066 1.0012776039598066z", "M-1.0012776 0.9987224L-0.774616 0.9987224L0.56848135 0.9835813800000001L0.56794702 0.9987224L0.5679320800000001 0.99914552L0.56808375 1.0012776L0.56896974 1.01373302L-0.7339801 1.0012776L-1.0012776 1.0012776L-0.8786234 0.9998949z"},
		{NonZero, "M0.9974447920803868 1.718178666643098L0.9974447920803868 1.0024362016795294L1.005034615441331899 0.9994720107590638z M0.9971400761003224 1.0025552079196132L1.0025552079196132 1.0004403385994465L1.0025552079196132 1.0025552079196132z M0.996941602106971899 1.00263272155621051L1.0025552079196132 1.0004403385994465L1.0025552079196132 1.716156561559842z M0.9974447920803868 1.0025552079196132L0.9974447920803868 1.0024362016795294L1.005034615441331899 0.9994720107590638z", "M0.9969416 1.00263272L0.99714008 1.00255521L0.99744479 1.00243621L0.99744479 1.0024362L1.0025552 1.00044034L1.00255521 1.00044034L1.00503462 0.99947201L1.00255521 1.23425561L1.00255521 1.71615656L1.00038252 1.43999488L0.99744479 1.7181786700000001L0.99744479 1.06659151z"},
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
		// overlapping polygons
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M2.5 5L7.5 5L5 10z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "M2.5 5L7.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M2.5 5L7.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M2.5 5L7.5 5L5 10z"},

		// non-overlapping polygons
		{"L10 0L5 10z", "M0 10L10 10L5 20z", ""},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", ""},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", ""},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "M2 2L8 2L5 8z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M2 2L8 2L5 8z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "L1 0L1 1L0 1z"},
		{"L2 0L2 1L0 1z", "L0 1L1 1L1 0z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "L1 0L1 1L0 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L1 1L2 1L2 0z", "M1 0L2 0L2 1L1 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 2L0 2z", "L1 0L1 1L0 1z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 0L5 10z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 -10L20 0L10 10z"},
		//{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "Q10 0 10 -10Q10 0 20 0Q10 0 10 10Q10 0 0 0z"}, // TODO

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 4L6 6L6 7L4 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L6 3L6 4L4 6z"},
		{"M1 3L6 3L6 4L4 6L5 7L1 7z", "M9 3L4 3L4 6L5 7L9 7z", "M4 3L6 3L6 4L4 6z"},
		{"L3 0L3 1L0 1z", "M1 -0.1L2 -0.1L2 1.1L1 1.1z", "M1 0L2 0L2 1L1 1z"},
		{"L10 0L10 10L0 10z", "M2 0L8 0L8 6L 2 6z", "M2 0L8 0L8 6L 2 6z"},

		// figure 10 from Martinez et al.
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "M1 2L2 2L2 3L1 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", ""},

		// fully parallel
		{"L10 0L10 5L7.5 7.5L5 5L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z", "M7.5 7.5L5 10L2.5 7.5L5 5z", ""},

		// subpaths on A cross at the same point on B
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1L1.5 0.5L1 0z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 -0.1L0.4 0.5L1 1.1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 -0.1L0.4 0.5L1 1.1L1 2L0 2z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0L0.5 0.5L1 1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 0L0.5 0.5L1 1L1 2L0 2z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z"},
		{"L1 0L1.1 0.5L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1L1.5 0.5L1 0zM1 0L1.1 0.5L1 1z"},
		{"L1 0L0.9 0.5L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1L1.5 0.5L1 0z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 1L3 1L3 2L2 2L2 3L1 3zM2 3L3 3L3 4L2 4z"},                                      // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M1 0L2 0L2 1L1 1zM1 2L2 2L2 3L1 3z"},                                 // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M0 2L2 2L2 3L0 3zM1 0L2 0L2 1L1 1z"},                                 // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9zM1 0L2 0L2 1L1 1z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M1 0L2 0L2 1L1 1z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z"},                                                  // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z"},                                                  // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "M5 1L5 9"},                                    // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", ""},                                          // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "M5 5L5 10"},                                  // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 10"},                                        // touch
		{"M5 0L10 0L10 5", "L10 0L10 10L0 10z", ""},                                      // boundary
		{"L5 0L5 5", "L10 0L10 10L0 10z", "M5 0L5 5"},                                    // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "M1 1L2 0M8 0L9 1"},                    // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", ""},                                  // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                                               // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "M5 0L5 1L6 0"},                           // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M6 0L7 1"},                               // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M6 0L7 1"},                               // touch with parallel
		{"M5 5L25 5", "L10 0L10 10L0 10zM20 0L30 0L30 10L20 10z", "M5 5L10 5M20 5L25 5"}, // cross twice

		// P intersects Q twice in the same point
		{"L0 -20L20 -20L20 20L0 20z", "L10 10L10 -10L-10 10L-10 -10z", "L10 -10L10 10z"},
		{"L20 0L20 20L-20 20L-20 0z", "L10 10L10 -10L-10 10L-10 -10z", "M-10 0L0 0L-10 10zM0 0L10 0L10 10z"},
		{"M-5 -5L0 -5L0 5L-5 5z", "L10 0L10 10L-10 10L-10 0L10 0L10 -10L-10 -10L-10 0z", "M-5 -5L0 -5L0 5L-5 5z"},
		{"M-5 -5L0 -5L0 5L-5 5z", "L-10 0L-10 10L10 10L10 0L-10 0L-10 -10L10 -10L10 0z", "M-5 -5L0 -5L0 5L-5 5z"},
		{"L20 0L20 20L0 20z", "L10 0L10 10L-10 10L-10 0L10 0L10 -10L-10 -10L-10 0z", "L10 0L10 10L0 10z"},

		// similar to holes and islands 4
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M1 4L5 4L5 6L1 6z"},

		// holes and islands 4
		//{"M0 4L6 4L6 6L0 6zM5 5A1 1 0 0 0 7 5A1 1 0 0 0 5 5zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z", "M1 3L5 3L5 7L1 7zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z", "M1 4L5 4L5 6L1 6zL0 6zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z"},

		// many overlapping segments
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "M1 0L2 0L2 3L1 3z"},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "M0 1L3 1L3 2L0 2z"},

		// bugs
		{"M23 15L24 15L24 16L23 16zM23.4 14L24.4 14L24.4 15L23.4 15z", "M15 16A1 1 0 0 1 16 15L24 15A1 1 0 0 1 25 16L25 24A1 1 0 0 1 24 25L16 25A1 1 0 0 1 15 24z", "M23 15L24 15L24 16L23 16z"},
		{"M23 15L24 15L24 16L23 16zM24 15.4L25 15.4L25 16.4L24 16.4z", "M14 14L24 14L24 24L14 24z", "M23 15L24 15L24 16L23 16z"},
		{"M0 1L2 1L2 2L0 2zM3 1L5 1L5 2L3 2z", "M1 0L4 0L4 3L1 3z", "M1 1L2 1L2 2L1 2zM3 1L4 1L4 2L3 2z"},
		{"L5 0L5 5L0 5zM1 1L1 4L4 4L4 1z", "M3 2L6 2L6 3L3 3z", "M4 2L5 2L5 3L4 3z"},
		{"M0.6203785454666688 7.0296030922171955L0.6187340865555582 7.030136875251485z", "M0.62500998 7.018654L0.61075695 7.0453795800000005z", ""},
		{"M0.6203785454666688 7.0296030922171955L0.6187340865555582 7.030136875251485z", "M0.62130568 6.99999002L0.62500998 7.018654L0.61075695 7.0453795800000005z", ""},
		{"M0.8785593994222234 7.0413469294202145L0.8763131971555573 7.043500136158196z", "M0.88303435 7.0075883900000005L0.87500998 7.06237368L1.06249861 7.18747447z", ""},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.And(q)
			test.T(t, r, MustParseSVGPath(tt.r))

			if p.Closed() {
				r = q.And(p)
				test.T(t, r, MustParseSVGPath(tt.r), "swapped arguments")
			}
		})
	}
}

func TestPathOr(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		//{"M0 1L4 1L4 3L0 3z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 1 2 3L0 3L0 1L4 1z"}, // TODO

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L4 0L4 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 1L4 1L4 3L2 3L2 2L0 2z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "L10 0L5 10z"},
		//{"M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z", "M10 0L5 5L0 0L5 -5z", "M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z"}, // TODO

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		//{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "A10 10 0 0 1 20 0A10 10 0 0 1 0 0z"}, // TODO
		{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10z"},

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L9 3L9 7L1 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L9 3L9 7L1 7z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "L2 0L2 1L0 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "L2 0L2 1L0 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "L3 0L3 1L0 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "L2 0L2 2L0 2z"},
		{"M2 0L2 2L0 2L0 0z", "M2 0L4 0L4 2L0 2L0 0z", "L4 0L4 2L0 2z"},
		{"M2 2L0 2L0 0L2 0z", "M2 0L4 0L4 2L0 2L0 0z", "L4 0L4 2L0 2z"},
		{"M2 0L2 1L0 1L0 0z", "M1 0L3 0L3 1L1 1z", "L3 0L3 1L0 1z"},
		{"M2 1L0 1L0 0L2 0z", "M1 0L3 0L3 1L1 1z", "L0 0L3 0L3 1L0 1z"},

		// figure 10 from Martinez et al.
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "L3 0L3 3L0 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", "L3 0L3 3L2 3L2 4L1 4L1 3L0 3z"},

		// fully parallel
		{"L10 0L10 5L7.5 7.5L5 5L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z", "M7.5 7.5L5 10L2.5 7.5L5 5z", "M0 0L10 0L10 5L7.5 7.5L10 10L10 15L0 15z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M0 1L1 1L1 0L3 0L3 1L4 1L4 2L3 2L3 3L4 3L4 2L5 2L5 5L2 5L2 4L1 4L1 3L0 3z"}, // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "L3 0L3 1L0 1zM0 2L3 2L3 3L0 3z"},                               // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "L3 0L3 1L0 1zM0 2L2 2L2 3L0 3z"},                               // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "L3 0L3 1L0 1zM0 2L2 2L2 3L0 3z"},               // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "L3 0L3 1L0 1zM0 2L4 2L4 3L0 3z"},                               // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z"},                                                                 // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "L7 0L7 4L0 4z"},                                                                 // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 1L5 9"},                     // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "M15 1L15 9M0 0L10 0L10 10L0 10z"},             // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 5L5 10L5 15"},              // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L10 10"},                     // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 5"},                     // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 1L2 0M8 0L9 1"},     // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0M8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "M0 0L10 0L10 10L0 10z"},                            // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 1L7 -1"},           // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 -1L7 1"},           // touch with parallel

		// similar to holes and islands 4
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M0 4L1 4L1 3L5 3L5 4L6 4L5 5L6 6L5 6L5 7L1 7L1 6L0 6zM6 4L7 5L6 6z"},

		// holes and islands 4
		//{"M0 4L6 4L6 6L0 6zM5 5A1 1 0 0 0 7 5A1 1 0 0 0 5 5zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z", "M1 3L5 3L5 7L1 7zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z", "M0 4L1 4L1 3L5 3L5 4L6 4A1 1 0 0 1 6 6L5 6L5 7L1 7L1 6L0 6zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z"},

		// many overlapping segments
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "L2 0L2 3L0 3z"},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "L3 0L3 2L0 2z"},

		// bugs
		// numerical precision error causes different path when swapping arguments
		{"M0.007659857738872233 -0.0005034030289721159L0.007949435635083546 0.000319400989951659L0 0.0003194009899516459L0 -0.05z", "M0 0.014404212206486022L0 -0.0003194009899516459L0.007724615472341156 -0.0003194009899516459L0.007949777738929242 0.0003203730407221883L0.00825498662783275 0.00120542818669378L0.008575413294551026 0.0018310816186470904L0.008865333294593825 0.002136281721490718L0.009353582183550202 0.0021210216648341884L0.010040231072537154 0.002410963667955457L0.010314933294807815 0.0028992914651979618L0.010543768850411084 0.003189014303174531L0.01087948440599007 0.003463702981264305L0.011093102183806993 0.0037383934158157217L0.011230417739383824 0.004043607069903032L0.011474542183862011 0.004348822892566773L0.011657653294975034 0.0046998237695987655L0.011627146628327978 0.00544739338054967L0.011642435517217107 0.006027324249046728L0.011749244406132677 0.006484945503459016L0.011962862183906964 0.007019104973011281L0.011962862183906964 0.007309080038069737L0.01185605329501982 0.007644618643766421L0.011581422183894574 0.00794986005330145L0.011474542183862011 0.008361939396820617L0.011535626628301545 0.008956947645827995L0.011764462183904811 0.009506176079227656L0.012329084406218271 0.009765642383030126L0.012847839961821704 0.010177739145603937L0.01295464885070885 0.010467511077436598L0.012878417739585757 0.010726983188021675z", "M0 -0.05L0.00765986 -0.0005034L0.00772462 -0.0003194L0.00794944 0.0003194L0.00794978 0.00032037L0.00825499 0.00120543L0.00857541 0.0018310800000000001L0.00886533 0.00213628L0.00935358 0.00212102L0.01004023 0.00241096L0.01031493 0.00289929L0.010543770000000001 0.00318901L0.01087948 0.0034637L0.0110931 0.0037383900000000003L0.01123042 0.00404361L0.01147454 0.00434882L0.01165765 0.00469982L0.011627150000000001 0.00544739L0.01164244 0.00602732L0.011749240000000001 0.00648495L0.01196286 0.0070191L0.01196286 0.00730908L0.01185605 0.00764462L0.01158142 0.00794986L0.01147454 0.00836194L0.01153563 0.00895695L0.011764460000000001 0.00950618L0.012329080000000001 0.00976564L0.012847840000000001 0.01017774L0.01295465 0.010467510000000001L0.01287842 0.01072698L0 0.01440421z"},

		// numerical precision error: two segments may be almost aligned such that the smaller segment finds that the next endpoint DOES intersect with the longer segment. This is because of scale issues between directions and positions for checking equality. This is handled in intersectionSubpath by checking for missing incoming/outgoing intersections
		{"M0.017448373295778197 0.0004042999012767723L0.01779169774029299 0.00014489013486240765L0.01822661329592279 -0.0001068884344590515L0.018493635518197493 -0.00031288798450646027L0.0184977364217076 -0.0003194009899516459L0.019171889409349777 -0.0003194009899516459z", "M0.01756073557099569 0.0003194009899516459L0.01779169774029299 0.00014489013486240765L0.01822661329592279 -0.0001068884344590515L0.018493635518197493 -0.00031288798450646027L0.018623342184881153 -0.0005188865466720927L0.018676746629324725 -0.0006485888382883331L0.018646239962649247 -0.0007706612256157541L0.01850124440704803 -0.0008316972891861951L0.018508853295941208 -0.0009535447596817904L0.01895137774044997 -0.0011137635594877793L0.019302311073843725 -0.001052727896691863L0.019393902184958733 -0.0008393267910520308L0.019271804407168247 -0.0004654796068876976z", "M0.01744837 0.0004043L0.017560740000000002 0.0003194L0.0177917 0.00014489L0.01822661 -0.00010689L0.01849364 -0.00031289L0.018497740000000002 -0.0003194L0.018623340000000002 -0.00051889L0.01867675 -0.00064859L0.01864624 -0.00077066L0.01850124 -0.0008317L0.01850885 -0.00095354L0.01895138 -0.00111376L0.01930231 -0.00105273L0.019393900000000002 -0.00083933L0.019271800000000002 -0.00046548L0.01895335 -0.0003194L0.01917189 -0.0003194z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.Or(q)
			test.T(t, r, MustParseSVGPath(tt.r))

			if p.Closed() {
				r = q.Or(p)
				test.T(t, r, MustParseSVGPath(tt.r), "swapped arguments")
			}
		})
	}
}

func TestPathXor(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		//{"M0 1L4 1L4 3L0 3z", "M4 3A1 1 0 0 0 2 3A1 1 0 0 0 4 3z", "M4 3A1 1 0 0 0 2 3L0 3L0 1L4 1zM4 3A1 1 0 0 1 2 3z"}, // TODO

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L4 0L4 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 1L4 1L4 3L2 3L2 2L0 2z"},

		// no overlap
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},

		// containment
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M0 0L10 0L5 10zM2 2L5 8L8 2z"},

		// equal
		{"L10 0L5 10z", "L10 0L5 10z", ""},
		//{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 10L20 0L10 -10zA10 10 0 0 1 20 0A10 10 0 0 1 0 0z"}, // TODO
		//{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10zQ10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z"}, // TODO

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L9 3L9 7L6 7L6 6L4 4L4 7L1 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L4 3L4 6L6 4L6 3L9 3L9 7L1 7z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "L1 0L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M0 1L1 1L1 0L2 0L2 2L0 2z"},
		{"L2 0L0 2z", "L2 2L0 2z", "L2 0L1 1zM0 2L1 1L2 2z"},

		// figure 10 from Martinez et al.
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "L3 0L3 3L2 3L2 2L1 2L1 3L0 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", "L3 0L3 3L2 3L2 4L1 4L1 3L0 3z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M0 1L1 1L1 3.0000000000000004L0 3zM1 0L3 0L3 1L1 1zM1 3.0000000000000004L2 3L2 4L1 4zM2 2L3 2L3 3L2 3zM2 4L3 4L3 3L4 3L4 2L5 2L5 5L2 5zM3 1L4 1L4 2L3 2z"}, // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "L1 0L1 1L0 1zM0 2L1 2L1 3L0 3zM2 0L3 0L3 1L2 1zM2 2L3 2L3 3L2 3z"},                                                                            // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},                                                                                                              // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "L1 0L1 1L0 1zM0 2L2 2L2 3L0 3zM0.1 2.1L0.1 2.9L1.9 2.9L1.9 2.1zM2 0L3 0L3 1L2 1z"},                                            // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "L1 0L1 1L0 1zM0 2L4 2L4 3L0 3zM2 0L3 0L3 1L2 1z"},                                                                                             // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                                                                              // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                                                                              // two inside the same

		// open
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                             // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "M15 1L15 9M0 0L10 0L10 10L0 10z"},             // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 10L5 15"},                  // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                               // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                             // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                     // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0M8 0L9 -1"}, // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM6 0L7 -1"},               // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 -1L6 0"},           // touch with parallel

		// multiple intersections in one point
		{"L2 0L2 2L4 2L4 4L2 2L0 2z", "L2 0L2 2L4 4L2 4L2 2L0 2z", "M2 2L4 2L4 4L2 4z"},
		{"L2 0L2 1.9L4 1.9L4 4L2 2L0 2z", "L2 0L2 2L4 4L1.9 4L1.9 2L0 2z", "M1.9 2L2 2L2 1.9L4 1.9L4 4L1.9 4z"}, // simple version of above
		{"L2 0L2 1L3 1L2 0L4 0L4 2L0 2z", "L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2L2 1L3 1z"},

		// similar to holes and islands 4
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M0 4L1 4L1 6L0 6zM1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7zM5 4L6 4L5 5zM5 5L6 6L5 6zM6 4L7 5L6 6z"},

		// holes and islands 4
		//{"M0 4L6 4L6 6L0 6zM5 5A1 1 0 0 0 7 5A1 1 0 0 0 5 5zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z", "M1 3L5 3L5 7L1 7zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z", "M0 4L1 4L1 6L0 6zM1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7zM5 4L6 4A1 1 0 0 1 5 5zM5 5A1 1 0 0 1 6 6L5 6zM6 4A1 1 0 0 1 6 6zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z"},

		// many overlapping segments
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "L1 0L1 3L0 3z"},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "L3 0L3 1L0 1z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.Xor(q)
			test.T(t, r, MustParseSVGPath(tt.r))

			if p.Closed() {
				r = q.Xor(p)
				test.T(t, r, MustParseSVGPath(tt.r), "swapped arguments")
			}
		})
	}
}

func TestPathNot(t *testing.T) {
	var tts = []struct {
		p, q string
		r    string
	}{
		// overlap
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5z"},

		{"M0 5L10 5L5 15z", "L10 0L5 10z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"M0 5L10 5L5 15z", "L5 10L10 0z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"M0 5L5 15L10 5z", "L10 0L5 10z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"M0 5L5 15L10 5z", "L5 10L10 0z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},

		// touching edges
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2z"},
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
		//{"A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 -10L20 0L10 10z", "A10 10 0 0 1 20 0A10 10 0 0 1 0 0zL10 10L20 0L10 -10z"}, // TODO
		//{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10zQ10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z"}, // TODO
		{"Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10z", ""},

		// partly parallel
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L4 3L4 7L1 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L4 3L4 7L1 7z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", ""},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "L1 0L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M0 1L1 1L1 0L2 0L2 2L0 2z"},

		// figure 10 from Martinez et al.
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "L3 0L3 3L2 3L2 2L1 2L1 3L0 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", "L3 0L3 3L0 3z"},

		// subpaths on A cross at the same point on B
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M1 -0.1L1.6 0.5L1 1.1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M1 0L1.5 0.5L1 1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M1 0.1L1.4 0.5L1 0.9z"},

		// subpaths
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 0L3 0L3 1L1 1zM1 3L2 3L2 4L1 4zM2 2L3 2L3 3L2 3z"},                                          // different winding
		{"M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 0L3 0L3 4L1 4z", "M0 1L1 1L1 3L0 3zM2 4L3 4L3 3L4 3L4 2L5 2L5 5L2 5zM3 1L4 1L4 2L3 2z"},                          // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "L1 0L1 1L0 1zM0 2L1 2L1 3L0 3z"},                                                  // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "L1 0L1 1L0 1z"},                                                                   // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "L1 0L1 1L0 1zM0 2L2 2L2 3L0 3zM0.1 2.1L0.1 2.9L1.9 2.9L1.9 2.1z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "L1 0L1 1L0 1zM0 2L2 2L2 3L0 3z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                  // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", ""},                                                                                                 // two inside the same

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
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M5 0L5 -1L6 0"},           // touch with parallel

		// similar to holes and islands 4
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M0 4L1 4L1 6L0 6zM5 4L6 4L5 5zM5 5L6 6L5 6zM6 4L7 5L6 6z"},
		{"M1 3L5 3L5 7L1 7z", "M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7z"},

		// holes and islands 4
		//{"M0 4L6 4L6 6L0 6zM5 5A1 1 0 0 0 7 5A1 1 0 0 0 5 5zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z", "M1 3L5 3L5 7L1 7zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z", "M0 4L1 4L1 6L0 6zM5 4L6 4A1 1 0 0 1 5 5zM5 5A1 1 0 0 1 6 6L5 6zM6 4A1 1 0 0 1 6 6zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z"},
		{"M1 3L5 3L5 7L1 7zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z", "M0 4L6 4L6 6L0 6zM5 5A1 1 0 0 0 7 5A1 1 0 0 0 5 5zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z", "M1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7z"},

		// many overlapping segments
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "L1 0L1 3L0 3z"},
		{"M1 0L2 0L2 3L1 3z", "L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", ""},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "L3 0L3 1L0 1z"},
		{"M0 1L3 1L3 2L0 2z", "L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", ""},
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
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5zM2.5 5L7.5 5L5 10z"},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L5 1L5 3L1 3z", "M0 0L2 0L2 1L1 1L1 2L0 2zM1 1L2 1L2 2L1 2zM4 0L6 0L6 2L5 2L5 1L4 1zM4 1L5 1L5 2L4 2z"},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L1 3L5 3L5 1z", "M0 0L2 0L2 1L1 1L1 2L0 2zM1 1L2 1L2 2L1 2zM4 0L6 0L6 2L5 2L5 1L4 1zM4 1L5 1L5 2L4 2z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, "x", tt.q), func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			q := MustParseSVGPath(tt.q)
			r := p.DivideBy(q)
			test.T(t, r, MustParseSVGPath(tt.r))
		})
	}
}
