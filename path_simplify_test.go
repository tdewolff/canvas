package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestPathSimplifyVisvalingamWhyatt(t *testing.T) {
	tests := []struct {
		p         string
		tolerance float64
		r         string
	}{
		// closed path
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 1.0, "M0 0L10 0L10 4L11 5L10 6L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 2.0, "M0 0L10 0L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 50.0, "M0 0L10 0L10 10L0 10z"},
		{"M0 0L10 0L10 4L11 5L10 6L10 10L0 10z", 51.0, ""},

		// open path
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 1.0, "M0 0L10 0L11 1L12 0L13 -5L14 0"},
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 2.0, "M0 0L12 0L13 -5L14 0"},
		{"M0 0L10 0L11 1L12 0L13 -5L14 0", 6.0, "M0 0L14 0"},

		// bugs
		{"M0 0L1 1L2 0zM2 0L4 2L4 0zM4 0L5 1L6 0z", 2.0, "M2 0L4 2L4 0z"},
		{"M2 0L4 2L4 0zM0 0L1 1L2 0zM4 0L5 1L6 0z", 2.0, "M2 0L4 2L4 0z"},
		{"M0 0L1 1L2 0zM4 0L5 1L6 0zM2 0L4 2L4 0z", 2.0, "M2 0L4 2L4 0z"},
		{"M0 0L40 0L40.1 0.1L40.2 0L40.3 0.5L40 40z", 2.0, "M0 0L40.2 0L40.3 0.5L40 40z"},
		{"M0 0L40 0L40.1 0.1L40.2 0L40.3 0.5L40 40z", 3.0, "M0 0L40.2 0L40 40z"},
		{"M0 0L10 0", 1.0, "M0 0L10 0"},
	}

	for _, tt := range tests {
		t.Run(tt.p, func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			r := MustParseSVGPath(tt.r)
			test.T(t, p.SimplifyVisvalingamWhyatt(tt.tolerance), r)
		})
	}

	p := MustParseSVGPath("M0 0L10 0L5 10z")
	test.T(t, NewVisvalingamWhyatt(func(_ Point) bool {
		return false
	}).Simplify(p.Split(), 1.0), p)
}

func TestPathFastClip(t *testing.T) {
	tests := []struct {
		p    string
		rect Rect
		r    string
	}{
		{"M-5 5L5 5L5 15L-5 15z", Rect{0, 0, 10, 10}, "M-5 5L5 5L5 15L-5 15z"},
		{"M1 5L9 5L9 15L1 15z", Rect{0, 0, 10, 10}, "M1 5L9 5L9 15L1 15z"},
		{"M1 5L9 5L9 15L5 20L1 15z", Rect{0, 0, 10, 10}, "M1 5L9 5L9 15L1 15z"},
		{"M-5 5L9 5L9 15L5 20L-5 15z", Rect{0, 0, 10, 10}, "M-5 5L9 5L9 15L-5 15z"},
		{"M-5 15L-5 5L9 5L9 15L5 20z", Rect{0, 0, 10, 10}, "M-5 5L9 5L9 15L-5 15z"},
		{"M20 2L30 2L30 8L20 8z", Rect{0, 0, 10, 10}, ""},
		{"M20 5L30 5L30 15L20 15z", Rect{0, 0, 10, 10}, ""},
		{"M20 -10L30 -10L30 20L20 20z", Rect{0, 0, 10, 10}, ""},
		{"M14 5L14 14L5 14z", Rect{0, 0, 10, 10}, "M14 14L5 14L14 5z"},
		{"M14 14L5 14L14 5z", Rect{0, 0, 10, 10}, "M5 14L14 5L14 14z"},
		{"M5 14L14 5L14 14z", Rect{0, 0, 10, 10}, "M5 14L14 5L14 14z"},
		//{"M16 5L16 16L5 16z", Rect{0, 0, 10, 10}, ""},
		{"M-10 -10L20 -10L20 20L-10 20z", Rect{0, 0, 10, 10}, "M20 -10L20 20L-10 20L-10 -10z"},
		{"M9 11L11 11", Rect{0, 0, 10, 10}, ""},
		{"M15 5L15 15L5 15", Rect{0, 0, 10, 10}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.p, func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			r := MustParseSVGPath(tt.r)
			test.T(t, p.FastClip(tt.rect.X0, tt.rect.Y0, tt.rect.X1, tt.rect.Y1, p.Closed()), r)
		})
	}
}

func TestPathLineClip(t *testing.T) {
	tests := []struct {
		p    string
		rect Rect
		r    string
	}{
		{"M-5 5L5 5L5 15L-5 15z", Rect{0, 0, 10, 10}, "M0 5L5 5L5 10"},
		{"M1 5L9 5L9 15L1 15z", Rect{0, 0, 10, 10}, "M1 10L1 5L9 5L9 10"},
		{"M1 5L9 5L9 15L5 20L1 15z", Rect{0, 0, 10, 10}, "M1 10L1 5L9 5L9 10"},
		{"M-5 5L9 5L9 15L5 20L-5 15z", Rect{0, 0, 10, 10}, "M0 5L9 5L9 10"},
		{"M-5 15L-5 5L9 5L9 15L5 20z", Rect{0, 0, 10, 10}, "M0 5L9 5L9 10"},
		{"M20 2L30 2L30 8L20 8z", Rect{0, 0, 10, 10}, ""},
		{"M20 5L30 5L30 15L20 15z", Rect{0, 0, 10, 10}, ""},
		{"M20 -10L30 -10L30 20L20 20z", Rect{0, 0, 10, 10}, ""},
		{"M14 5L14 14L5 14z", Rect{0, 0, 10, 10}, "M9 10L10 9"},
		{"M14 14L5 14L14 5z", Rect{0, 0, 10, 10}, "M9 10L10 9"},
		{"M5 14L14 5L14 14z", Rect{0, 0, 10, 10}, "M9 10L10 9"},
		//{"M16 5L16 16L5 16z", Rect{0, 0, 10, 10}, ""},
		{"M-10 -10L20 -10L20 20L-10 20z", Rect{0, 0, 10, 10}, ""},
		{"M9 11L11 11", Rect{0, 0, 10, 10}, ""},
		{"M15 5L15 15L5 15", Rect{0, 0, 10, 10}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.p, func(t *testing.T) {
			p := MustParseSVGPath(tt.p)
			r := MustParseSVGPath(tt.r)
			test.T(t, p.LineClip(tt.rect.X0, tt.rect.Y0, tt.rect.X1, tt.rect.Y1), r)
		})
	}
}
