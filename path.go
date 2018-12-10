package canvas

import (
	"math"
	"strings"

	"github.com/tdewolff/parse/strconv"
)

// PathCmd specifies the path command.
const (
	MoveToCmd float64 = 1.0 << iota
	LineToCmd
	QuadToCmd
	CubeToCmd
	ArcToCmd
	CloseCmd
)

// cmdLen returns the number of numbers the path command contains.
func cmdLen(cmd float64) int {
	switch cmd {
	case MoveToCmd, LineToCmd, CloseCmd:
		return 3
	case QuadToCmd:
		return 5
	case CubeToCmd:
		return 7
	case ArcToCmd:
		return 8
	}
	panic("unknown path command")
}

// Path defines a vector path in 2D.
type Path struct {
	d      []float64
	x0, y0 float64 // coords of last MoveTo
}

// IsEmpty returns true if p is an empty path.
func (p *Path) IsEmpty() bool {
	return len(p.d) == 0
}

// Copy returns a copy of p.
func (p *Path) Copy() *Path {
	q := &Path{}
	q.d = append(q.d, p.d...)
	q.x0 = p.x0
	q.y0 = p.y0
	return q
}

// Append appends path q to p.
func (p *Path) Append(q *Path) *Path {
	if len(q.d) == 0 {
		return p
	}

	if len(p.d) > 0 {
		if q.d[0] == MoveToCmd {
			x0, y0 := p.d[len(p.d)-2], p.d[len(p.d)-1]
			x1, y1 := q.d[1], q.d[2]
			if Equal(x0, x1) && Equal(y0, y1) {
				q.d = q.d[3:]
			}
		} else {
			// q implicitly starts at 0,0
			p.d = append(p.d, MoveToCmd, 0.0, 0.0)
		}
	}

	p.d = append(p.d, q.d...)
	p.x0 = q.x0
	p.y0 = q.y0
	return p
}

// Pos returns the current position of the path, which is the end point of the last command.
func (p *Path) Pos() (float64, float64) {
	if len(p.d) > 1 {
		return p.d[len(p.d)-2], p.d[len(p.d)-1]
	}
	return 0.0, 0.0
}

// Start returns the start point of the current path segment, ie. it returns the position of the last MoveTo command.
func (p *Path) StartPos() (float64, float64) {
	return p.x0, p.y0
}

////////////////////////////////////////////////////////////////

// MoveTo moves the path to x,y without connecting the path. It starts a new independent path segment.
// Multiple path segments can be useful when negating parts of a previous path by overlapping it
// with a path in the opposite direction.
func (p *Path) MoveTo(x, y float64) {
	p.d = append(p.d, MoveToCmd, x, y)
	p.x0, p.y0 = x, y
}

// LineTo adds a linear path to x,y.
func (p *Path) LineTo(x, y float64) {
	p.d = append(p.d, LineToCmd, x, y)
}

// Quadto adds a quadratic Bezier path with control point x1,y1 and end point x,y.
func (p *Path) QuadTo(x1, y1, x, y float64) {
	p.d = append(p.d, QuadToCmd, x1, y1, x, y)
}

// CubeTo adds a cubic Bezier path with control points x1,y1 and x2,y2 and end point x,y.
func (p *Path) CubeTo(x1, y1, x2, y2, x, y float64) {
	p.d = append(p.d, CubeToCmd, x1, y1, x2, y2, x, y)
}

// ArcTo adds an arc with radii rx and ry, with rot the rotation with respect to the coordinate system,
// large and sweep booleans (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs),
// and x,y the end position of the pen. The start positions of the pen was given by a previous command.
func (p *Path) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) {
	flarge := 0.0
	if large {
		flarge = 1.0
	}
	fsweep := 0.0
	if sweep {
		fsweep = 1.0
	}
	p.d = append(p.d, ArcToCmd, rx, ry, rot, flarge, fsweep, x, y)
}

// Close closes a path with a LineTo to the start of the path (the most recent MoveTo command).
// It also signals the path closes, as opposed to being just a LineTo command.
func (p *Path) Close() {
	p.d = append(p.d, CloseCmd, p.x0, p.y0)
}

////////////////////////////////////////////////////////////////

// Rect returns a rectangle at x,y with width and height of w and h respectively.
func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// Ellipse returns an ellipse at x,y with radii rx,ry.
func (p *Path) Ellipse(x, y, rx, ry float64) {
	p.MoveTo(x+rx, y)
	p.ArcTo(rx, ry, 0, false, false, x-rx, y)
	p.ArcTo(rx, ry, 0, false, false, x+rx, y)
	p.Close()
}

////////////////////////////////////////////////////////////////

// Split splits the path into its independent path segments. The path is split on the MoveTo and/or Close commands.
func (p *Path) Split() []*Path {
	ps := []*Path{}
	closed := false
	var i, j int
	var x0, y0 float64
	for j < len(p.d) {
		cmd := p.d[j]
		if j > i && cmd == MoveToCmd || closed {
			ps = append(ps, &Path{p.d[i:j], x0, y0})
			i = j
			closed = false
		}
		switch cmd {
		case MoveToCmd:
			x0, y0 = p.d[j+1], p.d[j+2]
		case CloseCmd:
			closed = true
		}
		j += cmdLen(cmd)
	}
	if j > i {
		ps = append(ps, &Path{p.d[i:j], x0, y0})
	}
	return ps
}

// Translate returns a copy of p that has the entire path translated by x,y.
func (p *Path) Translate(x, y float64) *Path {
	p = p.Copy()
	if len(p.d) > 0 && p.d[0] != MoveToCmd {
		p.d = append([]float64{MoveToCmd, 0.0, 0.0}, p.d...)
	}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd, LineToCmd, CloseCmd:
			p.d[i+1] += x
			p.d[i+2] += y
		case QuadToCmd:
			p.d[i+1] += x
			p.d[i+2] += y
			p.d[i+3] += x
			p.d[i+4] += y
		case CubeToCmd:
			p.d[i+1] += x
			p.d[i+2] += y
			p.d[i+3] += x
			p.d[i+4] += y
			p.d[i+5] += x
			p.d[i+6] += y
		case ArcToCmd:
			p.d[i+6] += x
			p.d[i+7] += y
		}
		i += cmdLen(cmd)
	}
	return p
}

// FlattenBeziers will return a copy of p with all Bezier curves flattened.
// It replaces the curves by linear segments, under the constraint that the maximum deviation is up to tolerance.
func (p *Path) FlattenBeziers(tolerance float64) *Path {
	p = p.Copy()
	start := Point{}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case QuadToCmd:
			c := Point{p.d[i+1], p.d[i+2]}
			end := Point{p.d[i+3], p.d[i+4]}
			c1 := start.Interpolate(c, 2.0/3.0)
			c2 := end.Interpolate(c, 2.0/3.0)

			q := flattenCubicBezier(start, c1, c2, end, 0.0, tolerance)
			p.d = append(p.d[:i], append(q.d, p.d[i+5:]...)...)
			i += len(q.d)
			if len(q.d) == 0 {
				continue
			}
		case CubeToCmd:
			c1 := Point{p.d[i+1], p.d[i+2]}
			c2 := Point{p.d[i+3], p.d[i+4]}
			end := Point{p.d[i+5], p.d[i+6]}

			q := flattenCubicBezier(start, c1, c2, end, 0.0, tolerance)
			p.d = append(p.d[:i], append(q.d, p.d[i+7:]...)...)
			i += len(q.d)
			if len(q.d) == 0 {
				continue
			}
		default:
			i += cmdLen(cmd)
		}
		start = Point{p.d[i-2], p.d[i-1]}
	}
	return p
}

// Reverse returns a copy of p that is the same path but in the reverse direction.
func (p *Path) Reverse() *Path {
	ip := &Path{}
	if len(p.d) == 0 {
		return ip
	}

	cmds := []float64{}
	for i := 0; i < len(p.d); {
		cmds = append(cmds, p.d[i])
		i += cmdLen(p.d[i])
	}

	end := Point{p.d[len(p.d)-2], p.d[len(p.d)-1]}
	if !end.IsZero() {
		ip.MoveTo(end.X, end.Y)
	}
	start := end
	closed := false

	i := len(p.d)
	for icmd := len(cmds) - 1; icmd >= 0; icmd-- {
		cmd := cmds[icmd]
		i -= cmdLen(cmd)
		end = Point{}
		if i > 0 {
			end = Point{p.d[i-2], p.d[i-1]}
		}

		switch cmd {
		case CloseCmd:
			if !start.Equals(end) {
				ip.LineTo(end.X, end.Y)
			}
			closed = true
		case MoveToCmd:
			if closed {
				ip.Close()
				closed = false
			}
			if !end.IsZero() {
				ip.MoveTo(end.X, end.Y)
			}
		case LineToCmd:
			if closed && (icmd == 0 || cmds[icmd-1] == MoveToCmd) {
				ip.Close()
				closed = false
			} else {
				ip.LineTo(end.X, end.Y)
			}
		case QuadToCmd:
			x1, y1 := p.d[i+1], p.d[i+2]
			ip.QuadTo(x1, y1, end.X, end.Y)
		case CubeToCmd:
			x1, y1 := p.d[i+3], p.d[i+4]
			x2, y2 := p.d[i+1], p.d[i+2]
			ip.CubeTo(x1, y1, x2, y2, end.X, end.Y)
		case ArcToCmd:
			rx, ry := p.d[i+1], p.d[i+2]
			rot, largeArc, sweep := p.d[i+3], p.d[i+4], p.d[i+5]
			if sweep == 0.0 {
				sweep = 1.0
			} else {
				sweep = 0.0
			}
			ip.ArcTo(rx, ry, rot, largeArc == 1.0, sweep == 1.0, end.X, end.Y)
		}
		start = end
	}
	if closed {
		ip.Close()
	}
	return ip
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

// ParseSVGPath parses an SVG path data string.
func ParseSVGPath(sPath string) *Path {
	path := []byte(sPath)
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

func (p *Path) String() string {
	return p.ToSVGPath()
}

// ToSVGPath returns a string that represents the path in the SVG path data format.
func (p *Path) ToSVGPath() string {
	svg := strings.Builder{}
	x, y := 0.0, 0.0
	if len(p.d) > 0 && p.d[0] != MoveToCmd {
		svg.WriteString("M0 0")
	}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			svg.WriteString("M")
			svg.WriteString(ftos(x))
			svg.WriteString(" ")
			svg.WriteString(ftos(y))
		case LineToCmd:
			xStart, yStart := x, y
			x, y = p.d[i+1], p.d[i+2]
			if Equal(x, xStart) && Equal(y, yStart) {
				// nothing
			} else if Equal(x, xStart) {
				svg.WriteString("V")
				svg.WriteString(ftos(y))
			} else if Equal(y, yStart) {
				svg.WriteString("H")
				svg.WriteString(ftos(x))
			} else {
				svg.WriteString("L")
				svg.WriteString(ftos(x))
				svg.WriteString(" ")
				svg.WriteString(ftos(y))
			}
		case QuadToCmd:
			x, y = p.d[i+3], p.d[i+4]
			svg.WriteString("Q")
			svg.WriteString(ftos(p.d[i+1]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+2]))
			svg.WriteString(" ")
			svg.WriteString(ftos(x))
			svg.WriteString(" ")
			svg.WriteString(ftos(y))
		case CubeToCmd:
			x, y = p.d[i+5], p.d[i+6]
			svg.WriteString("C")
			svg.WriteString(ftos(p.d[i+1]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+2]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+3]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+4]))
			svg.WriteString(" ")
			svg.WriteString(ftos(x))
			svg.WriteString(" ")
			svg.WriteString(ftos(y))
		case ArcToCmd:
			x, y = p.d[i+6], p.d[i+7]
			svg.WriteString("A")
			svg.WriteString(ftos(p.d[i+1]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+2]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+3]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+4]))
			svg.WriteString(" ")
			svg.WriteString(ftos(p.d[i+5]))
			svg.WriteString(" ")
			svg.WriteString(ftos(x))
			svg.WriteString(" ")
			svg.WriteString(ftos(y))
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			svg.WriteString("z")
		}
		i += cmdLen(cmd)
	}
	return svg.String()
}
