package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestShapes(t *testing.T) {
	defer setEpsilon(0.01)()

	test.T(t, Rectangle(0.0, 10.0), &Path{})
	test.T(t, Rectangle(5.0, 10.0), MustParseSVG("H5V10H0z"))
	test.T(t, RoundedRectangle(0.0, 10.0, 0.0), &Path{})
	test.T(t, RoundedRectangle(5.0, 10.0, 0.0), MustParseSVG("H5V10H0z"))
	test.T(t, RoundedRectangle(5.0, 10.0, 2.0), MustParseSVG("M0 2A2 2 0 0 1 2 0L3 0A2 2 0 0 1 5 2L5 8A2 2 0 0 1 3 10L2 10A2 2 0 0 1 0 8z"))
	test.T(t, RoundedRectangle(5.0, 10.0, -2.0), MustParseSVG("M0 2A2 2 0 0 0 2 0L3 0A2 2 0 0 0 5 2L5 8A2 2 0 0 0 3 10L2 10A2 2 0 0 0 0 8z"))
	test.T(t, BeveledRectangle(0.0, 10.0, 0.0), &Path{})
	test.T(t, BeveledRectangle(5.0, 10.0, 0.0), MustParseSVG("H5V10H0z"))
	test.T(t, BeveledRectangle(5.0, 10.0, 2.0), MustParseSVG("M0 2L2 0L3 0L5 2L5 8L3 10L2 10L0 8z"))
	test.T(t, Circle(0.0), &Path{})
	test.T(t, Circle(2.0), MustParseSVG("M2 0A2 2 0 0 1 -2 0A2 2 0 0 1 2 0z"))
	test.T(t, RegularPolygon(2, 2.0, true), &Path{})
	test.T(t, RegularPolygon(4, 0.0, true), &Path{})
	test.T(t, RegularPolygon(4, 2.0, true), MustParseSVG("M0 2L-2 0L0 -2L2 0z"))
	test.T(t, RegularPolygon(3, 2.0, true), MustParseSVG("M0 2L-1.7321 -1L1.7321 -1z"))
	test.T(t, RegularPolygon(3, 2.0, false), MustParseSVG("M-1.7321 1L0 -2L1.7321 1z"))
	test.T(t, StarPolygon(2, 4.0, 2.0, true), &Path{})
	test.T(t, StarPolygon(4, 4.0, 2.0, true), MustParseSVG("M0 4L-1.41 1.41L-4 0L-1.41 -1.41L0 -4L1.41 -1.41L4 0L1.41 1.41z"))
	test.T(t, StarPolygon(3, 4.0, 2.0, false), MustParseSVG("M-3.4641 2L-1.7321 -1L0 -4L1.7321 -1L3.4641 2L0 2z"))
}
