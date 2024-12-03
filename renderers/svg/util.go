package svg

import (
	"fmt"
	"math"
	"strings"

	"github.com/Seanld/canvas"
	"github.com/tdewolff/minify/v2"
)

////////////////////////////////////////////////////////////////

type num float64

func (f num) String() string {
	s := fmt.Sprintf("%.*g", canvas.Precision, f)
	if num(math.MaxInt32) < f || f < num(math.MinInt32) {
		if i := strings.IndexAny(s, ".eE"); i == -1 {
			s += ".0"
		}
	}
	return string(minify.Number([]byte(s), canvas.Precision))
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
