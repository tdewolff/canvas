package canvas

import "math"
import "strconv"

const epsilon = 1e-10

func Equal(a, b float64) bool {
	return math.Abs(a - b) < epsilon
}

func ftos(f float64) string {
	return strconv.FormatFloat(f, 'g', 10, 64)
}

////////////////////////////////////////////////////////////////

// arcToCenter changes between the SVG arc format to the center and angles format
// see https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
// and http://commons.oreilly.com/wiki/index.php/SVG_Essentials/Paths#Technique:_Converting_from_Other_Arc_Formats
func arcToCenter(x1, y1, rx, ry, rot float64, large, sweep bool, x2, y2 float64) (float64, float64, float64, float64) {
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

func angleToNormal(theta float64) Point {
	theta *= math.Pi / 180.0
	y, x := math.Sincos(theta)
	return Point{x, y}
}
