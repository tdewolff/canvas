package canvas

import (
	"math"
)

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
	if R < math.Abs(r0-r1) || r0+r1 < R {
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

////////////////////////////////////////////////////////////////

func ellipsePos(rx, ry, phi, cx, cy, theta float64) Point {
	sintheta, costheta := math.Sincos(theta)
	sinphi, cosphi := math.Sincos(phi)
	x := cx + rx*costheta*cosphi - ry*sintheta*sinphi
	y := cy + rx*costheta*sinphi + ry*sintheta*cosphi
	return Point{x, y}
}

func ellipseDeriv(rx, ry, phi float64, sweep bool, theta float64) Point {
	sintheta, costheta := math.Sincos(theta)
	sinphi, cosphi := math.Sincos(phi)
	dx := -rx*sintheta*cosphi - ry*costheta*sinphi
	dy := -rx*sintheta*sinphi + ry*costheta*cosphi
	if !sweep {
		return Point{-dx, -dy}
	}
	return Point{dx, dy}
}

func ellipseDeriv2(rx, ry, phi float64, sweep bool, theta float64) Point {
	sintheta, costheta := math.Sincos(theta)
	sinphi, cosphi := math.Sincos(phi)
	ddx := -rx*costheta*cosphi + ry*sintheta*sinphi
	ddy := -rx*costheta*sinphi - ry*sintheta*cosphi
	return Point{ddx, ddy}
}

func ellipseCurvatureRadius(rx, ry, phi float64, sweep bool, theta float64) float64 {
	dp := ellipseDeriv(rx, ry, phi, sweep, theta)
	ddp := ellipseDeriv2(rx, ry, phi, sweep, theta)
	a := dp.PerpDot(ddp)
	if equal(a, 0.0) {
		return math.NaN()
	}
	return math.Pow(dp.X*dp.X+dp.Y*dp.Y, 1.5) / a
}

// ellipseNormal returns the normal to the right at angle theta of the ellipse, given rotation phi.
func ellipseNormal(rx, ry, phi float64, sweep bool, theta, d float64) Point {
	return ellipseDeriv(rx, ry, phi, sweep, theta).Rot90CW().Norm(d)
}

// ellipseLength calculates the length of the elliptical arc
// it uses Gauss-Legendre (n=5) and has an error of ~1% or less (empirical)
func ellipseLength(rx, ry, theta1, theta2 float64) float64 {
	if theta2 < theta1 {
		theta1, theta2 = theta2, theta1
	}
	speed := func(theta float64) float64 {
		return ellipseDeriv(rx, ry, 0.0, true, theta).Length()
	}
	return gaussLegendre5(speed, theta1, theta2)
}

// ellipseToCenter converts to the center arc format and returns (centerX, centerY, angleFrom, angleTo) with angles in radians.
// when angleFrom with range [0, 2*PI) is bigger than angleTo with range (-2*PI, 4*PI), the ellipse runs clockwise. The angles are from before the ellipse has been stretched and rotated.
// See https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
func ellipseToCenter(x1, y1, rx, ry, phi float64, large, sweep bool, x2, y2 float64) (float64, float64, float64, float64) {
	if x1 == x2 && y1 == y2 {
		return x1, y1, 0.0, 0.0
	}

	sinphi, cosphi := math.Sincos(phi)
	x1p := cosphi*(x1-x2)/2.0 + sinphi*(y1-y2)/2.0
	y1p := -sinphi*(x1-x2)/2.0 + cosphi*(y1-y2)/2.0

	// reduce rouding errors
	raddiCheck := x1p*x1p/rx/rx + y1p*y1p/ry/ry
	if raddiCheck > 1.0 {
		rx *= math.Sqrt(raddiCheck)
		ry *= math.Sqrt(raddiCheck)
	}

	sq := (rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p) / (rx*rx*y1p*y1p + ry*ry*x1p*x1p)
	if sq < 0.0 {
		sq = 0.0
	}
	coef := math.Sqrt(sq)
	if large == sweep {
		coef = -coef
	}
	cxp := coef * rx * y1p / ry
	cyp := coef * -ry * x1p / rx
	cx := cosphi*cxp - sinphi*cyp + (x1+x2)/2.0
	cy := sinphi*cxp + cosphi*cyp + (y1+y2)/2.0

	// specify U and V vectors; theta = arccos(U*V / sqrt(U*U + V*V))
	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := -(x1p + cxp) / rx
	vy := -(y1p + cyp) / ry

	theta := math.Acos(ux / math.Sqrt(ux*ux+uy*uy))
	if uy < 0.0 {
		theta = -theta
	}
	theta = angleNorm(theta)

	deltaAcos := (ux*vx + uy*vy) / math.Sqrt((ux*ux+uy*uy)*(vx*vx+vy*vy))
	deltaAcos = math.Min(1.0, math.Max(-1.0, deltaAcos))
	delta := math.Acos(deltaAcos)
	if ux*vy-uy*vx < 0.0 {
		delta = -delta
	}
	if !sweep && delta > 0.0 { // clockwise in Cartesian
		delta -= 2.0 * math.Pi
	} else if sweep && delta < 0.0 { // counter clockwise in Cartesian
		delta += 2.0 * math.Pi
	}
	return cx, cy, theta, theta + delta
}

// scale ellipse if rx and ry are too small, see https://www.w3.org/TR/SVG/implnote.html#ArcCorrectionOutOfRangeRadii
func ellipseRadiiCorrection(start Point, rx, ry, phi float64, end Point) float64 {
	diff := start.Sub(end)
	sinphi, cosphi := math.Sincos(phi)
	x1p := (cosphi*diff.X + sinphi*diff.Y) / 2.0
	y1p := (-sinphi*diff.X + cosphi*diff.Y) / 2.0
	return math.Sqrt(x1p*x1p/rx/rx + y1p*y1p/ry/ry)
}

// splitEllipse returns the new mid point, the two largeArc parameters and the ok bool, the rest stays the same
func splitEllipse(rx, ry, phi, cx, cy, theta1, theta2, theta float64) (Point, bool, bool, bool) {
	if !angleBetween(theta, theta1, theta2) {
		return Point{}, false, false, false
	}

	mid := ellipsePos(rx, ry, phi, cx, cy, theta)
	largeArc0, largeArc1 := false, false
	if math.Abs(theta-theta1) > math.Pi {
		largeArc0 = true
	} else if math.Abs(theta-theta2) > math.Pi {
		largeArc1 = true
	}
	return mid, largeArc0, largeArc1, true
}

// see Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe,
// https://spaceroots.org/documents/ellipse/elliptical-arc.pdf
func ellipseToBeziers(start Point, rx, ry, phi float64, largeArc, sweep bool, end Point) *Path {
	p := &Path{}
	cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, largeArc, sweep, end.X, end.Y)

	dtheta := math.Pi / 2.0
	n := int(math.Ceil(math.Abs(theta1-theta0) / dtheta))
	dtheta = math.Abs(theta1-theta0) / float64(n) // evenly spread the n points, dalpha will get smaller
	kappa := math.Sin(dtheta) * (math.Sqrt(4.0+3.0*math.Pow(math.Tan(dtheta/2.0), 2.0)) - 1.0) / 3.0
	if !sweep {
		dtheta = -dtheta
	}

	p.MoveTo(start.X, start.Y)
	startDeriv := ellipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float64(i)*dtheta
		end := ellipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := ellipseDeriv(rx, ry, phi, sweep, theta)

		cp1 := start.Add(startDeriv.Mul(kappa))
		cp2 := end.Sub(endDeriv.Mul(kappa))
		p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y)
		startDeriv = endDeriv
		start = end
	}
	return p
}

func flattenEllipse(start Point, rx, ry, phi float64, largeArc, sweep bool, end Point) *Path {
	return ellipseToBeziers(start, rx, ry, phi, largeArc, sweep, end).Flatten()
}

////////////////////////////////////////////////////////////////
// Béziers /////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

func quadraticToCubicBezier(start, c, end Point) (Point, Point) {
	c1 := start.Interpolate(c, 2.0/3.0)
	c2 := end.Interpolate(c, 2.0/3.0)
	return c1, c2
}

func quadraticBezierPos(p0, p1, p2 Point, t float64) Point {
	p0 = p0.Mul(1.0 - 2.0*t + t*t)
	p1 = p1.Mul(2.0*t - 2.0*t*t)
	p2 = p2.Mul(t * t)
	return p0.Add(p1).Add(p2)
}

func quadraticBezierDeriv(p0, p1, p2 Point, t float64) Point {
	p0 = p0.Mul(-2.0 + 2.0*t)
	p1 = p1.Mul(2.0 - 4.0*t)
	p2 = p2.Mul(2.0 * t)
	return p0.Add(p1).Add(p2)
}

// see https://malczak.linuxpl.com/blog/quadratic-bezier-curve-length/
func quadraticBezierLength(p0, p1, p2 Point) float64 {
	a := p0.Sub(p1.Mul(2.0)).Add(p2)
	b := p1.Mul(2.0).Sub(p0.Mul(2.0))
	A := 4.0 * a.Dot(a)
	B := 4.0 * a.Dot(b)
	C := b.Dot(b)
	if equal(A, 0.0) {
		return 0.0
	}

	Sabc := 2.0 * math.Sqrt(A+B+C)
	A_2 := math.Sqrt(A)
	A_32 := 2.0 * A * A_2
	C_2 := 2.0 * math.Sqrt(C)
	BA := B / A_2
	return (A_32*Sabc + A_2*B*(Sabc-C_2) + (4.0*C*A-B*B)*math.Log((2.0*A_2+BA+Sabc)/(BA+C_2))) / (4.0 * A_32)
}

func splitQuadraticBezier(p0, p1, p2 Point, t float64) (Point, Point, Point, Point, Point, Point) {
	q0 := p0
	q1 := p0.Interpolate(p1, t)

	r2 := p2
	r1 := p1.Interpolate(p2, t)

	r0 := q1.Interpolate(r1, t)
	q2 := r0
	return q0, q1, q2, r0, r1, r2
}

func cubicBezierPos(p0, p1, p2, p3 Point, t float64) Point {
	p0 = p0.Mul(1.0 - 3.0*t + 3.0*t*t - t*t*t)
	p1 = p1.Mul(3.0*t - 6.0*t*t + 3.0*t*t*t)
	p2 = p2.Mul(3.0*t*t - 3.0*t*t*t)
	p3 = p3.Mul(t * t * t)
	return p0.Add(p1).Add(p2).Add(p3)
}

func cubicBezierDeriv(p0, p1, p2, p3 Point, t float64) Point {
	p0 = p0.Mul(-3.0 + 6.0*t - 3.0*t*t)
	p1 = p1.Mul(3.0 - 12.0*t + 9.0*t*t)
	p2 = p2.Mul(6.0*t - 9.0*t*t)
	p3 = p3.Mul(3.0 * t * t)
	return p0.Add(p1).Add(p2).Add(p3)
}

func cubicBezierDeriv2(p0, p1, p2, p3 Point, t float64) Point {
	p0 = p0.Mul(6.0 - 6.0*t)
	p1 = p1.Mul(18.0*t - 12.0)
	p2 = p2.Mul(6.0 - 18.0*t)
	p3 = p3.Mul(6.0 * t)
	return p0.Add(p1).Add(p2).Add(p3)
}

// negative when curve bends CW while following t
func cubicBezierCurvatureRadius(p0, p1, p2, p3 Point, t float64) float64 {
	dp := cubicBezierDeriv(p0, p1, p2, p3, t)
	ddp := cubicBezierDeriv2(p0, p1, p2, p3, t)
	a := dp.PerpDot(ddp) // negative when bending right ie. curve is CW at this point
	if equal(a, 0.0) {
		return math.NaN()
	}
	return math.Pow(dp.X*dp.X+dp.Y*dp.Y, 1.5) / a
}

// return the normal at the right-side of the curve (when increasing t)
func cubicBezierNormal(p0, p1, p2, p3 Point, t, d float64) Point {
	if t == 0.0 {
		n := p1.Sub(p0)
		if n.X == 0 && n.Y == 0 {
			n = p2.Sub(p0)
		}
		if n.X == 0 && n.Y == 0 {
			n = p3.Sub(p0)
		}
		if n.X == 0 && n.Y == 0 {
			return Point{}
		}
		return n.Rot90CW().Norm(d)
	} else if t == 1.0 {
		n := p3.Sub(p2)
		if n.X == 0 && n.Y == 0 {
			n = p3.Sub(p1)
		}
		if n.X == 0 && n.Y == 0 {
			n = p3.Sub(p0)
		}
		if n.X == 0 && n.Y == 0 {
			return Point{}
		}
		return n.Rot90CW().Norm(d)
	}
	panic("not implemented")
}

// cubicBezierLength calculates the length of the Bézier, taking care of inflection points
// it uses Gauss-Legendre (n=5) and has an error of ~1% or less (emperical)
func cubicBezierLength(p0, p1, p2, p3 Point) float64 {
	length := 0.0
	beziers := splitCubicBezierAtInflections(p0, p1, p2, p3)
	for _, bezier := range beziers {
		speed := func(t float64) float64 {
			return cubicBezierDeriv(bezier[0], bezier[1], bezier[2], bezier[3], t).Length()
		}
		length += gaussLegendre5(speed, 0.0, 1.0)
	}
	return length
}

func splitCubicBezierAtInflections(p0, p1, p2, p3 Point) [][4]Point {
	t1, t2 := findInflectionPointsCubicBezier(p0, p1, p2, p3)
	var beziers [][4]Point
	if t1 > 0.0 && t1 < 1.0 && t2 > 0.0 && t2 < 1.0 {
		p0, p1, p2, p3, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, t1)
		t2 = (t2 - t1) / (1.0 - t1)
		q0, q1, q2, q3, r0, r1, r2, r3 := splitCubicBezier(q0, q1, q2, q3, t2)
		beziers = append(beziers, [4]Point{p0, p1, p2, p3})
		beziers = append(beziers, [4]Point{q0, q1, q2, q3})
		beziers = append(beziers, [4]Point{r0, r1, r2, r3})
	} else if t1 > 0.0 && t1 < 1.0 || t2 > 0.0 && t2 < 1.0 {
		t := t1
		if t2 > 0.0 && t2 < 1.0 {
			t = t2
		}
		p0, p1, p2, p3, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, t)
		beziers = append(beziers, [4]Point{p0, p1, p2, p3})
		beziers = append(beziers, [4]Point{q0, q1, q2, q3})
	} else {
		beziers = append(beziers, [4]Point{p0, p1, p2, p3})
	}
	return beziers
}

func splitCubicBezier(p0, p1, p2, p3 Point, t float64) (Point, Point, Point, Point, Point, Point, Point, Point) {
	pm := p1.Interpolate(p2, t)

	q0 := p0
	q1 := p0.Interpolate(p1, t)
	q2 := q1.Interpolate(pm, t)

	r3 := p3
	r2 := p2.Interpolate(p3, t)
	r1 := pm.Interpolate(r2, t)

	r0 := q2.Interpolate(r1, t)
	q3 := r0
	return q0, q1, q2, q3, r0, r1, r2, r3
}

func addCubicBezierLine(p *Path, p0, p1, p2, p3 Point, t, d float64) {
	if p0.X == p3.X && p0.Y == p3.X && (p0.X == p1.X && p0.Y == p1.Y || p0.X == p2.X && p0.Y == p2.Y) {
		// Bézier has p0=p1=p3 or p0=p2=p3 and thus has no surface
		return
	}

	pos := Point{}
	if t == 0.0 {
		// line to beginning of path
		pos = p0
		if d != 0.0 {
			n := cubicBezierNormal(p0, p1, p2, p3, t, d)
			pos = pos.Add(n)
		}
	} else if t == 1.0 {
		// line to the end of the path
		pos = p3
		if d != 0.0 {
			n := cubicBezierNormal(p0, p1, p2, p3, t, d)
			pos = pos.Add(n)
		}
	} else {
		panic("not implemented")
	}
	p.LineTo(pos.X, pos.Y)
}

// split the curve and replace it by lines as long as maximum deviation = flatness is maintained
func flattenSmoothCubicBezier(p *Path, p0, p1, p2, p3 Point, d, flatness float64) {
	t := 0.0
	for t < 1.0 {
		s2nom := (p2.X-p0.X)*(p1.Y-p0.Y) - (p2.Y-p0.Y)*(p1.X-p0.X)
		denom := math.Hypot(p1.X-p0.X, p1.Y-p0.Y)
		if s2nom*denom == 0.0 {
			break
		}

		s2 := s2nom / denom
		r1 := denom
		effectiveFlatness := flatness / math.Abs(1.0+2.0*d*s2/3.0/r1/r1)
		t = 2.0 * math.Sqrt(effectiveFlatness/3.0/math.Abs(s2))
		if t >= 1.0 {
			break
		}
		_, _, _, _, p0, p1, p2, p3 = splitCubicBezier(p0, p1, p2, p3, t)
		addCubicBezierLine(p, p0, p1, p2, p3, 0.0, d)
	}
	addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
}

func findInflectionPointsCubicBezier(p0, p1, p2, p3 Point) (float64, float64) {
	// we omit multiplying bx,by,cx,cy with 3.0, so there is no need for divisions when calculating a,b,c
	ax := -p0.X + 3.0*p1.X - 3.0*p2.X + p3.X
	ay := -p0.Y + 3.0*p1.Y - 3.0*p2.Y + p3.Y
	bx := p0.X - 2.0*p1.X + p2.X
	by := p0.Y - 2.0*p1.Y + p2.Y
	cx := -p0.X + p1.X
	cy := -p0.Y + p1.Y

	a := (ay*bx - ax*by)
	b := (ay*cx - ax*cy)
	c := (by*cx - bx*cy)
	x1, x2 := solveQuadraticFormula(a, b, c)
	if 1.0 <= x1 || x1 < 0.0 {
		x1 = math.NaN()
	}
	if 1.0 <= x2 || x2 < 0.0 {
		x2 = math.NaN()
	} else if math.IsNaN(x1) {
		x1, x2 = x2, x1
	}
	return x1, x2
}

func findInflectionPointRange(p0, p1, p2, p3 Point, t, flatness float64) (float64, float64) {
	if math.IsNaN(t) {
		return math.Inf(1), math.Inf(1)
	}
	if t < 0.0 || t > 1.0 {
		panic("t outside 0.0--1.0 range")
	}

	// we state that s(t) = 3*s2*t^2 + (s3 - 3*s2)*t^3 (see paper on the r-s coordinate system)
	// with s(t) aligned perpendicular to the curve at t = 0
	// then we impose that s(tf) = flatness and find tf
	// at inflection points however, s2 = 0, so that s(t) = s3*t^3

	if t != 0.0 {
		_, _, _, _, p0, p1, p2, p3 = splitCubicBezier(p0, p1, p2, p3, t)
	}
	nr := p1.Sub(p0)
	ns := p3.Sub(p0)
	if nr.X == 0.0 && nr.Y == 0.0 {
		// if p0=p1, then rn (the velocity at t=0) needs adjustment
		// nr = lim[t->0](B'(t)) = 3*(p1-p0) + 6*t*((p1-p0)+(p2-p1)) + second order terms of t
		// if (p1-p0)->0, we use (p2-p1)=(p2-p0)
		nr = p2.Sub(p0)
	}

	if nr.X == 0.0 && nr.Y == 0.0 {
		// if rn is still zero, this curve has p0=p1=p2, so it is straight
		return 0.0, 1.0
	}

	s3 := math.Abs(ns.X*nr.Y-ns.Y*nr.X) / math.Hypot(nr.X, nr.Y)
	if s3 == 0.0 {
		return 0.0, 1.0 // can approximate whole curve linearly
	}

	tf := math.Cbrt(flatness / s3)
	return t - tf*(1.0-t), t + tf*(1.0-t)
}

func flattenCubicBezier(p0, p1, p2, p3 Point) *Path {
	return strokeCubicBezier(p0, p1, p2, p3, 0.0, Tolerance)
}

// see Flat, precise flattening of cubic Bézier path and offset curves, by T.F. Hain et al., 2005
// https://www.sciencedirect.com/science/article/pii/S0097849305001287
// see https://github.com/Manishearth/stylo-flat/blob/master/gfx/2d/Path.cpp for an example implementation
// or https://docs.rs/crate/lyon_bezier/0.4.1/source/src/flatten_cubic.rs
// p0, p1, p2, p3 are the start points, two control points and the end points respectively. With flatness defined as
// the maximum error from the orinal curve, and d the half width of the curve used for stroking (positive is to the right).
// TODO: use ellipse arcs for better results?
func strokeCubicBezier(p0, p1, p2, p3 Point, d, flatness float64) *Path {
	p := &Path{}
	// 0 <= t1 <= 1 if t1 exists
	// 0 <= t2 <= 1 and t1 < t2 if t2 exists
	t1, t2 := findInflectionPointsCubicBezier(p0, p1, p2, p3)
	if math.IsNaN(t1) && math.IsNaN(t2) {
		// There are no inflection points or cusps, approximate linearly by subdivision.
		flattenSmoothCubicBezier(p, p0, p1, p2, p3, d, flatness)
		return p
	}

	// t1min <= t1max; with t1min <= 1 and t2max >= 0
	// t2min <= t2max; with t2min <= 1 and t2max >= 0
	t1min, t1max := findInflectionPointRange(p0, p1, p2, p3, t1, flatness)
	t2min, t2max := findInflectionPointRange(p0, p1, p2, p3, t2, flatness)

	if math.IsNaN(t2) && t1min <= 0.0 && 1.0 <= t1max {
		// There is no second inflection point, and the first inflection point can be entirely approximated linearly.
		addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
		return p
	}

	if 0.0 < t1min {
		// Flatten up to t1min
		q0, q1, q2, q3, _, _, _, _ := splitCubicBezier(p0, p1, p2, p3, t1min)
		flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
	}

	if 0.0 < t1max && t1max < 1.0 && t1max < t2min {
		// t1 and t2 ranges do not overlap, approximate t1 linearly
		_, _, _, _, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, t1max)
		addCubicBezierLine(p, q0, q1, q2, q3, 0.0, d)
		if 1.0 <= t2min {
			// No t2 present, approximate the rest linearly by subdivision
			flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
			return p
		}
	} else if 1.0 <= t2min {
		// t1 and t2 overlap but past the curve, approximate linearly
		addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
		return p
	}

	// t1 and t2 exist and ranges might overlap
	if 0.0 < t2min {
		if t2min < t1max {
			// t2 range starts inside t1 range, approximate t1 range linearly
			_, _, _, _, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, t1max)
			addCubicBezierLine(p, q0, q1, q2, q3, 0.0, d)
		} else if 0.0 < t1max {
			// no overlap
			_, _, _, _, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, t1max)
			t2minq := (t2min - t1max) / (1 - t1max)
			q0, q1, q2, q3, _, _, _, _ = splitCubicBezier(q0, q1, q2, q3, t2minq)
			flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
		} else {
			// no t1, approximate up to t2min linearly by subdivision
			q0, q1, q2, q3, _, _, _, _ := splitCubicBezier(p0, p1, p2, p3, t2min)
			flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
		}
	}

	// handle (the rest of) t2
	if t2max < 1.0 {
		_, _, _, _, q0, q1, q2, q3 := splitCubicBezier(p0, p1, p2, p3, t2max)
		addCubicBezierLine(p, q0, q1, q2, q3, 0.0, d)
		flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
	} else {
		// t2max extends beyond 1
		addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
	}
	return p
}
