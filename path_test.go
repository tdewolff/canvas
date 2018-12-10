package canvas

import (
	"testing"
	"github.com/tdewolff/test"
	"strings"
)

func TestPathSplit(t *testing.T) {
	var tts = []struct {
		orig string
		split []string
	}{
		{"M5 5L6 6z", []string{"M5 5L6 6z"}},
		{"L5 5M10 10L20 20z", []string{"M0 0L5 5", "M10 10L20 20z"}},
		{"L5 5zL10 10", []string{"M0 0L5 5z", "M0 0L10 10"}},
		{"M5 5M10 10z", []string{"M5 5", "M10 10z"}},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			ps := ParseSVGPath(tt.orig).Split()
			if len(ps) != len(tt.split) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.ToSVGPath())
				}
				test.T(t, strings.Join(origs, "\n"), strings.Join(tt.split, "\n"))
			} else {
				for i, p := range ps {
					test.T(t, p.ToSVGPath(), tt.split[i])
				}
			}
		})
	}
}

func TestPathTranslate(t *testing.T) {
	var tts = []struct {
		dx, dy float64
		orig string
		translated string
	}{
		{10.0, 10.0, "M5 5L10 0Q10 10 15 5C15 0 20 0 20 5A5 5 0 0 0 30 5z", "M15 15L20 10Q20 20 25 15C25 10 30 10 30 15A5 5 0 0 0 40 15z"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := ParseSVGPath(tt.orig).Translate(tt.dx, tt.dy)
			test.T(t, p.ToSVGPath(), tt.translated)
		})
	}
}

func TestPathReverse(t *testing.T) {
	var tts = []struct {
		orig string
		inv string
	}{
		{"M5 5", "M5 5"},
		{"M5 5z", "M5 5z"},
		{"M5 5V10L10 5", "M10 5L5 10V5"},
		{"M5 5V10L10 5z", "M5 5H10L5 10z"},
		{"M5 5V10L10 5M10 10V20L20 10z", "M10 10H20L10 20zM10 5L5 10V5"},
		{"M5 5V10L10 5zM10 10V20L20 10z", "M10 10H20L10 20zM5 5H10L5 10z"},
		{"M5 5Q10 10 15 5", "M15 5Q10 10 5 5"},
		{"M5 5Q10 10 15 5z", "M5 5H15Q10 10 5 5z"},
		{"M5 5C5 10 10 10 10 5", "M10 5C10 10 5 10 5 5"},
		{"M5 5C5 10 10 10 10 5z", "M5 5H10C10 10 5 10 5 5z"},
		{"M5 5A2.5 5 0 0 0 10 5", "M10 5A2.5 5 0 0 1 5 5"}, // bottom-half of ellipse along y
		{"M5 5A2.5 5 0 0 1 10 5", "M10 5A2.5 5 0 0 0 5 5"},
		{"M5 5A2.5 5 0 1 0 10 5", "M10 5A2.5 5 0 1 1 5 5"},
		{"M5 5A2.5 5 0 1 1 10 5", "M10 5A2.5 5 0 1 0 5 5"},
		{"M5 5A5 2.5 90 0 0 10 5", "M10 5A5 2.5 90 0 1 5 5"}, // same shape
		{"M5 5A2.5 5 0 0 0 10 5z", "M5 5H10A2.5 5 0 0 1 5 5z"},
		{"L0 5L5 5", "M5 5H0V0"},
		{"L-1 5L5 5z", "M0 0L5 5H-1z"},
		{"Q0 5 5 5", "M5 5Q0 5 0 0"},
		{"Q0 5 5 5z", "M0 0L5 5Q0 5 0 0z"},
		{"C0 5 5 5 5 0", "M5 0C5 5 0 5 0 0"},
		{"C0 5 5 5 5 0z", "M0 0H5C5 5 0 5 0 0z"},
		{"A2.5 5 0 0 0 5 0", "M5 0A2.5 5 0 0 1 0 0"},
		{"A2.5 5 0 0 0 5 0z", "M0 0H5A2.5 5 0 0 1 0 0z"},
		{"M5 5L10 10zL15 10", "M15 10L5 5L10 10z"},
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := ParseSVGPath(tt.orig).Reverse()
			test.T(t, p.ToSVGPath(), tt.inv)
		})
	}
}
