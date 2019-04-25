package canvas

import "testing"
import "fmt"
import "github.com/tdewolff/test"

func TestPathStroke(t *testing.T) {
	var tts = []struct {
		orig   string
		w      float64
		cp     Capper
		jr     Joiner
		stroke string
	}{
		{"M10 10", 2.0, RoundCapper, RoundJoiner, ""},
		{"M10 10z", 2.0, RoundCapper, RoundJoiner, ""},
		{"M10 10L10 5", 2.0, RoundCapper, RoundJoiner, "M11 10L11 5A1 1 0 0 0 9 5L9 10A1 1 0 0 0 11 10z"},
		{"M10 10L10 5", 2.0, ButtCapper, RoundJoiner, "M11 10L11 5L9 5L9 10L11 10z"},
		{"M10 10L10 5", 2.0, SquareCapper, RoundJoiner, "M11 10L11 5L11 4L9 4L9 5L9 10L9 11L11 11L11 10z"},
		{"M10 10L10 5L15 5L10 10", 2.0, ButtCapper, RoundJoiner, "M11 10L11 5L10 6L15 6L14.293 4.2929L9.2929 9.2929L10.707 10.707L15.707 5.7071A1 1 0 0 0 15 4L10 4A1 1 0 0 0 9 5L9 10L11 10z"},
		{"M10 10L10 5L15 5L10 10z", 2.0, ButtCapper, RoundJoiner, "M11 10L11 5L10 6L15 6L14.293 4.2929L9.2929 9.2929L11 10zM9 10A1 1 0 0 0 10.707 10.707L15.707 5.7071A1 1 0 0 0 15 4L10 4A1 1 0 0 0 9 5L9 10z"},
		{"M10 10L10 5L15 5z", 2.0, ButtCapper, RoundJoiner, "M11 10L11 5L10 6L15 6L14.293 4.2929L9.2929 9.2929L11 10zM9 10A1 1 0 0 0 10.707 10.707L15.707 5.7071A1 1 0 0 0 15 4L10 4A1 1 0 0 0 9 5L9 10z"},
		{"M100 100A50 50 0 0 1 114.64 64.645", 2.0, ButtCapper, RoundJoiner, "M101 100A49 49 0 0 1 115.35 65.352L113.93 63.938A51 51 0 0 0 99 100L101 100z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprintf("%s_%g", tt.orig, tt.w), func(t *testing.T) {
			p, _ := ParseSVGPath(tt.orig)
			sp := p.Stroke(tt.w, tt.cp, tt.jr)
			test.T(t, sp.String(), tt.stroke)
		})
	}
}
