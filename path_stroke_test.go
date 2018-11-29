package canvas

import "testing"
import "github.com/tdewolff/test"

func TestPathStroke(t *testing.T) {
	var tts = []struct {
		orig string
		w float64
		cp Capper
		jr Joiner
		stroke string
	}{
		{"M10 10", 2.0, RoundCapper, RoundJoiner, ""},
		{"M10 10z", 2.0, RoundCapper, RoundJoiner, "M11 10A1 1 0 1 15 5z"},
		{"M10 10V5", 2.0, RoundCapper, RoundJoiner, "M11 10V5A1 1 0 0 1 9 5V10A1 1 0 0 1 11 10"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := ParseSVGPath(tt.orig)
			sp := p.Stroke(tt.w, tt.cp, tt.jr)
			test.T(t, sp.ToSVGPath(), tt.stroke)
		})
	}
}
