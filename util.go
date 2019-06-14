package canvas

import (
	"encoding/hex"
	"fmt"
	"image/color"
	"math"

	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

const Epsilon = 1e-10

// equal returns true if a and b are equal with tolerance Epsilon.
func equal(a, b float64) bool {
	return math.Abs(a-b) < Epsilon
}

// angleNorm returns the angle theta in the range [0,2PI).
func angleNorm(theta float64) float64 {
	theta = math.Mod(theta, 2.0*math.Pi)
	if theta < 0.0 {
		theta += 2.0 * math.Pi
	}
	return theta
}

// angleBetween is true when theta is in range (lower,upper) excluding the end points. Angles can be outside the [0,2PI) range.
func angleBetween(theta, lower, upper float64) bool {
	sweep := lower <= upper // true for CCW, ie along a positive angle
	theta = angleNorm(theta - lower)
	upper = angleNorm(upper - lower)
	if theta != 0.0 && (sweep && theta < upper || !sweep && theta > upper) {
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////

func toCSSColor(color color.RGBA) string {
	if color.A == 255 {
		buf := make([]byte, 7)
		buf[0] = '#'
		hex.Encode(buf[1:], []byte{color.R, color.G, color.B})
		return string(buf)
	} else {
		return fmt.Sprintf("rgba(%d,%d,%d,%g)", color.R, color.G, color.B, float64(color.A)/255.0)
	}
}

////////////////////////////////////////////////////////////////

func toF32Vec(x, y float64) f32.Vec2 {
	return f32.Vec2{float32(x), float32(y)}
}

func toP26_6(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{toI26_6(x), toI26_6(y)}
}

func fromP26_6(f fixed.Point26_6) Point {
	return Point{float64(f.X) / 64.0, float64(f.Y) / 64.0}
}

func toI26_6(f float64) fixed.Int26_6 {
	return fixed.Int26_6(f * 64.0)
}

func fromI26_6(f fixed.Int26_6) float64 {
	return float64(f) / 64.0
}

////////////////////////////////////////////////////////////////

// Point is a coordinate in 2D space. OP refers to the line that goes through the origin (0,0) and this point (x,y).
type Point struct {
	X, Y float64
}

// IsZero returns true if P is exactly zero.
func (p Point) IsZero() bool {
	return p.X == 0.0 && p.Y == 0.0
}

// Equals returns true if P and Q are equal with tolerance Epsilon.
func (p Point) Equals(q Point) bool {
	return equal(p.X, q.X) && equal(p.Y, q.Y)
}

// Neg negates x and y.
func (p Point) Neg() Point {
	return Point{-p.X, -p.Y}
}

// Add adds Q to P.
func (p Point) Add(q Point) Point {
	return Point{p.X + q.X, p.Y + q.Y}
}

// Sub subtracts Q from P.
func (p Point) Sub(q Point) Point {
	return Point{p.X - q.X, p.Y - q.Y}
}

// Mul multiplies x and y by f.
func (p Point) Mul(f float64) Point {
	return Point{f * p.X, f * p.Y}
}

// Div divides x and y by f.
func (p Point) Div(f float64) Point {
	return Point{p.X / f, p.Y / f}
}

// Rot90CW rotates the line OP by 90 degrees CW.
func (p Point) Rot90CW() Point {
	return Point{p.Y, -p.X}
}

// Rot90CCW rotates the line OP by 90 degrees CCW.
func (p Point) Rot90CCW() Point {
	return Point{-p.Y, p.X}
}

// Rot rotates the line OP by rot degrees CCW.
func (p Point) Rot(phi float64, p0 Point) Point {
	sinphi, cosphi := math.Sincos(phi)
	return Point{
		p0.X + cosphi*(p.X-p0.X) - sinphi*(p.Y-p0.Y),
		p0.Y + sinphi*(p.X-p0.X) + cosphi*(p.Y-p0.Y),
	}
}

// Dot returns the dot product between OP and OQ, ie. zero if perpendicular and |OP|*|OQ| if aligned.
func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

// PerpDot returns the perp dot product between OP and OQ, ie. zero if aligned and |OP|*|OQ| if perpendicular.
func (p Point) PerpDot(q Point) float64 {
	return p.X*q.Y - p.Y*q.X
}

// Length returns the length of OP.
func (p Point) Length() float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}

// Slope returns the slope between OP, ie. y/x.
func (p Point) Slope() float64 {
	return p.Y / p.X
}

// Angle returns the angle between the x-axis and OP.
func (p Point) Angle() float64 {
	return math.Atan2(p.Y, p.X)
}

// AngleBetween returns the angle between OP and OQ.
func (p Point) AngleBetween(q Point) float64 {
	return math.Atan2(p.PerpDot(q), p.Dot(q))
}

// Norm normalized OP to be of certain length.
func (p Point) Norm(length float64) Point {
	d := p.Length()
	if equal(d, 0.0) {
		return Point{}
	}
	return Point{p.X / d * length, p.Y / d * length}
}

// Interpolate returns a point on PQ that is linearly interpolated by t, ie. t=0 returns P and t=1 returns Q.
func (p Point) Interpolate(q Point, t float64) Point {
	return Point{(1-t)*p.X + t*q.X, (1-t)*p.Y + t*q.Y}
}

func (p Point) String() string {
	return fmt.Sprintf("[%g; %g]", p.X, p.Y)
}

////////////////////////////////////////////////////////////////

type Rect struct {
	X, Y, W, H float64
}

func (r Rect) Move(p Point) Rect {
	r.X += p.X
	r.Y += p.Y
	return r
}

func (r Rect) Add(q Rect) Rect {
	if q.W == 0.0 || q.H == 0 {
		return r
	} else if r.W == 0.0 || r.H == 0 {
		return q
	}
	x0 := math.Min(r.X, q.X)
	y0 := math.Min(r.Y, q.Y)
	x1 := math.Max(r.X+r.W, q.X+q.W)
	y1 := math.Max(r.Y+r.H, q.Y+q.H)
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func (r Rect) ToPath() *Path {
	return Rectangle(r.X, r.Y, r.W, r.H)
}

func (r Rect) String() string {
	return fmt.Sprintf("[%g; %g]--[%g; %g]", r.X, r.Y, r.X+r.W, r.Y+r.H)
}

////////////////////////////////////////////////////////////////

// Matrix is used for affine transformations. Be aware that concatenating transformation function will be evaluated right-to-left! So in Identity.Rotate(30).Translate(20,0) will first translate 20 points horizontally and then rotate 30 degrees counter clockwise.
type Matrix [2][3]float64

var Identity = Matrix{
	{1.0, 0.0, 0.0},
	{0.0, 1.0, 0.0},
}

func (m Matrix) Mul(q Matrix) Matrix {
	return Matrix{{
		m[0][0]*q[0][0] + m[0][1]*q[1][0],
		m[0][0]*q[0][1] + m[0][1]*q[1][1],
		m[0][0]*q[0][2] + m[0][1]*q[1][2] + m[0][2],
	}, {
		m[1][0]*q[0][0] + m[1][1]*q[1][0],
		m[1][0]*q[0][1] + m[1][1]*q[1][1],
		m[1][0]*q[0][2] + m[1][1]*q[1][2] + m[1][2],
	}}
}

func (m Matrix) Dot(p Point) Point {
	return Point{
		m[0][0]*p.X + m[0][1]*p.Y + m[0][2],
		m[1][0]*p.X + m[1][1]*p.Y + m[1][2],
	}
}

func (m Matrix) Translate(x, y float64) Matrix {
	return m.Mul(Matrix{
		{1.0, 0.0, x},
		{0.0, 1.0, y},
	})
}

func (m Matrix) Rotate(rot float64) Matrix {
	sintheta, costheta := math.Sincos(rot * math.Pi / 180.0)
	return m.Mul(Matrix{
		{costheta, -sintheta, 0.0},
		{sintheta, costheta, 0.0},
	})
}

func (m Matrix) Scale(x, y float64) Matrix {
	if equal(x, 0.0) && equal(y, 0.0) {
		panic("cannot scale affine transformation matrix to zero in x and y")
	}
	return m.Mul(Matrix{
		{x, 0.0, 0.0},
		{0.0, y, 0.0},
	})
}

func (m Matrix) Shear(x, y float64) Matrix {
	return m.Mul(Matrix{
		{1.0, x, 0.0},
		{y, 1.0, 0.0},
	})
}

func (m Matrix) RotateAt(rot, x, y float64) Matrix {
	return m.Translate(-x, -y).Rotate(rot).Translate(x, y)
}

func (m Matrix) ReflectX() Matrix {
	return m.Scale(-1.0, 1.0)
}

func (m Matrix) ReflectY() Matrix {
	return m.Scale(1.0, -1.0)
}

func (m Matrix) ReflectXAt(x float64) Matrix {
	return m.Translate(-x, 0.0).Scale(-1.0, 1.0).Translate(x, 0.0)
}

func (m Matrix) ReflectYAt(y float64) Matrix {
	return m.Translate(0.0, -y).Scale(1.0, -1.0).Translate(0.0, y)
}

func (m Matrix) T() Matrix {
	m[0][1], m[1][0] = m[1][0], m[0][1]
	return m
}

func (m Matrix) Det() float64 {
	return m[0][0]*m[1][1] - m[0][1]*m[1][0]
}

func (m Matrix) Inv() Matrix {
	det := m.Det()
	if equal(det, 0.0) {
		panic("determinant of affine transformation matrix is zero, should be impossible!")
	}
	return Matrix{{
		m[1][1] / det,
		-m[0][1] / det,
		-(m[1][1]*m[0][2] - m[0][1]*m[1][2]) / det,
	}, {
		-m[1][0] / det,
		m[0][0] / det,
		-(-m[1][0]*m[0][2] + m[0][0]*m[1][2]) / det,
	}}
}

func (m Matrix) Eigen() (float64, float64, Point, Point) {
	if equal(m[1][0], 0.0) && equal(m[0][1], 0.0) {
		return m[0][0], m[1][1], Point{1.0, 0.0}, Point{0.0, 1.0}
	}

	lambda1, lambda2 := solveQuadraticFormula(1.0, -m[0][0]-m[1][1], m.Det())
	if math.IsNaN(lambda1) && math.IsNaN(lambda2) {
		// either m[0][0] or m[1][1] is NaN
		panic("eigenvalues of affine transformation matrix do not exist, should be impossible!")
	} else if math.IsNaN(lambda2) {
		lambda2 = lambda1
	}

	// see http://www.math.harvard.edu/archive/21b_fall_04/exhibits/2dmatrices/index.html
	var v1, v2 Point
	if !equal(m[1][0], 0.0) {
		v1 = Point{lambda1 - m[1][1], m[1][0]}
		v2 = Point{lambda2 - m[1][1], m[1][0]}
	} else if !equal(m[0][1], 0.0) {
		v1 = Point{m[0][1], lambda1 - m[0][0]}
		v2 = Point{m[0][1], lambda2 - m[0][0]}
	}
	return lambda1, lambda2, v1, v2
}

func (m Matrix) pos() (float64, float64) {
	return m[0][2], m[1][2]
}

func (m Matrix) theta() float64 {
	return math.Atan2(-m[0][1], m[0][0])
}

func (m Matrix) scale() (float64, float64) {
	x := math.Copysign(math.Sqrt(m[0][0]*m[0][0]+m[0][1]*m[0][1]), m[0][0])
	y := math.Copysign(math.Sqrt(m[1][0]*m[1][0]+m[1][1]*m[1][1]), m[1][1])
	return x, y
}

func (m Matrix) String() string {
	return fmt.Sprintf("[%g, %g, %g; %g, %g, %g; 0, 0, 1]", m[0][0], m[0][1], m[0][2], m[1][0], m[1][1], m[1][2])
}

////////////////////////////////////////////////////////////////

// Numerically stable quadratic formula, lowest root is returned first
// see https://math.stackexchange.com/a/2007723
func solveQuadraticFormula(a, b, c float64) (float64, float64) {
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
		return -c / b, math.NaN()
	}

	if c == 0.0 {
		// no constant term, one solution at zero and one from solving linearly
		return 0.0, -b / a
	}

	discriminant := b*b - 4.0*a*c
	if discriminant < 0.0 {
		return math.NaN(), math.NaN()
	} else if discriminant == 0.0 {
		return -b / (2.0 * a), math.NaN()
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
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	return x1, x2
}

type gaussLegendreFunc func(func(float64) float64, float64, float64) float64

// Gauss-Legendre quadrature integration from a to b with n=3
// see https://pomax.github.io/bezierinfo/legendre-gauss.html for more values
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

// Gauss-Legendre quadrature integration from a to b with n=7
func gaussLegendre7(f func(float64) float64, a, b float64) float64 {
	c := (b - a) / 2.0
	d := (a + b) / 2.0
	Qd1 := f(-0.949108*c + d)
	Qd2 := f(-0.741531*c + d)
	Qd3 := f(-0.405845*c + d)
	Qd4 := f(d)
	Qd5 := f(0.405845*c + d)
	Qd6 := f(0.741531*c + d)
	Qd7 := f(0.949108*c + d)
	return c * (0.129485*(Qd1+Qd7) + 0.279705*(Qd2+Qd6) + 0.381830*(Qd3+Qd5) + 0.417959*Qd4)
}

// find value x for which f(x) = y in the interval x in [xmin, xmax] using the bisection method
func bisectionMethod(f func(float64) float64, y, xmin, xmax float64) float64 {
	const MaxIterations = 100
	const Tolerance = 0.001 // 0.1%

	n := 0
	toleranceX := math.Abs(xmax-xmin) * Tolerance
	toleranceY := math.Abs(f(xmax)-f(xmin)) * Tolerance

	var x float64
	for {
		x = (xmin + xmax) / 2.0
		if n >= MaxIterations {
			return x
		}

		dy := f(x) - y
		if math.Abs(dy) < toleranceY || math.Abs(xmax-xmin)/2.0 < toleranceX {
			return x
		} else if dy > 0.0 {
			xmax = x
		} else {
			xmin = x
		}
		n++
	}
	return x // MaxIterations reached
}

// polynomialApprox returns a function y(x) that maps the parameter x [xmin,xmax] to the integral of fp. For a circle tmin and tmax would be 0 and 2PI respectively for example. It also returns the total length of the curve.
// Implemented using M. Walter, A. Fournier, Approximate Arc Length Parametrization, Anais do IX SIBGRAPHI, p. 143--150, 1996
// see https://www.visgraf.impa.br/sibgrapi96/trabs/pdf/a14.pdf
func polynomialApprox3(gaussLegendre gaussLegendreFunc, fp func(float64) float64, xmin, xmax float64) (func(float64) float64, float64) {
	y1 := gaussLegendre(fp, xmin, xmin+(xmax-xmin)*1.0/3.0)
	y2 := gaussLegendre(fp, xmin, xmin+(xmax-xmin)*2.0/3.0)
	y3 := gaussLegendre(fp, xmin, xmax)

	// We have four points on the y(x) curve at x0=0, x1=1/3, x2=2/3 and x3=1
	// now obtain a polynomial that goes through these four points by solving the system of linear equations
	// y(x) = a*x^3 + b*x^2 + c*x + d  (NB: y0 = d = 0)
	// [y1; y2; y3] = [1/27, 1/9, 1/3;
	//                 8/27, 4/9, 2/3;
	//                    1,   1,   1] * [a; b; c]
	//
	// After inverting:
	// [a; b; c] = 0.5 * [ 27, -27,  9;
	//                    -45,  36, -9;
	//                     18,  -9,  2] * [y1; y2; y3]
	// NB: y0 = d = 0

	a := 13.5*y1 - 13.5*y2 + 4.5*y3
	b := -22.5*y1 + 18.0*y2 - 4.5*y3
	c := 9.0*y1 - 4.5*y2 + y3
	return func(x float64) float64 {
		x = (x - xmin) / (xmax - xmin)
		return a*x*x*x + b*x*x + c*x
	}, math.Abs(y3)
}

// invPolynomialApprox does the opposite of polynomialApprox, it returns a function x(y) that maps the parameter y [f(xmin),f(xmax)] to x [xmin,xmax]
func invPolynomialApprox3(gaussLegendre gaussLegendreFunc, fp func(float64) float64, xmin, xmax float64) (func(float64) float64, float64) {
	f := func(t float64) float64 {
		return math.Abs(gaussLegendre(fp, xmin, xmin+(xmax-xmin)*t))
	}
	f3 := f(1.0)
	t1 := bisectionMethod(f, (1.0/3.0)*f3, 0.0, 1.0)
	t2 := bisectionMethod(f, (2.0/3.0)*f3, 0.0, 1.0)
	t3 := 1.0

	// We have four points on the x(y) curve at y0=0, y1=1/3, y2=2/3 and y3=1
	// now obtain a polynomial that goes through these four points by solving the system of linear equations
	// x(y) = a*y^3 + b*y^2 + c*y + d  (NB: x0 = d = 0)
	// [x1; x2; x3] = [1/27, 1/9, 1/3;
	//                 8/27, 4/9, 2/3;
	//                    1,   1,   1] * [a*y3^3; b*y3^2; c*y3]
	//
	// After inverting:
	// [a*y3^3; b*y3^2; c*y3] = 0.5 * [ 27, -27,  9;
	//                                 -45,  36, -9;
	//                                  18,  -9,  2] * [x1; x2; x3]
	// NB: x0 = d = 0

	a := (27.0*t1 - 27.0*t2 + 9.0*t3) / (2.0 * f3 * f3 * f3)
	b := (-45.0*t1 + 36.0*t2 - 9.0*t3) / (2.0 * f3 * f3)
	c := (18.0*t1 - 9.0*t2 + 2.0*t3) / (2.0 * f3)
	return func(f float64) float64 {
		t := a*f*f*f + b*f*f + c*f
		return xmin + (xmax-xmin)*t
	}, f3
}

func invSpeedPolynomialChebyshevApprox(N int, gaussLegendre gaussLegendreFunc, fp func(float64) float64, tmin, tmax float64) (func(float64) float64, float64) {
	fLength := func(t float64) float64 {
		return math.Abs(gaussLegendre(fp, tmin, t))
	}
	totalLength := fLength(tmax)
	t := func(L float64) float64 {
		return bisectionMethod(fLength, L, tmin, tmax)
	}
	return polynomialChebyshevApprox(N, t, 0.0, totalLength, tmin, tmax), totalLength
}

func polynomialChebyshevApprox(N int, f func(float64) float64, xmin, xmax, ymin, ymax float64) func(float64) float64 {
	fs := make([]float64, N)
	for k := 0; k < N; k++ {
		u := math.Cos(math.Pi * (float64(k+1) - 0.5) / float64(N))
		fs[k] = f(xmin + (xmax-xmin)*(u+1.0)/2.0)
	}

	c := make([]float64, N)
	for j := 0; j < N; j++ {
		a := 0.0
		for k := 0; k < N; k++ {
			a += fs[k] * math.Cos(float64(j)*math.Pi*(float64(k+1)-0.5)/float64(N))
		}
		c[j] = (2.0 / float64(N)) * a
	}

	if ymax < ymin {
		ymin, ymax = ymax, ymin
	}
	return func(x float64) float64 {
		u := (x-xmin)/(xmax-xmin)*2.0 - 1.0
		a := 0.0
		for j := 0; j < N; j++ {
			a += c[j] * math.Cos(float64(j)*math.Acos(u))
		}
		y := -0.5*c[0] + a
		if !math.IsNaN(ymin) && !math.IsNaN(ymax) {
			y = math.Min(ymax, math.Max(ymin, y))
		}
		return y
	}
}
