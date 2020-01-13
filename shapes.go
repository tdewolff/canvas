package canvas

import (
	"math"
)

// Rectangle returns a rectangle with width w and height h.
func Rectangle(w, h float64) *Path {
	if equal(w, 0.0) || equal(h, 0.0) {
		return &Path{}
	}

	p := &Path{}
	p.LineTo(w, 0.0)
	p.LineTo(w, h)
	p.LineTo(0.0, h)
	p.Close()
	return p
}

// RoundedRectangle returns a rectangle with width w and height h with rounded corners of radius r. A negative radius will cast the corners inwards (ie. concave).
func RoundedRectangle(w, h, r float64) *Path {
	if equal(w, 0.0) || equal(h, 0.0) {
		return &Path{}
	} else if equal(r, 0.0) {
		return Rectangle(w, h)
	}

	sweep := true
	if r < 0.0 {
		sweep = false
		r = -r
	}
	r = math.Min(r, w/2.0)
	r = math.Min(r, h/2.0)

	p := &Path{}
	p.MoveTo(0.0, r)
	p.ArcTo(r, r, 0.0, false, sweep, r, 0.0)
	p.LineTo(w-r, 0.0)
	p.ArcTo(r, r, 0.0, false, sweep, w, r)
	p.LineTo(w, h-r)
	p.ArcTo(r, r, 0.0, false, sweep, w-r, h)
	p.LineTo(r, h)
	p.ArcTo(r, r, 0.0, false, sweep, 0.0, h-r)
	p.Close()
	return p
}

// BeveledRectangle returns a rectangle with width w and height h with beveled corners at distance r from the corner.
func BeveledRectangle(w, h, r float64) *Path {
	if equal(w, 0.0) || equal(h, 0.0) {
		return &Path{}
	} else if equal(r, 0.0) {
		return Rectangle(w, h)
	}

	r = math.Abs(r)
	r = math.Min(r, w/2.0)
	r = math.Min(r, h/2.0)

	p := &Path{}
	p.MoveTo(0.0, r)
	p.LineTo(r, 0.0)
	p.LineTo(w-r, 0.0)
	p.LineTo(w, r)
	p.LineTo(w, h-r)
	p.LineTo(w-r, h)
	p.LineTo(r, h)
	p.LineTo(0.0, h-r)
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
	p.ArcTo(rx, ry, 0.0, false, true, -rx, 0.0)
	p.ArcTo(rx, ry, 0.0, false, true, rx, 0.0)
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

// TODO: Grid
