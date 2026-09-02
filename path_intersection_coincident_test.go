package canvas

import (
	"math"
	"math/rand"
	"testing"
)

// polygonArea returns the shoelace area of a flattened path (all subpaths).
func polygonArea(p *Path) float64 {
	a := 0.0
	for _, sp := range p.Flatten(1e-3).Split() {
		var pts []Point
		d := sp.Data()
		for i := 0; i < len(d); {
			cmd := d[i]
			n := cmdLen(cmd)
			if cmd != CloseCmd {
				pts = append(pts, Point{d[i+n-3], d[i+n-2]})
			}
			i += n
		}
		for i := range pts {
			j := (i + 1) % len(pts)
			a += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
		}
	}
	return a / 2
}

// TestPathAndNearCoincidentEdge intersects a rounded rectangle with a plain
// rectangle of the same size. The rounded rectangle is shifted right by less
// than BentleyOttmannEpsilon, so its right edge nearly coincides with the
// clip's. The clip's right edge x=50.000000005 lies exactly on a tolerance
// square boundary (snap(x) - eps/2), which makes the raw-coordinate boundary
// test in breakupCrossingSegments disagree with snap() about which column the
// edge belongs to. The polygon walk then finds no next node and closes the
// contour early: the result has two subpaths and a third of the area missing.
func TestPathAndNearCoincidentEdge(t *testing.T) {
	const w, h, r = 50.000000005, 13.0, 2.0
	for _, tc := range []struct {
		name    string
		dx      float64
		flatten bool
	}{
		{"flattened dx=2e-9", 2e-9, true},
		{"flattened dx=5e-9", 5e-9, true},
		{"curves dx=2e-9", 2e-9, false},
		{"control dx=0", 0, true},
		{"control dx=2e-8", 2e-8, true},
	} {
		clip := Rectangle(w, h)
		p := RoundedRectangle(w, h, r).Translate(tc.dx, 0)
		if tc.flatten {
			p = p.Flatten(0.05)
		}
		res := p.And(clip)
		n, a0, a1 := len(res.Split()), polygonArea(p), polygonArea(res)
		if n != 1 || math.Abs(a1-a0) > 1e-3*a0 {
			t.Errorf("%s: And() returned %d subpaths with area %.4f, want 1 subpath with area %.4f", tc.name, n, a1, a0)
		}
	}
}

// TestPathAndNearCoincidentEdgeRandom repeats the scenario at random positions
// and sizes, with the right edge placed on a snap-rounding half-grid line in
// half of the cases. Without the vertical-segment guard in ToleranceEdgeY
// about 5% of the cases lose part of the shape.
func TestPathAndNearCoincidentEdgeRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	fails := 0
	for i := 0; i < 1000; i++ {
		x0 := rng.Float64() * 1000
		y0 := rng.Float64() * 1000
		w := 5 + rng.Float64()*500
		h := 5 + rng.Float64()*50
		if i%2 == 0 {
			w = math.Round((x0+w)/BentleyOttmannEpsilon)*BentleyOttmannEpsilon + BentleyOttmannEpsilon/2 - x0
		}
		dx := (rng.Float64()*2 - 1) * BentleyOttmannEpsilon
		dy := (rng.Float64()*2 - 1) * BentleyOttmannEpsilon
		clip := Rectangle(w, h).Translate(x0, y0)
		p := RoundedRectangle(w, h, 2).Translate(x0+dx, y0+dy).Flatten(0.05)
		res := p.And(clip)
		if a0, a1 := polygonArea(p), polygonArea(res); len(res.Split()) != 1 || math.Abs(a1-a0) > 1e-3*a0 {
			fails++
		}
	}
	if fails != 0 {
		t.Errorf("%d of 1000 random near-coincident intersections returned a wrong polygon", fails)
	}
}
