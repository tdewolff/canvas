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

	test.T(t, (&Polyline{}).ToPath(), MustParseSVG(""))
	test.T(t, (&Polyline{}).Add(10, 0).ToPath(), MustParseSVG(""))
	test.T(t, (&Polyline{}).Add(10, 0).Add(20, 10).ToPath(), MustParseSVG("M10 0L20 10"))
	test.T(t, (&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 0).ToPath(), MustParseSVG("M10 0L20 10z"))

	test.That(t, (&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(12, 5))
	test.That(t, !(&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(5, 5))

	FillRule = EvenOdd
	test.That(t, (&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(12, 5))
	test.That(t, !(&Polyline{}).Add(10, 0).Add(20, 10).Add(10, 10).Add(10, 0).Interior(5, 5))
	FillRule = NonZero
}

func TestPolylineSmoothen(t *testing.T) {
	test.T(t, (&Polyline{}).Smoothen(), MustParseSVG(""))
	test.T(t, (&Polyline{}).Add(0, 0).Add(10, 0).Smoothen(), MustParseSVG("M0 0L10 0"))
	test.T(t, (&Polyline{}).Add(0, 0).Add(5, 10).Add(10, 0).Add(5, -10).Smoothen(), MustParseSVG("M0 0C1.4444 5.1111 2.8889 10.222 5 10C7.1111 9.7778 9.8889 4.2222 10 0C10.111 -4.2222 7.5556 -7.1111 5 -10"))
	test.T(t, (&Polyline{}).Add(0, 0).Add(5, 10).Add(10, 0).Add(5, -10).Add(0, 0).Smoothen(), MustParseSVG("M0 0C0 5 2.5 10 5 10C7.5 10 10 5 10 0C10 -5 7.5 -10 5 -10C2.5 -10 0 -5 0 0z"))
}
