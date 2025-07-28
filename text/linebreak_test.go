package text

import (
	"fmt"
	"math"
	"testing"

	"github.com/tdewolff/test"
)

func TestLinebreak(t *testing.T) {
	lineWidth := 100.0
	P := Penalty(0.0, 0.0, false)
	G := Glue(0.0, 10.0, 0.0)
	g := Glue(10.0, -10.0, 0.0)

	var tests = []struct {
		items  []Item
		breaks string
		ratios []float64
	}{
		// full lines without spaces
		{[]Item{Box(100.0), P, Box(100.0)}, "1", []float64{0.0, 0.0}},
		{[]Item{Box(50.0), Box(50.0), P, Box(100.0)}, "2", []float64{0.0, 0.0}},
		{[]Item{Box(50.0), P, Box(50.0), P, Box(100.0)}, "3", []float64{0.0, 0.0}},

		// stretch line at spaces
		{[]Item{Box(50.0), G, Box(30.0), P, Box(100.0)}, "3", []float64{2.0, 0.0}},
		{[]Item{Box(50.0), G, P, g, Box(30.0), P, Box(100.0)}, "5", []float64{0.0, 0.0}},
		{[]Item{Box(50.0), P, Box(30.0), G, P, g, Box(100.0)}, "4", []float64{2.0, 0.0}},

		// line too short
		{[]Item{Box(80.0), P, Box(100.0)}, "1", []float64{0.0, 0.0}},
		{[]Item{Box(50.0), G, Box(20.0), P, Box(100.0)}, "3", []float64{0.0, 0.0}},
		{[]Item{Box(50.0), P, Box(40.0), G, Box(100.0)}, "", []float64{0.0}},
		{[]Item{Box(50.0), P, Box(40.0), g, Box(100.0)}, "3", []float64{0.0, 0.0}},
		{[]Item{Box(50.0), P, Box(40.0), P, Box(60.0), P, Box(50.0)}, "5>3", []float64{0.0, 0.0, 0.0}},
		{[]Item{Box(50.0), P, Box(40.0), P, Box(50.0), P, Box(60.0)}, "5>3", []float64{0.0, 0.0, 0.0}},
		{[]Item{Box(50.0), P, Box(40.0), P, Box(50.0), P, Box(40.0), P, Box(50.0)}, "7>3", []float64{0.0, 0.0, 0.0}},
		{[]Item{Box(30.0), P, Box(30.0), P, Box(30.0), P, Box(60.0), G, P, g, Box(60.0)}, "8>5", []float64{0.0, 0.0, 0.0}},

		// CJK
		{[]Item{Box(21.0), P, Box(21.0), P, Box(21.0), P, Box(21.0), P, Box(21.0), P, Box(21.0), P, Box(21.0), P, Box(21.0), P, Box(21.0)}, "15>7", []float64{0.0, 0.0, 0.0}},

		// line too long
		{[]Item{Box(120.0), P, Box(100.0)}, "1", []float64{0.0, 0.0}},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%v/%v", i, tt.breaks), func(t *testing.T) {
			tt.items = append(tt.items, Glue(0.0, math.Inf(1.0), 0.0))
			tt.items = append(tt.items, Penalty(0.0, -Infinity, true))

			breaks := KnuthLinebreak(tt.items, lineWidth, 2.0, 0)

			s := ""
			for i := len(breaks) - 2; 0 <= i; i-- {
				if 0 < len(s) {
					s += ">"
				}
				s += fmt.Sprintf("%d", breaks[i].Position)
			}
			test.String(t, s, tt.breaks)
			for i, ratio := range tt.ratios {
				test.T(t, breaks[i].Ratio, ratio, fmt.Sprintf("ratio of break %d", i))
			}
		})
	}
}
