//go:build !latex

package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestLaTeXMatrix(t *testing.T) {
	// \matrix (and \pmatrix) make TeX emit y1..y4 movements, whose operand
	// length was miscounted, desynchronising the DVI stream.
	for _, formula := range []string{`\matrix{a & b \cr c & d}`, `\pmatrix{a & b \cr c & d}`} {
		p, err := ParseLaTeX(formula)
		test.Error(t, err, formula)
		test.T(t, p.Empty(), false, formula)
	}
}
