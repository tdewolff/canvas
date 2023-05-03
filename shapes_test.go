package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestShapes(t *testing.T) {
	defer setEpsilon(1e-6)()

	test.T(t, Rectangle(0.0, 10.0), &Path{})
	test.T(t, Rectangle(5.0, 10.0), MustParseSVGPath("H5V10H0z"))
	test.T(t, RoundedRectangle(0.0, 10.0, 0.0), &Path{})
	test.T(t, RoundedRectangle(5.0, 10.0, 0.0), MustParseSVGPath("H5V10H0z"))
	test.T(t, RoundedRectangle(5.0, 10.0, 2.0), MustParseSVGPath("M0 2A2 2 0 0 1 2 0L3 0A2 2 0 0 1 5 2L5 8A2 2 0 0 1 3 10L2 10A2 2 0 0 1 0 8z"))
	test.T(t, RoundedRectangle(5.0, 10.0, -2.0), MustParseSVGPath("M0 2A2 2 0 0 0 2 0L3 0A2 2 0 0 0 5 2L5 8A2 2 0 0 0 3 10L2 10A2 2 0 0 0 0 8z"))
	test.T(t, BeveledRectangle(0.0, 10.0, 0.0), &Path{})
	test.T(t, BeveledRectangle(5.0, 10.0, 0.0), MustParseSVGPath("H5V10H0z"))
	test.T(t, BeveledRectangle(5.0, 10.0, 2.0), MustParseSVGPath("M0 2L2 0L3 0L5 2L5 8L3 10L2 10L0 8z"))
	test.T(t, Circle(0.0), &Path{})
	test.T(t, Circle(2.0), MustParseSVGPath("M2 0A2 2 0 0 1 -2 0A2 2 0 0 1 2 0z"))
	test.T(t, RegularPolygon(2, 2.0, true), &Path{})
	test.T(t, RegularPolygon(4, 0.0, true), &Path{})
	test.T(t, RegularPolygon(4, 2.0, true), MustParseSVGPath("M0 2L-2 0L0 -2L2 0z"))
	test.T(t, RegularPolygon(3, 2.0, true), MustParseSVGPath("M0 2L-1.732051 -1L1.732051 -1z"))
	test.T(t, RegularPolygon(3, 2.0, false), MustParseSVGPath("M-1.732051 1L0 -2L1.732051 1z"))
	test.T(t, StarPolygon(2, 4.0, 2.0, true), &Path{})
	test.T(t, StarPolygon(4, 4.0, 2.0, true), MustParseSVGPath("M0 4L-1.414214 1.414214L-4 0L-1.414214 -1.414214L0 -4L1.414214 -1.414214L4 0L1.414214 1.414214z"))
	test.T(t, StarPolygon(3, 4.0, 2.0, false), MustParseSVGPath("M-3.464102 2L0 -4L3.464102 2z"))
}
