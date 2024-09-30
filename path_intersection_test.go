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
		{"M-0.1997406229376793 296.9999999925494L-0.1997406229376793 158.88740153238177", "M-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834", Intersections{
			{Point{-0.1997406229376793, 158.88740153238177}, [2]float64{1.0, 0.0}, [2]float64{1.5 * math.Pi, 89.9999462 * math.Pi / 180.0}, true},
		}}, // #287
		{"M-0.1997406229376793 296.9999999925494L-0.1997406229376793 158.88740153238177", "M-0.19974062293732664 158.8874019079834L-0.19999999999964735 20.77454766193328", Intersections{
			{Point{-0.1997406229376793, 158.88740172019808}, [2]float64{0.9999999986, 1.359651238e-09}, [2]float64{270.0 * math.Pi / 180.0, 269.9998924 * math.Pi / 180.0}, false},
		}}, // #287
		{"M-0.1997406229376793 158.88740153238177L-0.19974062293732664 158.8874019079834", "M-0.19974062293732664 158.8874019079834L-0.19999999999964735 20.77454766193328", Intersections{
			{Point{-0.19974062293732664, 158.8874019079834}, [2]float64{1.0, 0.0}, [2]float64{89.9999462 * math.Pi / 180.0, 269.9998924 * math.Pi / 180.0}, true},
		}}, // #287
		{"M162.43449681368278 -9.999996185876771L162.43449681368278 -9.99998551284069", "M162.43449681368278 -9.999985512840682L162.2344968136828 -9.99998551284069", Intersections{
			{Point{162.43449681368278, -9.99998551284069}, [2]float64{1.0, 0.0}, [2]float64{0.5 * math.Pi, math.Pi}, true},
		}}, // #287
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
			{Point{4.0, 3.0}, [2]float64{0.0, 1.0}, [2]float64{math.Pi, 1.5 * math.Pi}, true},
			{Point{2.0, 3.0}, [2]float64{0.5, 0.0}, [2]float64{math.Pi, 0.5 * math.Pi}, true},
		}},

		// none
		{"M6 0L6 10", "A5 5 0 0 1 0 10", Intersections{}},
		{"M10 5L15 5", "A5 5 0 0 1 0 10", Intersections{}},
		{"M6 0L6 20", "A10 5 90 0 1 0 20", Intersections{}},

		// bugs
		{"M0 -0.7L1 -0.7", "M-0.7 0A0.7 0.7 0 0 1 0.7 0", Intersections{
			{Point{0.0, -0.7}, [2]float64{0.0, 0.5}, [2]float64{0.0, 0.0}, true},
		}}, // #200, at intersection the arc angle is deviated towards positive angle
		{"M30.23402723090112,37.620459766287226L30.170131507649785,37.66143576791836", "M30.242341004748596 37.609669236818846A0.8700000000000001 0.8700000000000001 0 0 1 28.82999999999447 36.9294", Intersections{
			{Point{30.170131507649785, 37.66143576791836}, [2]float64{1.0, 0.04553266140095003}, [2]float64{2.571361371137828, 2.570702752627385}, true}, // TODO
		}}, // #280
		{"M30.23402723090112,37.620459766287226L30.170131507649785,37.66143576791836", "M30.170131507649785 37.66143576791836A0.8700000000002787 0.8700000000002787 0 0 1 28.82999999999447 36.92939999999941", Intersections{
			{Point{30.170131507649785, 37.66143576791836}, [2]float64{1.0, 0.0}, [2]float64{2.571361371137828, 2.570702753023132}, true},
		}}, // #280
		{"M18.28586369751671 1.9033410129748447L18.285524146153797 1.9012793871179001", "M18.285524146153797 1.9012793871179001A0.09877777777777778 0.09877777777777778 0 0 0 18.188037109374996 1.8184178602430556", Intersections{
			{Point{18.285524146153797, 1.9012793871179001}, [2]float64{1.0, 0}, [2]float64{4.549153676432458, 4.550551548951796}, true},
		}}, // in preview
		{"M32761.5,32383.691L32761.52,32383.691", "M31511.49999999751,33633.691 A1250 1250 0 0 1 32761.50000000074,32383.691", Intersections{
			{Point{32761.5, 32383.691}, [2]float64{0.0, 1.0}, [2]float64{0.0, 0.0}, true},
		}}, // #293
		{"M73643.30051730774,34889.01290159931L73639.44503270132,34889.13797316706", "M73639.44503270132,34889.13797316706A1250 1250 0 0 1 73599.5139303418 34889.76998615123", Intersections{
			{Point{73639.44503270132, 34889.13797316706}, [2]float64{1.0, 0.0}, [2]float64{3.109164117290831, 3.1097912676269237}, true},
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

		{"M30.170131507649785 37.66143576791836A0.8700000000002787 0.8700000000002787 0 0 1 28.82999999999447 36.92939999999941", "M30.242341004748596 37.609669236818846A0.8700000000000001 0.8700000000000001 0 0 1 28.82999999999447 36.9294", Intersections{
			{Point{30.170131507649785, 37.66143576791836}, [2]float64{0.0, 0.0455326614}, [2]float64{2.5707027528269983, 2.5707027528269983}, true},
			{Point{28.82999999999447, 36.92939999999941}, [2]float64{1.0, 1.0}, [2]float64{1.5 * math.Pi, 1.5 * math.Pi}, true},
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
		{"L10 10L10 0L0 10z", []PathIntersection{
			{Point{5.0, 5.0}, 1, 0.5, 0.25 * math.Pi, false, false, false},
			{Point{5.0, 5.0}, 3, 0.5, 0.75 * math.Pi, true, false, false},
		}},

		// parallel segment
		{"L10 0L5 0L15 0", []PathIntersection{
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, true, false},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, false},
		}},
		{"L-5 0A5 5 0 0 1 5 0A5 5 0 0 1 -5 0z", []PathIntersection{
			{Point{5.0, 0.0}, 1, 0.5, 0.0, false, true, false},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, false},
		}},
		{"L-5 0A5 5 0 0 1 5 0A5 5 0 0 1 -5 0L0 0L0 1L1 0L0 -1z", []PathIntersection{
			{Point{0.0, 0.0}, 1, 0.0, math.Pi, false, true, false},
			{Point{10.0, 0.0}, 3, 0.5, 0.0, false, true, false},
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
		{NonZero, "M1 0L1 2.1L1.1 2.1L1.1 2L0 2L0 3L2 3L2 1L5 1L5 2L5.1 2.1L5.1 1.9L5 2L5 3L6 3L6 0z", "M5 1L2 1L2 3L0 3L0 2L1 2L1 0L6 0L6 3L5 3z"}, // CCW open first leg for CW path
		{Positive, "M4.428086304186892 0.375A0.375 0.375 0 0 1 4.428086304186892 -0.375L6.428086304186892 -0.375A0.375 0.375 0 0 1 6.428086304186892 0.375z", "M4.428086304186892 0.375A0.375 0.375 0 0 1 4.053086304186892 4.592425496802568e-17A0.375 0.375 0 0 1 4.428086304186892 -0.375L6.428086304186892 -0.375A0.375 0.375 0 0 1 6.803086304186892 -9.184850993605136e-17A0.375 0.375 0 0 1 6.428086304186892 0.375z"}, // two arcs as leftmost point

		// example from Subramaniam's thesis
		{NonZero, "L5.5 10L4.5 10L9 1.5L1 8L9 8L1.6 2L3 1L5 7L8 0z", "M3.349892008639309 6.090712742980562L0 0L8 0L6.479452054794521 3.547945205479452L9 1.5L6.592324805339266 6.047830923248053L9 8L5.5588235294117645 8L4.990463215258855 9.073569482288828L4.3999999999999995 8L1 8L3.349892008639309 6.090712742980561zM3.9753086419753085 3.9259259259259265L3 1L1.6 2L3.975308641975309 3.925925925925927zM4.990463215258855 9.073569482288828L5.5 10L4.5 10L4.990463215258855 9.073569482288828z"},
		{EvenOdd, "L5.5 10L4.5 10L9 1.5L1 8L9 8L1.6 2L3 1L5 7L8 0z", "M3.349892008639309 6.090712742980562L0 0L8 0L6.479452054794521 3.547945205479452L9 1.5L6.592324805339266 6.047830923248053L9 8L5.5588235294117645 8L4.990463215258855 9.073569482288828L4.3999999999999995 8L1 8L3.349892008639309 6.090712742980561zM3.349892008639309 6.090712742980562L4.4 8L5.5588235294117645 8L6.592324805339265 6.047830923248053L5.713467048710601 5.335243553008596L6.47945205479452 3.5479452054794525L4.995837669094692 4.753381893860562L3.975308641975309 3.925925925925927L4.409836065573771 5.229508196721312L3.349892008639309 6.090712742980561zM5.713467048710601 5.335243553008596L5 7L4.409836065573771 5.22950819672131L4.9958376690946915 4.753381893860562L5.713467048710601 5.335243553008596zM3.9753086419753085 3.9259259259259265L3 1L1.6 2L3.975308641975309 3.925925925925927zM4.990463215258855 9.073569482288828L5.5 10L4.5 10L4.990463215258855 9.073569482288828z"},

		// bugs
		{Negative, "M2 1L0 0L0 2L2 1L1 0L1 2z", "M1 1.5L0 2L0 0L1 0.5L1 0L2 1L1 2z"},
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

		// bugs
		//{"M0.007095306627675768 -0.001022210032758153L0.007659857738872233 -0.0005034030289721159L0.007949435635083546 0.0003194009899516459L0 0.0003194009899516459L0 -0.001022210032758153z", "M0 0.00120542818669378L0 -0.0003194009899516459L0.007724615472341156 -0.0003194009899516459L0.007949777738929242 0.0003203730407221883L0.00825498662783275 0.00120542818669378z", ""},
		//{"M1.05 0.0003194009899516459L0.5052266323285437 0.0003194009899516459L0.5051612978081295 0.00011549345222761076L0.5050397689192323 -0.00047961674448515623L0.5049743466970114 -0.0008355120402967486L0.5048980444747713 -0.0011673940887106937L0.5048293511414244 -0.001560308601426641L0.5047836266969767 -0.0019379609109790863L0.5047339911414213 -0.0022736490656711794L0.5046805866969493 -0.002601705409688293L0.5046310222524966 -0.0028458387375138727L0.5045547200302707 -0.00306326881441521L0.5044212444746989 -0.0031967779654706874L0.5042075555857792 -0.0032730687226489863L0.5040321244746337 -0.0032883268578274283L0.5038108266968493 -0.0032730687226489863L0.5035934400301159 -0.003253996046069574L0.503345475585661 -0.003173890711920535L0.5031204089189742 -0.0031128579761201536L0.5029601955855867 -0.003048010599314921L0.5027809244744645 -0.0029106864194687887L0.5026778844744513 -0.0027275868301330775L0.5026626666966649 -0.0025292280609363615L0.5026893333633495 -0.0022507613202691346L0.5027503466967005 -0.0019150729867618566L0.5028266489189122 -0.0016861930735387887L0.5028800533633841 -0.0014611266363431241L0.5029029511411522 -0.0012932797121720796L0.5028152000300281 -0.0011979118495020202L0.5026207822522224 -0.0011267785057924584L0.5023447289188567 -0.001019741675207797L0.5020141333632608 -0.0008851037736832268L0.5017051555854266 -0.0007935497596065488L0.5014113955853787 -0.0006791069676381767L0.5011482133631375 -0.0005455899916597673L0.5008697422519788 -0.00039681344387076933L0.5006408355852727 -0.00025185118376214177L0.5004539555852716 -0.00016411063077725885L0.5001907022518708 -0.00012977732215802007L0.4999465778073926 -0.00013359213556896066L0.49972151114070584 -0.00022896236116309865L0.49950405336291226 -0.0003853690727311232L0.4992523200295409 -0.0005684786456043867L0.498982382251711 -0.0007655000839434933L0.49875639114058856 -0.0009575838964366312L0.49851987558498934 -0.0012894650017329923L0.4983291555849547 -0.0016594903375306558L0.4981689422515956 -0.0020829174814167573L0.49803539558490684 -0.002517784257833L0.49791336891823335 -0.0028763553059150127L0.4978332266959882 -0.0032272942845992247L0.4977798222515446 -0.003589673917005598L0.49774170669596174 -0.003994009687929179L0.4976920711404347 -0.004421228254415155L0.49762721780707864 -0.00482937059510391L0.4975718933626183 -0.005138335582586251L0.49755475558485784 -0.005258600220173548L0.4975379022515227 -0.005318059291624877L0.497497511140395 -0.005329053601428768L0.49742312891817164 -0.005329053601428768L0.49726867558482013 -0.005315815554538972L0.49705690669588876 -0.005296743784882096L0.4968452089181028 -0.005285300719037878L0.4966373511402935 -0.0052680239275133545L0.49643134225134133 -0.0052471571441401466L0.4962387022513468 -0.005226065976415839L0.49605559114019115 -0.005184107984320008L0.49598092447349984 -0.005116795611940006L0.4959525511401921 -0.0049590598728030955L0.49593349336240067 -0.004642464770853394L0.4959105955846326 -0.00419999059863585L0.49586487114015654 -0.0038185436887374635L0.49582291558462543 -0.0034943111521812398L0.4957542222512643 -0.0031662616246848074L0.4956779200290242 -0.0028610970244216105L0.49562835558458573 -0.002597890815422943L0.4955672711401178 -0.0023346829937764824L0.4955635022512297 -0.0021553955831876692L0.4955864000289978 -0.00203714177517611L0.4956550222512419 -0.001964663473884798L0.4958114666957414 -0.00180826251234123L0.4958991466957201 -0.0017167104657431764L0.49603653336242814 -0.0015831967147050818L0.4961280533624546 -0.0014306090625524348L0.4961683022513341 -0.0012932797121720796L0.49617989336245216 -0.0010805529890092203L0.496169297806901 -0.0007488946672680186L0.4961166044735563 -0.0004540352540374215L0.49605175114018607 -9.162916924765341e-05L0.4960021866957476 0.0002097424296181316L0.496016705811158 0.0003194009899516459L0.4872449794770404 0.0003194009899516459L0.48709864891671373 0.00025933542694644984L0.48677907558337097 0.00012334752575782204L0.48646476447218845 -4.0465711023784934e-05L0.4861220800277124 -0.0001755550609061629L0.48586273780544786 -0.00027092519331972653L0.48560332447208054 -0.0003624803212005645L0.4852714489164782 -0.00044259089808917906L0.48495479113864803 -0.0005074422080326713L0.48469160891640684 -0.0004998126472344211L0.48462867558305334 -0.00046346001541053283L0.4845619022497232 -0.0003929986538508956L0.48451994669413523 -0.00018318467927258553L0.48448944002748817 0.00011055660668546352L0.48447114298788563 0.0003194009899516459L0.14307639098421987 0.0003194009899516459L0.1431007110910798 0.0002936690740256154L0.1432647644244298 0.00015072459333964616L0.1434822222022376 7.556186844226431e-06L0.14377591109115428 -0.00016590583570064155L0.14633178664706747 -0.0018767020275305413L0.14790341331392654 -0.0027695472216038297L0.1484069510918289 -0.0030823415757055272L0.14878078220296231 -0.003402762698883066L0.14905548442523298 -0.0037308104203361836L0.1492233066474995 -0.00402811552787341L0.1492996088697396 -0.004348753975435216L0.14928815998082712 -0.004554733185130999L0.14924236442527672 -0.004718752960584993L0.1491470044252594 -0.004882772109780831L0.14892577775856353 -0.005107820620978032L0.14887237331410574 -0.0052413234179482515L0.14879607109187987 -0.00572955868506142L0.14867397331404675 -0.006111212996117388L0.14851375998071603 -0.0062637840236732245L0.1482772444251168 -0.006340069334228815L0.14807891553620323 -0.006355326380059978L0.14788812442506583 -0.006309555226252428L0.1477508088695032 -0.006210384225653343L0.14768979553615225 -0.006111212996117388L0.14755247998058962 -0.005912869850249081L0.1473693688694624 -0.005718115734509865L0.1471938666471857 -0.005569357099716399L0.14686967109159355 -0.005405340571940087L0.14648815998043574 -0.005279466998018734L0.14573665775810696 -0.005199365440745396L0.14501950220243032 -0.0052108085295117235L0.14458081775795506 -0.005165036156185465L0.14322472886885862 -0.0050811200117806266L0.14293096886882495 -0.0050544193860275755L0.14271351109101715 -0.005006851843546656L0.14249228442430706 -0.0049323591711356585L0.14235496886874444 -0.0048771627101160675L0.1421985244242876 -0.00480648421131491L0.14059831109071297 -0.004310609568946688L0.13992311109063849 -0.004138959320627578L0.13949011553499702 -0.00409318580524598L0.13867567997932895 -0.004116072569075868L0.13816453331260448 -0.004188547240602247L0.1377620444236669 -0.0042857034970182895L0.137580853312528 -0.004353465929199274L0.1375417422014067 -0.00438308391231601L0.13753029331250843 -0.004403053601492957L0.13753029331250843 -0.004435588468567175L0.13755034664583832 -0.0044726108734920444L0.13765907553475643 -0.004545309321301261L0.1380252977570251 -0.004617783268670905L0.13827895109037058 -0.004688237700932518L0.13848879997931363 -0.004764525809434872L0.13951870220168416 -0.005561728437854185L0.14049527109072812 -0.006210384225653343L0.1407164977574098 -0.006301926695897464L0.14099496886855434 -0.006382026197300661L0.1412238755352746 -0.006439240035419402L0.14152140442419636 -0.006420168764506684L0.14166639997975494 -0.006362954900978934L0.1417998755353267 -0.006267598292438947L0.14188385775759116 -0.0061625935938565135L0.14195631997979774 -0.006082493683294388L0.14203262220203783 -0.005983322158527926L0.14211653331315688 -0.00588998404371921L0.14217754664650784 -0.005849821667496258L0.1422404799798329 -0.005838378748947548L0.1423130133131849 -0.005855655311094665L0.14236065775763507 -0.005907036214424011L0.14240453331321135 -0.006027298700004735L0.1424121422020903 -0.0061206365164849785L0.142389244424308 -0.006191312853147224L0.1423225422020522 -0.006296093112908352L0.1422309510909514 -0.006399078172691475L0.14217562664649108 -0.006477382551850042L0.14214135109094173 -0.006578347873045232L0.14215464886871132 -0.006715660329049911L0.14217562664649108 -0.006874062715098717L0.142204284424281 -0.007009579306995306L0.14226145775761267 -0.007146890384618132L0.14237018664653078 -0.007349042005699857L0.14248275553543976 -0.007509236954405196L0.14266394664655024 -0.00768468787939014L0.1428183999799444 -0.007824016044679638L0.1429919822021759 -0.00794404880379318L0.143291431091086 -0.008125331288340476L0.14351464886892984 -0.008279690028587083L0.1438770310911508 -0.008521996234705398L0.14432140442458774 -0.008766095919696681L0.14476584886907062 -0.008899587348238924L0.14505192886910834 -0.008944458323540516L0.1451682666468912 -0.008983496033891925L0.14526554664691105 -0.009018944039013377L0.14533516442470784 -0.009050353639324271L0.14540101331360233 -0.009065609722654244L0.145471555535849 -0.009071442929538875L0.14566419553587195 -0.009069423742644744L0.14621352886928207 -0.009040706407390076L0.14666935109157464 -0.009017822267139763L0.14699745775828887 -0.008994938114696538L0.14771269331392034 -0.00878898019405483L0.15041349331431775 -0.007545359262223883L0.1511688355366374 -0.007259296675357518L0.15152353775890504 -0.00719064137105363L0.15178295109227236 -0.007209712299939497L0.15199279998118698 -0.007293624286361933L0.15244684442568257 -0.007663149185518137L0.15280531553686671 -0.008064305586870546L0.1528853866479949 -0.008262638799649835L0.15297996442575368 -0.008843049852686136L0.1529311822035453 -0.009979170011618521L0.15279770664795933 -0.010654456263750944L0.1526679999813041 -0.011237975801364541L0.1525420622034801 -0.011596698411196371L0.15235518220346478 -0.011844594399391895L0.15217967998123072 -0.011917056033453832L0.1515541155366833 -0.011924683566803651L0.1509780444255 -0.011909428498810826L0.15069580442546737 -0.011901800962760944L0.15021509331430138 -0.011836966851859643L0.14992524442534716 -0.011871290805117951L0.1494330844252829 -0.0120009588254959L0.14885331553634273 -0.012164950173072953L0.14867020442518708 -0.012233821865635264L0.14850330664739886 -0.012279586707791168L0.14836883553624602 -0.01233858738820004L0.14829921775846344 -0.012372013593264342L0.14826721775845897 -0.012411048189008511L0.14825676442509916 -0.012440660617343724L0.14826344886957088 -0.012466907979899133L0.14827823998068368 -0.01249495001624723L0.14829153775846748 -0.012525459730923671L0.14829253331403436 -0.012547444659006146L0.14828968886956773 -0.012576159650151908L0.1482810844251219 -0.01260756664970586L0.14825911109177525 -0.01263246789734751L0.148244817758453 -0.012641889987264676L0.14814469331399494 -0.012682943355031284L0.1480865244250822 -0.01268585971297398L0.14803219553618874 -0.012666791215025341L0.1479644266473059 -0.012627532516063411L0.14789480886950912 -0.012536003524402872L0.14786145775839543 -0.012490238955436439L0.14781758220284757 -0.012463542934327165L0.14774988442505332 -0.012449185403525576L0.14748855109168346 -0.012454120805301727L0.14714238220274467 -0.012477900460325486L0.14706991998052388 -0.012497866386965484L0.14701459553607776 -0.012528376099609773L0.14697071998048727 -0.01257885167956374L0.14692968886937763 -0.01263246789734751L0.14689918220271636 -0.012669483238781254L0.14684385775824182 -0.012706722883393695L0.14677999998046687 -0.0127287077186935L0.14669800886936457 -0.0127325214134828L0.14649391998044337 -0.012739251462249968L0.146282151091512 -0.012721977668249451L0.14609143109147738 -0.012676213297410754L0.14580535109146808 -0.012580870701555114L0.1452216710913632 -0.012275772972785148L0.1448764266468885 -0.012075439241527874L0.14469907553572625 -0.011968654026546233L0.14455791998015854 -0.011878918346582168L0.14429473775788892 -0.01176450508270932L0.14407343998009026 -0.011756877520937792L0.1438751110911909 -0.011795015316195645L0.14355084442446753 -0.01187510457602059L0.1432647644244298 -0.011924683566803651L0.14313413331328206 -0.011957212737812029L0.14307212442439265 -0.011973365144598347L0.1430349333133023 -0.011978076262110449L0.1429815288688303 -0.011978076262110449L0.14291383109105027 -0.011960129145052179L0.14279841775770308 -0.011930516385461942L0.14269921775768069 -0.011917056033453832L0.14261431109100897 -0.01191324226631707L0.1424713066465273 -0.011935227507663626L0.14232631109096872 -0.011965737619902939L0.14223287109093974 -0.011968654026546233L0.14210124442425354 -0.01193993862936793L0.14202494220201345 -0.011907633784574045L0.1419115199798 -0.011900006248225736L0.14184666664644396 -0.011903820016570421L0.1416549510908709 -0.011934330151120776L0.14155383109083175 -0.011951379922720662L0.1414966577575001 -0.011966634975848933L0.14145278220195223 -0.011972467788837093L0.14139361775750103 -0.011981890023491815L0.141328764424145 -0.01198951754530242L0.14126775109082246 -0.012002977874601584L0.14118952886859404 -0.012012400102619836L0.1410884799796719 -0.012022046667240716L0.14100456886855284 -0.012025860424742518L0.14092826664634117 -0.012027655134048132L0.14086910220187576 -0.012037301695215774L0.14077950220186608 -0.012054351425987875L0.14067454220183606 -0.01206018422708155L0.1405868622018147 -0.01206018422708155L0.14045523553514272 -0.012067811734979728L0.14012143997956628 -0.012081047701386183L0.13953399109055908 -0.012111557710056786L0.13911052442384175 -0.012157322682440963L0.13893125331269118 -0.012191646379747567L0.13874430220155887 -0.012249076818449112L0.13858031997932585 -0.012329165231747652L0.138351413312634 -0.012462645588783516L0.1381377955348171 -0.012580870701555114L0.13751983997920547 -0.012802065199949197L0.13715361775689416 -0.012878338900861763L0.1367415999790751 -0.012916475700521346L0.13637537775680642 -0.012931730410898012L0.1356582222011582 -0.012931730410898012L0.13532257775665357 -0.013000376540588832L0.13502497775661482 -0.013122638166592537L0.13469694220101758 -0.013275184294613496L0.13427731553429112 -0.013465866192888143L0.1338042844231211 -0.013603156635554114L0.13339233775637638 -0.013656547244792705L0.13306423108969057 -0.013648920018951571L0.13265221331181465 -0.013580274925573121L0.13217925331177582 -0.013435357146022398L0.13162991997833728 -0.013237047813390745L0.1309813866449332 -0.012802065199949197L0.13040922664485777 -0.012206901347511234L0.12987226664478158 -0.011349025334681073L0.128570506644607 -0.009376563416495287L0.12795255108893855 -0.00869744302350739L0.12758632886666987 -0.0084609710967527L0.12719343997774502 -0.0084113881083141L0.1265029510887672 -0.008483855533626183L0.12593839997757073 -0.008644046250608994L0.125705724422005 -0.008716513283445693L0.12513349331078416 -0.008987310061172593L0.12462234664403127 -0.00918204935646827L0.12414931553288966 -0.009315539492476432L0.12367244442170033 -0.009368935430742908L0.12300869331049569 -0.009269771492597556L0.12214661331037746 -0.009094327041211159L0.12075041775462125 -0.008644046250608994L0.1199721777545335 -0.008277895162720483L0.11947247997666466 -0.0080223503668293L0.11860655997654135 -0.00771340612114102L0.11772151108755224 -0.0073851644482658685L0.11701196442075457 -0.0070037458201142044L0.11648559997625796 -0.006729122310915159L0.11602778664286006 -0.006416354509340749L0.1157302577539383 -0.006050184433405548L0.1155585955316667 -0.005710487099065631L0.11543272886498812 -0.00540152622662049L0.1152954133094255 -0.0050582337621705165L0.1152420088649535 -0.004707309740751953L0.11513896886495445 -0.004077927955975724L0.11502839108717922 -0.0038185436887374635L0.11482238219824126 -0.0036697784758104035L0.11450195553152298 -0.0035286417719078145L0.11405182219814947 -0.0034256498304614524L0.11248773330902395 -0.0039596794401290936L0.11022184886427056 -0.004981946175277585L0.10992431997532037 -0.005020089985649179L0.10965153775309489 -0.0049954087004806524L0.10929107553080541 -0.004909472842271612L0.10897825775299452 -0.00481792740470155L0.10857391997512877 -0.0047568970046683035L0.10758014219722156 -0.004585248539598297L0.10725971553053171 -0.004476425301305653L0.10655208886375078 -0.004173289425168036L0.10608673775259092 -0.004032154375551045L0.10521697775247674 -0.00389864782076188L0.10467525330793137 -0.003971122859141474L0.10392759108560767 -0.0039329781173478295L0.10323333330775597 -0.003753697377206322L0.10233683552984019 -0.0033798755550265014L0.10138316441859274 -0.002937388377716843L0.10084151108519279 -0.0026932555702217087L0.10038753775177156 -0.0023919022226550624L0.10028072886288442 -0.002197356533599759L0.1002578310850879 -0.0020066246105869823L0.10037231997401364 -0.0017128957929060107L0.10049626664070388 -0.0015048834138866596L0.10051724441845522 -0.0013123532593368736L0.10048673775180816 -0.0011691892516836333L0.10031123552953147 -0.0009593790682203007L0.100131466640633 -0.0008669276246422442L0.10007727997395932 -0.000719049780656178L0.10004364441840607 -0.0005417752148417776L0.09997772441839459 -0.0004847779205476854L0.09988698664062667 -0.00044371289386901935L0.09981523552947635 -0.00042755615133671654L0.09970394664058801 -0.0004553816486065898L0.0996190399739163 -0.0005009346415505433L0.09948143997387149 -0.0005666834574924451L0.0991241777516052 -0.0007482214745522242L0.09885772441825225 -0.0008624396854202132L0.09853559108483978 -0.0009562375175562465L0.09825448886260801 -0.0010352250068024205L0.09784161775144185 -0.0012021753584718908L0.09745804441804751 -0.0014157990556640243L0.09727941330690726 -0.0015594110281256235L0.09705683552911637 -0.0017936770277771075L0.09675525330683854 -0.0020250246682991246L0.09660001775125693 -0.0021596589971153435L0.09638625775122023 -0.002306185545506878L0.09608467552897082 -0.0025198037527331962L0.0958445333067317 -0.002584203150945541L0.09573623108447293 -0.0025375304259682707L0.09564833775111481 -0.0025343889907532002L0.09556634664004093 -0.0026165150077304133L0.09558981330665972 -0.002727811217326348L0.09550782219555742 -0.0027657326422456663L0.09524136886217605 -0.002988323999929321L0.0950510044177264 -0.0031171212950766858L0.0947610844176836 -0.0032548935839571413L0.09443020441761973 -0.003427669282686452L0.09390604441756523 -0.0037496585033238716L0.09368936886197332 -0.003928266117469548L0.09336439108413686 -0.004101039107936799L0.09308911997300129 -0.0043090389162045994L0.09271720886182777 -0.00463102153068462L0.0922223466395593 -0.005070350013468783L0.09177143108394148 -0.00546278008381762L0.09127649775051339 -0.00582581397215165L0.09073484441711344 -0.006115700387042011L0.09008766219480435 -0.006341191176019834L0.08907155552800816 -0.006560847234368339L0.08843326219455605 -0.006592931733152341L0.08787098663893289 -0.0065666807814039885L0.0875108088611114 -0.006499370575355101L0.08731461330553714 -0.006487703462212835L0.08680510219434723 -0.006534371895639879L0.08587688886089495 -0.00667213322547866L0.08522977774967444 -0.0068536454431722404L0.08436008886067725 -0.0070849658353324685L0.08356947552724137 -0.00733378531278106L0.08293985774935209 -0.007617828148497097L0.08264709330487108 -0.007790586300629343L0.08243333330483438 -0.007954593721777314L0.08196478219367975 -0.008159658207318898L0.08153726219360635 -0.008434945643898573L0.0810570488602167 -0.008718981198839515L0.08083738663795259 -0.008844844694976928L0.08057975108236803 -0.008953208158231973L0.08028691552677003 -0.009102403783614932L0.08011119997117078 -0.00925765642561771L0.07987112886004866 -0.009497713597014013L0.07945235552665508 -0.009934075767517925L0.07896922663771022 -0.010420238580010732L0.07863834663764635 -0.010774481114495416L0.0785973155265367 -0.010879923230007194L0.07851290663762711 -0.011011837682460168L0.07733317330412603 -0.010529046762428607L0.07591415108171873 -0.009948209789399698L0.07583215997058801 -0.009877988333158783L0.07567116441501298 -0.009839848835966336L0.07561256885945511 -0.00973732072570499L0.07564186663722694 -0.009488515169365996L0.07553647997053758 -0.009450375326395033L0.07536367997052196 -0.009409318869387562L0.07507674663715136 -0.00930118090951737L0.07487464885934969 -0.009277623848959138L0.07472239997044028 -0.009277623848959138L0.0745144710815282 -0.009245541355269893L0.07414846219256788 -0.009157594816002756L0.07386152885919728 -0.009113621478888945L0.07341935997025928 -0.009063814889586297L0.07305918219243779 -0.00895545170529033L0.07253793774789585 -0.008738948872561991L0.07226565330343249 -0.008659751190677412L0.07188200885893536 -0.008489913176688901L0.07134618663666004 -0.008276100296740196L0.0708278577476733 -0.007974561750771159L0.07072247108101237 -0.007936420563680713L0.07065512885876046 -0.00790142038570707L0.07046767996985182 -0.007760746305862654L0.0701016710809057 -0.007579237920865012L0.06934035552524165 -0.0071370183609644755L0.0688747199696138 -0.006835471819343297L0.0688688888585034 -0.006753578233855251L0.06881320885848652 -0.006618733935042087L0.06869907552515997 -0.006425553594795019L0.06841790219178279 -0.006200063248670062L0.06796407108060976 -0.005889759673010531L0.06771518219169081 -0.005708243382528622L0.06746039108055868 -0.005538393699168864L0.0670504355249193 -0.0053597927793731515L0.06678106663598271 -0.005198692317762266L0.06656439108041923 -0.005034674374783776L0.06626273774705282 -0.004762282043458299L0.06577377774696913 -0.0042558610675200725L0.0653140444135829 -0.0037812963383458964L0.0649567821912882 -0.0034738922738881683L0.06454099552459525 -0.0031548179913158947L0.06427155552454167 -0.0029762071613532726L0.06390554663558135 -0.0027038017771445766L0.0635687644133327 -0.0023349073830729594L0.06334917330218559 -0.0020126831670808087L0.0632789155244069 -0.0019101363740361421L0.06308271996880421 -0.0017492473673996756L0.06296851552436067 -0.0016642025862836363L0.06274892441321356 -0.0015764649179317303L0.06257612441316951 -0.0014739170836008952L0.06221594663537644 -0.001230897934163977L0.06196414219087387 -0.0010522791047122837L0.061800159968626645 -0.0010024631681062601L0.06161569774640441 -0.00089991396350797L0.061378471079692076 -0.0007096250753590994L0.06100072885743657 -0.00039636464508419067L0.060813351079602285 -0.00021190793634673355L0.06065519996849389 -8.310193042859737e-05L0.06054689774624933 -2.453321813788989e-05L0.06040339552399132 0.0001453389393333282L0.06021893330174066 0.0002595598298000823L0.06011347552394852 0.0002624770667125631L0.06004005817253244 0.0003194009899516459L0.023465610539176396 0.0003194009899516459L0.023452711074440913 0.00020592758603754646L0.023178079963273035 -0.0006371445342239213L0.022689759963199663 -0.0017088567271628108L0.022491431074286083 -0.002746659740395785L0.022430417740935127 -0.004211433953585697L0.02279656885211523 -0.005309981837683608L0.023437493296640355 -0.006271412560806766L0.024013493296720867 -0.006935090107376141L0.02430725329676875 -0.0073699077681652625L0.024368266630091284 -0.00782020191415711L0.024238559963436046 -0.008377061390490326L0.02388762663004229 -0.008644046250608994L0.023429813296644397 -0.008689814917147487L0.02307127107435747 -0.008346548729420533L0.022819466629883323 -0.008010908027003438L0.02252193774093314 -0.007827830174903738L0.0221939021853359 -0.007804945388514284L0.0218352888519604 -0.007988023338128869L0.021598773296403806 -0.008438086647657883L0.021354648851897196 -0.00914009522806225L0.020996035518521694 -0.010269252748727808L0.020458222185098407 -0.01122653431798426L0.019577013296100176 -0.012153508936592061L0.01821893329589841 -0.012969867163164395L0.016799911073491103 -0.013687056134557452L0.016418399962333297 -0.014068639756430912L0.01631159107341773 -0.014541520576756284L0.01635738662898234 -0.015045128440206668L0.016540497740123783 -0.01586905855235443L0.016906719962378247 -0.016601452532228222L0.017562862184718142 -0.017684879228198724L0.01801299551812008 -0.019142190841890283L0.018203715518126273 -0.020385700403551255L0.01823422218480175 -0.02118687472734848L0.018215164407038742 -0.022102420149053614L0.017925244406981733 -0.022785141467707604L0.017356853295780184 -0.02354051824659109L0.016754115517926493 -0.02400208101171586L0.016330719962297735 -0.024162213873623273L0.015758488851105312 -0.0241278997391845L0.015300746628824413 -0.023956328655842185L0.014957422184338043 -0.023704465549826637L0.01475525329541938 -0.02331915475281221L0.014747573295423422 -0.022979818226261273L0.014904017739866049 -0.02245320210290913L0.015243502184375757 -0.022003061312929617L0.015907253295580404 -0.021423276614200404L0.016738897740140146 -0.02079772837070948L0.01712033774020938 -0.020286561892433497L0.017272942184661133 -0.019828772270699346L0.017242435517999866 -0.019386230189226694L0.01687621329573119 -0.01868438902528169L0.016326879962321073 -0.01809693309967031L0.01572414218442475 -0.01776877075599259L0.014930684406579076 -0.017547601829363657L0.014404248850937051 -0.017166049108041648L0.014083822184218775 -0.016464171671259464L0.013938897739762979 -0.015480088342243903L0.01409143107312616 -0.014038131137667165L0.014564462184267768 -0.012847829436751113L0.0151824888510248 -0.012130626454435856L0.01604271996227169 -0.011440781197094907L0.017227146629068102 -0.010757430875756313L0.017745973295845374 -0.010726919905351906L0.018325742184785554 -0.010711664411999777L0.018341031073717318 -0.010253772744931666L0.01814270218480374 -0.00874321163315983L0.01805111107366031 -0.0073699077681652625L0.018447839962604462 -0.006500268045471103L0.018554648851520028 -0.005782959080747219L0.018341031073717318 -0.0051268924745357936L0.018341031073717318 -0.004318238452952983L0.018508853295941208 -0.002807692996483979L0.018493635518197493 -0.0016173044686951243L0.018295235518138497 -0.0011137635594877793L0.01821893329589841 -0.0008238433888152485L0.017410257740223756 -0.0004578500386713813L0.01673882662902315 -7.34640792643404e-08L0.016369671124152774 0.0003194009899516459L0.010468914030610676 0.0003194009899516459L0.009704586628060952 -0.00033577676242657617L0.009368871072453544 -0.0007017711526202675L0.009201048850215443 -0.0011137635594877793L0.009124746627975355 -0.0017851501268353331L0.008895839961269303 -0.00238023399394649L0.00871272885014207 -0.003189148882327686L0.008438097738988404 -0.0034180207879614954L0.008178684405621084 -0.004089371343440007L0.007858257738902807 -0.004150402691891486L0.007598844405535488 -0.003875760941127737L0.007629351072225177 -0.003356988399019656L0.008117671072270127 -0.002655109693719737L0.008148177738945606 -0.0024262337233693643L0.008010862183354561 -0.0020903223704920038L0.007965066627818373 -0.0018156674487528335L0.007553048849985089 -0.0015104932539315996L0.006896906627659405 -0.0014341993664714892L0.007095306627675768 -0.001022210032758153L0.007659857738872233 -0.0005034030289721159L0.007949435635083546 0.0003194009899516459L0 0.0003194009899516459L0 -0.05L1.05 -0.05z", "M0.025755075519185766 0.026637233801210414L0.023773137741130768 0.02581789270026036L0.021224942185241957 0.024841395040127168L0.017517066629139322 0.022949311568893904L0.013488764406375253 0.020904643883199014L0.010589564405947272 0.019561866322703736L0.008224479961185693 0.01834103260220843L0.0051574577385480325 0.016876212630108967L0.002105653293668297 0.015411442607586423L0 0.014404212206486022L0 -0.0003194009899516459L0.007724615472341156 -0.0003194009899516459L0.007949777738929242 0.0003203730407221883L0.00825498662783275 0.00120542818669378L0.008575413294551026 0.0018310816186470902L0.008865333294593825 0.002136281721490718L0.009353582183550202 0.0021210216648341884L0.010040231072537154 0.002410963667955457L0.010314933294807815 0.0028992914651979618L0.010543768850411084 0.003189014303174531L0.01087948440599007 0.003463702981264305L0.011093102183806991 0.0037383934158157217L0.011230417739383824 0.004043607069903032L0.011474542183862013 0.004348822892566773L0.011657653294975034 0.0046998237695987655L0.011627146628327978 0.00544739338054967L0.011642435517217109 0.006027324249046728L0.011749244406132675 0.0064849455034590164L0.011962862183906964 0.007019104973011281L0.011962862183906964 0.007309080038069737L0.01185605329501982 0.007644618643766421L0.011581422183894574 0.00794986005330145L0.011474542183862013 0.008361939396820617L0.011535626628301543 0.008956947645827995L0.011764462183904811 0.009506176079227657L0.012329084406218271 0.009765642383030126L0.012847839961821705 0.010177739145603937L0.01295464885070885 0.010467511077436598L0.012878417739585757 0.010726983188021677L0.012771608850698613 0.011016983283951731L0.01295464885070885 0.011352550853871435L0.013427679961907302 0.011444130953677245L0.013809191073065108 0.011398340879352986L0.014312728850924827 0.011276234253173811L0.014678951073179292 0.011322024197326641L0.014678951073179292 0.012069709063396772L0.014724675517655328 0.01251213094441539L0.014999377739897568 0.012680032475543612L0.0151824888510248 0.0125731859707372L0.015075679962137656 0.011428867590154823L0.015151911073260749 0.011032022042002154L0.015106186628813134 0.010299618194025584L0.015151911073260749 0.009094310208098477L0.015151911073260749 0.008194054742659773L0.015426613295502989 0.008163530330591584L0.01562494218444499 0.008575387277531377L0.015563928851094033 0.00901799761928146L0.015457119962206889 0.00935377402414872L0.015502915517743077 0.011718872424623328L0.015533422184418555 0.012359718224317362L0.015731751073360556 0.013153213048838097L0.01593015107337692 0.015121412842958648L0.01640318218453274 0.018447893454663244L0.016647306629010927 0.019332873168721676L0.017059324406844212 0.020675643570839952L0.01734924440690122 0.021316623009155933L0.017745973295845374 0.021530135271433437L0.018173208851450795 0.021926853302232985L0.018814062184887348 0.02207952448848971L0.019332817740519204 0.02217112746073724L0.019790631073888676 0.022140593114954754L0.020263662185087128 0.02235410947287164L0.020629884407370014 0.022384643970426055L0.021500213296363313 0.022457163489136178L0.021634826629707504 0.022423261160170682L0.021818648851962053 0.02227822235974486L0.021959093296430865 0.022225236170868357L0.02222156440757317 0.022229502005686186L0.02243646218538231 0.02226115900364789L0.022693528852087752 0.022277099770306563L0.022904444407686242 0.0222557705768196L0.023205955518818655 0.02217427070341671L0.023670595518893833 0.022132061463381092L0.024491644407888202 0.021974226215206727L0.025982986630324945 0.021644188523609387L0.02627759996367729 0.021691785639660566L0.026727377741536884 0.02177575426543399L0.027180711074933583 0.021980737188386L0.027244639963839745 0.022071217365876805L0.027297902186049328 0.022063808311941102L0.02736197329718948 0.022078626421119907L0.027427537741630204 0.022132735014750438L0.027250755519389713 0.02218684367656465L0.02695635551934572 0.022351415253552886L0.026470666630402206 0.02247085913947444L0.02586814218585687 0.022624205818388532L0.02534426663024192 0.022710421468261188L0.025273866630215025 0.022666640061913768L0.025216906630220137 0.02241136167359059L0.025183839963560217 0.022367131535332874L0.024794506630158253 0.022367131535332874L0.02441619551899521 0.022435160650076114L0.0241995910745203 0.022473104328526006L0.02397829329670742 0.022533724476033967L0.02377868440778741 0.02263835056156438L0.023590026629975114 0.02276767414387848L0.023468639963297733 0.022897896200063883L0.023348604407743778 0.023051917969212354L0.02312431996324449 0.02322480020465889L0.0229069332965679 0.023403969801094604L0.022571573296517045 0.02351892612622919L0.022426506629813048 0.023708200589581452L0.02235489774093935 0.023895455149826716L0.022522008852078557 0.023921275686831223L0.02266707551873992 0.024018495760941505L0.022841582185435527 0.023970671539927935L0.022999377741015792 0.023847631065805786L0.023153191074385404 0.02405599178368334L0.02319393774104128 0.024331936574427004L0.02325281774103871 0.024476084191974223L0.02337164440771744 0.02454770912264337L0.02355205329664045 0.02455242424764492L0.023893671074475265 0.024692081994757586L0.024246311074534788 0.024890342888255645L0.025601617741401128 0.025436182533454144L0.026909848852682217 0.02576041142290819L0.027398311074946946 0.02575142998026081L0.02762423107500922 0.025617382172058L0.027892391075042156 0.02527384447292036L0.028002897741714605 0.024939515323964656L0.02808040885284413 0.024846783791304006L0.028119662186171013 0.024853070668555688L0.028153226630635686 0.0248943844561893L0.028132604408384054 0.025019224186365818L0.028085102186167887 0.025149902134302238L0.02813317329727738 0.025290235424890284L0.02828378663065223 0.02559403050288722L0.028465617741801452 0.025833161177459374L0.02896602663072656 0.026118547978512652L0.029253386630784917 0.02618029596229121L0.029616479964161613 0.026192196493042275L0.029990737741968587 0.026124161427901527L0.03035105774205249 0.025996174963808016L0.03062391107539497 0.02582260796472724L0.030781848853209226 0.02562232194986791L0.030921155519877175 0.0254559415647293L0.03138529774220444 0.025268006601095294L0.03133061329772602 0.025903665686854538L0.031207591075485652 0.026287625396292924L0.03106984885323527 0.02670818871243341L0.030988782186582853 0.02675332148594123L0.03079230218655482 0.026790370813046138L0.0310076266310233 0.026976965182711865L0.031197279964374047 0.027023670037451097L0.03187233774224296 0.027276281309781325L0.03229310218674186 0.027367221731495306L0.0327557510756975 0.027420663353083796L0.033073119964640796 0.027434585130961864L0.03327272885356081 0.02740225068602342L0.033343057742470705 0.027372161710530918L0.03341836440915813 0.027291325761297003L0.03346650663138462 0.027200385500293578L0.03347255107581759 0.02712044804712832L0.033476248853588686 0.02696079812945129L0.033501422186915875 0.02685503880607598L0.03367038218694063 0.02660894168573691L0.03362017774252024 0.026322204392371873L0.03364101329803759 0.02620117802783284L0.03372065774250643 0.026143696237568292L0.033825831075873225 0.026154024991058122L0.03391678218699212 0.026206117872760615L0.03395660440921233 0.02635453802433574L0.033975235520330216 0.026774203830015608L0.03392524440920397 0.027101361879445562L0.03381729774254438 0.027453671446693306L0.033554471075831316 0.027946325871837985L0.03305697774241878 0.027997298188253694L0.03204321774227026 0.027799022953828967L0.03132478218661561 0.027794532019143503L0.031310417742162144 0.027884350802111157L0.03140961774218454 0.02795777779665798L0.03157118218665289 0.028060620709595696L0.03171112885334537 0.02802244754725791L0.031828604408900674 0.02809587477787545L0.03205452440894874 0.02812529061498026L0.032219786631202396 0.028172894913126356L0.03240481774231796 0.028145724528954474L0.03247877329791038 0.028235544046253835L0.0325590577423327 0.02832581285038316L0.032678951075709506 0.028347369607757855L0.03290394663129348 0.02829482503054237L0.033113866631310884 0.028365333580538277L0.03333224885355435 0.02854070725931024L0.03351535996471 0.028636365931362207L0.033719448853616996 0.028700138497768535L0.03415031107591915 0.028797144836588018L0.034586933298186295 0.02885597749140345L0.03475340440930097 0.028776710612746115L0.03479180440932339 0.028668701305321065L0.03474572440931922 0.02855552760300384L0.03477324440933671 0.028426860244181285L0.03484399996490595 0.028355228844915814L0.03486967107599526 0.028274615594881425L0.03483482663155257 0.028157850152851438L0.034832053298202936 0.028094527488150334L0.03489818663159383 0.028037941356132023L0.03501751107602047 0.02802559121816728L0.03514885329826711 0.028101039389071047L0.03519649774273148 0.028188164227472612L0.03520865774271442 0.028252385226664956L0.035271804409404695 0.02828000477678927L0.03537832885383807 0.02827079825814849L0.03548207996497865 0.028218478323481122L0.03559322663164721 0.028123718775844964L0.0356740799650197 0.028103733969047084L0.0361480355206254 0.028579779085546875L0.03652947552066621 0.028915483884546234L0.03683468440959814 0.028976562230425884L0.03736872885414755 0.028762788399845363L0.03767393774306527 0.028869675182164656L0.037963857743108065 0.02899183183043874L0.03823848885426173 0.029144303575620256L0.03852840885430453 0.029205382246857425L0.03916926218771266 0.028808596988255886L0.039321866632178626 0.02844235419891561L0.03967279996555817 0.028167730293318982L0.040206844410064946 0.028213538247129577L0.04042053329899886 0.028121922388365306L0.04086298663239063 0.02764902598937624L0.04129022218799605 0.027603218637537452L0.04161071996583132 0.02772514713791452L0.041748035521393945 0.02796945427220976L0.04145811552137957 0.028381276612634565L0.04113768885466129 0.02883913607436739L0.04119870218799804 0.02941893372489801L0.041275004410223914 0.029892074167449323L0.04109189329909668 0.030395535454204037L0.04078668441015054 0.030563282340722253L0.04016111996563154 0.030746524668231245L0.03938287996551537 0.03085319182422097L0.0379790755208802 0.030563282340722253L0.03683468440959814 0.030273374813845066L0.03535457774272288 0.029693565630068974L0.034133884409243365 0.02976991495893344L0.03283688885350955 0.02970883548496772L0.030273404408717397 0.028915483884546234L0.027450506630529503 0.02764902598937624z", ""},
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
