package canvas

import "testing"
import "fmt"
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
		{"M10 10z", 2.0, RoundCapper, RoundJoiner, ""},
		{"M10 10V5", 2.0, RoundCapper, RoundJoiner, "M11 10V5A1 1 0 0 1 9 5V10A1 1 0 0 1 11 10z"},
		{"M10 10V5", 2.0, ButtCapper, RoundJoiner, "M11 10V5H9V10H11z"},
		{"M10 10V5", 2.0, SquareCapper, RoundJoiner, "M11 10V5V4H9V5V10V11H11V10z"},
		{"M10 10V5H15L10 10", 2.0, ButtCapper, RoundJoiner, "M11 10V5L10 6H15L14.293 4.2929L9.2929 9.2929L10.707 10.707L15.707 5.7071A1 1 0 1 1 15 4H10A1 1 0 0 1 9 5V10H11z"},
		{"M10 10V5H15L10 10z", 2.0, ButtCapper, RoundJoiner, "M11 10V5L10 6H15L14.293 4.2929L9.2929 9.2929L11 10H9A1 1 0 1 1 10.707 10.707L15.707 5.7071A1 1 0 1 1 15 4H10A1 1 0 0 1 9 5V10z"},
		{"M10 10V5H15z", 2.0, ButtCapper, RoundJoiner, "M11 10V5L10 6H15L14.293 4.2929L9.2929 9.2929L11 10H9A1 1 0 1 1 10.707 10.707L15.707 5.7071A1 1 0 1 1 15 4H10A1 1 0 0 1 9 5V10z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprintf("%s_%g", tt.orig, tt.w), func(t *testing.T) {
			p := ParseSVGPath(tt.orig)
			sp := p.Stroke(tt.w, tt.cp, tt.jr)
			test.T(t, sp.ToSVGPath(), tt.stroke)
		})
	}
}
