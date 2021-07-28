package canvas

import (
	"fmt"
	"math"
)

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

type intersection struct {
	pos      Point
	s, t     float64 // line or Bézier curve parameter, or arc angle, of intersection
	crossing bool    // instead of tangential non-crossing/touching
}

type intersections []intersection

func (zs *intersections) add(pos Point, s, t float64, crossing bool) {
	*zs = append(*zs, intersection{
		pos:      pos,
		s:        s,
		t:        t,
		crossing: crossing,
	})
}

func (zs intersections) Has() bool {
	return 0 < len(zs)
}

func (zs intersections) HasCrossing() bool {
	for _, z := range zs {
		if z.crossing {
			return true
		}
	}
	return false
}

// http://www.cs.swan.ac.uk/~cssimon/line_intersection.html
func intersectionLineLine(a0, a1, b0, b1 Point) (Point, bool) {
	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if Equal(div, 0.0) {
		return Point{}, false
	}

	ta := db.PerpDot(a0.Sub(b0)) / div
	tb := da.PerpDot(a0.Sub(b0)) / div
	if 0.0 <= ta && ta <= 1.0 && 0.0 <= tb && tb <= 1.0 {
		return a0.Interpolate(a1, ta), true
	}
	return Point{}, false
}

// http://mathworld.wolfram.com/Circle-LineIntersection.html
func intersectionRayCircle(l0, l1, c Point, r float64) (Point, Point, bool) {
	d := l1.Sub(l0).Norm(1.0) // along line direction, anchored in l0, its length is 1
	D := l0.Sub(c).PerpDot(d)
	discriminant := r*r - D*D
	if discriminant < 0 {
		return Point{}, Point{}, false
	}
	discriminant = math.Sqrt(discriminant)

	ax := D * d.Y
	bx := d.X * discriminant
	if d.Y < 0.0 {
		bx = -bx
	}
	ay := -D * d.X
	by := math.Abs(d.Y) * discriminant
	return c.Add(Point{ax + bx, ay + by}), c.Add(Point{ax - bx, ay - by}), true
}

// https://math.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
func intersectionCircleCircle(c0 Point, r0 float64, c1 Point, r1 float64) (Point, Point, bool) {
	R := c0.Sub(c1).Length()
	if R < math.Abs(r0-r1) || r0+r1 < R || c0.Equals(c1) {
		return Point{}, Point{}, false
	}
	R2 := R * R

	k := r0*r0 - r1*r1
	a := 0.5
	b := 0.5 * k / R2
	c := 0.5 * math.Sqrt(2.0*(r0*r0+r1*r1)/R2-k*k/(R2*R2)-1.0)

	i0 := c0.Add(c1).Mul(a)
	i1 := c1.Sub(c0).Mul(b)
	i2 := Point{c1.Y - c0.Y, c0.X - c1.X}.Mul(c)
	return i0.Add(i1).Add(i2), i0.Add(i1).Sub(i2), true
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineQuad(l0, l1, p0, p1, p2 Point) intersections {
	// write line as A.X = bias
	A := Point{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p0.Sub(p1.Mul(2.0)).Add(p2))
	b := A.Dot(p1.Sub(p0).Mul(2.0))
	c := A.Dot(p0) - bias

	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
		}
	}

	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	if horizontal {
		if l1.X < l0.X {
			l0, l1 = l1, l0
		}
	} else if l1.Y < l0.Y {
		l0, l1 = l1, l0
	}

	zs := intersections{}
	for _, root := range roots {
		if 0.0 <= root && root <= 1.0 {
			pos := quadraticBezierPos(p0, p1, p2, root)
			dif := A.Dot(quadraticBezierDeriv(p0, p1, p2, root))
			if horizontal {
				if l0.X <= pos.X && pos.X <= l1.X {
					zs.add(pos, (pos.X-l0.X)/(l1.X-l0.X), root, !Equal(dif, 0.0))
				}
			} else if l0.Y <= pos.Y && pos.Y <= l1.Y {
				zs.add(pos, (pos.Y-l0.Y)/(l1.Y-l0.Y), root, !Equal(dif, 0.0))
			}
		}
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineCube(l0, l1, p0, p1, p2, p3 Point) intersections {
	// write line as A.X = bias
	A := Point{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p3.Sub(p0).Add(p1.Mul(3.0)).Sub(p2.Mul(3.0)))
	b := A.Dot(p0.Mul(3.0).Sub(p1.Mul(6.0)).Add(p2.Mul(3.0)))
	c := A.Dot(p1.Mul(3.0).Sub(p0.Mul(3.0)))
	d := A.Dot(p0) - bias

	roots := []float64{}
	r0, r1, r2 := solveCubicFormula(a, b, c, d)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
			if !math.IsNaN(r2) {
				roots = append(roots, r2)
			}
		}
	}

	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	if horizontal {
		if l1.X < l0.X {
			l0, l1 = l1, l0
		}
	} else if l1.Y < l0.Y {
		l0, l1 = l1, l0
	}

	zs := intersections{}
	for _, root := range roots {
		if 0.0 <= root && root <= 1.0 {
			pos := cubicBezierPos(p0, p1, p2, p3, root)
			dif := A.Dot(cubicBezierDeriv(p0, p1, p2, p3, root))
			if horizontal {
				if l0.X <= pos.X && pos.X <= l1.X {
					zs.add(pos, (pos.X-l0.X)/(l1.X-l0.X), root, !Equal(dif, 0.0))
				}
			} else if l0.Y <= pos.Y && pos.Y <= l1.Y {
				zs.add(pos, (pos.Y-l0.Y)/(l1.Y-l0.Y), root, !Equal(dif, 0.0))
			}
		}
	}
	return zs
}

func intersectionLineEllipse(l0, l1, center, radius Point, phi, theta0, theta1 float64) intersections {
	// we take the ellipse center as the origin
	l0 = l0.Sub(center)
	l1 = l1.Sub(center)

	// write ellipse as Ax^2 + Bxy + Cy^2 = 1 and line as Dx + Ey = F
	sin, cos := math.Sincos(phi)
	A := cos*cos/(radius.X*radius.X) + sin*sin/(radius.Y*radius.Y)
	B := 2.0 * cos * sin * (1.0/(radius.X*radius.X) - 1.0/(radius.Y*radius.Y))
	C := sin*sin/(radius.X*radius.X) + cos*cos/(radius.Y*radius.Y)
	D := l1.Y - l0.Y
	E := l0.X - l1.X
	F := l0.Dot(Point{D, E})

	// rewrite as a polynomial: ax^2 + bx + c = 0
	var a, b, c float64
	horizontal := math.Abs(D) <= math.Abs(E)
	if horizontal {
		a = (A*E*E - B*D*E + C*D*D)
		b = (B*F*E - 2.0*C*F*D)
		c = (C*F*F - E*E)
	} else {
		a = (C*D*D - B*E*D + A*E*E)
		b = (B*F*D - 2.0*A*F*E)
		c = (A*F*F - D*D)
	}
	fmt.Println(a, b, c, horizontal)

	// find solutions
	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
		}
	}
	fmt.Println(roots)

	zs := intersections{}
	for _, root := range roots {
		// get intersection position with center as origin
		var x, y float64
		if horizontal {
			x = root
			y = (F - D*x) / E
		} else {
			y = root
			x = (F - E*x) / D
		}

		angle := math.Atan2(y, x)
		fmt.Println("intersection", x, y, "angle", angle*180.0/math.Pi)
	}
	return zs
}

// For Bézier-Bézier interesections:
// see T.W. Sederberg, "Computer Aided Geometric Design", 2012
// see T.W. Sederberg and S.R. Parry, "Comparison of three curve intersection algorithms", 1986
