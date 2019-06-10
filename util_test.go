package canvas

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestMatrix(t *testing.T) {
	var tts = []struct {
		m                     Matrix
		p                     Point
		res                   Point
		theta, xscale, yscale float64
	}{
		{Identity.Translate(3.0, 5.0), Point{1.0, 10.0}, Point{4.0, 15.0}, 0.0, 1.0, 1.0},
		{Identity.Scale(2.0, 1.5), Point{1.0, 10.0}, Point{2.0, 15.0}, 0.0, 2.0, 1.5},
		{Identity.Rotate(90.0), Point{1.0, 10.0}, Point{-10.0, 1.0}, math.Pi / 2.0, 1.0, 1.0},
		{Identity.Translate(0.0, 10.0).Rotate(90.0).Translate(0.0, -10.0), Point{1.0, 10.0}, Point{0.0, 11.0}, math.Pi / 2.0, 1.0, 1.0},
		{Identity.Scale(2.0, 1.5).Rotate(90.0), Point{1.0, 10.0}, Point{-20.0, 1.5}, math.Pi / 2.0, 2.0, 1.5},
		{Identity.Rotate(90.0).Scale(2.0, 1.5), Point{1.0, 10.0}, Point{-15.0, 2.0}, math.Pi / 2.0, 1.5, 2.0},
	}
	for j, tt := range tts {
		t.Run(fmt.Sprintf("%v", j), func(t *testing.T) {
			p := tt.m.Dot(tt.p)
			test.That(t, p.Equals(tt.res), p, "!=", tt.res)
			test.T(t, tt.m.theta(), tt.theta, "theta")
			xscale, yscale := tt.m.scale()
			test.T(t, xscale, tt.xscale, "xscale")
			test.T(t, yscale, tt.yscale, "yscale")
		})
	}
}
