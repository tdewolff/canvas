package canvas

import (
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

func TestParse(t *testing.T) {
	var tts = []struct {
		orig string
		err  string
	}{
		{"5", "bad path: path should start with command"},
		{"MM", "bad path: number should follow command 'M'"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			_, err := ParseSVG(tt.orig)
			test.T(t, err.Error(), tt.err)
		})
	}
}

func TestPath(t *testing.T) {
	var tts = []struct {
		orig string
		res  string
	}{
		{"A10 10 0 0 0 40 0", "A20 20 0 0 0 40 0"},  // scale ellipse
		{"A10 5 90 0 0 40 0", "A40 20 90 0 0 40 0"}, // scale ellipse
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)
			test.T(t, p.String(), tt.res)
		})
	}
}

func TestPathDirection(t *testing.T) {
	var tts = []struct {
		orig      string
		direction float64
	}{
		{"L10 0L10 10z", 2 * math.Pi},
		{"L10 0L10 -10z", -2 * math.Pi},
		{"L10 0z", 0.0},
		{"M0 0z", 0.0},
		{"L10 0L20 0z", 0.0},
		{"L10 0L20 0L10 1z", 2 * math.Pi},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)
			test.Float(t, p.direction(), tt.direction)
		})
	}
}

func TestPathBounds(t *testing.T) {
	var tts = []struct {
		orig   string
		bounds Rect
	}{
		{"Q50 100 100 0z", Rect{0, 0, 100, 50}},
		{"Q0 0 100 0z", Rect{0, 0, 100, 0}},
		{"Q100 0 100 0z", Rect{0, 0, 100, 0}},
		{"C0 100 100 100 100 0z", Rect{0, 0, 100, 75}},
		{"C0 0 100 90 100 0z", Rect{0, 0, 100, 40}},
		{"C0 90 100 0 100 0z", Rect{0, 0, 100, 40}},
		{"C0 0 100 0 100 0z", Rect{0, 0, 100, 0}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			bounds := p.Bounds()
			test.Float(t, bounds.X, tt.bounds.X)
			test.Float(t, bounds.Y, tt.bounds.Y)
			test.Float(t, bounds.W, tt.bounds.W)
			test.Float(t, bounds.H, tt.bounds.H)
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
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			length := p.Length()
			if math.Abs(tt.length-length)/length > 0.01 {
				test.Fail(t, length, "!=", tt.length, "±1%")
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
		{"M5 5M10 10z", []string{"M5 5", "M10 10z"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			ps := p.Split()
			if len(ps) != len(tt.split) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.split, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p.String(), tt.split[i])
				}
			}
		})
	}
}

func TestPathSplitAt(t *testing.T) {
	var tts = []struct {
		orig  string
		d     []float64
		split []string
	}{
		{"L4 3L8 0z", []float64{0.0, 5.0, 10.0, 18.0}, []string{"L4 3", "M4 3L8 0", "M8 0L0 0"}},
		{"L4 3L8 0z", []float64{5.0, 20.0}, []string{"L4 3", "M4 3L8 0L0 0"}},
		{"L4 3L8 0z", []float64{2.5, 7.5, 14.0}, []string{"L2 1.5", "M2 1.5L4 3L6 1.5", "M6 1.5L8 0L4 0", "M4 0L0 0"}},
		{"C10 0 10 0 20 0", []float64{10.0}, []string{"C5 0 7.5 0 10 0", "M10 0C12.5 0 15 0 20 0"}},
		{"A10 10 0 0 1 -20 0", []float64{15.707963}, []string{"A10 10 0 0 1 -10 10", "M-10 10A10 10 0 0 1 -20 0"}},
		{"A10 10 0 0 0 20 0", []float64{15.707963}, []string{"A10 10 0 0 0 10 10", "M10 10A10 10 0 0 0 20 0"}},
		{"A10 10 0 1 0 2.9289 -7.0711", []float64{15.707963}, []string{"A10 10 0 0 0 10.024 9.9999", "M10.024 9.9999A10 10 0 1 0 2.9289 -7.0711"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			ps := p.SplitAt(tt.d...)
			if len(ps) != len(tt.split) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.split, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p.String(), tt.split[i])
				}
			}
		})
	}
}

func TestPathTranslate(t *testing.T) {
	var tts = []struct {
		dx, dy     float64
		orig       string
		translated string
	}{
		{10.0, 10.0, "M5 5L10 0Q10 10 15 5C15 0 20 0 20 5A5 5 0 0 0 30 5z", "M15 15L20 10Q20 20 25 15C25 10 30 10 30 15A5 5 0 0 0 40 15z"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			p = p.Translate(tt.dx, tt.dy)
			test.T(t, p.String(), tt.translated)
		})
	}
}

func TestPathReverse(t *testing.T) {
	var tts = []struct {
		orig string
		inv  string
	}{
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
		{"M5 5A2.5 5 0 0 0 10 5", "M10 5A2.5 5 0 0 1 5 5"}, // bottom-half of ellipse along y
		{"M5 5A2.5 5 0 0 1 10 5", "M10 5A2.5 5 0 0 0 5 5"},
		{"M5 5A2.5 5 0 1 0 10 5", "M10 5A2.5 5 0 1 1 5 5"},
		{"M5 5A2.5 5 0 1 1 10 5", "M10 5A2.5 5 0 1 0 5 5"},
		{"M5 5A5 2.5 90 0 0 10 5", "M10 5A5 2.5 90 0 1 5 5"}, // same shape
		{"M5 5A2.5 5 0 0 0 10 5z", "M5 5L10 5A2.5 5 0 0 1 5 5z"},
		{"L0 5L5 5", "M5 5L0 5L0 0"},
		{"L-1 5L5 5z", "L5 5L-1 5z"},
		{"Q0 5 5 5", "M5 5Q0 5 0 0"},
		{"Q0 5 5 5z", "L5 5Q0 5 0 0z"},
		{"C0 5 5 5 5 0", "M5 0C5 5 0 5 0 0"},
		{"C0 5 5 5 5 0z", "L5 0C5 5 0 5 0 0z"},
		{"A2.5 5 0 0 0 5 0", "M5 0A2.5 5 0 0 1 0 0"},
		{"A2.5 5 0 0 0 5 0z", "L5 0A2.5 5 0 0 1 0 0z"},
		{"M5 5L10 10zL15 10", "M15 10L5 5L10 10z"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			p = p.Reverse()
			test.T(t, p.String(), tt.inv)
		})
	}
}

func TestPathOptimize(t *testing.T) {
	var tts = []struct {
		orig string
		opt  string
	}{
		{"M0 0", ""},
		{"M10 10z", ""},
		{"M10 10M20 20", "M20 20"},
		{"M10 10L20 20zz", "M10 10L20 20z"},
		{"M10 10L20 20L20 20", "M10 10L20 20"},
		{"M10 10L20 20L30 30", "M10 10L30 30"},
		{"L10 10A5 5 0 0 0 10 10", "L10 10"},
		{"Q0 0 10 10", "L10 10"},
		{"Q10 10 10 10", "L10 10"},
		{"C0 0 0 0 10 10", "L10 10"},
		{"C0 0 10 10 10 10", "L10 10"},
		{"C10 10 0 0 10 10", "L10 10"},
		{"C10 10 10 10 10 10", "L10 10"},
		{"C10 10 10 10 10 10L20 20", "L20 20"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p, err := ParseSVG(tt.orig)
			test.Error(t, err)

			opt := p.Optimize().String()
			if strings.HasPrefix(opt, "M0 0") {
				opt = opt[4:]
			}
			test.T(t, opt, tt.opt)
		})
	}
}

func plotPathLengthParametrization(filename string, speed, length func(float64) float64, tmin, tmax float64) {
	T3, totalLength := invPolynomialApprox3(gaussLegendre5, speed, tmin, tmax)
	Tc, _ := invSpeedPolynomialChebyshevApprox(10, gaussLegendre5, speed, tmin, tmax)

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
	anchors1.GlyphStyle.Color = SteelBlue
	anchors1.GlyphStyle.Radius = 5.0

	line1, err := plotter.NewLine(model1Data)
	if err != nil {
		panic(err)
	}
	line1.LineStyle.Color = SteelBlue

	line2, err := plotter.NewLine(model2Data)
	if err != nil {
		panic(err)
	}
	line2.LineStyle.Color = OrangeRed

	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.X.Label.Text = "L"
	p.Y.Label.Text = "t"
	p.Add(scatter, line1, line2, anchors1)

	p.Legend.Add("real", scatter)
	p.Legend.Add("Simple polynomial", line1)
	p.Legend.Add("Chebyshev", line2)

	if err := p.Save(8*vg.Inch, 16*vg.Inch, filename); err != nil {
		panic(err)
	}
}

func TestPathLengthParametrization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping length parametrization test")
	}
	_ = os.Mkdir("test", 0755)

	start := Point{0.0, 0.0}
	cp := Point{0.0, 10.0}
	end := Point{100.0, 0.0}
	speed := func(t float64) float64 {
		return quadraticBezierDeriv(start, cp, end, t).Length()
	}
	length := func(t float64) float64 {
		p0, p1, p2, _, _, _ := splitQuadraticBezier(start, cp, end, t)
		return quadraticBezierLength(p0, p1, p2)
	}
	plotPathLengthParametrization("test/quadratic_bezier_parametrization.png", speed, length, 0.0, 1.0)

	start = Point{0.0, 0.0}
	cp1 := Point{0.0, 10.0}
	cp2 := Point{100.0, 10.0}
	end = Point{100.0, 0.0}
	speed = func(t float64) float64 {
		return cubicBezierDeriv(start, cp1, cp2, end, t).Length()
	}
	length = func(t float64) float64 {
		p0, p1, p2, p3, _, _, _, _ := splitCubicBezier(start, cp1, cp2, end, t)
		return cubicBezierLength(p0, p1, p2, p3)
	}
	plotPathLengthParametrization("test/cubic_bezier_parametrization.png", speed, length, 0.0, 1.0)

	start = Point{0.0, 0.0}
	cp1 = Point{10.0, 10.0}
	cp2 = Point{0.0, 10.0}
	end = Point{10.0, 10.0}
	speed = func(t float64) float64 {
		return cubicBezierDeriv(start, cp1, cp2, end, t).Length()
	}
	length = func(t float64) float64 {
		p0, p1, p2, p3, _, _, _, _ := splitCubicBezier(start, cp1, cp2, end, t)
		return cubicBezierLength(p0, p1, p2, p3)
	}
	plotPathLengthParametrization("test/cubic_bezier_parametrization_inflection.png", speed, length, 0.0, 1.0)

	rx, ry := 100.0, 10.0
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
	plotPathLengthParametrization("test/ellipse_parametrization.png", speed, length, theta1, theta2)
}
