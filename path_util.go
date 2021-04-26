package canvas

import (
	"math"
)

// EllipsePos returns the position on the ellipse at angle theta.
func EllipsePos(rx, ry, phi, cx, cy, theta float64) Point {
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

func ellipseDeriv2(rx, ry, phi float64, theta float64) Point {
	sintheta, costheta := math.Sincos(theta)
	sinphi, cosphi := math.Sincos(phi)
	ddx := -rx*costheta*cosphi + ry*sintheta*sinphi
	ddy := -rx*costheta*sinphi - ry*sintheta*cosphi
	return Point{ddx, ddy}
}

func ellipseCurvatureRadius(rx, ry float64, sweep bool, theta float64) float64 {
	// phi has no influence on the curvature
	dp := ellipseDeriv(rx, ry, 0.0, sweep, theta)
	ddp := ellipseDeriv2(rx, ry, 0.0, theta)
	a := dp.PerpDot(ddp)
	if Equal(a, 0.0) {
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

// ellipseToCenter converts to the center arc format and returns (centerX, centerY, angleFrom, angleTo) with angles in radians. When angleFrom with range [0, 2*PI) is bigger than angleTo with range (-2*PI, 4*PI), the ellipse runs clockwise. The angles are from before the ellipse has been stretched and rotated. See https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
func ellipseToCenter(x1, y1, rx, ry, phi float64, large, sweep bool, x2, y2 float64) (float64, float64, float64, float64) {
	if Equal(x1, x2) && Equal(y1, y2) {
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

// ellipseSplit returns the new mid point, the two large parameters and the ok bool, the rest stays the same
func ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, theta float64) (Point, bool, bool, bool) {
	if !angleBetween(theta, theta0, theta1) {
		return Point{}, false, false, false
	}

	mid := EllipsePos(rx, ry, phi, cx, cy, theta)
	large0, large1 := false, false
	if math.Abs(theta-theta0) > math.Pi {
		large0 = true
	} else if math.Abs(theta-theta1) > math.Pi {
		large1 = true
	}
	return mid, large0, large1, true
}

func arcToQuad(start Point, rx, ry, phi float64, large, sweep bool, end Point) *Path {
	p := &Path{}
	p.MoveTo(start.X, start.Y)
	for _, bezier := range ellipseToQuadraticBeziers(start, rx, ry, phi, large, sweep, end) {
		p.QuadTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y)
	}
	return p
}

func arcToCube(start Point, rx, ry, phi float64, large, sweep bool, end Point) *Path {
	p := &Path{}
	p.MoveTo(start.X, start.Y)
	for _, bezier := range ellipseToCubicBeziers(start, rx, ry, phi, large, sweep, end) {
		p.CubeTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y, bezier[3].X, bezier[3].Y)
	}
	return p
}

//func ellipseToQuadraticBezierError(a, b, n1, n2 float64) float64 {
//	if a < b {
//		a, b = b, a
//	}
//	ba := b / a
//	c0, c1 := 0.0, 0.0
//	if ba < 0.25 {
//		c0 += ((3.92478*ba*ba - 13.5822*ba - 0.233377) / (ba + 0.0128206))
//		c0 += ((-1.08814*ba*ba + 0.859987*ba + 0.000362265) / (ba + 0.000229036)) * math.Cos(1.0*(n1+n2))
//		c0 += ((-0.942512*ba*ba + 0.390456*ba + 0.0080909) / (ba + 0.00723895)) * math.Cos(2.0*(n1+n2))
//		c0 += ((-0.736228*ba*ba + 0.20998*ba + 0.0129867) / (ba + 0.0103456)) * math.Cos(3.0*(n1+n2))
//		c1 += ((-0.395018*ba*ba + 6.82464*ba + 0.0995293) / (ba + 0.0122198))
//		c1 += ((-0.545608*ba*ba + 0.0774863*ba + 0.0267327) / (ba + 0.0132482)) * math.Cos(1.0*(n1+n2))
//		c1 += ((0.0534754*ba*ba - 0.0884167*ba + 0.012595) / (ba + 0.0343396)) * math.Cos(2.0*(n1+n2))
//		c1 += ((0.209052*ba*ba - 0.0599987*ba - 0.00723897) / (ba + 0.00789976)) * math.Cos(3.0*(n1+n2))
//	} else {
//		c0 += ((0.0863805*ba*ba - 11.5595*ba - 2.68765) / (ba + 0.181224))
//		c0 += ((0.242856*ba*ba - 1.81073*ba + 1.56876) / (ba + 1.68544)) * math.Cos(1.0*(n1+n2))
//		c0 += ((0.233337*ba*ba - 0.455621*ba + 0.222856) / (ba + 0.403469)) * math.Cos(2.0*(n1+n2))
//		c0 += ((0.0612978*ba*ba - 0.104879*ba + 0.0446799) / (ba + 0.00867312)) * math.Cos(3.0*(n1+n2))
//		c1 += ((0.028973*ba*ba + 6.68407*ba + 0.171472) / (ba + 0.0211706))
//		c1 += ((0.0307674*ba*ba - 0.0517815*ba + 0.0216803) / (ba - 0.0749348)) * math.Cos(1.0*(n1+n2))
//		c1 += ((-0.0471179*ba*ba + 0.1288*ba - 0.0781702) / (ba + 2.0)) * math.Cos(2.0*(n1+n2))
//		c1 += ((-0.0309683*ba*ba + 0.0531557*ba - 0.0227191) / (ba + 0.0434511)) * math.Cos(3.0*(n1+n2))
//	}
//	return ((0.02*ba*ba + 2.83*ba + 0.125) / (ba + 0.01)) * a * math.Exp(c0+c1*math.Abs(n2-n1))
//}

// see Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe, https://spaceroots.org/documents/ellipse/elliptical-arc.pdf
func ellipseToQuadraticBeziers(start Point, rx, ry, phi float64, large, sweep bool, end Point) [][3]Point {
	cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

	dtheta := math.Pi / 2.0 // TODO: use error measure to determine dtheta?
	n := int(math.Ceil(math.Abs(theta1-theta0) / dtheta))
	dtheta = math.Abs(theta1-theta0) / float64(n) // evenly spread the n points, dalpha will get smaller
	kappa := math.Tan(dtheta / 2.0)
	if !sweep {
		dtheta = -dtheta
	}

	beziers := [][3]Point{}
	startDeriv := ellipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float64(i)*dtheta
		end := EllipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := ellipseDeriv(rx, ry, phi, sweep, theta)

		cp := start.Add(startDeriv.Mul(kappa))
		beziers = append(beziers, [3]Point{start, cp, end})

		startDeriv = endDeriv
		start = end
	}
	return beziers
}

//func ellipseToCubicBezierError(a, b, n1, n2 float64) float64 {
//	if a < b {
//		a, b = b, a
//	}
//	ba := b / a
//	c0, c1 := 0.0, 0.0
//	if ba < 0.25 {
//		c0 += ((3.85268*ba*ba - 21.229*ba - 0.330434) / (ba + 0.0127842))
//		c0 += ((-1.61486*ba*ba + 0.706564*ba + 0.225945) / (ba + 0.263682)) * math.Cos(1.0*(n1+n2))
//		c0 += ((-0.910164*ba*ba + 0.388383*ba + 0.00551445) / (ba + 0.00671814)) * math.Cos(2.0*(n1+n2))
//		c0 += ((-0.630184*ba*ba + 0.192402*ba + 0.0098871) / (ba + 0.0102527)) * math.Cos(3.0*(n1+n2))
//		c1 += ((-0.162211*ba*ba + 9.94329*ba + 0.13723) / (ba + 0.0124084))
//		c1 += ((-0.253135*ba*ba + 0.00187735*ba + 0.0230286) / (ba + 0.01264)) * math.Cos(1.0*(n1+n2))
//		c1 += ((-0.0695069*ba*ba - 0.0437594*ba + 0.0120636) / (ba + 0.0163087)) * math.Cos(2.0*(n1+n2))
//		c1 += ((-0.0328856*ba*ba - 0.00926032*ba - 0.00173573) / (ba + 0.00527385)) * math.Cos(3.0*(n1+n2))
//	} else {
//		c0 += ((0.0899116*ba*ba - 19.2349*ba - 4.11711) / (ba + 0.183362))
//		c0 += ((0.138148*ba*ba - 1.45804*ba + 1.32044) / (ba + 1.38474)) * math.Cos(1.0*(n1+n2))
//		c0 += ((0.230903*ba*ba - 0.450262*ba + 0.219963) / (ba + 0.414038)) * math.Cos(2.0*(n1+n2))
//		c0 += ((0.0590565*ba*ba - 0.101062*ba + 0.0430592) / (ba + 0.0204699)) * math.Cos(3.0*(n1+n2))
//		c1 += ((0.0164649*ba*ba + 9.89394*ba + 0.0919496) / (ba + 0.00760802))
//		c1 += ((0.0191603*ba*ba - 0.0322058*ba + 0.0134667) / (ba - 0.0825018)) * math.Cos(1.0*(n1+n2))
//		c1 += ((0.0156192*ba*ba - 0.017535*ba + 0.00326508) / (ba - 0.228157)) * math.Cos(2.0*(n1+n2))
//		c1 += ((-0.0236752*ba*ba + 0.0405821*ba - 0.0173086) / (ba + 0.176187)) * math.Cos(3.0*(n1+n2))
//	}
//	return ((0.001*ba*ba + 4.98*ba + 0.207) / (ba + 0.0067)) * a * math.Exp(c0+c1*math.Abs(n2-n1))
//}
//
//func distanceEllipseCubicBezier(start, cp1, cp2, end Point, rx, ry, phi, cx, cy, theta0, theta1 float64) float64 {
//	N := 100
//	hausdorff := 0.0
//	for i := 0; i <= N; i++ {
//		t := float64(i) / float64(N)
//		pos := cubicBezierPos(start, cp1, cp2, end, t)
//
//		dist := func(theta float64) float64 { return EllipsePos(rx, ry, phi, cx, cy, theta).Sub(pos).Length() }
//		theta := gradientDescent(dist, theta0, theta1)
//		theta2 := lookupMin(dist, theta0, theta1)
//		fmt.Println("gradientDescent, loopup:", dist(theta), dist(theta2))
//
//		d := dist(theta)
//		if hausdorff < d {
//			hausdorff = d
//		}
//	}
//	return hausdorff
//}

// see Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe, https://spaceroots.org/documents/ellipse/elliptical-arc.pdf
func ellipseToCubicBeziers(start Point, rx, ry, phi float64, large, sweep bool, end Point) [][4]Point {
	cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

	dtheta := math.Pi / 2.0 // TODO: use error measure to determine dtheta?
	n := int(math.Ceil(math.Abs(theta1-theta0) / dtheta))
	dtheta = math.Abs(theta1-theta0) / float64(n) // evenly spread the n points, dalpha will get smaller
	kappa := math.Sin(dtheta) * (math.Sqrt(4.0+3.0*math.Pow(math.Tan(dtheta/2.0), 2.0)) - 1.0) / 3.0
	if !sweep {
		dtheta = -dtheta
	}

	beziers := [][4]Point{}
	startDeriv := ellipseDeriv(rx, ry, phi, sweep, theta0)
	for i := 1; i < n+1; i++ {
		theta := theta0 + float64(i)*dtheta
		end := EllipsePos(rx, ry, phi, cx, cy, theta)
		endDeriv := ellipseDeriv(rx, ry, phi, sweep, theta)

		cp1 := start.Add(startDeriv.Mul(kappa))
		cp2 := end.Sub(endDeriv.Mul(kappa))
		beziers = append(beziers, [4]Point{start, cp1, cp2, end})

		startDeriv = endDeriv
		start = end
	}
	return beziers
}

func flattenEllipticArc(start Point, rx, ry, phi float64, large, sweep bool, end Point) *Path {
	// TODO: (flatten ellipse) use direct algorithm
	return arcToCube(start, rx, ry, phi, large, sweep, end).Flatten()
}

////////////////////////////////////////////////////////////////
// Béziers /////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

func quadraticToCubicBezier(p0, p1, p2 Point) (Point, Point) {
	c1 := p0.Interpolate(p1, 2.0/3.0)
	c2 := p2.Interpolate(p1, 2.0/3.0)
	return c1, c2
}

// see http://www.caffeineowl.com/graphics/2d/vectorial/cubic2quad01.html
func cubicToQuadraticBeziers(p0, p1, p2, p3 Point) [][3]Point {
	// TODO: misses theoretic background for optimal number of quads
	quads := [][3]Point{}
	endQuads := [][3]Point{}
	for {
		// dist = sqrt(3)/36 * ||p3 - 3*p2 + 3*p1 - p0||
		dist := math.Sqrt(3.0) / 36.0 * p3.Sub(p2.Mul(3.0)).Add(p1.Mul(3.0)).Sub(p0).Length()
		t := math.Cbrt(Tolerance / dist)

		// cp = (3*p2 - p3 + 3*p1 - p0) / 4
		if t >= 1.0 {
			// approximate by one quadratic bezier
			pcp := p2.Mul(3.0).Sub(p3).Add(p1.Mul(3.0)).Sub(p0).Div(4.0)
			quads = append(quads, [3]Point{p0, pcp, p3})
			break
		} else if t >= 0.5 {
			// approximate by two quadratic beziers
			r0, r1, r2, r3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, 0.5)
			rcp := r2.Mul(3.0).Sub(r3).Add(r1.Mul(3.0)).Sub(r0).Div(4.0)
			qcp := q2.Mul(3.0).Sub(q3).Add(q1.Mul(3.0)).Sub(q0).Div(4.0)
			quads = append(quads, [3]Point{r0, rcp, r3}, [3]Point{q0, qcp, q3})
			break
		} else {
			// approximate start and end by two quadratic beziers, and reevaluate the middle part
			r0, r1, r2, r3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, 1-t)
			r0, r1, r2, r3, p0, p1, p2, p3 = cubicBezierSplit(r0, r1, r2, r3, t/(1-t))
			rcp := r2.Mul(3.0).Sub(r3).Add(r1.Mul(3.0)).Sub(r0).Div(4.0)
			qcp := q2.Mul(3.0).Sub(q3).Add(q1.Mul(3.0)).Sub(q0).Div(4.0)
			quads = append(quads, [3]Point{r0, rcp, r3})
			endQuads = append([][3]Point{{q0, qcp, q3}}, endQuads...)
		}
	}
	return append(quads, endQuads...)
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
	if Equal(A, 0.0) {
		// p1 is in the middle between p0 and p2, so it is a straight line from p0 to p2
		return p2.Sub(p0).Length()
	}

	Sabc := 2.0 * math.Sqrt(A+B+C)
	A2 := math.Sqrt(A)
	A32 := 2.0 * A * A2
	C2 := 2.0 * math.Sqrt(C)
	BA := B / A2
	return (A32*Sabc + A2*B*(Sabc-C2) + (4.0*C*A-B*B)*math.Log((2.0*A2+BA+Sabc)/(BA+C2))) / (4.0 * A32)
}

func quadraticBezierSplit(p0, p1, p2 Point, t float64) (Point, Point, Point, Point, Point, Point) {
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
	if Equal(a, 0.0) {
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
	panic("not implemented") // not needed
}

// cubicBezierLength calculates the length of the Bézier, taking care of inflection points. It uses Gauss-Legendre (n=5) and has an error of ~1% or less (empirical).
func cubicBezierLength(p0, p1, p2, p3 Point) float64 {
	t1, t2 := findInflectionPointsCubicBezier(p0, p1, p2, p3)
	var beziers [][4]Point
	if t1 > 0.0 && t1 < 1.0 && t2 > 0.0 && t2 < 1.0 {
		p0, p1, p2, p3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1)
		t2 = (t2 - t1) / (1.0 - t1)
		q0, q1, q2, q3, r0, r1, r2, r3 := cubicBezierSplit(q0, q1, q2, q3, t2)
		beziers = append(beziers, [4]Point{p0, p1, p2, p3})
		beziers = append(beziers, [4]Point{q0, q1, q2, q3})
		beziers = append(beziers, [4]Point{r0, r1, r2, r3})
	} else if t1 > 0.0 && t1 < 1.0 {
		p0, p1, p2, p3, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1)
		beziers = append(beziers, [4]Point{p0, p1, p2, p3})
		beziers = append(beziers, [4]Point{q0, q1, q2, q3})
	} else {
		beziers = append(beziers, [4]Point{p0, p1, p2, p3})
	}

	length := 0.0
	for _, bezier := range beziers {
		speed := func(t float64) float64 {
			return cubicBezierDeriv(bezier[0], bezier[1], bezier[2], bezier[3], t).Length()
		}
		length += gaussLegendre7(speed, 0.0, 1.0)
	}
	return length
}

func cubicBezierNumInflections(p0, p1, p2, p3 Point) int {
	t1, t2 := findInflectionPointsCubicBezier(p0, p1, p2, p3)
	if !math.IsNaN(t2) {
		return 2
	} else if !math.IsNaN(t1) {
		return 1
	}
	return 0
}

func cubicBezierSplit(p0, p1, p2, p3 Point, t float64) (Point, Point, Point, Point, Point, Point, Point, Point) {
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
		// Bézier has p0=p1=p3 or p0=p2=p3 and thus has no surface or length
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

// split the curve and replace it by lines as long as maximum deviation = flatness is maintained.
func flattenSmoothCubicBezier(p *Path, p0, p1, p2, p3 Point, d, flatness float64) {
	// TODO: by also iterating in the reverse direction, we can take the t half way between the t's found one direction and t's found the reverse direction. this will improve smoothness without add points, it will distribute the points equally along the curve (and not close to the end points)
	t := 0.0
	for t < 1.0 {
		s2nom := (p2.X-p0.X)*(p1.Y-p0.Y) - (p2.Y-p0.Y)*(p1.X-p0.X)
		denom := math.Hypot(p1.X-p0.X, p1.Y-p0.Y)
		if s2nom*denom == 0.0 {
			break
		}

		s2 := s2nom / denom
		//r1 := denom
		effectiveFlatness := flatness // / (1.0 + 2.0*d*s2/3.0/r1/r1)  // TODO: (stroke bezier) unclear what to do when negative
		t = 2.0 * math.Sqrt(effectiveFlatness/3.0/math.Abs(s2))
		if t >= 1.0 {
			break
		}
		_, _, _, _, p0, p1, p2, p3 = cubicBezierSplit(p0, p1, p2, p3, t)
		addCubicBezierLine(p, p0, p1, p2, p3, 0.0, d)
	}
	addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
}

func findInflectionPointsCubicBezier(p0, p1, p2, p3 Point) (float64, float64) {
	// see www.faculty.idc.ac.il/arik/quality/appendixa.html
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
	if x1 < 0.0 || 1.0 <= x1 {
		x1 = math.NaN()
	}
	if x2 < 0.0 || 1.0 <= x2 {
		x2 = math.NaN()
	} else if math.IsNaN(x1) {
		x1, x2 = x2, x1
	}
	return x1, x2
}

func findInflectionPointRangeCubicBezier(p0, p1, p2, p3 Point, t, flatness float64) (float64, float64) {
	// find the range around an inflection point that we consider flat within the flatness criterion
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

	if !Equal(t, 0.0) {
		_, _, _, _, p0, p1, p2, p3 = cubicBezierSplit(p0, p1, p2, p3, t)
	}
	nr := p1.Sub(p0)
	ns := p3.Sub(p0)
	if Equal(nr.X, 0.0) && Equal(nr.Y, 0.0) {
		// if p0=p1, then rn (the velocity at t=0) needs adjustment
		// nr = lim[t->0](B'(t)) = 3*(p1-p0) + 6*t*((p1-p0)+(p2-p1)) + second order terms of t
		// if (p1-p0)->0, we use (p2-p1)=(p2-p0)
		nr = p2.Sub(p0)
	}

	if Equal(nr.X, 0.0) && Equal(nr.Y, 0.0) {
		// if rn is still zero, this curve has p0=p1=p2, so it is straight
		return 0.0, 1.0
	}

	s3 := math.Abs(ns.X*nr.Y-ns.Y*nr.X) / math.Hypot(nr.X, nr.Y)
	if Equal(s3, 0.0) {
		return 0.0, 1.0 // can approximate whole curve linearly
	}

	tf := math.Cbrt(flatness / s3)
	return t - tf*(1.0-t), t + tf*(1.0-t)
}

func flattenQuadraticBezier(p0, p1, p2 Point) *Path {
	cp1, cp2 := quadraticToCubicBezier(p0, p1, p2)
	return strokeCubicBezier(p0, cp1, cp2, p2, 0.0, Tolerance)
}

func flattenCubicBezier(p0, p1, p2, p3 Point) *Path {
	return strokeCubicBezier(p0, p1, p2, p3, 0.0, Tolerance)
}

// see Flat, precise flattening of cubic Bézier path and offset curves, by T.F. Hain et al., 2005,  https://www.sciencedirect.com/science/article/pii/S0097849305001287
// see https://github.com/Manishearth/stylo-flat/blob/master/gfx/2d/Path.cpp for an example implementation
// or https://docs.rs/crate/lyon_bezier/0.4.1/source/src/flatten_cubic.rs
// p0, p1, p2, p3 are the start points, two control points and the end points respectively. With flatness defined as the maximum error from the orinal curve, and d the half width of the curve used for stroking (positive is to the right).
func strokeCubicBezier(p0, p1, p2, p3 Point, d, flatness float64) *Path {
	p := &Path{}
	start := p0.Add(cubicBezierNormal(p0, p1, p2, p3, 0.0, d))
	p.MoveTo(start.X, start.Y)

	// 0 <= t1 <= 1 if t1 exists
	// 0 <= t1 <= t2 <= 1 if t1 and t2 both exist
	t1, t2 := findInflectionPointsCubicBezier(p0, p1, p2, p3)
	if math.IsNaN(t1) && math.IsNaN(t2) {
		// There are no inflection points or cusps, approximate linearly by subdivision.
		flattenSmoothCubicBezier(p, p0, p1, p2, p3, d, flatness)
		return p
	}

	// t1min <= t1max; with 0 <= t1max and t1min <= 1
	// t2min <= t2max; with 0 <= t2max and t2min <= 1
	t1min, t1max := findInflectionPointRangeCubicBezier(p0, p1, p2, p3, t1, flatness)
	t2min, t2max := findInflectionPointRangeCubicBezier(p0, p1, p2, p3, t2, flatness)

	if math.IsNaN(t2) && t1min <= 0.0 && 1.0 <= t1max {
		// There is no second inflection point, and the first inflection point can be entirely approximated linearly.
		addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
		return p
	}

	if 0.0 < t1min {
		// Flatten up to t1min
		q0, q1, q2, q3, _, _, _, _ := cubicBezierSplit(p0, p1, p2, p3, t1min)
		flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
	}

	if 0.0 < t1max && t1max < 1.0 && t1max < t2min {
		// t1 and t2 ranges do not overlap, approximate t1 linearly
		_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1max)
		addCubicBezierLine(p, q0, q1, q2, q3, 0.0, d)
		if 1.0 <= t2min {
			// No t2 present, approximate the rest linearly by subdivision
			flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
			return p
		}
	} else if 1.0 <= t2min {
		// No t2 present and t1max is past the end of the curve, approximate linearly
		addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
		return p
	}

	// t1 and t2 exist and ranges might overlap
	if 0.0 < t2min {
		if t2min < t1max {
			// t2 range starts inside t1 range, approximate t1 range linearly
			_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1max)
			addCubicBezierLine(p, q0, q1, q2, q3, 0.0, d)
		} else {
			// no overlap
			_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t1max)
			t2minq := (t2min - t1max) / (1 - t1max)
			q0, q1, q2, q3, _, _, _, _ = cubicBezierSplit(q0, q1, q2, q3, t2minq)
			flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
		}
	}

	// handle (the rest of) t2
	if t2max < 1.0 {
		_, _, _, _, q0, q1, q2, q3 := cubicBezierSplit(p0, p1, p2, p3, t2max)
		addCubicBezierLine(p, q0, q1, q2, q3, 0.0, d)
		flattenSmoothCubicBezier(p, q0, q1, q2, q3, d, flatness)
	} else {
		// t2max extends beyond 1
		addCubicBezierLine(p, p0, p1, p2, p3, 1.0, d)
	}
	return p
}
