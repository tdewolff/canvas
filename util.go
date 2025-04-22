package canvas

import (
	"encoding/hex"
	"fmt"
	"image/color"
	"math"
	"strings"

	"golang.org/x/image/math/fixed"

	"github.com/tdewolff/minify/v2"
)

const minNormalFloat64 = 0x1p-1022

// Epsilon is the smallest number below which we assume the value to be zero. This is to avoid numerical floating point issues.
var Epsilon = 1e-10

// Precision is the number of significant digits at which floating point value will be printed to output formats.
var Precision = 8

// Equal returns true if a and b are equal within an absolute tolerance of Epsilon or within a relative tolerance of Epsilon (relative to the largest of the two).
func Equal(a, b float64) bool {
	// avoid math.Abs
	if a < b {
		return b-a <= Epsilon
	}
	return a-b <= Epsilon
	//return math.Abs(a-b) <= Epsilon // this is much quicker

	// See https://floating-point-gui.de/errors/comparison/ and
	// https://randomascii.wordpress.com/2012/02/25/comparing-floating-point-numbers-2012-edition/
	// for more information. To be blunt, the code below may not be appropriate for all cases,
	// especially for small numbers. Since most comparisons involve millimeter scale (the
	// coordinates in a canvas), this is probably OK. We should make sure that computations
	// resulting in small numbers (below Epsilon) should be insignificant in their difference.
	diff := math.Abs(a - b)
	abs := a == b || diff <= Epsilon // handle infinities and absolute epsilon
	if !abs && (a != 0.0 || b != 0.0) {
		// handle relative epsilon for large numbers (relative to largest number)
		return diff/math.Max(math.Abs(a), math.Abs(b)) <= Epsilon
	}
	return abs
}

// Interval returns true if f is in closed interval [lower-Epsilon,upper+Epsilon] where lower and upper can be interchanged.
func Interval(f, lower, upper float64) bool {
	if upper < lower {
		lower, upper = upper, lower
	}
	return lower-Epsilon <= f && f <= upper+Epsilon
}

// IntervalExclusive returns true if f is in open interval [lower+Epsilon,upper-Epsilon] where lower and upper can be interchanged.
func IntervalExclusive(f, lower, upper float64) bool {
	if upper < lower {
		lower, upper = upper, lower
	}
	return lower+Epsilon < f && f < upper-Epsilon
}

// angleEqual returns true if both angles are equal.
func angleEqual(a, b float64) bool {
	return angleBetween(a, b, b) // angleBetween will add Epsilon to lower and upper
}

// angleNorm returns the angle theta in the range [0,2PI).
func angleNorm(theta float64) float64 {
	theta = math.Mod(theta, 2.0*math.Pi)
	if theta < 0.0 {
		theta += 2.0 * math.Pi
	}
	return theta
}

// angleTime returns the time [0.0,1.0] of theta between [lower,upper]. When outside of [lower,upper], the result will also be outside of [0.0,1.0].
func angleTime(theta, lower, upper float64) float64 {
	sweep := true
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
		sweep = false
	}
	theta = angleNorm(theta - lower + Epsilon)
	upper = angleNorm(upper - lower)

	t := (theta - Epsilon) / upper
	if !sweep {
		t = 1.0 - t
	}
	if Equal(t, 0.0) {
		return 0.0
	} else if Equal(t, 1.0) {
		return 1.0
	}
	return t
}

// angleBetween is true when theta is in range [lower,upper] including the end points. Angles can be outside the [0,2PI) range.
func angleBetween(theta, lower, upper float64) bool {
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
	}
	theta = angleNorm(theta - lower + Epsilon)
	upper = angleNorm(upper - lower + 2.0*Epsilon)
	return theta <= upper
}

// angleBetweenExclusive is true when theta is in range (lower,upper) excluding the end points. Angles can be outside the [0,2PI) range.
func angleBetweenExclusive(theta, lower, upper float64) bool {
	if upper < lower {
		// sweep is false, ie direction is along negative angle (clockwise)
		lower, upper = upper, lower
	}
	theta = angleNorm(theta - lower)
	upper = angleNorm(upper - lower)
	if 0.0 < theta && theta < upper {
		return true
	}
	return false
}

// snap "gridsnaps" the floating point to a grid of the given spacing
func snap(val, spacing float64) float64 {
	return math.Round(val/spacing) * spacing
}

////////////////////////////////////////////////////////////////

type numEps float64

func (f numEps) String() string {
	s := fmt.Sprintf("%.*g", int(math.Ceil(-math.Log10(Epsilon))), f)
	if dot := strings.IndexByte(s, '.'); dot != -1 {
		for dot < len(s) && s[len(s)-1] == '0' {
			s = s[:len(s)-1]
		}
		if dot < len(s) && s[len(s)-1] == '.' {
			s = s[:len(s)-1]
		}
	}
	return s
}

type num float64

func (f num) String() string {
	s := fmt.Sprintf("%.*g", Precision, f)
	if num(math.MaxInt32) < f || f < num(math.MinInt32) {
		if i := strings.IndexAny(s, ".eE"); i == -1 {
			s += ".0"
		}
	}
	return string(minify.Number([]byte(s), Precision))
}

type dec float64

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", Precision, f)
	s = string(minify.Decimal([]byte(s), Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}

// CSSColor is a string formatter to convert a color.RGBA to a CSS color (hexadecimal or using rgba()).
type CSSColor color.RGBA

func (color CSSColor) String() string {
	if color.A == 255 {
		buf := make([]byte, 7)
		buf[0] = '#'
		hex.Encode(buf[1:], []byte{color.R, color.G, color.B})
		if buf[1] == buf[2] && buf[3] == buf[4] && buf[5] == buf[6] {
			buf[2] = buf[3]
			buf[3] = buf[5]
			buf = buf[:4]
		}
		return string(buf)
	} else if color.A == 0 {
		return "rgba(0,0,0,0)"
	}
	a := float64(color.A) / 255.0
	return fmt.Sprintf("rgba(%d,%d,%d,%v)", int(float64(color.R)/a), int(float64(color.G)/a), int(float64(color.B)/a), dec(a))
}

func rgbaColor(col color.Color) color.RGBA {
	r, g, b, a := col.RGBA()
	// RGBA returns an alpha-premultiplied color so that c <= a. We silently correct the color by clipping r,g,b to a
	if a < r {
		r = a
	}
	if a < g {
		g = a
	}
	if a < b {
		b = a
	}
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

////////////////////////////////////////////////////////////////

func toP26_6(p Point) fixed.Point26_6 {
	return fixed.Point26_6{X: toI26_6(p.X), Y: toI26_6(p.Y)}
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

// Origin is the coordinate system's origin.
var Origin = Point{0.0, 0.0}

// Point is a coordinate in 2D space. OP refers to the line that goes through the origin (0,0) and this point (x,y).
type Point struct {
	X, Y float64
}

// PolarPoint returns a point from polar coordinates, with angle in radians CCW and radius the distance from (0,0).
func PolarPoint(angle, radius float64) Point {
	return Point{
		X: radius * math.Cos(angle),
		Y: radius * math.Sin(angle),
	}
}

// IsZero returns true if P is exactly zero.
func (p Point) IsZero() bool {
	return p.X == 0.0 && p.Y == 0.0
}

// Equals returns true if P and Q are equal with tolerance Epsilon.
func (p Point) Equals(q Point) bool {
	return Equal(p.X, q.X) && Equal(p.Y, q.Y)
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

// Rot rotates the line OP by phi radians CCW.
func (p Point) Rot(phi float64, p0 Point) Point {
	sinphi, cosphi := math.Sincos(phi)
	return Point{
		p0.X + cosphi*(p.X-p0.X) - sinphi*(p.Y-p0.Y),
		p0.Y + sinphi*(p.X-p0.X) + cosphi*(p.Y-p0.Y),
	}
}

// Hadamard returns the Hadamard product, or the element-wise product, of the point.
func (p Point) Hadamard(q Point) Point {
	return Point{
		p.X * q.X,
		p.Y * q.Y,
	}
}

// Dot returns the dot product between OP and OQ, i.e. zero if perpendicular and |OP|*|OQ| if aligned.
func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

// PerpDot returns the perp dot product between OP and OQ, i.e. zero if aligned and |OP|*|OQ| if perpendicular. This is the cross product in two dimensions.
func (p Point) PerpDot(q Point) float64 {
	return p.X*q.Y - p.Y*q.X
}

// Length returns the length of OP.
func (p Point) Length() float64 {
	return math.Hypot(p.X, p.Y) // sqrt(p.X^2 + p.Y^2)
}

// Slope returns the slope between OP, i.e. y/x.
func (p Point) Slope() float64 {
	return p.Y / p.X
}

// Angle returns the angle in radians [0,2PI) between the x-axis and OP.
func (p Point) Angle() float64 {
	return angleNorm(math.Atan2(p.Y, p.X))
}

// AngleBetween returns the angle between OP and OQ.
func (p Point) AngleBetween(q Point) float64 {
	return math.Atan2(p.PerpDot(q), p.Dot(q))
}

// Norm normalises OP to be of given length.
func (p Point) Norm(length float64) Point {
	d := p.Length()
	if d == 0.0 {
		return Point{}
	}
	return Point{p.X / d * length, p.Y / d * length}
}

// Transform transforms the point by affine transformation matrix m.
func (p Point) Transform(m Matrix) Point {
	return m.Dot(p)
}

// Interpolate returns a point on PQ that is linearly interpolated by t in [0,1], i.e. t=0 returns P and t=1 returns Q.
func (p Point) Interpolate(q Point, t float64) Point {
	return Point{(1-t)*p.X + t*q.X, (1-t)*p.Y + t*q.Y}
}

func (p Point) InterpolateX(q Point, x float64) Point {
	return p.Interpolate(q, (x-p.X)/(q.X-p.X))
}

func (p Point) InterpolateY(q Point, y float64) Point {
	return p.Interpolate(q, (y-p.Y)/(q.Y-p.Y))
}

// Gridsnap snaps point to a grid with the given spacing.
func (p Point) Gridsnap(spacing float64) Point {
	return Point{snap(p.X, spacing), snap(p.Y, spacing)}
}

// String returns the string representation of a point, such as "(x,y)".
func (p Point) String() string {
	return fmt.Sprintf("(%v,%v)", p.X, p.Y)
}

////////////////////////////////////////////////////////////////

// Rect is a rectangle in 2D defined by a position and its width and height.
type Rect struct {
	X0, Y0, X1, Y1 float64
}

func RectFromSize(x, y, w, h float64) Rect {
	return Rect{x, y, x + w, y + h}
}

func RectFromPoints(ps ...Point) Rect {
	if len(ps) == 0 {
		return Rect{}
	}
	r := Rect{ps[0].X, ps[0].Y, ps[0].X, ps[0].Y}
	for _, p := range ps[1:] {
		r = r.AddPoint(p)
	}
	return r
}

func (r Rect) Zero() bool {
	return r == Rect{}
}

func (r Rect) Empty() bool {
	return Equal(r.W(), 0.0) || Equal(r.H(), 0.0)
}

// W returns the width of the rectangle.
func (r Rect) W() float64 {
	return r.X1 - r.X0
}

// H returns the height of the rectangle.
func (r Rect) H() float64 {
	return r.Y1 - r.Y0
}

// Area returns the area of the rectangle.
func (r Rect) Area() float64 {
	return (r.X1 - r.X0) * (r.Y1 - r.Y0)
}

// Center returns the center point.
func (r Rect) Center() Point {
	return Point{(r.X0 + r.X1) / 2.0, (r.Y0 + r.Y1) / 2.0}
}

// Equals returns true if rectangles are equal with tolerance Epsilon.
func (r Rect) Equals(q Rect) bool {
	return Equal(r.X0, q.X0) && Equal(r.X1, q.X1) && Equal(r.Y0, q.Y0) && Equal(r.Y1, q.Y1)
}

// Translate translates the rect.
func (r Rect) Translate(x, y float64) Rect {
	r.X0 += x
	r.Y0 += y
	r.X1 += x
	r.Y1 += y
	return r
}

// Transform transforms the rectangle by affine transformation matrix m and returns the new bounds of that rectangle.
func (r Rect) Transform(m Matrix) Rect {
	p0 := m.Dot(Point{r.X0, r.Y0})
	p1 := m.Dot(Point{r.X1, r.Y0})
	p2 := m.Dot(Point{r.X1, r.Y1})
	p3 := m.Dot(Point{r.X0, r.Y1})
	x0 := math.Min(p0.X, math.Min(p1.X, math.Min(p2.X, p3.X)))
	y0 := math.Min(p0.Y, math.Min(p1.Y, math.Min(p2.Y, p3.Y)))
	x1 := math.Max(p0.X, math.Max(p1.X, math.Max(p2.X, p3.X)))
	y1 := math.Max(p0.Y, math.Max(p1.Y, math.Max(p2.Y, p3.Y)))
	return Rect{x0, y0, x1, y1}
}

// Add returns a rect that encompasses both the current rect and the given rect.
func (r Rect) Add(q Rect) Rect {
	x0 := math.Min(r.X0, q.X0)
	y0 := math.Min(r.Y0, q.Y0)
	x1 := math.Max(r.X1, q.X1)
	y1 := math.Max(r.Y1, q.Y1)
	return Rect{x0, y0, x1, y1}
}

// AddPoint returns a rect that encompasses both the current rect and the given point.
func (r Rect) AddPoint(p Point) Rect {
	x0 := math.Min(r.X0, p.X)
	y0 := math.Min(r.Y0, p.Y)
	x1 := math.Max(r.X1, p.X)
	y1 := math.Max(r.Y1, p.Y)
	return Rect{x0, y0, x1, y1}
}

// Expand expands the rectangle.
func (r Rect) Expand(d float64) Rect {
	r.X0 -= d
	r.Y0 -= d
	r.X1 += d
	r.Y1 += d
	return r
}

// ContainsPoint returns true if the rectangle contains or touches an edge.
func (r Rect) ContainsPoint(p Point) bool {
	return cohenSutherlandOutcode(r, p, 0.0) == 0
}

// TouchesPoint returns true if the rectangle touches a point (within +-Epsilon).
func (r Rect) TouchesPoint(p Point) bool {
	return Interval(p.X, r.X0, r.X1) && Interval(p.Y, r.Y0, r.Y1)
}

// ClosestPoint returns a point in the rectangle closest to the given point.
func (r Rect) ClosestPoint(p Point) Point {
	if r.X0 <= p.X && p.X <= r.X1 {
		if r.Y0 <= p.Y && p.Y <= r.Y1 {
			// inside
			return p
		} else if r.Y1 < p.Y {
			return Point{p.X, r.Y1}
		} else {
			return Point{p.X, r.Y0}
		}
	} else if r.X1 < p.X {
		if r.Y0 <= p.Y && p.Y <= r.Y1 {
			return Point{r.X1, p.Y}
		} else if r.Y1 < p.Y {
			return Point{r.X1, r.Y1}
		} else {
			return Point{r.X1, r.Y0}
		}
	} else {
		if r.Y0 <= p.Y && p.Y <= r.Y1 {
			return Point{r.X0, p.Y}
		} else if r.Y1 < p.Y {
			return Point{r.X0, r.Y1}
		} else {
			return Point{r.X0, r.Y0}
		}
	}
}

// DistanceToPoint returns the distance between the rectangle and a point.
func (r Rect) DistanceToPoint(p Point) float64 {
	var q Point
	if r.X0 <= p.X && p.X <= r.X1 {
		if r.Y0 <= p.Y && p.Y <= r.Y1 {
			// inside
			return 0.0
		} else if r.Y1 < p.Y {
			return p.Y - r.Y1
		} else {
			return r.Y0 - p.Y
		}
	} else if r.X1 < p.X {
		if r.Y0 <= p.Y && p.Y <= r.Y1 {
			return p.X - r.X1
		} else if r.Y1 < p.Y {
			q = Point{r.X1, r.Y1}
		} else {
			q = Point{r.X1, r.Y0}
		}
	} else {
		if r.Y0 <= p.Y && p.Y <= r.Y1 {
			return r.X0 - p.X
		} else if r.Y1 < p.Y {
			q = Point{r.X0, r.Y1}
		} else {
			q = Point{r.X0, r.Y0}
		}
	}
	return p.Sub(q).Length()
}

func (r Rect) ContainsLine(a, b Point) bool {
	return r.ContainsPoint(a) && r.ContainsPoint(b)
}

func (r Rect) OverlapsLine(a, b Point) bool {
	_, _, overlaps, _ := cohenSutherlandLineClip(r, a, b, 0.0)
	return overlaps
}

func (r Rect) TouchesLine(a, b Point) bool {
	_, _, _, touches := cohenSutherlandLineClip(r, a, b, Epsilon)
	return touches
}

// Contains returns true if r contains q.
func (r Rect) Contains(q Rect) bool {
	return r.X0 <= q.X0 && q.X1 <= r.X1 && r.Y0 <= q.Y0 && q.Y1 <= r.Y1
}

// Overlaps returns true if both rectangles overlap.
func (r Rect) Overlaps(q Rect) bool {
	if q.X1 <= r.X0 || r.X1 <= q.X0 {
		// left or right
		return false
	} else if q.Y1 <= r.Y0 || r.Y1 <= q.Y0 {
		// below or above
		return false
	}
	return true
}

// Touches returns true if both rectangles touch (or overlap).
func (r Rect) Touches(q Rect) bool {
	if q.X1+Epsilon < r.X0 || r.X1 < q.X0-Epsilon {
		// left or right
		return false
	} else if q.Y1+Epsilon < r.Y0 || r.Y1 < q.Y0-Epsilon {
		// below or above
		return false
	}
	return true
}

// And returns the rectangle that is the overlap of both.
func (r Rect) And(q Rect) Rect {
	x0 := math.Max(r.X0, q.X0)
	y0 := math.Max(r.Y0, q.Y0)
	x1 := math.Min(r.X1, q.X1)
	y1 := math.Min(r.Y1, q.Y1)
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return Rect{x0, y0, x1, y1}

}

// ToPath converts the rectangle to a path.
func (r Rect) ToPath() *Path {
	return Rectangle(r.X1-r.X0, r.Y1-r.Y0).Translate(r.X0, r.Y0)
}

// String returns a string representation of r such as "(xmin,ymin)-(xmax,ymax)".
func (r Rect) String() string {
	return fmt.Sprintf("(%g,%g)-(%g,%g)", r.X0, r.Y0, r.X1, r.Y1)
}

////////////////////////////////////////////////////////////////

// Matrix is used for affine transformations, which are transformations such as translation, scaling, reflection, rotation, shear stretching. See https://en.wikipedia.org/wiki/Affine_transformation#Image_transformation for an overview of the transformations. The affine transformation matrix contains all transformations in a matrix, where we can concatenate transformations to apply them sequentially. Be aware that concatenated transformations will be evaluated right-to-left! So that Identity.Rotate(30).Translate(20,0) will first translate 20 points horizontally and then rotate 30 degrees counter clockwise.
type Matrix [2][3]float64

// Identity is the identity affine transformation matrix, i.e. transforms any point to itself.
var Identity = Matrix{
	{1.0, 0.0, 0.0},
	{0.0, 1.0, 0.0},
}

// Mul multiplies the current matrix by the given matrix, i.e. combining transformations.
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

// Dot returns the dot product between the matrix and the given vector, i.e. applying the transformation.
func (m Matrix) Dot(p Point) Point {
	return Point{
		m[0][0]*p.X + m[0][1]*p.Y + m[0][2],
		m[1][0]*p.X + m[1][1]*p.Y + m[1][2],
	}
}

// Translate adds a translation in x and y.
func (m Matrix) Translate(x, y float64) Matrix {
	return m.Mul(Matrix{
		{1.0, 0.0, x},
		{0.0, 1.0, y},
	})
}

// Rotate adds a rotation transformation with rot in degree counter clockwise.
func (m Matrix) Rotate(rot float64) Matrix {
	// TODO: this accepts degrees but Point.Angle returns radians
	sintheta, costheta := math.Sincos(rot * math.Pi / 180.0)
	return m.Mul(Matrix{
		{costheta, -sintheta, 0.0},
		{sintheta, costheta, 0.0},
	})
}

// RotateAbout adds a rotation transformation about (x,y) with rot in degrees counter clockwise.
func (m Matrix) RotateAbout(rot, x, y float64) Matrix {
	// TODO: accept Point
	return m.Translate(x, y).Rotate(rot).Translate(-x, -y)
}

// Scale adds a scaling transformation in sx and sy. When scale is negative it will flip those axes.
func (m Matrix) Scale(sx, sy float64) Matrix {
	return m.Mul(Matrix{
		{sx, 0.0, 0.0},
		{0.0, sy, 0.0},
	})
}

// ScaleAbout adds a scaling transformation about (x,y) in sx and sy. When scale is negative it will flip those axes.
func (m Matrix) ScaleAbout(sx, sy, x, y float64) Matrix {
	// TODO: accept Point
	return m.Translate(x, y).Scale(sx, sy).Translate(-x, -y)
}

// Shear adds a shear transformation with sx the horizontal shear and sy the vertical shear.
func (m Matrix) Shear(sx, sy float64) Matrix {
	return m.Mul(Matrix{
		{1.0, sx, 0.0},
		{sy, 1.0, 0.0},
	})
}

// ShearAbout adds a shear transformation about (x,y) with sx the horizontal shear and sy the vertical shear.
func (m Matrix) ShearAbout(sx, sy, x, y float64) Matrix {
	return m.Translate(x, y).Shear(sx, sy).Translate(-x, -y)
}

// ReflectX adds a horizontal reflection transformation, i.e. Scale(-1,1).
func (m Matrix) ReflectX() Matrix {
	return m.Scale(-1.0, 1.0)
}

// ReflectXAbout adds a horizontal reflection transformation about x.
func (m Matrix) ReflectXAbout(x float64) Matrix {
	return m.Translate(x, 0.0).Scale(-1.0, 1.0).Translate(-x, 0.0)
}

// ReflectY adds a vertical reflection transformation, i.e. Scale(1,-1).
func (m Matrix) ReflectY() Matrix {
	return m.Scale(1.0, -1.0)
}

// ReflectYAbout adds a vertical reflection transformation about y.
func (m Matrix) ReflectYAbout(y float64) Matrix {
	return m.Translate(0.0, y).Scale(1.0, -1.0).Translate(0.0, -y)
}

// T returns the matrix transpose.
func (m Matrix) T() Matrix {
	m[0][1], m[1][0] = m[1][0], m[0][1]
	return m
}

// Det returns the matrix determinant.
func (m Matrix) Det() float64 {
	return m[0][0]*m[1][1] - m[0][1]*m[1][0]
}

// Inv returns the matrix inverse.
func (m Matrix) Inv() Matrix {
	det := m.Det()
	if Equal(det, 0.0) {
		panic("determinant of affine transformation matrix is zero")
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

// Eigen returns the matrix eigenvalues and eigenvectors. The first eigenvalue is related to the first eigenvector, and so for the second pair. Eigenvectors are normalized.
func (m Matrix) Eigen() (float64, float64, Point, Point) {
	if Equal(m[1][0], 0.0) && Equal(m[0][1], 0.0) {
		return m[0][0], m[1][1], Point{1.0, 0.0}, Point{0.0, 1.0}
	}

	lambda1, lambda2 := solveQuadraticFormula(1.0, -m[0][0]-m[1][1], m.Det())
	if math.IsNaN(lambda1) && math.IsNaN(lambda2) {
		// either m[0][0] or m[1][1] is NaN or the the affine matrix has no real eigenvalues
		return lambda1, lambda2, Point{}, Point{}
	} else if math.IsNaN(lambda2) {
		lambda2 = lambda1
	}

	// see http://www.math.harvard.edu/archive/21b_fall_04/exhibits/2dmatrices/index.html
	var v1, v2 Point
	if !Equal(m[1][0], 0.0) {
		v1 = Point{lambda1 - m[1][1], m[1][0]}.Norm(1.0)
		v2 = Point{lambda2 - m[1][1], m[1][0]}.Norm(1.0)
	} else if !Equal(m[0][1], 0.0) {
		v1 = Point{m[0][1], lambda1 - m[0][0]}.Norm(1.0)
		v2 = Point{m[0][1], lambda2 - m[0][0]}.Norm(1.0)
	}
	return lambda1, lambda2, v1, v2
}

// Pos extracts the translation component as (tx,ty).
func (m Matrix) Pos() (float64, float64) {
	return m[0][2], m[1][2]
}

// Decompose extracts the translation, rotation, scaling and rotation components (applied in the reverse order) as (tx, ty, theta, sx, sy, phi) with rotation counter clockwise. This corresponds to Identity.Translate(tx, ty).Rotate(phi).Scale(sx, sy).Rotate(theta).
func (m Matrix) Decompose() (float64, float64, float64, float64, float64, float64) {
	// see https://math.stackexchange.com/questions/861674/decompose-a-2d-arbitrary-transform-into-only-scaling-and-rotation
	E := (m[0][0] + m[1][1]) / 2.0
	F := (m[0][0] - m[1][1]) / 2.0
	G := (m[1][0] + m[0][1]) / 2.0
	H := (m[1][0] - m[0][1]) / 2.0

	Q, R := math.Sqrt(E*E+H*H), math.Sqrt(F*F+G*G)
	sx, sy := Q+R, Q-R

	a1, a2 := math.Atan2(G, F), math.Atan2(H, E)
	theta := (a2 - a1) / 2.0 * 180.0 / math.Pi
	phi := (a2 + a1) / 2.0 * 180.0 / math.Pi
	if Equal(sx, 1.0) && Equal(sy, 1.0) {
		theta += phi
		phi = 0.0
	}
	return m[0][2], m[1][2], phi, sx, sy, theta
}

// IsTranslation is true if the matrix consists of only translational components, i.e. no rotation, scaling, or skew transformations.
func (m Matrix) IsTranslation() bool {
	return Equal(m[0][0], 1.0) && Equal(m[0][1], 0.0) && Equal(m[1][0], 0.0) && Equal(m[1][1], 1.0)
}

// IsRigid is true if the matrix is orthogonal and consists of only translation, rotation, and reflection transformations.
func (m Matrix) IsRigid() bool {
	a := m[0][0]*m[0][0] + m[0][1]*m[0][1]
	b := m[1][0]*m[1][0] + m[1][1]*m[1][1]
	c := m[0][0]*m[1][0] + m[0][1]*m[1][1]
	return Equal(a, 1.0) && Equal(b, 1.0) && Equal(c, 0.0)
}

// IsSimilarity is true if the matrix consists of only translation, rotation, reflection, and scaling transformations.
func (m Matrix) IsSimilarity() bool {
	a := m[0][0]*m[0][0] + m[0][1]*m[0][1]
	b := m[1][0]*m[1][0] + m[1][1]*m[1][1]
	c := m[0][0]*m[1][0] + m[0][1]*m[1][1]
	return Equal(a, b) && Equal(c, 0.0)
}

// Equals returns true if both matrices are equal with a tolerance of Epsilon.
func (m Matrix) Equals(q Matrix) bool {
	return Equal(m[0][0], q[0][0]) && Equal(m[0][1], q[0][1]) && Equal(m[1][0], q[1][0]) && Equal(m[1][1], q[1][1]) && Equal(m[0][2], q[0][2]) && Equal(m[1][2], q[1][2])
}

// String returns a string representation of the affine transformation matrix as six values, where [a b c; d e f; g h i] will be written as "a b d e c f" as g, h and i have fixed values (0, 0 and 1 respectively).
func (m Matrix) String() string {
	return fmt.Sprintf("(%g %g; %g %g) + (%g,%g)", m[0][0], m[0][1], m[1][0], m[1][1], m[0][2], m[1][2])
}

// ToSVG writes out the matrix in SVG notation, taking care of the proper order of transformations.
func (m Matrix) ToSVG(h float64) string {
	s := &strings.Builder{}
	tx, ty, phi, sx, sy, theta := m.Decompose()
	if !Equal(m[0][2], 0.0) || !Equal(m[1][2], 0.0) {
		fmt.Fprintf(s, " translate(%v,%v)", dec(tx), dec(h-ty))
	}
	if !Equal(phi, 0.0) {
		fmt.Fprintf(s, " rotate(%v)", dec(-phi))
	}
	if !Equal(sx, 1.0) || !Equal(sy, 1.0) {
		fmt.Fprintf(s, " scale(%v,%v)", dec(sx), dec(sy))
	}
	if !Equal(theta, 0.0) {
		fmt.Fprintf(s, " rotate(%v)", dec(-theta))
	}

	matrix := fmt.Sprintf("matrix(%v,%v,%v,%v,%v,%v)", dec(m[0][0]), -dec(m[1][0]), -dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(h-m[1][2]))
	if s.Len() == 0 {
		return ""
	} else if len(matrix) < s.Len()-1 {
		return matrix
	}
	return s.String()[1:]
}

////////////////////////////////////////////////////////////////

func cohenSutherlandOutcode(rect Rect, p Point, eps float64) int {
	code := 0b0000
	if p.X < rect.X0-eps {
		code |= 0b0001 // left
	} else if rect.X1+eps < p.X {
		code |= 0b0010 // right
	}
	if p.Y < rect.Y0-eps {
		code |= 0b0100 // bottom
	} else if rect.Y1+eps < p.Y {
		code |= 0b1000 // top
	}
	return code
}

// return whether line is inside the rectangle, either entirely or partially.
func cohenSutherlandLineClip(rect Rect, a, b Point, eps float64) (Point, Point, bool, bool) {
	outcode0 := cohenSutherlandOutcode(rect, a, eps)
	outcode1 := cohenSutherlandOutcode(rect, b, eps)
	if outcode0 == 0 && outcode1 == 0 {
		return a, b, true, false
	}
	for {
		if (outcode0 | outcode1) == 0 {
			// both inside
			return a, b, true, true
		} else if (outcode0 & outcode1) != 0 {
			// both in same region outside
			return a, b, false, false
		}

		// pick point outside
		outcodeOut := outcode0
		if outcode0 < outcode1 {
			outcodeOut = outcode1
		}

		// intersect with rectangle
		var c Point
		if (outcodeOut & 0b1000) != 0 {
			// above
			c.X = a.X + (b.X-a.X)*(rect.Y1-a.Y)/(b.Y-a.Y)
			c.Y = rect.Y1
		} else if (outcodeOut & 0b0100) != 0 {
			// below
			c.X = a.X + (b.X-a.X)*(rect.Y0-a.Y)/(b.Y-a.Y)
			c.Y = rect.Y0
		} else if (outcodeOut & 0b0010) != 0 {
			// right
			c.X = rect.X1
			c.Y = a.Y + (b.Y-a.Y)*(rect.X1-a.X)/(b.X-a.X)
		} else if (outcodeOut & 0b0001) != 0 {
			// left
			c.X = rect.X0
			c.Y = a.Y + (b.Y-a.Y)*(rect.X0-a.X)/(b.X-a.X)
		}

		// prepare next pass
		if outcodeOut == outcode0 {
			outcode0 = cohenSutherlandOutcode(rect, c, eps)
			a = c
		} else {
			outcode1 = cohenSutherlandOutcode(rect, c, eps)
			b = c
		}
	}
}

// Numerically stable quadratic formula, lowest root is returned first, see https://math.stackexchange.com/a/2007723
func solveQuadraticFormula(a, b, c float64) (float64, float64) {
	if Equal(a, 0.0) {
		if Equal(b, 0.0) {
			if Equal(c, 0.0) {
				// all terms disappear, all x satisfy the solution
				return 0.0, math.NaN()
			}
			// linear term disappears, no solutions
			return math.NaN(), math.NaN()
		}
		// quadratic term disappears, solve linear equation
		return -c / b, math.NaN()
	}

	if Equal(c, 0.0) {
		// no constant term, one solution at zero and one from solving linearly
		if Equal(b, 0.0) {
			return 0.0, math.NaN()
		}
		return 0.0, -b / a
	}

	discriminant := b*b - 4.0*a*c
	if discriminant < 0.0 {
		return math.NaN(), math.NaN()
	} else if Equal(discriminant, 0.0) {
		return -b / (2.0 * a), math.NaN()
	}

	// Avoid catastrophic cancellation, which occurs when we subtract two nearly equal numbers and causes a large error. This can be the case when 4*a*c is small so that sqrt(discriminant) -> b, and the sign of b and in front of the radical are the same. Instead, we calculate x where b and the radical have different signs, and then use this result in the analytical equivalent of the formula, called the Citardauq Formula.
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

// see https://www.geometrictools.com/Documentation/LowDegreePolynomialRoots.pdf
// see https://github.com/thelonious/kld-polynomial/blob/development/lib/Polynomial.js
func solveCubicFormula(a, b, c, d float64) (float64, float64, float64) {
	var x1, x2, x3 float64
	x2, x3 = math.NaN(), math.NaN() // x1 is always set to a number below
	if Equal(a, 0.0) {
		x1, x2 = solveQuadraticFormula(b, c, d)
	} else {
		// obtain monic polynomial: x^3 + f.x^2 + g.x + h = 0
		b /= a
		c /= a
		d /= a

		// obtain depressed polynomial: x^3 + c1.x + c0
		bthird := b / 3.0
		c0 := d - bthird*(c-2.0*bthird*bthird)
		c1 := c - b*bthird
		if Equal(c0, 0.0) {
			if c1 < 0.0 {
				tmp := math.Sqrt(-c1)
				x1 = -tmp - bthird
				x2 = tmp - bthird
				x3 = 0.0 - bthird
			} else {
				x1 = 0.0 - bthird
			}
		} else if Equal(c1, 0.0) {
			if 0.0 < c0 {
				x1 = -math.Cbrt(c0) - bthird
			} else {
				x1 = math.Cbrt(-c0) - bthird
			}
		} else {
			delta := -(4.0*c1*c1*c1 + 27.0*c0*c0)
			if Equal(delta, 0.0) {
				delta = 0.0
			}

			if delta < 0.0 {
				betaRe := -c0 / 2.0
				betaIm := math.Sqrt(-delta / 108.0)
				tmp := betaRe - betaIm
				if 0.0 <= tmp {
					x1 = math.Cbrt(tmp)
				} else {
					x1 = -math.Cbrt(-tmp)
				}
				tmp = betaRe + betaIm
				if 0.0 <= tmp {
					x1 += math.Cbrt(tmp)
				} else {
					x1 -= math.Cbrt(-tmp)
				}
				x1 -= bthird
			} else if 0.0 < delta {
				betaRe := -c0 / 2.0
				betaIm := math.Sqrt(delta / 108.0)
				theta := math.Atan2(betaIm, betaRe) / 3.0
				sintheta, costheta := math.Sincos(theta)
				distance := math.Sqrt(-c1 / 3.0) // same as rhoPowThird
				tmp := distance * sintheta * math.Sqrt(3.0)
				x1 = 2.0*distance*costheta - bthird
				x2 = -distance*costheta - tmp - bthird
				x3 = -distance*costheta + tmp - bthird
			} else {
				tmp := -3.0 * c0 / (2.0 * c1)
				x1 = tmp - bthird
				x2 = -2.0*tmp - bthird
			}
		}
	}

	// sort
	if x3 < x2 || math.IsNaN(x2) {
		x2, x3 = x3, x2
	}
	if x2 < x1 || math.IsNaN(x1) {
		x1, x2 = x2, x1
	}
	if x3 < x2 || math.IsNaN(x2) {
		x2, x3 = x3, x2
	}
	return x1, x2, x3
}

type gaussLegendreFunc func(func(float64) float64, float64, float64) float64

// Gauss-Legendre quadrature integration from a to b with n=3, see https://pomax.github.io/bezierinfo/legendre-gauss.html for more values
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

//func lookupMin(f func(float64) float64, xmin, xmax float64) float64 {
//	const MaxIterations = 1000
//	min := math.Inf(1)
//	for i := 0; i <= MaxIterations; i++ {
//		t := float64(i) / float64(MaxIterations)
//		x := xmin + t*(xmax-xmin)
//		y := f(x)
//		if y < min {
//			min = y
//		}
//	}
//	return min
//}
//
//func gradientDescent(f func(float64) float64, xmin, xmax float64) float64 {
//	const MaxIterations = 100
//	const Delta = 0.0001
//	const Rate = 0.01
//
//	x := (xmin + xmax) / 2.0
//	for i := 0; i < MaxIterations; i++ {
//		dydx := (f(x+Delta) - f(x-Delta)) / 2.0 / Delta
//		x -= Rate * dydx
//	}
//	return x
//}

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
}

// polynomialApprox returns a function y(x) that maps the parameter x [xmin,xmax] to the integral of fp. For a circle tmin and tmax would be 0 and 2PI respectively for example. It also returns the total length of the curve. Implemented using M. Walter, A. Fournier, Approximate Arc Length Parametrization, Anais do IX SIBGRAPHI, p. 143--150, 1996, see https://www.visgraf.impa.br/sibgrapi96/trabs/pdf/a14.pdf
//func polynomialApprox3(gaussLegendre gaussLegendreFunc, fp func(float64) float64, xmin, xmax float64) (func(float64) float64, float64) {
//	y1 := gaussLegendre(fp, xmin, xmin+(xmax-xmin)*1.0/3.0)
//	y2 := gaussLegendre(fp, xmin, xmin+(xmax-xmin)*2.0/3.0)
//	y3 := gaussLegendre(fp, xmin, xmax)
//
//	// We have four points on the y(x) curve at x0=0, x1=1/3, x2=2/3 and x3=1
//	// now obtain a polynomial that goes through these four points by solving the system of linear equations
//	// y(x) = a*x^3 + b*x^2 + c*x + d  (NB: y0 = d = 0)
//	// [y1; y2; y3] = [1/27, 1/9, 1/3;
//	//                 8/27, 4/9, 2/3;
//	//                    1,   1,   1] * [a; b; c]
//	//
//	// After inverting:
//	// [a; b; c] = 0.5 * [ 27, -27,  9;
//	//                    -45,  36, -9;
//	//                     18,  -9,  2] * [y1; y2; y3]
//	// NB: y0 = d = 0
//
//	a := 13.5*y1 - 13.5*y2 + 4.5*y3
//	b := -22.5*y1 + 18.0*y2 - 4.5*y3
//	c := 9.0*y1 - 4.5*y2 + y3
//	return func(x float64) float64 {
//		x = (x - xmin) / (xmax - xmin)
//		return a*x*x*x + b*x*x + c*x
//	}, math.Abs(y3)
//}
//
//// invPolynomialApprox does the opposite of polynomialApprox, it returns a function x(y) that maps the parameter y [f(xmin),f(xmax)] to x [xmin,xmax]
//func invPolynomialApprox3(gaussLegendre gaussLegendreFunc, fp func(float64) float64, xmin, xmax float64) (func(float64) float64, float64) {
//	f := func(t float64) float64 {
//		return math.Abs(gaussLegendre(fp, xmin, xmin+(xmax-xmin)*t))
//	}
//	f3 := f(1.0)
//	t1 := bisectionMethod(f, (1.0/3.0)*f3, 0.0, 1.0)
//	t2 := bisectionMethod(f, (2.0/3.0)*f3, 0.0, 1.0)
//	t3 := 1.0
//
//	// We have four points on the x(y) curve at y0=0, y1=1/3, y2=2/3 and y3=1
//	// now obtain a polynomial that goes through these four points by solving the system of linear equations
//	// x(y) = a*y^3 + b*y^2 + c*y + d  (NB: x0 = d = 0)
//	// [x1; x2; x3] = [1/27, 1/9, 1/3;
//	//                 8/27, 4/9, 2/3;
//	//                    1,   1,   1] * [a*y3^3; b*y3^2; c*y3]
//	//
//	// After inverting:
//	// [a*y3^3; b*y3^2; c*y3] = 0.5 * [ 27, -27,  9;
//	//                                 -45,  36, -9;
//	//                                  18,  -9,  2] * [x1; x2; x3]
//	// NB: x0 = d = 0
//
//	a := (27.0*t1 - 27.0*t2 + 9.0*t3) / (2.0 * f3 * f3 * f3)
//	b := (-45.0*t1 + 36.0*t2 - 9.0*t3) / (2.0 * f3 * f3)
//	c := (18.0*t1 - 9.0*t2 + 2.0*t3) / (2.0 * f3)
//	return func(f float64) float64 {
//		t := a*f*f*f + b*f*f + c*f
//		return xmin + (xmax-xmin)*t
//	}, f3
//}

func invSpeedPolynomialChebyshevApprox(N int, gaussLegendre gaussLegendreFunc, fp func(float64) float64, tmin, tmax float64) (func(float64) float64, float64) {
	// TODO: find better way to determine N. For Arc 10 seems fine, for some Quads 10 is too low, for Cube depending on inflection points is maybe not the best indicator
	// TODO: track efficiency, how many times is fp called? Does a look-up table make more sense?
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
		x = math.Min(xmax, math.Max(xmin, x))
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
