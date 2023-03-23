// +build !fribidi js

package text

import (
	"github.com/benoitkugler/textprocessing/fribidi"
)

// EmbeddingLevels returns the embedding levels for each rune of a mixed LTR/RTL string. A change in level means a change in direction.
func EmbeddingLevels(str []rune) []int {
	pbase := fribidi.CharType(fribidi.ON)
	vis, _ := fribidi.LogicalToVisual(fribidi.DefaultFlags, str, &pbase)

	levels := make([]int, len(str))
	for i, level := range vis.EmbeddingLevels {
		levels[i] = int(level)
	}
	return levels
}
