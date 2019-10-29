package canvas

import "math"

// intersection between two line segments
// see http://www.cs.swan.ac.uk/~cssimon/line_intersection.html
func intersectionLineLine(a0, a1, b0, b1 Point) (Point, bool) {
	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if equal(div, 0.0) {
		return Point{}, false
	}

	ta := db.PerpDot(a0.Sub(b0)) / div
	tb := da.PerpDot(a0.Sub(b0)) / div
	if 0.0 <= ta && ta <= 1.0 && 0.0 <= tb && tb <= 1.0 {
		return a0.Interpolate(a1, ta), true
	}
	return Point{}, false
}

//func intersectionLineQuad(a0, a1, p0, p1, p2 Point) (Point, Point, bool) {
//}

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
