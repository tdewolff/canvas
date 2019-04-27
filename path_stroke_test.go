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
		{"M10 10L10 5", 2.0, RoundCapper, RoundJoiner, "M9 10L9 5A1 1 0 0 1 11 5L11 10A1 1 0 0 1 9 10z"},
		{"M10 10L10 5", 2.0, ButtCapper, RoundJoiner, "M9 10L9 5L11 5L11 10L9 10z"},
		{"M10 10L10 5", 2.0, SquareCapper, RoundJoiner, "M9 10L9 5L9 4L11 4L11 5L11 10L11 11L9 11L9 10z"},
		{"M10 10L10 5L15 5L10 10", 2.0, ButtCapper, RoundJoiner, "M9 10L9 5A1 1 0 0 1 10 4L15 4A1 1 0 0 1 15.707 5.7071L10.707 10.707L9.2929 9.2929L14.293 4.2929L15 6L10 6L11 5L11 10L9 10z"},
		{"M10 10L10 5L15 5L10 10z", 2.0, ButtCapper, RoundJoiner, "M9 10L9 5A1 1 0 0 1 10 4L15 4A1 1 0 0 1 15.707 5.7071L10.707 10.707A1 1 0 0 1 9 10zM11 10L9.2929 9.2929L14.293 4.2929L15 6L10 6L11 5L11 10z"},
		{"M10 10L10 5L15 5z", 2.0, ButtCapper, RoundJoiner, "M9 10L9 5A1 1 0 0 1 10 4L15 4A1 1 0 0 1 15.707 5.7071L10.707 10.707A1 1 0 0 1 9 10zM11 10L9.2929 9.2929L14.293 4.2929L15 6L10 6L11 5L11 10z"},
		{"M100 100A50 50 0 0 1 114.64 64.645", 2.0, ButtCapper, RoundJoiner, "M99 100A51 51 0 0 1 113.93 63.938L115.35 65.352A49 49 0 0 0 101 100L99 100z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprintf("%s", tt.orig), func(t *testing.T) {
			p, err := Parse(tt.orig)
			test.Error(t, err)

			sp := p.Stroke(tt.w, tt.cp, tt.jr)
			test.T(t, sp.String(), tt.stroke)
		})
	}
}
