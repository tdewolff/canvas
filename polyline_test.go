package canvas

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestPolyline(t *testing.T) {
	p := &Polyline{}
	p.Add(10, 0)
	p.Add(20, 10)
	test.T(t, len(p.Coords()), 2)
	test.T(t, p.Coords()[0], Point{10, 0})
	test.T(t, p.Coords()[1], Point{20, 10})

	test.T(t, (&Polyline{}).ToPath(), MustParseSVGPath(""))
	test.T(t, (&Polyline{}).Add(10, 0).ToPath(), MustParseSVGPath(""))
	test.T(t, (&Polyline{}).Add(10, 0).Add(20, 10).ToPath(), MustParseSVGPath("M10 0L20 10"))
	test.T(t, (&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 0).ToPath(), MustParseSVGPath("M10 0L20 10z"))

	test.That(t, (&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(12, 5, NonZero))
	test.That(t, !(&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(5, 5, NonZero))

	test.That(t, (&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(12, 5, EvenOdd))
	test.That(t, !(&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(5, 5, EvenOdd))
}

func TestPolylineSmoothen(t *testing.T) {
	test.T(t, (&Polyline{}).Smoothen(), MustParseSVGPath(""))
	test.T(t, (&Polyline{}).Add(0, 0).Add(10, 0).Smoothen(), MustParseSVGPath("L10 0"))
	test.T(t, (&Polyline{}).Add(0, 0).Add(5, 10).Add(10, 0).Add(5, -10).Smoothen(), MustParseSVGPath("C1.4444444444 5.1111111111 2.8888888889 10.2222222222 5 10C7.1111111111 9.7777777778 9.8888888889 4.2222222222 10 0C10.1111111111 -4.2222222222 7.5555555556 -7.1111111111 5 -10"))
	test.T(t, (&Polyline{}).Add(0, 0).Add(5, 10).Add(10, 0).Add(5, -10).Add(0, 0).Smoothen(), MustParseSVGPath("C0 5 2.5 10 5 10C7.5 10 10 5 10 0C10 -5 7.5 -10 5 -10C2.5 -10 0 -5 0 0z"))
}
