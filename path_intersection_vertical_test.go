package canvas

import (
	"math"
	"math/rand"
	"testing"
)

// TestSettleVerticalFirstSegment strokes a merged grid of nearly-exact cells whose corners carry
// noise large enough to leave the shared edges a fraction of the snapping grid apart. The stroke
// outline then holds segments whose two endpoints fall in the same grid column while differing in
// x, which the sweep calls left-to-right because it compares verticality exactly and only snaps to
// the grid after the intersection phase. Splitting such a segment turns the piece before the split
// vertical, and when it points downwards that piece has to be reversed to keep left-endpoints at
// the bottom, which is impossible once its left-endpoint sits in the sweep status: "first segment
// became vertical and needs reversal, but was already in the sweep status". The settled outline of
// the 10x10 grid is the single outer square of side 1.2.
func TestSettleVerticalFirstSegment(t *testing.T) {
	DebugPathIntersection = true
	defer func() { DebugPathIntersection = false }()

	for _, eps := range []float64{1e-16, 1e-14, 1e-13, 1e-12} {
		for _, seed := range []int64{1, 2, 3, 4} {
			rng := rand.New(rand.NewSource(seed))
			merged := noisyGrid(10, eps, rng).Merge()

			// Stroke settles the outline with Settle(Positive)
			var res *Path
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("eps %g seed %d: panic: %v", eps, seed, r)
					}
				}()
				res = merged.Stroke(0.2, ButtCap, MiterJoin, 0.01)
			}()
			if res == nil {
				continue
			}
			if n, a := len(res.Split()), polygonArea(res); n != 1 || math.Abs(a-1.44) > 1e-6 {
				t.Errorf("eps %g seed %d: settled outline has %d subpaths and area %.6f, want 1 subpath of area 1.44", eps, seed, n, a)
			}
		}
	}
}
