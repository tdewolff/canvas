package pdf

import (
	"fmt"
	"math"
	"strings"

	"github.com/Seanld/canvas"
	"github.com/tdewolff/minify/v2"
)

const mmPerPt = 25.4 / 72.0
const ptPerMm = 72 / 25.4

////////////////////////////////////////////////////////////////

func float64sEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, f := range a {
		if f != b[i] {
			return false
		}
	}
	return true
}

type dec float64

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", canvas.Precision, f)
	s = string(minify.Decimal([]byte(s), canvas.Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}
