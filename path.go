package canvas

import (
	"math"

	"github.com/tdewolff/strconv"
)

type PathCmd int

const (
	MoveToCmd PathCmd = iota
	LineToCmd
	QuadToCmd
	CubeToCmd
	ArcToCmd
	CloseCmd
)

type Path struct {
	cmds []PathCmd
	d    []float64
	x0   float64
	y0   float64
}

func (p *Path) IsEmpty() bool {
	return len(p.cmds) == 0
}

func (p *Path) Append(p2 *Path) {
	p.cmds = append(p.cmds, p2.cmds...)
	p.d = append(p.d, p2.d...)
}

func (p *Path) Pos() (float64, float64) {
	if len(p.cmds) > 0 && p.cmds[len(p.cmds)-1] == CloseCmd {
		return p.x0, p.y0
	}
	if len(p.d) > 1 {
		return p.d[len(p.d)-2], p.d[len(p.d)-1]
	}
	return 0.0, 0.0
}

func (p *Path) Translate(x, y float64) {
	i := 0
	for _, cmd := range p.cmds {
		switch cmd {
		case MoveToCmd, LineToCmd:
			p.d[i+0] += x
			p.d[i+1] += y
			i += 2
		case QuadToCmd:
			p.d[i+0] += x
			p.d[i+1] += y
			p.d[i+2] += x
			p.d[i+3] += y
			i += 4
		case CubeToCmd:
			p.d[i+0] += x
			p.d[i+1] += y
			p.d[i+2] += x
			p.d[i+3] += y
			p.d[i+4] += x
			p.d[i+5] += y
			i += 6
		case ArcToCmd:
			p.d[i+5] += x
			p.d[i+6] += y
			i += 7
		}
	}
}

////////////////////////////////////////////////////////////////

func (p *Path) MoveTo(x, y float64) {
	p.cmds = append(p.cmds, MoveToCmd)
	p.d = append(p.d, x, y)
	p.x0, p.y0 = x, y
}

func (p *Path) LineTo(x, y float64) {
	p.cmds = append(p.cmds, LineToCmd)
	p.d = append(p.d, x, y)
}

func (p *Path) QuadTo(x1, y1, x, y float64) {
	p.cmds = append(p.cmds, QuadToCmd)
	p.d = append(p.d, x1, y1, x, y)
}

func (p *Path) CubeTo(x1, y1, x2, y2, x, y float64) {
	p.cmds = append(p.cmds, CubeToCmd)
	p.d = append(p.d, x1, y1, x2, y2, x, y)
}

// ArcTo defines an arc with radii rx and ry, with rot the rotation with respect to the coordinate system,
// start and end are the start and end angles respectively of our arc, in degrees counter-clockwise from 3 o'clock.
func (p *Path) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) {
	p.cmds = append(p.cmds, ArcToCmd)
	flarge := 0.0
	if large {
		flarge = 1.0
	}
	fsweep := 0.0
	if sweep {
		fsweep = 1.0
	}
	p.d = append(p.d, rx, ry, rot, flarge, fsweep, x, y)
}

func (p *Path) Close() {
	p.cmds = append(p.cmds, CloseCmd)
}

////////////////////////////////////////////////////////////////

func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

func (p *Path) Ellipse(x, y, rx, ry float64) {
	p.MoveTo(x+rx, y)
	p.ArcTo(rx, ry, 0, false, false, x-rx, y)
	p.ArcTo(rx, ry, 0, false, false, x+rx, y)
	p.Close()
}

func arcToCenter(x1, y1, rx, ry, rot float64, large, sweep bool, x2, y2 float64) (float64, float64, float64, float64) {
	// see https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes
	// and http://commons.oreilly.com/wiki/index.php/SVG_Essentials/Paths#Technique:_Converting_from_Other_Arc_Formats
	if x1 == x2 && y1 == y2 {
		return x1, y1, 0, 0
	}

	rot *= math.Pi / 180.0
	x1p := math.Cos(rot)*(x1-x2)/2 + math.Sin(rot)*(y1-y2)/2
	y1p := -math.Sin(rot)*(x1-x2)/2 + math.Cos(rot)*(y1-y2)/2

	// reduce rouding errors
	raddiCheck := x1p*x1p/rx/rx + y1p*y1p/ry/ry
	if raddiCheck > 1.0 {
		rx *= math.Sqrt(raddiCheck)
		ry *= math.Sqrt(raddiCheck)
	}

	sq := (rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p) / (rx*rx*y1p*y1p + ry*ry*x1p*x1p)
	if sq < 0 {
		sq = 0
	}
	coef := math.Sqrt(sq)
	if large == sweep {
		coef = -coef
	}
	cxp := coef * rx * y1p / ry
	cyp := coef * -ry * x1p / rx
	cx := math.Cos(rot)*cxp - math.Sin(rot)*cyp + (x1+x2)/2
	cy := math.Sin(rot)*cxp + math.Cos(rot)*cyp + (y1+y2)/2

	// specify U and V vectors; theta = arccos(U*V / sqrt(U*U + V*V))
	ux := (x1p - cxp) / rx
	uy := (y1p - cyp) / ry
	vx := -(x1p + cxp) / rx
	vy := -(y1p + cyp) / ry

	theta := math.Acos(ux / math.Sqrt(ux*ux+uy*uy))
	if uy < 0 {
		theta = -theta
	}
	theta *= 180 / math.Pi

	delta := math.Acos((ux*vx + uy*vy) / math.Sqrt((ux*ux+uy*uy)*(vx*vx+vy*vy)))
	if ux*vy-uy*vx < 0 {
		delta = -delta
	}
	delta *= 180 / math.Pi
	if !sweep && delta > 0 {
		delta -= 360
	} else if sweep && delta < 0 {
		delta += 360
	}

	return cx, cy, theta, theta + delta
}

////////////////////////////////////////////////////////////////

func skipCommaWhitespace(path []byte) int {
	i := 0
	for i < len(path) && (path[i] == ' ' || path[i] == ',' || path[i] == '\n' || path[i] == '\r' || path[i] == '\t') {
		i++
	}
	return i
}

func parseNum(path []byte) (float64, int) {
	i := skipCommaWhitespace(path)
	f, n := strconv.ParseFloat(path[i:])
	return f, i + n
}

func ParseSVGPath(path []byte) *Path {
	p := &Path{}

	var prevCmd byte
	cpx, cpy := 0.0, 0.0 // control points

	i := 0
	for i < len(path) {
		i += skipCommaWhitespace(path[i:])
		cmd := prevCmd
		if path[i] >= 'A' {
			cmd = path[i]
			i++
		}
		x, y := p.Pos()
		switch cmd {
		case 'M', 'm':
			a, n := parseNum(path[i:])
			i += n
			b, n := parseNum(path[i:])
			i += n
			if cmd == 'm' {
				a += x
				b += y
			}
			p.MoveTo(a, b)
		case 'Z', 'z':
			p.Close()
		case 'L', 'l':
			a, n := parseNum(path[i:])
			i += n
			b, n := parseNum(path[i:])
			i += n
			if cmd == 'l' {
				a += x
				b += y
			}
			p.LineTo(a, b)
		case 'H', 'h':
			a, n := parseNum(path[i:])
			i += n
			if cmd == 'h' {
				a += x
			}
			p.LineTo(a, y)
		case 'V', 'v':
			b, n := parseNum(path[i:])
			i += n
			if cmd == 'v' {
				b += y
			}
			p.LineTo(x, b)
		case 'C', 'c':
			a, n := parseNum(path[i:])
			i += n
			b, n := parseNum(path[i:])
			i += n
			c, n := parseNum(path[i:])
			i += n
			d, n := parseNum(path[i:])
			i += n
			e, n := parseNum(path[i:])
			i += n
			f, n := parseNum(path[i:])
			i += n
			if cmd == 'c' {
				a += x
				b += y
				c += x
				d += y
				e += x
				f += y
			}
			p.CubeTo(a, b, c, d, e, f)
			cpx, cpy = c, d
		case 'S', 's':
			c, n := parseNum(path[i:])
			i += n
			d, n := parseNum(path[i:])
			i += n
			e, n := parseNum(path[i:])
			i += n
			f, n := parseNum(path[i:])
			i += n
			if cmd == 's' {
				c += x
				d += y
				e += x
				f += y
			}
			a, b := x, y
			if prevCmd == 'C' || prevCmd == 'c' || prevCmd == 'S' || prevCmd == 's' {
				a, b = 2*x-cpx, 2*y-cpy
			}
			p.CubeTo(a, b, c, d, e, f)
			cpx, cpy = c, d
		case 'Q', 'q':
			a, n := parseNum(path[i:])
			i += n
			b, n := parseNum(path[i:])
			i += n
			c, n := parseNum(path[i:])
			i += n
			d, n := parseNum(path[i:])
			i += n
			if cmd == 'q' {
				a += x
				b += y
				c += x
				d += y
			}
			p.QuadTo(a, b, c, d)
			cpx, cpy = a, b
		case 'T', 't':
			c, n := parseNum(path[i:])
			i += n
			d, n := parseNum(path[i:])
			i += n
			if cmd == 't' {
				c += x
				d += y
			}
			a, b := x, y
			if prevCmd == 'Q' || prevCmd == 'q' || prevCmd == 'T' || prevCmd == 't' {
				a, b = 2*x-cpx, 2*y-cpy
			}
			p.QuadTo(a, b, c, d)
			cpx, cpy = a, b
		case 'A', 'a':
			a, n := parseNum(path[i:])
			i += n
			b, n := parseNum(path[i:])
			i += n
			c, n := parseNum(path[i:])
			i += n
			d, n := parseNum(path[i:])
			i += n
			e, n := parseNum(path[i:])
			i += n
			f, n := parseNum(path[i:])
			i += n
			g, n := parseNum(path[i:])
			i += n
			if cmd == 'a' {
				f += x
				g += y
			}
			large := math.Abs(d-1.0) < 1e-10
			sweep := math.Abs(e-1.0) < 1e-10
			p.ArcTo(a, b, c, large, sweep, f, g)
		}
		prevCmd = cmd
	}
	return p
}
