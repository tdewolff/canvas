package canvas

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/tdewolff/parse/strconv"
)

// Tolerance is the maximum deviation from the original path in millimeters when e.g. flatting
var Tolerance = 0.01

// PathCmd specifies the path command.
const (
	NullCmd   = 0.0
	MoveToCmd = 1.0 << iota
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
	case CubeToCmd, ArcToCmd:
		return 7
	case NullCmd:
		return 0
	}
	panic("unknown path command")
}

func fromArcFlags(f float64) (bool, bool) {
	largeArc := (f == 1.0 || f == 3.0)
	sweep := (f == 2.0 || f == 3.0)
	return largeArc, sweep
}

func toArcFlags(largeArc, sweep bool) float64 {
	f := 0.0
	if largeArc {
		f += 1.0
	}
	if sweep {
		f += 2.0
	}
	return f
}

// Path defines a vector path in 2D using a series of connected commands (MoveTo, LineTo, QuadTo, CubeTo, ArcTo and Close).
// Each command consists of a number of float64 values (depending on the command) that fully define the action. The first value is the command itself (as a float64). The last two values are the end point position of the pen after the action (x,y). QuadTo defined one control point (x,y) in between, CubeTo defines two control points, and ArcTo defines (rx,ry,phi,largeArc+sweep) i.e. the radius in x and y, its rotation (in radians) and the largeArc and sweep booleans in one float64.
// Only valid commands are appended, so that LineTo has a non-zero length, QuadTo's and CubeTo's control point(s) don't (both) overlap with the start and end point, and ArcTo has non-zero radii and has non-zero length. For ArcTo we also make sure the angle is is in the range [0, 2*PI) and we scale the radii up if they appear too small to fit the arc.
type Path struct {
	d  []float64
	i0 int // index of last MoveTo
}

// Empty returns true if p is an empty path.
func (p *Path) Empty() bool {
	return len(p.d) == 0
}

// Copy returns a copy of p.
func (p *Path) Copy() *Path {
	q := &Path{}
	q.d = append(q.d, p.d...)
	q.i0 = p.i0
	return q
}

// Append appends path q to p.
func (p *Path) Append(q *Path) *Path {
	if len(q.d) == 0 {
		return p
	} else if len(p.d) > 0 && q.d[0] != MoveToCmd {
		p.MoveTo(0.0, 0.0)
	}
	p.d = append(p.d, q.d...)
	p.i0 = q.i0
	return p
}

// Join joins path q to p.
func (p *Path) Join(q *Path) *Path {
	if len(q.d) == 0 {
		return p
	} else if len(p.d) > 0 && q.d[0] == MoveToCmd {
		x0, y0 := p.d[len(p.d)-2], p.d[len(p.d)-1]
		x1, y1 := q.d[1], q.d[2]
		if equal(x0, x1) && equal(y0, y1) {
			q.d = q.d[3:]
		}
	}
	p.d = append(p.d, q.d...)
	p.i0 = q.i0
	return p
}

// Pos returns the current position of the path, which is the end point of the last command.
func (p *Path) Pos() (float64, float64) {
	if len(p.d) > 0 {
		return p.d[len(p.d)-2], p.d[len(p.d)-1]
	}
	return 0.0, 0.0
}

// StartPos returns the start point of the current path segment, ie. it returns the position of the last MoveTo command.
func (p *Path) StartPos() (float64, float64) {
	if len(p.d) > 0 && p.d[p.i0] == MoveToCmd {
		return p.d[p.i0+1], p.d[p.i0+2]
	}
	return 0.0, 0.0
}

////////////////////////////////////////////////////////////////

// MoveTo moves the path to x,y without connecting the path. It starts a new independent path segment.
// Multiple path segments can be useful when negating parts of a previous path by overlapping it
// with a path in the opposite direction.
func (p *Path) MoveTo(x, y float64) *Path {
	p.i0 = len(p.d)
	p.d = append(p.d, MoveToCmd, x, y)
	return p
}

// LineTo adds a linear path to x2,y2.
func (p *Path) LineTo(x2, y2 float64) *Path {
	x1, y1 := p.Pos()
	if equal(x1, x2) && equal(y1, y2) {
		return p
	}
	p.d = append(p.d, LineToCmd, x2, y2)
	return p
}

// Quadto adds a quadratic Bezier path with control point cpx,cpy and end point x2,y2.
func (p *Path) QuadTo(cpx, cpy, x2, y2 float64) *Path {
	x1, y1 := p.Pos()
	if equal(cpx, x1) && equal(cpy, y1) || equal(cpx, x2) && equal(cpy, y2) {
		return p.LineTo(x2, y2)
	}
	p.d = append(p.d, QuadToCmd, cpx, cpy, x2, y2)
	return p
}

// CubeTo adds a cubic Bezier path with control points cpx1,cpy1 and cpx2,cpy2 and end point x2,y2.
func (p *Path) CubeTo(cpx1, cpy1, cpx2, cpy2, x2, y2 float64) *Path {
	x1, y1 := p.Pos()
	if (equal(cpx1, x1) && equal(cpy1, y1) || equal(cpx1, x2) && equal(cpy1, y2)) &&
		(equal(cpx2, x1) && equal(cpy2, y1) || equal(cpx2, x2) && equal(cpy2, y2)) {
		return p.LineTo(x2, y2)
	}
	p.d = append(p.d, CubeToCmd, cpx1, cpy1, cpx2, cpy2, x2, y2)
	return p
}

// ArcTo adds an arc with radii rx and ry, with rot the rotation with respect to the coordinate system in degrees,
// large and sweep booleans (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs),
// and x2,y2 the end position of the pen. The start positions of the pen was given by a previous command.
func (p *Path) ArcTo(rx, ry, rot float64, largeArc, sweep bool, x2, y2 float64) *Path {
	x1, y1 := p.Pos()
	if equal(x1, x2) && equal(y1, y2) {
		return p
	}

	rot = math.Mod(rot, 360.0)
	if rot < 0.0 {
		rot += 360.0
	}
	if equal(rx, 0.0) || equal(ry, 0.0) {
		return p.LineTo(x2, y2)
	}
	rx = math.Abs(rx)
	ry = math.Abs(ry)

	// scale ellipse if rx and ry are too small, see https://www.w3.org/TR/SVG/implnote.html#ArcCorrectionOutOfRangeRadii
	phi := rot * math.Pi / 180.0
	x1p := (math.Cos(phi)*(x1-x2) + math.Sin(phi)*(y1-y2)) / 2.0
	y1p := (math.Cos(phi)*(y1-y2) - math.Sin(phi)*(x1-x2)) / 2.0
	lambda := x1p*x1p/rx/rx + y1p*y1p/ry/ry
	if lambda > 1.0 {
		rx = math.Sqrt(lambda) * rx
		ry = math.Sqrt(lambda) * ry
	}

	p.d = append(p.d, ArcToCmd, rx, ry, rot, toArcFlags(largeArc, sweep), x2, y2)
	return p
}

// Close closes a path with a LineTo to the start of the path (the most recent MoveTo command).
// It also signals the path closes, as opposed to being just a LineTo command.
func (p *Path) Close() *Path {
	x0, y0 := p.StartPos()
	p.d = append(p.d, CloseCmd, x0, y0)
	return p
}

////////////////////////////////////////////////////////////////

// Rectangle returns a rectangle at x,y with width and height of w and h respectively.
func Rectangle(x, y, w, h float64) *Path {
	p := &Path{}
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
	return p
}

// Ellipse returns an ellipse at x,y with radii rx,ry.
func Ellipse(x, y, rx, ry float64) *Path {
	p := &Path{}
	p.MoveTo(x+rx, y)
	p.ArcTo(rx, ry, 0, false, false, x-rx, y)
	p.ArcTo(rx, ry, 0, false, false, x+rx, y)
	p.Close()
	return p
}

////////////////////////////////////////////////////////////////

// direction returns the angle the last path segment makes (ie. positive or negative) to determine its direction.
// Approaches all commands to be linear, so does not incorporate the angle a path command makes.
func (p *Path) direction() float64 {
	theta := 0.0
	first := true
	var a0, a, b Point
	var start, end Point
	for i := p.i0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)

		end = Point{p.d[i-2], p.d[i-1]}
		b = end.Sub(start)
		if first {
			a0 = b
			first = false
		} else {
			theta += a.Angle(b)
		}
		start = end
		a = b
	}
	if !first {
		theta += b.Angle(a0)
	}
	return theta
}

// CW returns true when the last path segment has a clockwise direction.
func (p *Path) CW() bool {
	return p.direction() < 0.0
}

// CCW returns true when the last path segment has a counter clockwise direction.
func (p *Path) CCW() bool {
	return p.direction() > 0.0
}

// Bounds returns the bounding box rectangle of the path.
func (p *Path) Bounds() Rect {
	xmin, xmax := math.Inf(1), math.Inf(-1)
	ymin, ymax := math.Inf(1), math.Inf(-1)

	if len(p.d) > 0 && p.d[0] != MoveToCmd {
		xmin = 0.0
		xmax = 0.0
		ymin = 0.0
		ymax = 0.0
	}

	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
		case LineToCmd, CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
		case QuadToCmd:
			c := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}

			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			tdenom := (start.X - 2*c.X + end.X)
			if tdenom != 0.0 {
				t := (start.X - c.X) / tdenom
				x := quadraticBezierAt(start, c, end, t)
				xmin = math.Min(xmin, x.X)
				xmax = math.Max(xmax, x.X)
			}

			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
			tdenom = (start.Y - 2*c.Y + end.Y)
			if tdenom != 0.0 {
				t := (start.Y - c.Y) / tdenom
				y := quadraticBezierAt(start, c, end, t)
				ymin = math.Min(ymin, y.Y)
				ymax = math.Max(ymax, y.Y)
			}
		case CubeToCmd:
			c1 := Point{p.d[i+1], p.d[i+2]}
			c2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}

			a := -start.X + 3*c1.X - 3*c2.X + end.X
			b := 2*start.X - 4*c1.X + 2*c2.X
			c := -start.X + c1.X
			t1, t2 := solveQuadraticFormula(a, b, c)

			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			if !math.IsNaN(t1) {
				x1 := cubicBezierAt(start, c1, c2, end, t1)
				xmin = math.Min(xmin, x1.X)
				xmax = math.Max(xmax, x1.X)
			}
			if !math.IsNaN(t2) {
				x2 := cubicBezierAt(start, c1, c2, end, t2)
				xmin = math.Min(xmin, x2.X)
				xmax = math.Max(xmax, x2.X)
			}

			a = -start.Y + 3*c1.Y - 3*c2.Y + end.Y
			b = 2*start.Y - 4*c1.Y + 2*c2.Y
			c = -start.Y + c1.Y
			t1, t2 = solveQuadraticFormula(a, b, c)

			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
			if !math.IsNaN(t1) {
				y1 := cubicBezierAt(start, c1, c2, end, t1)
				ymin = math.Min(ymin, y1.Y)
				ymax = math.Max(ymax, y1.Y)
			}
			if !math.IsNaN(t2) {
				y2 := cubicBezierAt(start, c1, c2, end, t2)
				ymin = math.Min(ymin, y2.Y)
				ymax = math.Max(ymax, y2.Y)
			}
		case ArcToCmd:
			rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			cx, cy, angle0, angle1 := ellipseToCenter(start.X, start.Y, rx, ry, rot, largeArc, sweep, end.X, end.Y)

			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)

			rot *= math.Pi / 180.0
			cos := math.Cos(rot)
			sin := math.Sin(rot)

			tx := -math.Atan(ry / rx * math.Tan(rot))
			ty := math.Atan(ry / rx / math.Tan(rot))

			dx := math.Abs(rx*math.Cos(tx)*cos - ry*math.Sin(tx)*sin)
			dy := math.Abs(rx*math.Cos(ty)*sin + ry*math.Sin(ty)*cos)

			tx *= 180.0 / math.Pi
			ty *= 180.0 / math.Pi

			if angle1 < angle0 {
				angle0, angle1 = angle1, angle0
			}

			if angle0 < tx && tx < angle1 {
				xmin = math.Min(xmin, cx-dx)
			}
			if angle0 < tx+180.0 && tx+180.0 < angle1 {
				xmax = math.Max(xmax, cx+dx)
			}
			// y is inverted
			if angle0 < ty && ty < angle1 {
				ymax = math.Max(ymax, cy+dy)
			}
			if angle0 < ty+180.0 && ty+180.0 < angle1 {
				ymin = math.Min(ymin, cy-dy)
			}
		}
		i += cmdLen(cmd)
		start = end
	}
	return Rect{xmin, ymin, xmax - xmin, ymax - ymin}
}

// Length returns the length of the path in millimeters. The length is approximated for cubic Béziers.
func (p *Path) Length() float64 {
	d := 0.0
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
		case LineToCmd, CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			d += end.Sub(start).Length()
		case QuadToCmd:
			c := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			c1, c2 := quadraticToCubicBezier(start, c, end)
			d += cubicBezierLength(start, c1, c2, end)
		case CubeToCmd:
			c1 := Point{p.d[i+1], p.d[i+2]}
			c2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			d += cubicBezierLength(start, c1, c2, end)
		case ArcToCmd:
			rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			d += ellipseLength(start, rx, ry, rot, largeArc, sweep, end)
		}
		i += cmdLen(cmd)
		start = end
	}
	return d
}

// Translate translates the path by (x,y).
func (p *Path) Translate(x, y float64) *Path {
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
			p.d[i+5] += x
			p.d[i+6] += y
		}
		i += cmdLen(cmd)
	}
	return p
}

// Scale scales the path by (x,y).
func (p *Path) Scale(x, y float64) *Path {
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd, LineToCmd, CloseCmd:
			p.d[i+1] *= x
			p.d[i+2] *= y
		case QuadToCmd:
			p.d[i+1] *= x
			p.d[i+2] *= y
			p.d[i+3] *= x
			p.d[i+4] *= y
		case CubeToCmd:
			p.d[i+1] *= x
			p.d[i+2] *= y
			p.d[i+3] *= x
			p.d[i+4] *= y
			p.d[i+5] *= x
			p.d[i+6] *= y
		case ArcToCmd:
			p.d[i+1] *= math.Abs(x)
			p.d[i+2] *= math.Abs(y)
			largeArc, sweep := fromArcFlags(p.d[i+4])
			if x*y < 0.0 {
				p.d[i+3] *= -1.0
				sweep = !sweep
			}
			p.d[i+4] = toArcFlags(largeArc, sweep)
			p.d[i+5] *= x
			p.d[i+6] *= y
		}
		i += cmdLen(cmd)
	}
	return p
}

// Rotate rotates the path by rot in degrees around point (x,y).
func (p *Path) Rotate(rot, x, y float64) *Path {
	mid := Point{x, y}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd, LineToCmd, CloseCmd:
			end := Point{p.d[i+1], p.d[i+2]}.Rot(rot, mid)
			p.d[i+1] = end.X
			p.d[i+2] = end.Y
		case QuadToCmd:
			c := Point{p.d[i+1], p.d[i+2]}.Rot(rot, mid)
			end := Point{p.d[i+3], p.d[i+4]}.Rot(rot, mid)
			p.d[i+1] = c.X
			p.d[i+2] = c.Y
			p.d[i+3] = end.X
			p.d[i+4] = end.Y
		case CubeToCmd:
			c1 := Point{p.d[i+1], p.d[i+2]}.Rot(rot, mid)
			c2 := Point{p.d[i+3], p.d[i+4]}.Rot(rot, mid)
			end := Point{p.d[i+5], p.d[i+6]}.Rot(rot, mid)
			p.d[i+1] = c1.X
			p.d[i+2] = c1.Y
			p.d[i+3] = c2.X
			p.d[i+4] = c2.Y
			p.d[i+5] = end.X
			p.d[i+6] = end.Y
		case ArcToCmd:
			end := Point{p.d[i+5], p.d[i+6]}.Rot(rot, mid)
			p.d[i+5] = end.X
			p.d[i+6] = end.Y
		}
		i += cmdLen(cmd)
	}
	return p
}

// Flatten flattens all Bézier and arc curves into linear segments. It uses Tolerance as the maximum deviation.
func (p *Path) Flatten() *Path {
	return p.Replace(nil, flattenCubicBezier, flattenEllipse)
}

type LineReplacer func(Point, Point) *Path
type BezierReplacer func(Point, Point, Point, Point) *Path
type ArcReplacer func(Point, float64, float64, float64, bool, bool, Point) *Path

func (p *Path) Replace(line LineReplacer, bezier BezierReplacer, arc ArcReplacer) *Path {
	start := Point{}
	for i := 0; i < len(p.d); {
		var q *Path
		cmd := p.d[i]
		switch cmd {
		case LineToCmd, CloseCmd:
			if line != nil {
				end := Point{p.d[i+1], p.d[i+2]}
				q = line(start, end)
				if cmd == CloseCmd {
					q.Close()
				}
			}
		case QuadToCmd:
			if bezier != nil {
				c := Point{p.d[i+1], p.d[i+2]}
				end := Point{p.d[i+3], p.d[i+4]}
				c1, c2 := quadraticToCubicBezier(start, c, end)
				q = bezier(start, c1, c2, end)
			}
		case CubeToCmd:
			if bezier != nil {
				c1 := Point{p.d[i+1], p.d[i+2]}
				c2 := Point{p.d[i+3], p.d[i+4]}
				end := Point{p.d[i+5], p.d[i+6]}
				q = bezier(start, c1, c2, end)
			}
		case ArcToCmd:
			if arc != nil {
				rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]
				largeArc, sweep := fromArcFlags(p.d[i+4])
				end := Point{p.d[i+5], p.d[i+6]}
				q = arc(start, rx, ry, rot, largeArc, sweep, end)
			}
		}

		if q != nil {
			p.d = append(p.d[:i], append(q.d, p.d[i+cmdLen(cmd):]...)...)
			i += len(q.d)
			if q.Empty() {
				continue
			}
		} else {
			i += cmdLen(cmd)
		}
		start = Point{p.d[i-2], p.d[i-1]}
	}
	return p
}

// Split splits the path into its independent path segments. The path is split on the MoveTo and/or Close commands.
func (p *Path) Split() []*Path {
	ps := []*Path{}
	closed := false
	var i, j, i0 int
	for j < len(p.d) {
		cmd := p.d[j]
		if j > i && cmd == MoveToCmd || closed {
			ps = append(ps, &Path{p.d[i:j], i0})
			i = j
			closed = false
		}
		switch cmd {
		case MoveToCmd:
			i0 = j
		case CloseCmd:
			closed = true
		}
		j += cmdLen(cmd)
	}
	if j > i {
		ps = append(ps, &Path{p.d[i:j], i0})
	}
	return ps
}

// SplitAt splits the path into seperate paths at the specified intervals (given in millimeters) along the path.
func (p *Path) SplitAt(ts ...float64) []*Path {
	if len(ts) == 0 {
		return []*Path{}
	}

	sort.Float64s(ts)
	if ts[0] == 0.0 {
		ts = ts[1:]
	}

	j := 0   // index into T
	T := 0.0 // current position along curve

	p = p.Replace(nil, nil, flattenEllipse)

	qs := []*Path{}
	q := &Path{}
	push := func() {
		qs = append(qs, q)
		q = &Path{}
	}

	if len(p.d) > 0 && p.d[0] == MoveToCmd {
		q.MoveTo(p.d[1], p.d[2])
	}
	for _, ps := range p.Split() {
		var start, end Point
		for i := 0; i < len(ps.d); {
			cmd := ps.d[i]
			switch cmd {
			case MoveToCmd:
				end = Point{p.d[i+1], p.d[i+2]}
			case LineToCmd, CloseCmd:
				end = Point{p.d[i+1], p.d[i+2]}

				if j == len(ts) {
					q.LineTo(end.X, end.Y)
				} else {
					dT := end.Sub(start).Length()

					Tcurve := T
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						tpos := (ts[j] - T) / dT
						pos := start.Interpolate(end, tpos)
						Tcurve = ts[j]

						q.LineTo(pos.X, pos.Y)
						push()
						q.MoveTo(pos.X, pos.Y)
						j++
					}
					if Tcurve < T+dT {
						q.LineTo(end.X, end.Y)
					}
					T += dT
				}
			case QuadToCmd, CubeToCmd:
				var c1, c2 Point
				if cmd == QuadToCmd {
					c := Point{p.d[i+1], p.d[i+2]}
					c1, c2 = quadraticToCubicBezier(start, c, end)
					end = Point{p.d[i+3], p.d[i+4]}
				} else {
					c1 = Point{p.d[i+1], p.d[i+2]}
					c2 = Point{p.d[i+3], p.d[i+4]}
					end = Point{p.d[i+5], p.d[i+6]}
				}

				if j == len(ts) {
					q.CubeTo(c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y)
				} else {
					dT := cubicBezierLength(start, c1, c2, end)

					Tcurve, dTcurve := T, dT
					r0, r1, r2, r3 := start, c1, c2, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						tpos := (ts[j] - Tcurve) / dTcurve
						_, c1, c2, _, r0, r1, r2, r3 = splitCubicBezier(r0, r1, r2, r3, tpos)
						dTcurve = dT - (ts[j] - T)
						Tcurve = ts[j]

						q.CubeTo(c1.X, c1.Y, c2.X, c2.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if Tcurve < T+dT {
						q.CubeTo(r1.X, r1.Y, r2.X, r2.Y, r3.X, r3.Y)
					}
					T += dT
				}
			case ArcToCmd:
				panic("arcs should have been replaced")
				// TODO: implement
			}
			i += cmdLen(cmd)
			start = end
		}
	}
	if len(q.d) > 3 {
		qs = append(qs, q)
	}
	return qs
}

// Dash returns a new path that consists of dashes. Each parameter represents a length in millimeters along the original path, and will be either a dash or a space alternatingly.
func (p *Path) Dash(d ...float64) *Path {
	p = p.Replace(nil, nil, flattenEllipse) // TODO: replaces ellipses twice, also in SplitAt, bad?

	length := p.Length()
	if len(d) == 0 || length <= d[0] {
		return p
	}

	i := 0 // index in d
	pos := 0.0
	t := []float64{}
	for pos < length {
		if len(d) <= i {
			i = 0
		}
		pos += d[i]
		t = append(t, pos)
		i++
	}

	ps := p.SplitAt(t...)
	q := &Path{}
	for j := 0; j < len(ps); j += 2 {
		q.Append(ps[j])
	}
	return q
}

// Reverse returns a new path that is the same path as p but in the reverse direction.
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
			rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			ip.ArcTo(rx, ry, rot, largeArc, !sweep, end.X, end.Y)
		}
		start = end
	}
	if closed {
		ip.Close()
	}
	return ip
}

func (p *Path) Optimize() *Path {
	var start, end Point
	var prevCmd float64
	// TODO: run reverse
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			if len(p.d) > i+3 && p.d[i+3] == MoveToCmd || i == 0 && end.X == 0.0 && end.Y == 0.0 {
				p.d = append(p.d[:i], p.d[i+3:]...)
				cmd = NullCmd
			}
			// TODO: remove MoveTo + Close combination
		case LineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			if start == end {
				p.d = append(p.d[:i], p.d[i+3:]...)
				cmd = NullCmd
			}
			// TODO: remove if followed by Close with the same end point
		case CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			if prevCmd == NullCmd || prevCmd == CloseCmd {
				p.d = append(p.d[:i], p.d[i+3:]...)
				cmd = NullCmd
			}
		case QuadToCmd:
			c := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			if c == start || c == end {
				p.d = append(p.d[:i+1], p.d[i+3:]...)
				p.d[i] = LineToCmd
				cmd = LineToCmd
			}
		case CubeToCmd:
			c1 := Point{p.d[i+1], p.d[i+2]}
			c2 := Point{p.d[i+3], p.d[i+4]}
			end := Point{p.d[i+5], p.d[i+6]}
			if (c1 == start || c1 == end) && (c2 == start || c2 == end) {
				p.d = append(p.d[:i+1], p.d[i+5:]...)
				p.d[i] = LineToCmd
				cmd = LineToCmd
			}
		case ArcToCmd:
			end = Point{p.d[i+5], p.d[i+6]}
			if start == end {
				p.d = append(p.d[:i], p.d[i+7:]...)
				cmd = NullCmd
			}
		}
		if cmd == LineToCmd {
			// TODO: combine line elements that have the same direction
		}
		i += cmdLen(cmd)
		start = end
		prevCmd = cmd
	}
	return p
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
func ParseSVGPath(s string) (*Path, error) {
	// TODO: add error handling
	if len(s) == 0 {
		return &Path{}, nil
	}

	path := []byte(s)
	if path[0] < 'A' {
		return nil, fmt.Errorf("bad path: does not start with command")
	}

	var prevCmd byte
	cpx, cpy := 0.0, 0.0 // control points

	i := 0
	p := &Path{}
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
			p.ArcTo(a, b, c, d == 1.0, e == 1.0, f, g)
		default:
			return nil, fmt.Errorf("unknown command in SVG path: %c", cmd)
		}
		prevCmd = cmd
	}
	return p, nil
}

// String returns a string that represents the path similar to the SVG path data format (but not necessarily valid SVG).
func (p *Path) String() string {
	sb := strings.Builder{}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			fmt.Fprintf(&sb, "M%.5g %.5g", p.d[i+1], p.d[i+2])
		case LineToCmd:
			fmt.Fprintf(&sb, "L%.5g %.5g", p.d[i+1], p.d[i+2])
		case QuadToCmd:
			fmt.Fprintf(&sb, "Q%.5g %.5g %.5g %.5g", p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4])
		case CubeToCmd:
			fmt.Fprintf(&sb, "C%.5g %.5g %.5g %.5g %.5g %.5g", p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4], p.d[i+5], p.d[i+6])
		case ArcToCmd:
			largeArc, sweep := fromArcFlags(p.d[i+4])
			sLargeArc := "0"
			if largeArc {
				sLargeArc = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			fmt.Fprintf(&sb, "A%.5g %.5g %.5g %s %s %.5g %.5g", p.d[i+1], p.d[i+2], p.d[i+3], sLargeArc, sSweep, p.d[i+5], p.d[i+6])
		case CloseCmd:
			fmt.Fprintf(&sb, "z")
		}
		i += cmdLen(cmd)
	}
	return sb.String()
}

// ToSVG returns a string that represents the path in the SVG path data format with minifications.
func (p *Path) ToSVG() string {
	sb := strings.Builder{}
	x, y := 0.0, 0.0
	if len(p.d) > 0 && p.d[0] != MoveToCmd {
		fmt.Fprintf(&sb, "M0 0")
	}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, "M%.5g %.5g", x, y)
		case LineToCmd:
			xStart, yStart := x, y
			x, y = p.d[i+1], p.d[i+2]
			if equal(x, xStart) && equal(y, yStart) {
				// nothing
			} else if equal(x, xStart) {
				fmt.Fprintf(&sb, "V%.5g", y)
			} else if equal(y, yStart) {
				fmt.Fprintf(&sb, "H%.5g", x)
			} else {
				fmt.Fprintf(&sb, "L%.5g %.5g", x, y)
			}
		case QuadToCmd:
			x, y = p.d[i+3], p.d[i+4]
			fmt.Fprintf(&sb, "Q%.5g %.5g %.5g %.5g", p.d[i+1], p.d[i+2], x, y)
		case CubeToCmd:
			x, y = p.d[i+5], p.d[i+6]
			fmt.Fprintf(&sb, "C%.5g %.5g %.5g %.5g %.5g %.5g", p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4], x, y)
		case ArcToCmd:
			x, y = p.d[i+5], p.d[i+6]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			sLargeArc := "0"
			if largeArc {
				sLargeArc = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			fmt.Fprintf(&sb, "A%.5g %.5g %.5g %s %s %.5g %.5g", p.d[i+1], p.d[i+2], p.d[i+3], sLargeArc, sSweep, p.d[i+5], p.d[i+6])
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, "z")
		}
		i += cmdLen(cmd)
	}
	return sb.String()
}

var psEllipseDef = `/ellipse {
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
xrad yrad scale
0 0 1 startangle endangle arc
savematrix setmatrix
} def /ellipsen {
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
xrad yrad scale
0 0 1 startangle endangle arcn
savematrix setmatrix
} def`

// ToPS returns a string that represents the path in the PostScript data format.
func (p *Path) ToPS() string {
	sb := strings.Builder{}
	if len(p.d) > 0 && p.d[0] != MoveToCmd {
		fmt.Fprintf(&sb, " 0 0 moveto")
	}

	var cmd float64
	x, y := 0.0, 0.0
	ellipsesDefined := false
	for i := 0; i < len(p.d); {
		cmd = p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %.5g %.5g moveto", x, y)
		case LineToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %.5g %.5g lineto", x, y)
		case QuadToCmd, CubeToCmd:
			var start, c1, c2 Point
			start = Point{x, y}
			if cmd == QuadToCmd {
				x, y = p.d[i+3], p.d[i+4]
				c1, c2 = quadraticToCubicBezier(start, Point{p.d[i+1], p.d[i+2]}, Point{x, y})
			} else {
				c1 = Point{p.d[i+1], p.d[i+2]}
				c2 = Point{p.d[i+3], p.d[i+4]}
				x, y = p.d[i+5], p.d[i+6]
			}
			fmt.Fprintf(&sb, " %.5g %.5g %.5g %.5g %.5g %.5g curveto", c1.X, c1.Y, c2.X, c2.Y, x, y)
		case ArcToCmd:
			x0, y0 := x, y
			rx, ry, rot := p.d[i+1], p.d[i+2], p.d[i+3]
			largeArc, sweep := fromArcFlags(p.d[i+4])
			x, y = p.d[i+5], p.d[i+6]

			isEllipse := !equal(rx, ry)
			if isEllipse && !ellipsesDefined {
				fmt.Fprintf(&sb, " %s", psEllipseDef)
				ellipsesDefined = true
			}

			cx, cy, theta0, theta1 := ellipseToCenter(x0, y0, rx, ry, rot, largeArc, sweep, x, y)
			if !equal(rot, 0.0) {
				fmt.Fprintf(&sb, " %.5g %.5g translate %.5g rotate %.5g %.5g translate", cx, cy, rot, -cx, -cy)
			}
			if isEllipse {
				fmt.Fprintf(&sb, " %.5g %.5g %.5g %.5g %.5g %.5g ellipse", cx, cy, rx, ry, theta0, theta1)
			} else {
				fmt.Fprintf(&sb, " %.5g %.5g %.5g %.5g %.5g arc", cx, cy, rx, theta0, theta1)
			}
			if !sweep {
				fmt.Fprintf(&sb, "n")
			}
			if !equal(rot, 0.0) {
				fmt.Fprintf(&sb, " initmatrix")
			}
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " closepath")
		}
		i += cmdLen(cmd)
	}
	if cmd != CloseCmd {
		fmt.Fprintf(&sb, " closepath")
	}
	fmt.Fprintf(&sb, " fill")
	return sb.String()[1:] // remove the first space
}

// ToPDF returns a string that represents the path in the PDF data format.
func (p *Path) ToPDF() string {
	p = p.Copy().Replace(nil, nil, ellipseToBeziers)

	sb := strings.Builder{}
	if len(p.d) > 0 && p.d[0] != MoveToCmd {
		fmt.Fprintf(&sb, " 0 0 m")
	}

	var cmd float64
	x, y := 0.0, 0.0
	for i := 0; i < len(p.d); {
		cmd = p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %.5g %.5g m", x, y)
		case LineToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %.5g %.5g l", x, y)
		case QuadToCmd, CubeToCmd:
			var start, c1, c2 Point
			start = Point{x, y}
			if cmd == QuadToCmd {
				x, y = p.d[i+3], p.d[i+4]
				c1, c2 = quadraticToCubicBezier(start, Point{p.d[i+1], p.d[i+2]}, Point{x, y})
			} else {
				c1 = Point{p.d[i+1], p.d[i+2]}
				c2 = Point{p.d[i+3], p.d[i+4]}
				x, y = p.d[i+5], p.d[i+6]
			}
			fmt.Fprintf(&sb, " %.5g %.5g %.5g %.5g %.5g %.5g c", c1.X, c1.Y, c2.X, c2.Y, x, y)
		case ArcToCmd:
			panic("arcs should have been replaced")
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " h")
		}
		i += cmdLen(cmd)
	}
	if cmd != CloseCmd {
		fmt.Fprintf(&sb, " h")
	}
	fmt.Fprintf(&sb, " f")
	return sb.String()[1:] // remove the first space
}
