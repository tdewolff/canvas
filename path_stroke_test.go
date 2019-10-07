package canvas

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestPathStroke(t *testing.T) {
	// TODO: remove last lineTo if followed by close to the same location (butt and square cappers), fix in path.Close() func?
	Tolerance = 1.0
	Epsilon = 1e-3
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
		{"M10 10L10 5", 2.0, ButtCapper, RoundJoiner, "M9 10L9 5L11 5L11 10z"},
		{"M10 10L10 5", 2.0, SquareCapper, RoundJoiner, "M9 10L9 5L9 4L11 4L11 5L11 10L11 11L9 11z"},

		{"M0 0L10 0L20 0", 2.0, ButtCapper, RoundJoiner, "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"M0 0L10 0L10 10", 2.0, ButtCapper, RoundJoiner, "M0 -1L10 -1A1 1 0 0 1 11 0L11 10L9 10L9 1L0 1z"},
		{"M0 0L10 0L10 -10", 2.0, ButtCapper, RoundJoiner, "M0 -1L9 -1L9 -10L11 -10L11 0A1 1 0 0 1 10 1L0 1z"},

		{"M0 0L10 0L20 0", 2.0, ButtCapper, BevelJoiner, "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"M0 0L10 0L10 10", 2.0, ButtCapper, BevelJoiner, "M0 -1L10 -1L11 0L11 10L9 10L9 1L0 1z"},
		{"M0 0L10 0L10 -10", 2.0, ButtCapper, BevelJoiner, "M0 -1L9 -1L9 -10L11 -10L11 0L10 1L0 1z"},

		{"M0 0L10 0L20 0", 2.0, ButtCapper, MiterClipJoiner(BevelJoiner, 4.0), "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"M0 0L10 0L5 0", 2.0, ButtCapper, MiterClipJoiner(BevelJoiner, 4.0), "M0 -1L10 -1L10 1L5 1L5 -1L10 -1L10 1L0 1z"},
		{"M0 0L10 0L10 10", 2.0, ButtCapper, MiterClipJoiner(BevelJoiner, 2.0), "M0 -1L10 -1L11 0L11 10L9 10L9 1L0 1z"},
		{"M0 0L10 0L10 10", 2.0, ButtCapper, MiterClipJoiner(BevelJoiner, 4.0), "M0 -1L10 -1L11 -1L11 0L11 10L9 10L9 1L0 1z"},
		{"M0 0L10 0L10 -10", 2.0, ButtCapper, MiterClipJoiner(BevelJoiner, 4.0), "M0 -1L9 -1L9 -10L11 -10L11 0L11 1L10 1L0 1z"},

		{"M0 0L10 0L20 0", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 4.0), "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"M0 0L10 0L5 0", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 4.0), "M0 -1L10 -1L10 1L5 1L5 -1L10 -1L10 1L0 1z"},
		{"M0 0L10 0L10 10", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 2.0), "M0 -1L10 -1L11 0L11 10L9 10L9 1L0 1z"},

		{"M0 0L10 0L10 10L0 10z", 2.0, ButtCapper, BevelJoiner, "M0 -1L10 -1L11 0L11 10L10 11L0 11L-1 10L-1 0zM1 1L1 9L9 9L9 1z"},
		{"M0 0L0 10L10 10L10 0z", 2.0, ButtCapper, BevelJoiner, "M-1 0L-1 10L0 11L10 11L11 10L11 0L10 -1L0 -1zM1 1L9 1L9 9L1 9z"},
		{"M0 0Q10 0 10 10", 2.0, ButtCapper, BevelJoiner, "M0 -1L9.6467 3.7346L11 10L9 10L7.6849 4.3366L0 1z"},
		{"M0 0C0 10 10 10 10 0", 2.0, ButtCapper, BevelJoiner, "M1 0L3.4056 6.0146L6.9801 5.7289L9 0L11 0L9.7996 5.3497L2.7451 7.9408L-1 0z"},
		{"M0 0A10 5 0 0 0 20 0", 2.0, ButtCapper, BevelJoiner, "M1 0A9 4 0 0 0 19 0L21 0A11 6 0 0 1 -1 0z"},
		{"M0 0A10 5 0 0 1 20 0", 2.0, ButtCapper, BevelJoiner, "M-1 0A11 6 0 0 1 21 0L19 0A9 4 0 0 0 1 0z"},
		{"M5 2L2 2A2 2 0 0 0 0 0", 2.0, ButtCapper, BevelJoiner, "M5 3L2 3L1 2A1 1 0 0 0 0 1L0 -1A3 3 0 0 1 3 2L2 1L5 1z"},

		// two circle quadrants joining at 90 degrees
		{"M0 0A10 10 0 0 1 10 10A10 10 0 0 1 0 0z", 2.0, ButtCapper, ArcsJoiner, "M0 -1A11 11 0 0 1 11 10A11 11 0 0 1 10.958 10.958A11 11 0 0 1 10 11A11 11 0 0 1 -1 0A11 11 0 0 1 -0.958 -0.958A11 11 0 0 1 0 -1zM0 1L1 0A9 9 0 0 0 10 9L9 10A9 9 0 0 0 0 1z"},

		// circles joining at one point (10,0), stroke will never join
		{"M0 0A5 5 0 0 0 10 0A10 10 0 0 1 0 10", 2.0, ButtCapper, ArcsJoiner, "M1 0A4 4 0 0 0 9 0L11 0A11 11 0 0 1 0 11L0 9A9 9 0 0 0 9 0L11 0A6 6 0 0 1 -1 0z"},

		// circle and line intersecting in one point
		{"M0 0A2 2 0 0 1 2 2L5 2", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 10.0), "M0 -1A3 3 0 0 1 3 2L2 1L5 1L5 3L2 3L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1z"},
		{"M0 4A2 2 0 0 0 2 2L5 2", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 10.0), "M0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L2 1L5 1L5 3L2 3L3 2A3 3 0 0 1 0 5z"},
		{"M5 2L2 2A2 2 0 0 0 0 0", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 10.0), "M5 3L2 3L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L0-1A3 3 0 0 1 3 2L2 1L5 1z"},
		{"M5 2L2 2A2 2 0 0 1 0 4", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 10.0), "M5 3L2 3L3 2A3 3 0 0 1 0 5L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L2 1L5 1z"},

		// cut by limit
		{"M0 0A2 2 0 0 1 2 2L5 2", 2.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 1.0), "M0 -1A3 3 0 0 1 3 2L2 1L5 1L5 3L2 3L1 2A1 1 0 0 0 0 1z"},

		// no intersection
		{"M0 0A2 2 0 0 1 2 2L5 2", 3.0, ButtCapper, ArcsClipJoiner(BevelJoiner, 10.0), "M0 -1.5A3.5 3.5 0 0 1 3.5 2L2 .5L5 .5L5 3.5L2 3.5L.5 2A.5 .5 0 0 0 0 1.5z"},
	}
	for j, tt := range tts {
		t.Run(fmt.Sprintf("%v", j), func(t *testing.T) {
			stroke := MustParseSVG(tt.orig).Stroke(tt.w, tt.cp, tt.jr)
			test.T(t, stroke, MustParseSVG(tt.stroke))
		})
	}
}

func TestPathStrokeEllipse(t *testing.T) {
	rx, ry := 20.0, 10.0
	nphi := 12
	ntheta := 120
	for iphi := 0; iphi < nphi; iphi++ {
		phi := float64(iphi) / float64(nphi) * math.Pi
		for itheta := 0; itheta < ntheta; itheta++ {
			theta := float64(itheta) / float64(ntheta) * 2.0 * math.Pi
			outer := ellipsePos(rx+1.0, ry+1.0, phi, 0.0, 0.0, theta)
			inner := ellipsePos(rx-1.0, ry-1.0, phi, 0.0, 0.0, theta)
			test.Float(t, outer.Sub(inner).Length(), 2.0, fmt.Sprintf("phi=%g theta=%g", phi, theta))
		}
	}
}

func TestPathOffset(t *testing.T) {
	// TODO: cut at intersection on inner bends
	var tts = []struct {
		orig   string
		w      float64
		offset string
	}{
		{"M0 0L10 0L10 10L0 10z", 0.0, "M0 0L10 0L10 10L0 10z"},
		{"M0 0L10 0L10 10L0 10", 1.0, ""},
		{"M0 0L10 0L10 10L0 10z", 1.0, "M0 -1L10 -1A1 1 0 0 1 11 0L11 10A1 1 0 0 1 10 11L0 11A1 1 0 0 1 -1 10L-1 0A1 1 0 0 1 0 -1z"},
		{"M0 0L10 0L10 10L0 10z", -1.0, "M1 1L9 1L9 9L1 9z"},
	}
	for j, tt := range tts {
		t.Run(fmt.Sprintf("%v", j), func(t *testing.T) {
			offset := MustParseSVG(tt.orig).Offset(tt.w)
			test.T(t, offset, MustParseSVG(tt.offset))
		})
	}
}
