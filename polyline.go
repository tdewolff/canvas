package canvas

type Polyline struct {
	coords []Point
}

func PolylineFromPath(p *Path) *Polyline {
	return &Polyline{p.Flatten().Coords()}
}

func PolylineFromPathCoords(p *Path) *Polyline {
	return &Polyline{p.Coords()}
}

func (p *Polyline) ToPath() *Path {
	if len(p.coords) < 2 {
		return &Path{}
	}

	q := &Path{}
	q.MoveTo(p.coords[0].X, p.coords[0].Y)
	for _, coord := range p.coords[1 : len(p.coords)-1] {
		q.LineTo(coord.X, coord.Y)
	}
	if p.coords[0].Equals(p.coords[len(p.coords)-1]) {
		p.Close()
	} else {
		q.LineTo(p.coords[len(p.coords)-1].X, p.coords[len(p.coords)-1].Y)
	}
	return q
}

// Smoothen returns a new path that smoothens out a path using cubic Béziers between all the path points. This is equivalent of saying all path commands are linear and are replaced by cubic Béziers so that the curvature at is smooth along the whole path.
func (p *Polyline) Smoothen() *Path {
	if len(p.d) == 0 {
		return &Path{}
	}

	K := p.Coords()
	if len(K) == 2 { // there are only two coordinates, that's a straight line
		q := &Path{}
		q.MoveTo(K[0].X, K[0].Y)
		q.LineTo(K[1].X, K[1].Y)
		return q
	}

	var p1, p2 []Point
	if p.Closed() {
		// see http://www.jacos.nl/jacos_html/spline/circular/index.html
		n := len(K) - 1
		p1 = make([]Point, n+1)
		p2 = make([]Point, n)

		a := make([]float64, n)
		b := make([]float64, n)
		c := make([]float64, n)
		d := make([]Point, n)
		for i := 0; i < n; i++ {
			a[i] = 1.0
			b[i] = 4.0
			c[i] = 1.0
			d[i] = K[i].Mul(4.0).Add(K[i+1].Mul(2.0))
		}

		lc := make([]float64, n)
		lc[0] = a[0]
		lr := c[n-1]

		for i := 0; i < n-3; i++ {
			m := a[i+1] / b[i]
			b[i+1] -= m * c[i]
			d[i+1] = d[i+1].Sub(d[i].Mul(m))
			lc[i+1] = -m * lc[i]
			m = lr / b[i]
			b[n-1] -= m * lc[i]
			lr = -m * c[i]
			d[n-1] = d[n-1].Sub(d[i].Mul(m))
		}

		i := n - 3
		m := a[i+1] / b[i]
		b[i+1] -= m * c[i]
		d[i+1] = d[i+1].Sub(d[i].Mul(m))
		c[i+1] -= m * lc[i]
		m = lr / b[i]
		b[n-1] -= m * lc[i]
		a[n-1] -= m * c[i]
		d[n-1] = d[n-1].Sub(d[i].Mul(m))

		i = n - 2
		m = a[i+1] / b[i]
		b[i+1] -= m * c[i]
		d[i+1] = d[i+1].Sub(d[i].Mul(m))

		p1[n-1] = d[n-1].Div(b[n-1])
		lc[n-2] = 0.0
		for i := n - 2; i >= 0; i-- {
			p1[i] = d[i].Sub(p1[i+1].Mul(c[i])).Sub(p1[n-1].Mul(lc[i])).Div(b[i])
		}
		p1[n] = p1[0]
		for i := 0; i < n; i++ {
			p2[i] = K[i+1].Mul(2.0).Sub(p1[i+1])
		}
	} else {
		// see https://www.particleincell.com/2012/bezier-splines/
		n := len(K) - 1
		p1 = make([]Point, n)
		p2 = make([]Point, n)

		a := make([]float64, n)
		b := make([]float64, n)
		c := make([]float64, n)
		d := make([]Point, n)
		b[0] = 2.0
		c[0] = 1.0
		d[0] = K[0].Add(K[1].Mul(2.0))
		for i := 1; i < n-1; i++ {
			a[i] = 1.0
			b[i] = 4.0
			c[i] = 1.0
			d[i] = K[i].Mul(4.0).Add(K[i+1].Mul(2.0))
		}
		a[n-1] = 2.0
		b[n-1] = 7.0
		d[n-1] = K[n].Add(K[n-1].Mul(8.0))

		// solve with tridiagonal matrix algorithm
		for i := 1; i < n; i++ {
			w := a[i] / b[i-1]
			b[i] -= w * c[i-1]
			d[i] = d[i].Sub(d[i-1].Mul(w))
		}

		p1[n-1] = d[n-1].Div(b[n-1])
		for i := n - 2; i >= 0; i-- {
			p1[i] = d[i].Sub(p1[i+1].Mul(c[i])).Div(b[i])
		}
		for i := 0; i < n-1; i++ {
			p2[i] = K[i+1].Mul(2.0).Sub(p1[i+1])
		}
		p2[n-1] = K[n].Add(p1[n-1]).Mul(0.5)
	}

	q := &Path{}
	q.MoveTo(K[0].X, K[0].Y)
	for i := 0; i < len(K)-1; i++ {
		q.CubeTo(p1[i].X, p1[i].Y, p2[i].X, p2[i].Y, K[i+1].X, K[i+1].Y)
	}
	if p.Closed() {
		q.Close()
	}
	return q
}
