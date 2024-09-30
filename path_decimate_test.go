package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestDecimate(t *testing.T) {
	tests := []struct {
		p         string
		tolerance float64
		r         string
	}{
		// closed path
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 0.5, "M0 0L10 0L10 4L11 5L10 6L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 2.0, "M0 0L10 0L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 3.0, "M0 0L10 0L11 5L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 5.0, "M0 0L10 0L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 50.0, "M0 0L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 51.0, ""},

		// open path
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 1.0, "M0 0L10 0L11 1L12 0L13 -5L14 0"},
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 2.0, "M0 0L12 0L13 -5L14 0"},
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 6.0, "M0 0L11 1L13 -5L14 0"},
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 30.0, "M0 0L14 0"},
	}

	for _, tt := range tests {
		t.Run(tt.p, func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			r := MustParseSVGPath(tt.r)
			test.T(t, p.Decimate(tt.tolerance), r)
		})
	}
}
