package canvas

// Polyline defines a list of points in 2D space that form a polyline. If the last coordinate equals the first coordinate, we assume the polyline to close itself.
type Polyline struct {
	coords []Point
}

// PolylineFromPath returns a polyline from the given path by approximating it by linear line segments, i.e. by flattening.
func PolylineFromPath(p *Path) *Polyline {
	return &Polyline{p.Flatten().Coords()}
}

// PolylineFromPathCoords returns a polyline from the given path from each of the start/end coordinates of the segments, i.e. converting all non-linear segments to linear ones.
func PolylineFromPathCoords(p *Path) *Polyline {
	return &Polyline{p.Coords()}
}

// Add adds a new point to the polyline.
func (p *Polyline) Add(x, y float64) *Polyline {
	p.coords = append(p.coords, Point{x, y})
	return p
}

// Coords returns the list of coordinates of the polyline.
func (p *Polyline) Coords() []Point {
	return p.coords
}

// ToPath convertes the polyline to a path. If the last coordinate equals the first one, we close the path.
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
		q.Close()
	} else {
		q.LineTo(p.coords[len(p.coords)-1].X, p.coords[len(p.coords)-1].Y)
	}
	return q
}

// FillCount returns the number of times the test point is enclosed by the polyline. Counter clockwise enclosures are counted positively and clockwise enclosures negatively.
func (p *Polyline) FillCount(x, y float64) int {
	test := Point{x, y}
	count := 0
	prevCoord := p.coords[0]
	for _, coord := range p.coords[1:] {
		// see https://wrf.ecse.rpi.edu//Research/Short_Notes/pnpoly.html
		if (test.Y < coord.Y) != (test.Y < prevCoord.Y) &&
			test.X < (prevCoord.X-coord.X)*(test.Y-coord.Y)/(prevCoord.Y-coord.Y)+coord.X {
			if prevCoord.Y < coord.Y {
				count--
			} else {
				count++
			}
		}
		prevCoord = coord
	}
	return count
}

// Interior is true when the point (x,y) is in the interior of the path, i.e. gets filled. This depends on the FillRule.
func (p *Polyline) Interior(x, y float64, fillRule FillRule) bool {
	fillCount := p.FillCount(x, y)
	if fillRule == NonZero {
		return fillCount != 0
	}
	return fillCount%2 != 0
}

// Smoothen returns a new path that smoothens out a path using cubic Béziers between all the path points. It makes sure that the curvature is smooth along the whole path. If the path is closed it will be smooth between start and end segments too.
func (p *Polyline) Smoothen() *Path {
	K := p.coords
	if len(K) < 2 {
		return &Path{}
	} else if len(K) == 2 { // there are only two coordinates, that's a straight line
		q := &Path{}
		q.MoveTo(K[0].X, K[0].Y)
		q.LineTo(K[1].X, K[1].Y)
		return q
	}

	var p1, p2 []Point
	closed := K[0].Equals(K[len(K)-1])
	if closed {
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
	if closed {
		q.Close()
	}
	return q
}
