package canvas

import (
	"encoding/gob"
	"os"
	"testing"

	"github.com/tdewolff/test"
)

// unitSquareGridPaths creates a 10×10 grid of adjacent closed squares sharing edges,
// analogous to Voronoi cells for a regular grid of sites.
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

// TestSettleMergedVoronoiLikeGrid verifies that settling the merged grid with NonZero rule
// produces a non-empty result.
func TestSettleMergedVoronoiLikeGrid(t *testing.T) {
	merged := unitSquareGridPaths().Merge()
	got := merged.Settle(NonZero)
	test.T(t, 0 < got.Len(), true)
}

// TestSettleVoronoiDenseGridStrokeOutline is the canonical regression test for the
// bentleyOttmann polygon-walk bug at shared vertices.
//
// The fixture is the stroke outline produced by stroking 100 adjacent Voronoi cells
// (computed via floating-point arithmetic in github.com/aldernero/gaul), merged into one
// path, stroked at width 0.2 with ButtCap/MiterJoin/tolerance 0.05, and captured before
// the internal Settle call. Because the Voronoi algorithm accumulates floating-point error,
// coordinates are near-but-not-exactly on the rational grid. That creates near-coincident
// vertices at which the old one-directional event-ring search found no matching result
// segment and panicked:
//
//	next node for result polygon is nil, probably buggy intersection code
//
// An exact unit-square grid (unitSquareGridPaths) uses perfectly rational coordinates
// and does NOT trigger this code path; only floating-point imprecision exposes the flaw.
func TestSettleVoronoiDenseGridStrokeOutline(t *testing.T) {
	f, err := os.Open("testdata/voronoi_stroke_pre_settle.gob")
	if err != nil {
		t.Skipf("testdata not found: %v", err)
	}
	defer f.Close()

	var p Path
	if err := gob.NewDecoder(f).Decode(&p); err != nil {
		t.Fatalf("gob decode: %v", err)
	}
	// This panics in bentleyOttmann on the pre-fix code.
	result := p.Settle(Positive)
	test.T(t, 0 < result.Len(), true)
}
