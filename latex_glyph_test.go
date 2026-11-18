//go:build !latex

package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestLaTeXUprightG(t *testing.T) {
	// lmroman10regular's "g" loads with an empty outline, so every upright g
	// (\rm g, \log, \arg, text in \hbox) drew as a speck. It now falls back to
	// another design size of the same face.
	p, err := ParseLaTeX(`{\rm g}`)
	test.Error(t, err)
	test.T(t, 1.0 < p.Bounds().W(), true, "width of an upright g (mm)")
}
