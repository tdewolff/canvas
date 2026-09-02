package canvas

import (
	"math"
	"math/rand"
	"testing"
)

// noisyGrid builds n×n adjacent cells of size 0.1 whose corners carry random noise of magnitude
// eps, mimicking floating-point drift in computed geometry such as Voronoi cells.
func noisyGrid(n int, eps float64, rng *rand.Rand) Paths {
	var ps Paths
	nz := func() float64 { return (rng.Float64()*2 - 1) * eps }
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x, y := 0.05+0.1*float64(i), 0.05+0.1*float64(j)
			p := &Path{}
			p.MoveTo(x+nz(), y+nz())
			p.LineTo(x+0.1+nz(), y+nz())
			p.LineTo(x+0.1+nz(), y+0.1+nz())
			p.LineTo(x+nz(), y+0.1+nz())
			p.Close()
			ps = append(ps, p)
		}
	}
	return ps
}

// TestSettleOverlappingStackOrder strokes a merged grid of nearly-exact cells, which settles the
// outline with Settle(Positive). The stroke outlines of adjacent cells overlap exactly along the shared edges, so the
// sweep sees stacks of identical segments starting in the same point. Windings must be computed
// bottom-up along the sweep status; computing them in the square's CompareH order instead reads
// a not-yet-computed neighbour, the windings above the stack come out wrong, and the result
// polygon walk finds no continuation ("next node for result polygon is nil" in debug mode, a
// silently truncated contour otherwise). The settled outline of the 10×10 grid is the single
// outer square of side 1.2.
func TestSettleOverlappingStackOrder(t *testing.T) {
	DebugPathIntersection = true
	defer func() { DebugPathIntersection = false }()

	for _, seed := range []int64{6, 9, 12, 13} {
		rng := rand.New(rand.NewSource(seed))
		merged := noisyGrid(10, 1e-17, rng).Merge()

		// Stroke settles the outline with Settle(Positive)
		var res *Path
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("seed %d: panic: %v", seed, r)
				}
			}()
			res = merged.Stroke(0.2, ButtCap, MiterJoin, 0.01)
		}()
		if res == nil {
			continue
		}
		if n, a := len(res.Split()), polygonArea(res); n != 1 || math.Abs(a-1.44) > 1e-6 {
			t.Errorf("seed %d: settled outline has %d subpaths and area %.6f, want 1 subpath of area 1.44", seed, n, a)
		}
	}
}

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
