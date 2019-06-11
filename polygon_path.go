package canvas

type PolygonPath struct {
	*Path
}

func (p *Path) ToPolygon() PolygonPath {
	return PolygonPath{p.Flatten()}
}

func (p *Path) ToPolygonCoords() PolygonPath {
	q := &Path{}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)

		end := Point{p.d[i-2], p.d[i-1]}
		switch cmd {
		case MoveToCmd:
			q.MoveTo(end.X, end.Y)
		case CloseCmd:
			q.Close()
		case LineToCmd, QuadToCmd, CubeToCmd, ArcToCmd:
			q.LineTo(end.X, end.Y)
		}
	}
	return PolygonPath{q}
}

func (p *PolygonPath) QuadTo(cpx, cpy, x1, y1 float64) *Path {
	panic("not allowed on polygon")
}

func (p *PolygonPath) CubeTo(cpx1, cpy1, cpx2, cpy2, x1, y1 float64) *Path {
	panic("not allowed on polygon")
}

func (p *PolygonPath) ArcTo(rx, ry, rot float64, largeArc, sweep bool, x1, y1 float64) *Path {
	panic("not allowed on polygon")
}

func (p *PolygonPath) Arc(rx, ry, rot, theta0, theta1 float64) *Path {
	panic("not allowed on polygon")
}

func (p *PolygonPath) Replace(line LineReplacer, bezier BezierReplacer, arc ArcReplacer) *Path {
	q := p.Path.Replace(line, nil, nil)
	for i := 0; i < len(q.d); {
		cmd := q.d[i]
		i += cmdLen(cmd)

		if cmd != MoveToCmd && cmd != CloseCmd && cmd != LineToCmd {
			panic("not allowed on polygon")
		}
	}
	return q
}

func (p *PolygonPath) ToPath() *Path {
	return p.Path
}

// Smoothen returns a new path that smoothens out a path using cubic Béziers between all the path points. This is equivalent of saying all path commands are linear and are replaced by cubic Béziers so that the curvature at is smooth along the whole path.
func (p *PolygonPath) Smoothen() *Path {
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
