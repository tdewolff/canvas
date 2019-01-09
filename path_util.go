package canvas

import (
	"math"
)

// ellipseToEndpoints converts to the endpoint arc format and returns (startX, startY, largeArc, sweep, endX, endY)
// see https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
func ellipseToEndpoint(cx, cy, rx, ry, rot, theta1, theta2 float64) (float64, float64, bool, bool, float64, float64) {
	x1 := math.Cos(rot)*rx*math.Cos(theta1) - math.Sin(rot)*ry*math.Sin(theta1) + cx
	y1 := math.Sin(rot)*rx*math.Cos(theta1) + math.Cos(rot)*ry*math.Sin(theta1) + cy
	x2 := math.Cos(rot)*rx*math.Cos(theta2) - math.Sin(rot)*ry*math.Sin(theta2) + cx
	y2 := math.Sin(rot)*rx*math.Cos(theta2) + math.Cos(rot)*ry*math.Sin(theta2) + cy
	largeArc := math.Abs(theta2-theta1) > 180.0
	sweep := (theta2 - theta1) > 0.0
	return x1, y1, largeArc, sweep, x2, y2
}

// ellipseToCenter converts to the center arc format and returns (centerX, centerY, angleFrom, angleTo)
// see https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
// TODO: return in radians?
func ellipseToCenter(x1, y1, rx, ry, rot float64, large, sweep bool, x2, y2 float64) (float64, float64, float64, float64) {
	if x1 == x2 && y1 == y2 {
		return x1, y1, 0.0, 0.0
	}

	rot *= math.Pi / 180.0
	x1p := math.Cos(rot)*(x1-x2)/2.0 + math.Sin(rot)*(y1-y2)/2.0
	y1p := -math.Sin(rot)*(x1-x2)/2.0 + math.Cos(rot)*(y1-y2)/2.0

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
	cx := math.Cos(rot)*cxp - math.Sin(rot)*cyp + (x1+x2)/2.0
	cy := math.Sin(rot)*cxp + math.Cos(rot)*cyp + (y1+y2)/2.0

	// specify U and V vectors; theta = arccos(U*V / sqrt(U*U + V*V))
	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := -(x1p + cxp) / rx
	vy := -(y1p + cyp) / ry

	theta := math.Acos(ux / math.Sqrt(ux*ux+uy*uy))
	if uy < 0.0 {
		theta = -theta
	}
	theta *= 180.0 / math.Pi

	delta := math.Acos((ux*vx + uy*vy) / math.Sqrt((ux*ux+uy*uy)*(vx*vx+vy*vy)))
	if ux*vy-uy*vx < 0.0 {
		delta = -delta
	}
	delta *= 180.0 / math.Pi
	if !sweep && delta > 0.0 {
		delta -= 360.0
	} else if sweep && delta < 0.0 {
		delta += 360.0
	}
	return cx, cy, theta, theta + delta
}

func splitEllipse(start Point, rx, ry, rot float64, largeArc, sweep bool, end Point) (Point, bool, bool, bool, bool) {
	panic("not implemented")
}

func ellipseNormal(theta float64) Point {
	theta *= math.Pi / 180.0
	y, x := math.Sincos(theta)
	return Point{x, y}
}

////////////////////////////////////////////////////////////////

// Gauss-Legendre quadrature integration from a to b with n=3
func gaussLegendre3(f func(float64) float64, a, b float64) float64 {
	c := (b - a) / 2.0
	d := (a + b) / 2.0
	Qd1 := f(-0.774596669*c + d)
	Qd2 := f(d)
	Qd3 := f(0.774596669*c + d)
	return c * ((5.0/9.0)*(Qd1+Qd3) + (8.0/9.0)*Qd2)
}

// Gauss-Legendre quadrature integration from a to b with n=5
func gaussLegendre5(f func(float64) float64, a, b float64) float64 {
	c := (b - a) / 2.0
	d := (a + b) / 2.0
	Qd1 := f(-0.90618*c + d)
	Qd2 := f(-0.538469*c + d)
	Qd3 := f(d)
	Qd4 := f(0.538469*c + d)
	Qd5 := f(0.90618*c + d)
	return c * (0.236927*(Qd1+Qd5) + 0.478629*(Qd2+Qd4) + 0.568889*Qd3)
}

// find parametric value t at a given length s on the curve using the bisection method
func bisectionMethod(f func(float64) float64, s float64) float64 {
	tmin, tmax := 0.0, 1.0
	for {
		t := (tmin + tmax) / 2.0
		ds := f(t) - s
		if math.Abs(ds) < 0.1 || (tmax-tmin)/2.0 < 0.1 {
			return t
		} else if ds > 0.0 {
			tmax = t
		} else {
			tmin = t
		}
	}
}

func cubicBezierSpeedAt(p0, p1, p2, p3 Point, t float64) float64 {
	p0 = p0.Mul(-3.0 + 6.0*t - 3.0*t*t)
	p1 = p1.Mul(3.0 - 12.0*t + 9.0*t*t)
	p2 = p2.Mul(6.0*t - 9.0*t*t)
	p3 = p3.Mul(3.0 * t * t)
	return p0.Add(p1).Add(p2).Add(p3).Length()
}

func cubicBezierLength(p0, p1, p2, p3 Point) float64 {
	// TODO: split at inflection points
	deriv := func(t float64) float64 { return cubicBezierSpeedAt(p0, p1, p2, p3, t) }
	return gaussLegendre3(deriv, 0.0, 1.0)
}

// cubicBezierLength returns a function that maps t=[0,1] to its lengths L(t)
// implemented using M. Walter, A. Fournier, Approximate Arc Length Parametrization, Anais do IX SIBGRAPHI, p. 143--150, 1996
// see https://www.visgraf.impa.br/sibgrapi96/trabs/pdf/a14.pdf
func cubicBezierLengthFunc(p0, p1, p2, p3 Point) func(float64) float64 {
	// TODO: split at inflection points
	speed := func(t float64) float64 { return cubicBezierSpeedAt(p0, p1, p2, p3, t) }
	s1 := gaussLegendre3(speed, 0.0, 1.0/3.0)
	s2 := gaussLegendre3(speed, 0.0, 2.0/3.0)
	s3 := gaussLegendre3(speed, 0.0, 1.0)

	// We have three points on the s(t) curve at t0=0, t1=1/3, t2=2/3 and t3=1
	// now obtain a polynomial that goes through these four points by solving the system of linear equations
	// s(t) = a*t^3 + b*t^2 + c*t + d  (s0=0; d=0)
	// [s1; s2; s3] = [1/27, 1/9, 1/3;
	//                 8/27, 4/9, 2/3;
	//                    1,   1,   1] * [a; b; c]
	//
	// After inverting (note that d=0):
	// [a; b; c] = 0.5 * [ 27, -27,  9;
	//                    -45,  36, -9;
	//                     18,  -9,  2] * [s1; s2; s3]

	a := 13.5*s1 - 13.5*s2 + 4.5*s3
	b := -22.5*s1 + 18.0*s2 - 4.5*s3
	c := 9.0*s1 - 4.5*s2 + s3
	return func(t float64) float64 {
		return a*t*t*t + b*t*t + c*t
	}
}

func cubicBezierInverseLengthFunc(p0, p1, p2, p3 Point) func(float64) float64 {
	// TODO: split at inflection points
	speed := func(t float64) float64 { return cubicBezierSpeedAt(p0, p1, p2, p3, t) }
	gaussLegendre := func(s float64) float64 { return gaussLegendre3(speed, 0.0, s) }
	s3 := gaussLegendre(1.0)
	t1 := bisectionMethod(gaussLegendre, (1.0/3.0)*s3)
	t2 := bisectionMethod(gaussLegendre, (2.0/3.0)*s3)

	div := 1.0 / (s3 * s3 * s3)
	a := div * (13.5*t1 - 13.5*t2 + 4.5)
	b := div * (-22.5*t1*s3 + 18.0*t2*s3 - 4.5*s3)
	c := div * (9.0*t1*s3*s3 - 4.5*t2*s3*s3 + 1.0*s3*s3)

	return func(s float64) float64 {
		return a*s*s*s + b*s*s + c*s
	}
}

func ellipseSpeedAt(rx, ry, phi float64) float64 {
	dx := rx * math.Cos(phi)
	dy := -ry * math.Sin(phi)
	return math.Sqrt(dx*dx + dy*dy)
}

func ellipseLength(start Point, rx, ry, rot float64, largeArc, sweep bool, end Point) float64 {
	_, _, angle0, angle1 := ellipseToCenter(start.X, start.Y, rx, ry, rot, largeArc, sweep, end.X, end.Y)
	phi0 := angle0 * math.Pi / 180.0
	phi1 := angle1 * math.Pi / 180.0
	if phi1 < phi0 {
		phi0, phi1 = phi1, phi0
	}

	speed := func(phi float64) float64 { return ellipseSpeedAt(rx, ry, phi) }
	return gaussLegendre5(speed, phi0, phi1)
}

////////////////////////////////////////////////////////////////

func quadraticToCubicBezier(start, c, end Point) (Point, Point) {
	c1 := start.Interpolate(c, 2.0/3.0)
	c2 := end.Interpolate(c, 2.0/3.0)
	return c1, c2
}

func cubicBezierNormal(p0, p1, p2, p3 Point, t float64) Point {
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
		return n.Rot90CW()
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
		return n.Rot90CW()
	}
	panic("not implemented")
}

func addCubicBezierLine(p *Path, p0, p1, p2, p3 Point, t, d float64) {
	if p0.X == p3.X && p0.Y == p3.X && (p0.X == p1.X && p0.Y == p1.Y || p0.X == p2.X && p0.Y == p2.Y) {
		// Bezier has p0=p1=p3 or p0=p2=p3 and thus has no surface
		return
	}

	pos := Point{}
	if t == 0.0 {
		// line to beginning of path
		pos = p0
		if d != 0.0 {
			n := cubicBezierNormal(p0, p1, p2, p3, t)
			pos = pos.Add(n.Norm(d))
		}
	} else if t == 1.0 {
		// line to the end of the path
		pos = p3
		if d != 0.0 {
			n := cubicBezierNormal(p0, p1, p2, p3, t)
			pos = pos.Add(n.Norm(d))
		}
	} else {
		panic("not implemented")
	}
	p.LineTo(pos.X, pos.Y)
}

func quadraticBezierAt(p0, p1, p2 Point, t float64) Point {
	p0 = p0.Mul((1 - t) * (1 - t))
	p1 = p1.Mul(2 * t * (1 - t))
	p2 = p2.Mul(t * t)
	return p0.Add(p1).Add(p2)
}

func cubicBezierAt(p0, p1, p2, p3 Point, t float64) Point {
	p0 = p0.Mul((1 - t) * (1 - t) * (1 - t))
	p1 = p1.Mul(3 * t * (1 - t) * (1 - t))
	p2 = p2.Mul(3 * t * t * (1 - t))
	p3 = p3.Mul(t * t * t)
	return p0.Add(p1).Add(p2).Add(p3)
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

func solveQuadraticFormula(a, b, c float64) (float64, float64) {
	// Numerically stable quadratic formula
	// see https://math.stackexchange.com/a/2007723
	if a == 0.0 {
		if b == 0.0 {
			if c == 0.0 {
				// all terms disappear, all x satisfy the solution
				return 0.0, math.NaN()
			}
			// linear term disappears, no solutions
			return math.NaN(), math.NaN()
		}
		// quadratic term disappears, solve linear equation
		x1 := -c / b
		if 1.0 <= x1 || x1 < 0.0 {
			x1 = math.NaN()
		}
		return x1, math.NaN()
	}

	if c == 0.0 {
		// no constant term, one solution at zero and one from solving linearly
		x2 := -b / a
		if 1.0 <= x2 || x2 < 0.0 {
			x2 = math.NaN()
		}
		return 0.0, x2
	}

	discriminant := b*b - 4.0*a*c
	if discriminant < 0.0 {
		return math.NaN(), math.NaN()
	} else if discriminant == 0.0 {
		x1 := -b / (2.0 * a)
		if 1.0 <= x1 || x1 < 0.0 {
			x1 = math.NaN()
		}
		return x1, math.NaN()
	}

	// Avoid catastrophic cancellation, which occurs when we subtract two nearly equal numbers and causes a large error
	// this can be the case when 4*a*c is small so that sqrt(discriminant) -> b, and the sign of b and in front of the radical are the same
	// instead we calculate x where b and the radical have different signs, and then use this result in the analytical equivalent
	// of the formula, called the Citardauq Formula.
	q := math.Sqrt(discriminant)
	if b < 0.0 {
		// apply sign of b
		q = -q
	}
	x1 := -(b + q) / (2.0 * a)
	x2 := c / (a * x1)
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if 1.0 <= x1 || x1 < 0.0 {
		x1 = math.NaN()
	}
	if 1.0 <= x2 || x2 < 0.0 {
		x2 = math.NaN()
	}
	return x1, x2
}

func findInflectionPointsCubicBezier(p0, p1, p2, p3 Point) (float64, float64) {
	// We omit multiplying bx,by,cx,cy with 3.0, so there is no need for divisions when calculating a,b,c
	ax := -p0.X + 3.0*p1.X - 3.0*p2.X + p3.X
	ay := -p0.Y + 3.0*p1.Y - 3.0*p2.Y + p3.Y
	bx := p0.X - 2.0*p1.X + p2.X
	by := p0.Y - 2.0*p1.Y + p2.Y
	cx := -p0.X + p1.X
	cy := -p0.Y + p1.Y

	a := (ay*bx - ax*by)
	b := (ay*cx - ax*cy)
	c := (by*cx - bx*cy)
	return solveQuadraticFormula(a, b, c)
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

// see Flat, precise flattening of cubic Bezier path and offset curves, by T.F. Hain et al., 2005
// https://www.sciencedirect.com/science/article/pii/S0097849305001287
// see https://github.com/Manishearth/stylo-flat/blob/master/gfx/2d/Path.cpp for an example implementation
// or https://docs.rs/crate/lyon_bezier/0.4.1/source/src/flatten_cubic.rs
// p0, p1, p2, p3 are the start points, two control points and the end points respectively. With flatness defined as
// the maximum error from the orinal curve, and d the half width of the curve used for stroking (positive is to the right).
// TODO: use ellipse arcs for better results
func flattenCubicBezier(p0, p1, p2, p3 Point, d, flatness float64) *Path {
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

func flattenEllipse(start Point, rx, ry, rot float64, largeArc, sweep bool, end Point, tolerance float64) *Path {
	panic("not implemented")
}
