package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestLaTeX(t *testing.T) {
	_, _, err := ParseLaTeX(`$x=\frac{5}{2}$`)
	test.Error(t, err)

	_, _, err = ParseLaTeX(`$x=\frac{5}{2}`)
	if err == nil {
		test.Fail(t)
	}
}
