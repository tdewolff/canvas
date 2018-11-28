package canvas

import "testing"
import "github.com/tdewolff/test"

func TestPathInvert(t *testing.T) {
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
	}
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			p := ParseSVGPath(tt.orig)
			ip := p.Invert()
			test.T(t, ip.ToSVGPath(), tt.inv)
		})
	}
}
