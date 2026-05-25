package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

// unitSquareGridPaths mimics a dense Voronoi diagram: many adjacent closed cells sharing edges.
func unitSquareGridPaths() Paths {
	var ps Paths
	for x := 0.05; x < 1.0; x += 0.1 {
		for y := 0.05; y < 1.0; y += 0.1 {
			p := &Path{}
			p.MoveTo(x, y)
			p.LineTo(x+0.1, y)
			p.LineTo(x+0.1, y+0.1)
			p.LineTo(x, y+0.1)
			p.Close()
			ps = append(ps, p)
		}
	}
	return ps
}

func TestStrokeMergedVoronoiLikeGrid(t *testing.T) {
	origFast := FastStroke
	FastStroke = false
	t.Cleanup(func() { FastStroke = origFast })

	merged := unitSquareGridPaths().Merge()
	// Panics in bentleyOttmann Settle during Stroke (Positive fill rule).
	_ = merged.Stroke(0.2, ButtCap, MiterJoin, Tolerance)
}

func TestSettleMergedVoronoiLikeGrid(t *testing.T) {
	merged := unitSquareGridPaths().Merge()
	// Settle alone succeeds for the same geometry.
	got := merged.Settle(NonZero)
	test.T(t, 0 < got.Len(), true)
}

func TestSettlePositiveMergedVoronoiLikeGrid(t *testing.T) {
	merged := unitSquareGridPaths().Merge()
	// Path.Stroke uses Settle(Positive); this is the fill rule that panics on dense grids.
	_ = merged.Settle(Positive)
}

// TestStrokeMergedGridWithRasterTolerance matches stroke tolerance used when rendering
// through the rasterizer at DPMM(2), as in Voronoi diagram visualization tests.
func TestStrokeMergedGridWithRasterTolerance(t *testing.T) {
	origFast := FastStroke
	FastStroke = false
	t.Cleanup(func() { FastStroke = origFast })

	merged := unitSquareGridPaths().Merge()
	tol := PixelTolerance / 2.0
	_ = merged.Stroke(0.2, ButtCap, MiterJoin, tol)
}
