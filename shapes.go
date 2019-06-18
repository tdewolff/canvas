package canvas

import (
	"math"
)

// Rectangle returns a rectangle at x,y with width and height of w and h respectively.
func Rectangle(x, y, w, h float64) *Path {
	if equal(w, 0.0) || equal(h, 0.0) {
		return &Path{}
	}

	p := &Path{}
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
	return p
}

// RoundedRectangle returns a rectangle at x,y with width w and height h with rounded corners of radius r. A negative radius will cast the corners inwards (ie. concave).
func RoundedRectangle(x, y, w, h, r float64) *Path {
	if equal(w, 0.0) || equal(h, 0.0) {
		return &Path{}
	} else if equal(r, 0.0) {
		return Rectangle(x, y, w, h)
	}

	sweep := true
	if r < 0.0 {
		sweep = false
		r = -r
	}
	r = math.Min(r, w/2.0)
	r = math.Min(r, h/2.0)

	p := &Path{}
	p.MoveTo(x, y+r)
	p.ArcTo(r, r, 0.0, false, sweep, x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(r, r, 0.0, false, sweep, x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(r, r, 0.0, false, sweep, x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.ArcTo(r, r, 0.0, false, sweep, x, y+h-r)
	p.Close()
	return p
}

// BeveledRectangle returns a rectangle at x,y with width w and height h with beveled corners at distance r from the corner.
func BeveledRectangle(x, y, w, h, r float64) *Path {
	if equal(w, 0.0) || equal(h, 0.0) {
		return &Path{}
	} else if equal(r, 0.0) {
		return Rectangle(x, y, w, h)
	}

	r = math.Abs(r)
	r = math.Min(r, w/2.0)
	r = math.Min(r, h/2.0)

	p := &Path{}
	p.MoveTo(x, y+r)
	p.LineTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.LineTo(x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.LineTo(x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.LineTo(x, y+h-r)
	p.Close()
	return p
}

// Circle returns a circle with radius r.
func Circle(r float64) *Path {
	return Ellipse(r, r)
}

// Ellipse returns an ellipse with radii rx,ry.
func Ellipse(rx, ry float64) *Path {
	if equal(rx, 0.0) || equal(ry, 0.0) {
		return &Path{}
	}

	p := &Path{}
	p.MoveTo(rx, 0.0)
	p.ArcTo(rx, ry, 0.0, false, false, -rx, 0.0)
	p.ArcTo(rx, ry, 0.0, false, false, rx, 0.0)
	p.Close()
	return p
}

// RegularPolygon returns a regular polygon with radius r and rotation rot in degrees. It uses n vertices/edges, so when n approaches infinity this will return a path that approximates a circle. n must be 3 or more. The up boolean defines whether the first point will point north or not.
func RegularPolygon(n int, r float64, up bool) *Path {
	return RegularStarPolygon(n, 1, r, up)
}

// RegularStarPolygon returns a regular star polygon with radius r and rotation rot in degrees. It uses n vertices of density d. This will result in a self-intersection star in counter clockwise direction. If n/2 < d the star will be clockwise and if n and d are not coprime a regular polygon will be obtained, possible with multiple windings. n must be 3 or more and d 2 or more. The up boolean defines whether the first point will point north or not.
func RegularStarPolygon(n, d int, r float64, up bool) *Path {
	if n < 3 || d < 1 || n == d*2 || equal(r, 0.0) {
		return &Path{}
	}

	dtheta := 2.0 * math.Pi / float64(n)
	theta0 := 0.5 * math.Pi
	if !up {
		theta0 += dtheta / 2.0
	}

	p := &Path{}
	for i := 0; i == 0 || i%n != 0; i += d {
		theta := theta0 + float64(i)*dtheta
		sintheta, costheta := math.Sincos(theta)
		if i == 0 {
			p.MoveTo(r*costheta, r*sintheta)
		} else {
			p.LineTo(r*costheta, r*sintheta)
		}
	}
	p.Close()
	return p
}

// StarPolygon returns a star polygon of n points with alternating radius R and r. The up boolean defines whether the first point (true) or second point (false) will be pointing north.
func StarPolygon(n int, R, r float64, up bool) *Path {
	if n < 3 || equal(R, 0.0) || equal(r, 0.0) {
		return &Path{}
	}

	n *= 2
	dtheta := 2.0 * math.Pi / float64(n)
	theta0 := 0.5 * math.Pi
	if !up {
		theta0 += dtheta
	}

	p := &Path{}
	for i := 0; i < n; i++ {
		theta := theta0 + float64(i)*dtheta
		sintheta, costheta := math.Sincos(theta)
		if i == 0 {
			p.MoveTo(R*costheta, R*sintheta)
		} else if i%2 == 0 {
			p.LineTo(R*costheta, R*sintheta)
		} else {
			p.LineTo(r*costheta, r*sintheta)
		}
	}
	p.Close()
	return p
}
