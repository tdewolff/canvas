package svg

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/minify/v2"
)

////////////////////////////////////////////////////////////////

// splitAlpha splits an alpha premultiplied color into an opaque color and an opacity in [0,1].
// SVG 1.1 has no rgba() color notation, translucent paints must be written as an opaque color
// with the alpha in a separate fill-opacity/stroke-opacity/stop-opacity property. Renderers that
// implement SVG 1.1 instead of CSS Color 4 discard an rgba() value and fall back to black.
func splitAlpha(col color.RGBA) (canvas.CSSColor, float64) {
	if col.A == 255 {
		return canvas.CSSColor(col), 1.0
	}
	nrgba := color.NRGBAModel.Convert(col).(color.NRGBA)
	return canvas.CSSColor{R: nrgba.R, G: nrgba.G, B: nrgba.B, A: 255}, float64(col.A) / 255.0
}

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
