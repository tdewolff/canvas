package canvas

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/tdewolff/parse/v2/strconv"
	"golang.org/x/image/vector"
)

// Tolerance is the maximum deviation from the original path in millimeters when e.g. flatting.
var Tolerance = 0.01

// FillRule is the algorithm to specify which area is to be filled and which not, in particular when multiple subpaths overlap. The NonZero rule is the default and will fill any point that is being enclosed by an unequal number of paths winding clockwise and counter clockwise, otherwise it will not be filled. The EvenOdd rule will fill any point that is being enclosed by an uneven number of paths, whichever their direction.
type FillRule int

// see FillRule
const (
	NonZero FillRule = iota
	EvenOdd
)

const (
	MoveToCmd = 1.0 << iota //  1.0
	LineToCmd               //  2.0
	QuadToCmd               //  4.0
	CubeToCmd               //  8.0
	ArcToCmd                // 16.0
	CloseCmd                // 32.0
)

// cmdLen returns the number of values (float64s) the path command contains.
func cmdLen(cmd float64) int {
	switch cmd {
	case MoveToCmd, LineToCmd, CloseCmd:
		return 4
	case QuadToCmd:
		return 6
	case CubeToCmd, ArcToCmd:
		return 8
	}
	panic(fmt.Sprintf("unknown path command '%f'", cmd))
}

// toArcFlags converts to the largeArc and sweep boolean flags given its value in the path.
func toArcFlags(f float64) (bool, bool) {
	large := (f == 1.0 || f == 3.0)
	sweep := (f == 2.0 || f == 3.0)
	return large, sweep
}

// fromArcFlags converts the largeArc and sweep boolean flags to a value stored in the path.
func fromArcFlags(large, sweep bool) float64 {
	f := 0.0
	if large {
		f += 1.0
	}
	if sweep {
		f += 2.0
	}
	return f
}

// Path defines a vector path in 2D using a series of commands (MoveTo, LineTo, QuadTo, CubeTo, ArcTo and Close). Each command consists of a number of float64 values (depending on the command) that fully define the action. The first value is the command itself (as a float64). The last two values is the end point position of the pen after the action (x,y). QuadTo defined one control point (x,y) in between, CubeTo defines two control points, and ArcTo defines (rx,ry,phi,large+sweep) i.e. the radius in x and y, its rotation (in radians) and the large and sweep booleans in one float64.
// Only valid commands are appended, so that LineTo has a non-zero length, QuadTo's and CubeTo's control point(s) don't (both) overlap with the start and end point, and ArcTo has non-zero radii and has non-zero length. For ArcTo we also make sure the angle is in the range [0, 2*PI) and we scale the radii up if they appear too small to fit the arc.
type Path struct {
	d []float64
	// TODO: optimization: cache bounds and path len until changes (clearCache()), set bounds directly for predefined shapes
}

// Empty returns true if p is an empty path or consists of only MoveTos and Closes.
func (p *Path) Empty() bool {
	return len(p.d) <= cmdLen(MoveToCmd)
}

// Equals returns true if p and q are equal within tolerance Epsilon.
func (p *Path) Equals(q *Path) bool {
	if len(p.d) != len(q.d) {
		return false
	}
	for i := 0; i < len(p.d); i++ {
		if !Equal(p.d[i], q.d[i]) {
			return false
		}
	}
	return true
}

// Closed returns true if the last subpath of p is a closed path.
func (p *Path) Closed() bool {
	return 0 < len(p.d) && p.d[len(p.d)-1] == CloseCmd
}

// Copy returns a copy of p.
func (p *Path) Copy() *Path {
	q := &Path{}
	q.d = append(q.d, p.d...)
	return q
}

// Append appends path q to p and returns a new path if successful (otherwise either p or q are returned).
func (p *Path) Append(q *Path) *Path {
	if q == nil || q.Empty() {
		return p
	} else if p.Empty() {
		return q
	}
	return &Path{append(p.d, q.d...)}
}

// Join joins path q to p and returns a new path if successful (otherwise either p or q are returned). It's like executing the commands in q to p in sequence, where if the first MoveTo of q doesn't coincide with p it will fallback to appending the paths.
func (p *Path) Join(q *Path) *Path {
	if q == nil || q.Empty() {
		return p
	} else if p.Empty() {
		return q
	}

	if !Equal(p.d[len(p.d)-3], q.d[1]) || !Equal(p.d[len(p.d)-2], q.d[2]) {
		p.d[len(p.d)-3] = (p.d[len(p.d)-3] + q.d[1]) / 2.0
		p.d[len(p.d)-2] = (p.d[len(p.d)-2] + q.d[2]) / 2.0
	}

	q.d = q.d[cmdLen(MoveToCmd):]

	// add the first command through the command functions to use the optimization features
	// q is not empty, so starts with a MoveTo followed by other commands
	cmd := q.d[0]
	switch cmd {
	case MoveToCmd:
		p.MoveTo(q.d[1], q.d[2])
	case LineToCmd:
		p.LineTo(q.d[1], q.d[2])
	case QuadToCmd:
		p.QuadTo(q.d[1], q.d[2], q.d[3], q.d[4])
	case CubeToCmd:
		p.CubeTo(q.d[1], q.d[2], q.d[3], q.d[4], q.d[5], q.d[6])
	case ArcToCmd:
		large, sweep := toArcFlags(q.d[4])
		p.ArcTo(q.d[1], q.d[2], q.d[3]*180.0/math.Pi, large, sweep, q.d[5], q.d[6])
	case CloseCmd:
		p.Close()
	}

	i := len(p.d)
	end := p.StartPos()
	p = &Path{append(p.d, q.d[cmdLen(cmd):]...)}

	// repair close commands
	for i < len(p.d) {
		cmd := p.d[i]
		if cmd == MoveToCmd {
			break
		} else if cmd == CloseCmd {
			p.d[i+1] = end.X
			p.d[i+2] = end.Y
			break
		}
		i += cmdLen(cmd)
	}
	return p

}

// Pos returns the current position of the path, which is the end point of the last command.
func (p *Path) Pos() Point {
	if 0 < len(p.d) {
		return Point{p.d[len(p.d)-3], p.d[len(p.d)-2]}
	}
	return Point{}
}

// StartPos returns the start point of the current subpath, i.e. it returns the position of the last MoveTo command.
func (p *Path) StartPos() Point {
	for i := len(p.d); 0 < i; {
		cmd := p.d[i-1]
		if cmd == MoveToCmd {
			return Point{p.d[i-3], p.d[i-2]}
		}
		i -= cmdLen(cmd)
	}
	return Point{}
}

// Coords returns all the coordinates of the segment start/end points.
func (p *Path) Coords() []Point {
	coords := []Point{}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)
		if cmd != CloseCmd || !Equal(coords[len(coords)-1].X, p.d[i-3]) || !Equal(coords[len(coords)-1].Y, p.d[i-2]) {
			coords = append(coords, Point{p.d[i-3], p.d[i-2]})
		}
	}
	return coords
}

////////////////////////////////////////////////////////////////

// MoveTo moves the path to (x,y) without connecting the path. It starts a new independent subpath. Multiple subpaths can be useful when negating parts of a previous path by overlapping it with a path in the opposite direction. The behaviour for overlapping paths depends on the FillRule.
func (p *Path) MoveTo(x, y float64) {
	if 0 < len(p.d) && p.d[len(p.d)-1] == MoveToCmd {
		p.d[len(p.d)-3] = x
		p.d[len(p.d)-2] = y
		return
	}
	p.d = append(p.d, MoveToCmd, x, y, MoveToCmd)
}

// LineTo adds a linear path to (x,y).
func (p *Path) LineTo(x, y float64) {
	start := p.Pos()
	end := Point{x, y}
	if start.Equals(end) {
		return
	} else if cmdLen(LineToCmd) <= len(p.d) && p.d[len(p.d)-1] == LineToCmd {
		prevStart := Point{}
		if cmdLen(LineToCmd) < len(p.d) {
			prevStart = Point{p.d[len(p.d)-cmdLen(LineToCmd)-3], p.d[len(p.d)-cmdLen(LineToCmd)-2]}
		}
		if Equal(end.Sub(start).AngleBetween(start.Sub(prevStart)), 0.0) {
			p.d[len(p.d)-3] = x
			p.d[len(p.d)-2] = y
			return
		}
	}

	if len(p.d) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p.d[len(p.d)-1] == CloseCmd {
		p.MoveTo(p.d[len(p.d)-3], p.d[len(p.d)-2])
	}
	p.d = append(p.d, LineToCmd, end.X, end.Y, LineToCmd)
}

// QuadTo adds a quadratic Bézier path with control point (cpx,cpy) and end point (x,y).
func (p *Path) QuadTo(cpx, cpy, x, y float64) {
	start := p.Pos()
	cp := Point{cpx, cpy}
	end := Point{x, y}
	if start.Equals(end) && start.Equals(cp) {
		return
	} else if !start.Equals(end) && Equal(end.Sub(start).AngleBetween(cp.Sub(start)), 0.0) && Equal(end.Sub(start).AngleBetween(end.Sub(cp)), 0.0) {
		p.LineTo(end.X, end.Y)
		return
	}

	if len(p.d) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p.d[len(p.d)-1] == CloseCmd {
		p.MoveTo(p.d[len(p.d)-3], p.d[len(p.d)-2])
	}
	p.d = append(p.d, QuadToCmd, cp.X, cp.Y, end.X, end.Y, QuadToCmd)
}

// CubeTo adds a cubic Bézier path with control points (cpx1,cpy1) and (cpx2,cpy2) and end point (x,y).
func (p *Path) CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float64) {
	start := p.Pos()
	cp1 := Point{cpx1, cpy1}
	cp2 := Point{cpx2, cpy2}
	end := Point{x, y}
	if start.Equals(end) && start.Equals(cp1) && start.Equals(cp2) {
		return
	} else if !start.Equals(end) && Equal(end.Sub(start).AngleBetween(cp1.Sub(start)), 0.0) && Equal(end.Sub(start).AngleBetween(end.Sub(cp1)), 0.0) && Equal(end.Sub(start).AngleBetween(cp2.Sub(start)), 0.0) && Equal(end.Sub(start).AngleBetween(end.Sub(cp2)), 0.0) {
		p.LineTo(end.X, end.Y)
		return
	}

	if len(p.d) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p.d[len(p.d)-1] == CloseCmd {
		p.MoveTo(p.d[len(p.d)-3], p.d[len(p.d)-2])
	}
	p.d = append(p.d, CubeToCmd, cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y, CubeToCmd)
}

// ArcTo adds an arc with radii rx and ry, with rot the counter clockwise rotation with respect to the coordinate system in degrees, large and sweep booleans (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs), and (x,y) the end position of the pen. The start position of the pen was given by a previous command's end point.
func (p *Path) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) {
	start := p.Pos()
	end := Point{x, y}
	if start.Equals(end) {
		return
	}
	if Equal(rx, 0.0) || Equal(ry, 0.0) {
		p.LineTo(end.X, end.Y)
		return
	}

	rx = math.Abs(rx)
	ry = math.Abs(ry)
	if rx < ry {
		rx, ry = ry, rx
		rot += 90.0
	}

	phi := angleNorm(rot * math.Pi / 180.0)
	if math.Pi <= phi { // phi is canonical within 0 <= phi < 180
		phi -= math.Pi
	}

	// scale ellipse if rx and ry are too small
	lambda := ellipseRadiiCorrection(start, rx, ry, phi, end)
	if lambda > 1.0 {
		rx *= lambda
		ry *= lambda
	}

	if len(p.d) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p.d[len(p.d)-1] == CloseCmd {
		p.MoveTo(p.d[len(p.d)-3], p.d[len(p.d)-2])
	}
	p.d = append(p.d, ArcToCmd, rx, ry, phi, fromArcFlags(large, sweep), end.X, end.Y, ArcToCmd)
}

// Arc adds an elliptical arc with radii rx and ry, with rot the counter clockwise rotation in degrees, and theta0 and theta1 the angles in degrees of the ellipse (before rot is applies) between which the arc will run. If theta0 < theta1, the arc will run in a CCW direction. If the difference between theta0 and theta1 is bigger than 360 degrees, one full circle will be drawn and the remaining part of diff % 360, e.g. a difference of 810 degrees will draw one full circle and an arc over 90 degrees.
func (p *Path) Arc(rx, ry, rot, theta0, theta1 float64) {
	phi := rot * math.Pi / 180.0
	theta0 *= math.Pi / 180.0
	theta1 *= math.Pi / 180.0
	dtheta := math.Abs(theta1 - theta0)

	sweep := theta0 < theta1
	large := math.Mod(dtheta, 2.0*math.Pi) > math.Pi
	p0 := EllipsePos(rx, ry, phi, 0.0, 0.0, theta0)
	p1 := EllipsePos(rx, ry, phi, 0.0, 0.0, theta1)

	start := p.Pos()
	center := start.Sub(p0)
	if dtheta >= 2.0*math.Pi {
		startOpposite := center.Sub(p0)
		p.ArcTo(rx, ry, rot, large, sweep, startOpposite.X, startOpposite.Y)
		p.ArcTo(rx, ry, rot, large, sweep, start.X, start.Y)
		if Equal(math.Mod(dtheta, 2.0*math.Pi), 0.0) {
			return
		}
	}
	end := center.Add(p1)
	p.ArcTo(rx, ry, rot, large, sweep, end.X, end.Y)
}

// Close closes a (sub)path with a LineTo to the start of the path (the most recent MoveTo command). It also signals the path closes as opposed to being just a LineTo command, which can be significant for stroking purposes for example.
func (p *Path) Close() {
	end := p.StartPos()
	if len(p.d) == 0 || p.d[len(p.d)-1] == CloseCmd {
		return
	} else if p.d[len(p.d)-1] == MoveToCmd {
		p.d = p.d[:len(p.d)-cmdLen(MoveToCmd)]
		return
	} else if p.d[len(p.d)-1] == LineToCmd && Equal(p.d[len(p.d)-3], end.X) && Equal(p.d[len(p.d)-2], end.Y) {
		p.d[len(p.d)-1] = CloseCmd
		p.d[len(p.d)-cmdLen(LineToCmd)] = CloseCmd
		return
	} else if cmdLen(LineToCmd) <= len(p.d) && p.d[len(p.d)-1] == LineToCmd {
		start := Point{p.d[len(p.d)-3], p.d[len(p.d)-2]}
		prevStart := Point{}
		if cmdLen(LineToCmd) < len(p.d) {
			prevStart = Point{p.d[len(p.d)-cmdLen(LineToCmd)-3], p.d[len(p.d)-cmdLen(LineToCmd)-2]}
		}
		if Equal(end.Sub(start).AngleBetween(start.Sub(prevStart)), 0.0) {
			p.d[len(p.d)-cmdLen(LineToCmd)] = CloseCmd
			p.d[len(p.d)-3] = end.X
			p.d[len(p.d)-2] = end.Y
			p.d[len(p.d)-1] = CloseCmd
			return
		}
	}
	p.d = append(p.d, CloseCmd, end.X, end.Y, CloseCmd)
}

////////////////////////////////////////////////////////////////

func (p *Path) simplifyToCoords() []Point {
	coords := p.Coords()
	if len(coords) == 3 {
		// if there are just two commands, linearizing them gives us an area of no surface. To avoid this we add extra coordinates halfway for QuadTo, CubeTo and ArcTo.
		coords = []Point{}
		for i := 0; i < len(p.d); {
			cmd := p.d[i]
			if cmd == QuadToCmd {
				p0 := Point{p.d[i-3], p.d[i-2]}
				p1 := Point{p.d[i+1], p.d[i+2]}
				p2 := Point{p.d[i+3], p.d[i+4]}
				_, _, _, coord, _, _ := quadraticBezierSplit(p0, p1, p2, 0.5)
				coords = append(coords, coord)
			} else if cmd == CubeToCmd {
				p0 := Point{p.d[i-3], p.d[i-2]}
				p1 := Point{p.d[i+1], p.d[i+2]}
				p2 := Point{p.d[i+3], p.d[i+4]}
				p3 := Point{p.d[i+5], p.d[i+6]}
				_, _, _, _, coord, _, _, _ := cubicBezierSplit(p0, p1, p2, p3, 0.5)
				coords = append(coords, coord)
			} else if cmd == ArcToCmd {
				rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
				large, sweep := toArcFlags(p.d[i+4])
				cx, cy, theta0, theta1 := ellipseToCenter(p.d[i-3], p.d[i-2], rx, ry, phi, large, sweep, p.d[i+5], p.d[i+6])
				coord, _, _, _ := ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, (theta0+theta1)/2.0)
				coords = append(coords, coord)
			}
			i += cmdLen(cmd)
			if cmd != CloseCmd || !Equal(coords[len(coords)-1].X, p.d[i-3]) || !Equal(coords[len(coords)-1].Y, p.d[i-2]) {
				coords = append(coords, Point{p.d[i-3], p.d[i-2]})
			}
		}
	}
	return coords
}

// CCW returns true when the path has (mostly) a counter clockwise direction. It does not need the path to be closed and will return true for a empty or straight line.
func (p *Path) CCW() bool {
	// use the Shoelace formula
	area := 0.0
	for _, ps := range p.Split() {
		coords := ps.simplifyToCoords()
		for i := 1; i < len(coords); i++ {
			area += (coords[i].X - coords[i-1].X) * (coords[i-1].Y + coords[i].Y)
		}
	}
	return area <= 0.0
}

// Filling returns whether each subpath gets filled or not. A path may not be filling when it negates another path and depends on the FillRule. If a subpath is not closed, it is implicitly assumed to be closed. If the path has no area it will return false.
func (p *Path) Filling(fillRule FillRule) []bool {
	var pls []*Polyline
	var ccw []bool
	for _, ps := range p.Split() {
		ps.Close()

		coords := ps.simplifyToCoords()
		polyline := &Polyline{coords}
		pls = append(pls, polyline) // no need for flattening as we pick our test point to be inside the polyline
		ccw = append(ccw, ps.CCW())
	}

	testPoints := make([]Point, 0, len(pls))
	for i, pl := range pls {
		offset := pl.coords[1].Sub(pl.coords[0]).Rot90CW().Norm(Epsilon)
		if ccw[i] {
			offset = offset.Neg()
		}
		testPoints = append(testPoints, pl.coords[0].Interpolate(pl.coords[1], 0.5).Add(offset))
	}

	fillCounts := make([]int, len(testPoints))
	for _, pl := range pls {
		for i, test := range testPoints {
			fillCounts[i] += pl.FillCount(test.X, test.Y)
		}
	}

	fillings := make([]bool, len(fillCounts))
	for i := range fillCounts {
		if fillRule == NonZero {
			fillings[i] = fillCounts[i] != 0
		} else {
			fillings[i] = fillCounts[i]%2 != 0
		}
	}
	return fillings
}

// Interior is true when the point (x,y) is in the interior of the path, i.e. gets filled. This depends on the FillRule.
func (p *Path) Interior(x, y float64, fillRule FillRule) bool {
	fillCount := 0
	test := Point{x, y}
	for _, ps := range p.Split() {
		fillCount += PolylineFromPath(ps).FillCount(test.X, test.Y) // uses flattening
	}
	if fillRule == NonZero {
		return fillCount != 0
	}
	return fillCount%2 != 0
}

// Bounds returns the bounding box rectangle of the path.
func (p *Path) Bounds() Rect {
	if len(p.d) == 0 {
		return Rect{}
	}

	xmin, xmax := math.Inf(1), math.Inf(-1)
	ymin, ymax := math.Inf(1), math.Inf(-1)

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
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}

			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			if tdenom := (start.X - 2*cp.X + end.X); tdenom != 0.0 {
				if t := (start.X - cp.X) / tdenom; 0.0 < t && t < 1.0 {
					x := quadraticBezierPos(start, cp, end, t)
					xmin = math.Min(xmin, x.X)
					xmax = math.Max(xmax, x.X)
				}
			}

			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
			if tdenom := (start.Y - 2*cp.Y + end.Y); tdenom != 0.0 {
				if t := (start.Y - cp.Y) / tdenom; 0.0 < t && t < 1.0 {
					y := quadraticBezierPos(start, cp, end, t)
					ymin = math.Min(ymin, y.Y)
					ymax = math.Max(ymax, y.Y)
				}
			}
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}

			a := -start.X + 3*cp1.X - 3*cp2.X + end.X
			b := 2*start.X - 4*cp1.X + 2*cp2.X
			c := -start.X + cp1.X
			t1, t2 := solveQuadraticFormula(a, b, c)

			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			if !math.IsNaN(t1) && 0.0 < t1 && t1 < 1.0 {
				x1 := cubicBezierPos(start, cp1, cp2, end, t1)
				xmin = math.Min(xmin, x1.X)
				xmax = math.Max(xmax, x1.X)
			}
			if !math.IsNaN(t2) && 0.0 < t2 && t2 < 1.0 {
				x2 := cubicBezierPos(start, cp1, cp2, end, t2)
				xmin = math.Min(xmin, x2.X)
				xmax = math.Max(xmax, x2.X)
			}

			a = -start.Y + 3*cp1.Y - 3*cp2.Y + end.Y
			b = 2*start.Y - 4*cp1.Y + 2*cp2.Y
			c = -start.Y + cp1.Y
			t1, t2 = solveQuadraticFormula(a, b, c)

			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
			if !math.IsNaN(t1) && 0.0 < t1 && t1 < 1.0 {
				y1 := cubicBezierPos(start, cp1, cp2, end, t1)
				ymin = math.Min(ymin, y1.Y)
				ymax = math.Max(ymax, y1.Y)
			}
			if !math.IsNaN(t2) && 0.0 < t2 && t2 < 1.0 {
				y2 := cubicBezierPos(start, cp1, cp2, end, t2)
				ymin = math.Min(ymin, y2.Y)
				ymax = math.Max(ymax, y2.Y)
			}
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			cx, cy, theta1, theta2 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

			// find the four extremes (top, bottom, left, right) and apply those who are between theta1 and theta2
			// x(theta) = cx + rx*cos(theta)*cos(phi) - ry*sin(theta)*sin(phi)
			// y(theta) = cy + rx*cos(theta)*sin(phi) + ry*sin(theta)*cos(phi)
			// be aware that positive rotation appears clockwise in SVGs (non-Cartesian coordinate system)
			// we can now find the angles of the extremes

			sinphi, cosphi := math.Sincos(phi)
			thetaRight := math.Atan2(-ry*sinphi, rx*cosphi)
			thetaTop := math.Atan2(rx*cosphi, ry*sinphi)
			thetaLeft := thetaRight + math.Pi
			thetaBottom := thetaTop + math.Pi

			dx := math.Sqrt(rx*rx*cosphi*cosphi + ry*ry*sinphi*sinphi)
			dy := math.Sqrt(rx*rx*sinphi*sinphi + ry*ry*cosphi*cosphi)
			if angleBetween(thetaLeft, theta1, theta2) {
				xmin = math.Min(xmin, cx-dx)
			}
			if angleBetween(thetaRight, theta1, theta2) {
				xmax = math.Max(xmax, cx+dx)
			}
			if angleBetween(thetaBottom, theta1, theta2) {
				ymin = math.Min(ymin, cy-dy)
			}
			if angleBetween(thetaTop, theta1, theta2) {
				ymax = math.Max(ymax, cy+dy)
			}
			xmin = math.Min(xmin, end.X)
			xmax = math.Max(xmax, end.X)
			ymin = math.Min(ymin, end.Y)
			ymax = math.Max(ymax, end.Y)
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
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			d += quadraticBezierLength(start, cp, end)
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			d += cubicBezierLength(start, cp1, cp2, end)
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			_, _, theta1, theta2 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			d += ellipseLength(rx, ry, theta1, theta2)
		}
		i += cmdLen(cmd)
		start = end
	}
	return d
}

// Transform transforms the path by the given transformation matrix and returns a new path.
func (p *Path) Transform(m Matrix) *Path {
	p = p.Copy()
	_, _, _, xscale, yscale, _ := m.Decompose()
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd, LineToCmd, CloseCmd:
			end := m.Dot(Point{p.d[i+1], p.d[i+2]})
			p.d[i+1] = end.X
			p.d[i+2] = end.Y
		case QuadToCmd:
			cp := m.Dot(Point{p.d[i+1], p.d[i+2]})
			end := m.Dot(Point{p.d[i+3], p.d[i+4]})
			p.d[i+1] = cp.X
			p.d[i+2] = cp.Y
			p.d[i+3] = end.X
			p.d[i+4] = end.Y
		case CubeToCmd:
			cp1 := m.Dot(Point{p.d[i+1], p.d[i+2]})
			cp2 := m.Dot(Point{p.d[i+3], p.d[i+4]})
			end := m.Dot(Point{p.d[i+5], p.d[i+6]})
			p.d[i+1] = cp1.X
			p.d[i+2] = cp1.Y
			p.d[i+3] = cp2.X
			p.d[i+4] = cp2.Y
			p.d[i+5] = end.X
			p.d[i+6] = end.Y
		case ArcToCmd:
			rx := p.d[i+1]
			ry := p.d[i+2]
			phi := p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			end := m.Dot(Point{p.d[i+5], p.d[i+6]})

			// For ellipses written as the conic section equation in matrix form, we have:
			// (x, y) E (x; y) = 0, with E = (1/rx^2, 0; 0, 1/ry^2)
			// for our transformed ellipse we have (x', y') = T (x, y), with T the affine transformation matrix
			// so that (T^-1 (x'; y'))^T E (T^-1 (x'; y') = 0  =>  (x', y') T^(-1,T) E T^(-1) (x'; y') = 0
			// we define Q = T^(-1,T) E T^(-1) the new ellipse equation which is typically rotated from the x-axis.
			// That's why we find the eigenvalues and eigenvectors (the new direction and length of the major and minor axes).
			T := m.Rotate(phi * 180.0 / math.Pi)
			invT := T.Inv()
			Q := Identity.Scale(1.0/rx/rx, 1.0/ry/ry)
			Q = invT.T().Mul(Q).Mul(invT)

			lambda1, lambda2, v1, v2 := Q.Eigen()
			rx = 1 / math.Sqrt(lambda1)
			ry = 1 / math.Sqrt(lambda2)
			phi = v1.Angle()
			if rx < ry {
				rx, ry = ry, rx
				phi = v2.Angle()
			}
			phi = angleNorm(phi)
			if math.Pi <= phi { // phi is canonical within 0 <= phi < 180
				phi -= math.Pi
			}

			if xscale*yscale < 0.0 { // flip x or y axis needs flipping of the sweep
				sweep = !sweep
			}
			p.d[i+1] = rx
			p.d[i+2] = ry
			p.d[i+3] = phi
			p.d[i+4] = fromArcFlags(large, sweep)
			p.d[i+5] = end.X
			p.d[i+6] = end.Y
		}
		i += cmdLen(cmd)
	}
	return p
}

// Translate translates the path by (x,y) and returns a new path.
func (p *Path) Translate(x, y float64) *Path {
	return p.Transform(Identity.Translate(x, y))
}

// Flatten flattens all Bézier and arc curves into linear segments and returns a new path. It uses Tolerance as the maximum deviation.
func (p *Path) Flatten() *Path {
	return p.replace(nil, flattenQuadraticBezier, flattenCubicBezier, flattenEllipticArc)
}

// ReplaceArcs replaces ArcTo commands by CubeTo commands.
func (p *Path) ReplaceArcs() *Path {
	return p.replace(nil, nil, nil, arcToCube)
}

// replace replaces path segments by their respective functions, each returning the path that will replace the segment or nil if no replacement is to be performed. The line function will take the start and end points. The bezier function will take the start point, control point 1 and 2, and the end point (i.e. a cubic Bézier, quadratic Béziers will be implicitly converted to cubic ones). The arc function will take a start point, the major and minor radii, the radial rotaton counter clockwise, the large and sweep booleans, and the end point. The replacing path will replace the path segment without any checks, you need to make sure the be moved so that its start point connects with the last end point of the base path before the replacement. If the end point of the replacing path is different that the end point of what is replaced, the path that follows will be displaced.
func (p *Path) replace(
	line func(Point, Point) *Path,
	quad func(Point, Point, Point) *Path,
	cube func(Point, Point, Point, Point) *Path,
	arc func(Point, float64, float64, float64, bool, bool, Point) *Path,
) *Path {
	p = p.Copy()

	var start, end Point
	for i := 0; i < len(p.d); {
		var q *Path
		cmd := p.d[i]
		switch cmd {
		case LineToCmd, CloseCmd:
			if line != nil {
				end = Point{p.d[i+1], p.d[i+2]}
				q = line(start, end)
				if cmd == CloseCmd {
					q.Close()
				}
			}
		case QuadToCmd:
			if quad != nil {
				cp := Point{p.d[i+1], p.d[i+2]}
				end = Point{p.d[i+3], p.d[i+4]}
				q = quad(start, cp, end)
			}
		case CubeToCmd:
			if cube != nil {
				cp1 := Point{p.d[i+1], p.d[i+2]}
				cp2 := Point{p.d[i+3], p.d[i+4]}
				end = Point{p.d[i+5], p.d[i+6]}
				q = cube(start, cp1, cp2, end)
			}
		case ArcToCmd:
			if arc != nil {
				rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
				large, sweep := toArcFlags(p.d[i+4])
				end = Point{p.d[i+5], p.d[i+6]}
				q = arc(start, rx, ry, phi, large, sweep, end)
			}
		}

		if q != nil {
			r := &Path{append([]float64{MoveToCmd, end.X, end.Y, MoveToCmd}, p.d[i+cmdLen(cmd):]...)}

			p.d = p.d[: i : i+cmdLen(cmd)] // make sure not to overwrite the rest of the path
			p = p.Join(q)
			if cmd != CloseCmd {
				p.LineTo(end.X, end.Y)
			}

			i = len(p.d)
			p = p.Join(r) // join the rest of the base path
		} else {
			i += cmdLen(cmd)
		}
		start = Point{p.d[i-3], p.d[i-2]}
	}
	return p
}

// Markers returns an array of start, mid and end marker paths along the path at the coordinates between commands. Align will align the markers with the path direction so that the markers orient towards the path's left.
func (p *Path) Markers(first, mid, last *Path, align bool) []*Path {
	markers := []*Path{}
	for _, ps := range p.Split() {
		isFirst := true
		closed := ps.Closed()

		var start, end Point
		var n0Start, n1Prev, n0, n1 Point
		for i := 0; i < len(ps.d); {
			cmd := ps.d[i]
			i += cmdLen(cmd)

			start = end
			end = Point{ps.d[i-3], ps.d[i-2]}

			if align {
				n1Prev = n1
				switch cmd {
				case LineToCmd, CloseCmd:
					n := end.Sub(start).Rot90CW().Norm(1.0)
					n0, n1 = n, n
				case QuadToCmd, CubeToCmd:
					var cp1, cp2 Point
					if cmd == QuadToCmd {
						cp := Point{p.d[i-5], p.d[i-4]}
						cp1, cp2 = quadraticToCubicBezier(start, cp, end)
					} else {
						cp1 = Point{p.d[i-7], p.d[i-6]}
						cp2 = Point{p.d[i-5], p.d[i-4]}
					}
					n0 = cubicBezierNormal(start, cp1, cp2, end, 0.0, 1.0)
					n1 = cubicBezierNormal(start, cp1, cp2, end, 1.0, 1.0)
				case ArcToCmd:
					rx, ry, phi := p.d[i-7], p.d[i-6], p.d[i-5]
					large, sweep := toArcFlags(p.d[i-4])
					_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
					n0 = ellipseNormal(rx, ry, phi, sweep, theta0, 1.0)
					n1 = ellipseNormal(rx, ry, phi, sweep, theta1, 1.0)
				}
			}

			if cmd == MoveToCmd {
				continue
			}

			q := mid
			angle := n1Prev.Add(n0).Angle()
			if isFirst {
				n0Start = n0
				isFirst = false
				if closed {
					continue
				}
				q = first
				angle = n0.Angle()
			}

			m := Identity.Translate(start.X, start.Y)
			if align {
				m = m.Rotate((angle * 180.0 / math.Pi) + 90.0)
			}
			markers = append(markers, q.Transform(m))
		}

		q := last
		angle := n1.Angle()
		if closed {
			q = mid
			angle = n1.Add(n0Start).Angle()
		}

		m := Identity.Translate(end.X, end.Y)
		if align {
			m = m.Rotate((angle * 180.0 / math.Pi) + 90.0)
		}
		markers = append(markers, q.Transform(m))
	}
	return markers
}

// Split splits the path into its independent subpaths. The path is split before each MoveTo command. None of the subpaths shall be empty.
func (p *Path) Split() []*Path {
	ps := []*Path{}

	var i, j int
	for j < len(p.d) {
		cmd := p.d[j]
		if i < j && cmd == MoveToCmd {
			ps = append(ps, &Path{p.d[i:j:j]})
			i = j
		}
		j += cmdLen(cmd)
	}
	if i+cmdLen(MoveToCmd) < j {
		ps = append(ps, &Path{p.d[i:j:j]})
	}
	return ps
}

// SplitAt splits the path into separate paths at the specified intervals (given in millimeters) along the path.
func (p *Path) SplitAt(ts ...float64) []*Path {
	if len(ts) == 0 {
		return []*Path{p}
	}

	sort.Float64s(ts)
	if ts[0] == 0.0 {
		ts = ts[1:]
	}

	j := 0   // index into ts
	T := 0.0 // current position along curve

	qs := []*Path{}
	q := &Path{}
	push := func() {
		qs = append(qs, q)
		q = &Path{}
	}

	if 0 < len(p.d) && p.d[0] == MoveToCmd {
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
			case QuadToCmd:
				cp := Point{p.d[i+1], p.d[i+2]}
				end = Point{p.d[i+3], p.d[i+4]}

				if j == len(ts) {
					q.QuadTo(cp.X, cp.Y, end.X, end.Y)
				} else {
					speed := func(t float64) float64 {
						return quadraticBezierDeriv(start, cp, end, t).Length()
					}
					invL, dT := invSpeedPolynomialChebyshevApprox(20, gaussLegendre7, speed, 0.0, 1.0)

					t0 := 0.0
					r0, r1, r2 := start, cp, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						t := invL(ts[j] - T)
						tsub := (t - t0) / (1.0 - t0)
						t0 = t

						var q1 Point
						_, q1, _, r0, r1, r2 = quadraticBezierSplit(r0, r1, r2, tsub)

						q.QuadTo(q1.X, q1.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if !Equal(t0, 1.0) {
						q.QuadTo(r1.X, r1.Y, r2.X, r2.Y)
					}
					T += dT
				}
			case CubeToCmd:
				cp1 := Point{p.d[i+1], p.d[i+2]}
				cp2 := Point{p.d[i+3], p.d[i+4]}
				end = Point{p.d[i+5], p.d[i+6]}

				if j == len(ts) {
					q.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y)
				} else {
					speed := func(t float64) float64 {
						// splitting on inflection points does not improve output
						return cubicBezierDeriv(start, cp1, cp2, end, t).Length()
					}
					N := 20 + 20*cubicBezierNumInflections(start, cp1, cp2, end) // TODO: needs better N
					invL, dT := invSpeedPolynomialChebyshevApprox(N, gaussLegendre7, speed, 0.0, 1.0)

					t0 := 0.0
					r0, r1, r2, r3 := start, cp1, cp2, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						t := invL(ts[j] - T)
						tsub := (t - t0) / (1.0 - t0)
						t0 = t

						var q1, q2 Point
						_, q1, q2, _, r0, r1, r2, r3 = cubicBezierSplit(r0, r1, r2, r3, tsub)

						q.CubeTo(q1.X, q1.Y, q2.X, q2.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if !Equal(t0, 1.0) {
						q.CubeTo(r1.X, r1.Y, r2.X, r2.Y, r3.X, r3.Y)
					}
					T += dT
				}
			case ArcToCmd:
				rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
				large, sweep := toArcFlags(p.d[i+4])
				end = Point{p.d[i+5], p.d[i+6]}
				cx, cy, theta1, theta2 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

				if j == len(ts) {
					q.ArcTo(rx, ry, phi*180.0/math.Pi, large, sweep, end.X, end.Y)
				} else {
					speed := func(theta float64) float64 {
						return ellipseDeriv(rx, ry, 0.0, true, theta).Length()
					}
					invL, dT := invSpeedPolynomialChebyshevApprox(10, gaussLegendre7, speed, theta1, theta2)

					startTheta := theta1
					nextLarge := large
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						theta := invL(ts[j] - T)
						mid, large1, large2, ok := ellipseSplit(rx, ry, phi, cx, cy, startTheta, theta2, theta)
						if !ok {
							panic("theta not in elliptic arc range for splitting")
						}

						q.ArcTo(rx, ry, phi*180.0/math.Pi, large1, sweep, mid.X, mid.Y)
						push()
						q.MoveTo(mid.X, mid.Y)
						startTheta = theta
						nextLarge = large2
						j++
					}
					if !Equal(startTheta, theta2) {
						q.ArcTo(rx, ry, phi*180.0/math.Pi, nextLarge, sweep, end.X, end.Y)
					}
					T += dT
				}
			}
			i += cmdLen(cmd)
			start = end
		}
	}
	if cmdLen(MoveToCmd) < len(q.d) {
		qs = append(qs, q)
	}
	return qs
}

//type intersection struct {
//	i int     // index into path
//	t float64 // parametric value
//}
//
//func (p *Path) SplitAtIntersections(q *Path) ([]*Path, []*Path) {
//	selfIntersect := p == q
//	ps := []*Path{}
//	qs := []*Path{}
//	for _, pp := range p.Split() {
//		for _, qq := range q.Split() {
//			qu := []intersection{}
//			for {
//				_ = pp
//				_ = qq
//				// add to ps
//			}
//
//			if !selfIntersect {
//				sort.Slice(qu, func(i, j int) bool {
//					return qu[i].i < qu[j].i || qu[i].i == qu[j].i && qu[i].t < qu[j].t
//				})
//
//				for _, _ = range qu {
//					// add to qs
//				}
//			}
//		}
//	}
//
//	if selfIntersect {
//		return ps, ps
//	}
//	return ps, qs
//}

func dashStart(offset float64, d []float64) (int, float64) {
	i0 := 0 // index in d
	for d[i0] <= offset {
		offset -= d[i0]
		i0++
		if i0 == len(d) {
			i0 = 0
		}
	}
	pos0 := -offset // negative if offset is halfway into dash
	if offset < 0.0 {
		dTotal := 0.0
		for _, dd := range d {
			dTotal += dd
		}
		pos0 = -(dTotal + offset) // handle negative offsets
	}
	return i0, pos0
}

// dashCanonical returns an optimized dash array.
func dashCanonical(offset float64, d []float64) (float64, []float64) {
	if len(d) == 0 {
		return 0.0, []float64{}
	}

	// remove zeros except first and last
	for i := 1; i < len(d)-1; i++ {
		if Equal(d[i], 0.0) {
			d[i-1] += d[i+1]
			d = append(d[:i], d[i+2:]...)
			i--
		}
	}

	// remove first zero, collapse with second and last
	if Equal(d[0], 0.0) {
		if len(d) < 3 {
			return 0.0, []float64{0.0}
		}
		offset -= d[1]
		d[len(d)-1] += d[1]
		d = d[2:]
	}

	// remove last zero, collapse with fist and second to last
	if Equal(d[len(d)-1], 0.0) {
		if len(d) < 3 {
			return 0.0, []float64{}
		}
		offset += d[len(d)-2]
		d[0] += d[len(d)-2]
		d = d[:len(d)-2]
	}

	// if there are zeros or negatives, don't draw any dashes
	for i := 0; i < len(d); i++ {
		if d[i] < 0.0 || Equal(d[i], 0.0) {
			return 0.0, []float64{0.0}
		}
	}

	// remove repeated patterns
REPEAT:
	for len(d)%2 == 0 {
		mid := len(d) / 2
		for i := 0; i < mid; i++ {
			if !Equal(d[i], d[mid+i]) {
				break REPEAT
			}
		}
		d = d[:mid]
	}
	return offset, d
}

func (p *Path) checkDash(offset float64, d []float64) (*Path, []float64) {
	offset, d = dashCanonical(offset, d)
	if len(d) == 0 {
		return p, d
	} else if len(d) == 1 && d[0] == 0.0 {
		return &Path{}, d
	}

	length := p.Length()
	i, pos := dashStart(offset, d)
	if length <= d[i]-pos {
		if i%2 == 0 {
			return p, nil // first dash covers whole path
		}
		return &Path{}, nil // first space covers whole path
	}
	return p, d
}

// Dash returns a new path that consists of dashes. The elements in d specify the width of the dashes and gaps. It will alternate between dashes and gaps when picking widths. If d is an array of odd length, it is equivalent of passing d twice in sequence. The offset specifies the offset used into d (or negative offset into the path). Dash will be applied to each subpath independently.
func (p *Path) Dash(offset float64, d ...float64) *Path {
	offset, d = dashCanonical(offset, d)
	if len(d) == 0 {
		return p
	} else if len(d) == 1 && d[0] == 0.0 {
		return &Path{}
	}

	if len(d)%2 == 1 {
		// if d is uneven length, dash and space lengths alternate. Duplicate d so that uneven indices are always spaces
		d = append(d, d...)
	}

	i0, pos0 := dashStart(offset, d)

	q := &Path{}
	for _, ps := range p.Split() {
		i := i0
		pos := pos0

		t := []float64{}
		length := ps.Length()
		for pos+d[i]+Epsilon < length {
			pos += d[i]
			if 0.0 < pos {
				t = append(t, pos)
			}
			i++
			if i == len(d) {
				i = 0
			}
		}

		j0 := 0
		endsInDash := i%2 == 0
		if len(t)%2 == 1 && endsInDash || len(t)%2 == 0 && !endsInDash {
			j0 = 1
		}

		qd := &Path{}
		pd := ps.SplitAt(t...)
		for j := j0; j < len(pd)-1; j += 2 {
			qd = qd.Append(pd[j])
		}
		if endsInDash {
			if ps.Closed() {
				qd = pd[len(pd)-1].Join(qd)
			} else {
				qd = qd.Append(pd[len(pd)-1])
			}
		}
		q = q.Append(qd)
	}
	return q
}

// Reverse returns a new path that is the same path as p but in the reverse direction.
func (p *Path) Reverse() *Path {
	rp := &Path{}
	if len(p.d) == 0 {
		return rp
	}

	end := Point{p.d[len(p.d)-3], p.d[len(p.d)-2]}
	if !end.IsZero() {
		rp.MoveTo(end.X, end.Y)
	}
	start := end
	closed := false

	for i := len(p.d); 0 < i; {
		cmd := p.d[i-1]
		i -= cmdLen(cmd)

		end = Point{}
		if i > 0 {
			end = Point{p.d[i-3], p.d[i-2]}
		}

		switch cmd {
		case CloseCmd:
			if !start.Equals(end) {
				rp.LineTo(end.X, end.Y)
			}
			closed = true
		case MoveToCmd:
			if closed {
				rp.Close()
				closed = false
			}
			if !end.IsZero() {
				rp.MoveTo(end.X, end.Y)
			}
		case LineToCmd:
			if closed && (i == 0 || p.d[i-1] == MoveToCmd) {
				rp.Close()
				closed = false
			} else {
				rp.LineTo(end.X, end.Y)
			}
		case QuadToCmd:
			cx, cy := p.d[i+1], p.d[i+2]
			rp.QuadTo(cx, cy, end.X, end.Y)
		case CubeToCmd:
			cx1, cy1 := p.d[i+3], p.d[i+4]
			cx2, cy2 := p.d[i+1], p.d[i+2]
			rp.CubeTo(cx1, cy1, cx2, cy2, end.X, end.Y)
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			rp.ArcTo(rx, ry, phi*180.0/math.Pi, large, !sweep, end.X, end.Y)
		}
		start = end
	}
	return rp
}

// Segment is a path command.
type Segment struct {
	Cmd        float64
	Start, End Point
	args       [4]float64
}

// CP1 returns the first control point for quadratic and cubic Béziers.
func (seg Segment) CP1() Point {
	return Point{seg.args[0], seg.args[1]}
}

// CP2 returns the second control point for cubic Béziers.
func (seg Segment) CP2() Point {
	return Point{seg.args[2], seg.args[3]}
}

// Arc returns the arguments for arcs (rx,ry,rot,large,sweep).
func (seg Segment) Arc() (float64, float64, float64, bool, bool) {
	large, sweep := toArcFlags(seg.args[3])
	return seg.args[0], seg.args[1], seg.args[2], large, sweep
}

// Segments returns the path segments as a slice of segment structures.
func (p *Path) Segments() []Segment {
	segs := []Segment{}
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			segs = append(segs, Segment{
				Cmd:   cmd,
				Start: start,
				End:   end,
			})
		case LineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			segs = append(segs, Segment{
				Cmd:   cmd,
				Start: start,
				End:   end,
			})
		case QuadToCmd:
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			segs = append(segs, Segment{
				Cmd:   cmd,
				Start: start,
				End:   end,
				args:  [4]float64{cp.X, cp.Y, 0.0, 0.0},
			})
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			segs = append(segs, Segment{
				Cmd:   cmd,
				Start: start,
				End:   end,
				args:  [4]float64{cp1.X, cp1.Y, cp2.X, cp2.Y},
			})
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]*180.0/math.Pi
			flags := p.d[i+4]
			end = Point{p.d[i+5], p.d[i+6]}
			segs = append(segs, Segment{
				Cmd:   cmd,
				Start: start,
				End:   end,
				args:  [4]float64{rx, ry, phi, flags},
			})
		case CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			segs = append(segs, Segment{
				Cmd:   cmd,
				Start: start,
				End:   end,
			})
		}
		start = end
		i += cmdLen(cmd)
	}
	return segs
}

// Iterate iterates over the path commands and calls the respective functions move, line, quad, cube, arc, close when encountering MoveTo, LineTo, QuadTo, CubeTo, ArcTo, Close commands respectively.
// DEPRECATED
func (p *Path) Iterate(
	move func(Point, Point),
	line func(Point, Point),
	quad func(Point, Point, Point),
	cube func(Point, Point, Point, Point),
	arc func(Point, float64, float64, float64, bool, bool, Point),
	close func(Point, Point),
) {
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			move(start, end)
		case LineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			line(start, end)
		case QuadToCmd:
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			quad(start, cp, end)
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			cube(start, cp1, cp2, end)
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]*180.0/math.Pi
			large, sweep := toArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			arc(start, rx, ry, phi, large, sweep, end)
		case CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			close(start, end)
		}
		start = end
		i += cmdLen(cmd)
	}
}

////////////////////////////////////////////////////////////////

func skipCommaWhitespace(path []byte) int {
	i := 0
	for i < len(path) && (path[i] == ' ' || path[i] == ',' || path[i] == '\n' || path[i] == '\r' || path[i] == '\t') {
		i++
	}
	return i
}

// MustParseSVG parses an SVG path data string and panics if it fails.
func MustParseSVG(s string) *Path {
	p, err := ParseSVG(s)
	if err != nil {
		panic(err)
	}
	return p
}

// ParseSVG parses an SVG path data string.
func ParseSVG(s string) (*Path, error) {
	if len(s) == 0 {
		return &Path{}, nil
	}

	i := 0
	path := []byte(s)
	i += skipCommaWhitespace(path[i:])
	if path[0] == ',' || path[i] < 'A' {
		return nil, fmt.Errorf("bad path: path should start with command")
	}

	cmdLens := map[byte]int{
		'M': 2,
		'Z': 0,
		'L': 2,
		'H': 1,
		'V': 1,
		'C': 6,
		'S': 4,
		'Q': 4,
		'T': 2,
		'A': 7,
	}
	f := [7]float64{}

	p := &Path{}
	var q, c Point
	var p0, p1 Point
	prevCmd := byte('z')
	for {
		i += skipCommaWhitespace(path[i:])
		if len(path) <= i {
			break
		}

		cmd := prevCmd
		repeat := true
		if cmd == 'z' || cmd == 'Z' || !(path[i] >= '0' && path[i] <= '9' || path[i] == '.' || path[i] == '-' || path[i] == '+') {
			cmd = path[i]
			repeat = false
			i++
			i += skipCommaWhitespace(path[i:])
		}

		CMD := cmd
		if 'a' <= cmd && cmd <= 'z' {
			CMD -= 'a' - 'A'
		}
		for j := 0; j < cmdLens[CMD]; j++ {
			if CMD == 'A' && (j == 3 || j == 4) {
				// parse largeArc and sweep booleans for A command
				if i < len(path) && path[i] == '1' {
					f[j] = 1.0
				} else if i < len(path) && path[i] == '0' {
					f[j] = 0.0
				} else {
					return nil, fmt.Errorf("bad path: largeArc and sweep flags should be 0 or 1 in command '%c' at position %d", cmd, i+1)
				}
				i++
			} else {
				num, n := strconv.ParseFloat(path[i:])
				if n == 0 {
					if repeat && j == 0 && i < len(path) {
						return nil, fmt.Errorf("bad path: unknown command '%c' at position %d", path[i], i+1)
					} else if 1 < cmdLens[CMD] {
						return nil, fmt.Errorf("bad path: sets of %d numbers should follow command '%c' at position %d", cmdLens[CMD], cmd, i+1)
					} else {
						return nil, fmt.Errorf("bad path: number should follow command '%c' at position %d", cmd, i+1)
					}
				}
				f[j] = num
				i += n
			}
			i += skipCommaWhitespace(path[i:])
		}

		switch cmd {
		case 'M', 'm':
			p1 = Point{f[0], f[1]}
			if cmd == 'm' {
				p1 = p1.Add(p0)
				cmd = 'l'
			} else {
				cmd = 'L'
			}
			p.MoveTo(p1.X, p1.Y)
		case 'Z', 'z':
			p1 = p.StartPos()
			p.Close()
		case 'L', 'l':
			p1 = Point{f[0], f[1]}
			if cmd == 'l' {
				p1 = p1.Add(p0)
			}
			p.LineTo(p1.X, p1.Y)
		case 'H', 'h':
			p1.X = f[0]
			if cmd == 'h' {
				p1.X += p0.X
			}
			p.LineTo(p1.X, p1.Y)
		case 'V', 'v':
			p1.Y = f[0]
			if cmd == 'v' {
				p1.Y += p0.Y
			}
			p.LineTo(p1.X, p1.Y)
		case 'C', 'c':
			cp1 := Point{f[0], f[1]}
			cp2 := Point{f[2], f[3]}
			p1 = Point{f[4], f[5]}
			if cmd == 'c' {
				cp1 = cp1.Add(p0)
				cp2 = cp2.Add(p0)
				p1 = p1.Add(p0)
			}
			p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y)
			c = cp2
		case 'S', 's':
			cp1 := p0
			cp2 := Point{f[0], f[1]}
			p1 = Point{f[2], f[3]}
			if cmd == 's' {
				cp2 = cp2.Add(p0)
				p1 = p1.Add(p0)
			}
			if prevCmd == 'C' || prevCmd == 'c' || prevCmd == 'S' || prevCmd == 's' {
				cp1 = p0.Mul(2.0).Sub(c)
			}
			p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y)
			c = cp2
		case 'Q', 'q':
			cp := Point{f[0], f[1]}
			p1 = Point{f[2], f[3]}
			if cmd == 'q' {
				cp = cp.Add(p0)
				p1 = p1.Add(p0)
			}
			p.QuadTo(cp.X, cp.Y, p1.X, p1.Y)
			q = cp
		case 'T', 't':
			cp := p0
			p1 = Point{f[0], f[1]}
			if cmd == 't' {
				p1 = p1.Add(p0)
			}
			if prevCmd == 'Q' || prevCmd == 'q' || prevCmd == 'T' || prevCmd == 't' {
				cp = p0.Mul(2.0).Sub(q)
			}
			p.QuadTo(cp.X, cp.Y, p1.X, p1.Y)
			q = cp
		case 'A', 'a':
			rx := f[0]
			ry := f[1]
			rot := f[2]
			large := f[3] == 1.0
			sweep := f[4] == 1.0
			p1 = Point{f[5], f[6]}
			if cmd == 'a' {
				p1 = p1.Add(p0)
			}
			p.ArcTo(rx, ry, rot, large, sweep, p1.X, p1.Y)
		default:
			return nil, fmt.Errorf("bad path: unknown command '%c' at position %d", cmd, i+1)
		}
		prevCmd = cmd
		p0 = p1
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
			fmt.Fprintf(&sb, "M%g %g", p.d[i+1], p.d[i+2])
		case LineToCmd:
			fmt.Fprintf(&sb, "L%g %g", p.d[i+1], p.d[i+2])
		case QuadToCmd:
			fmt.Fprintf(&sb, "Q%g %g %g %g", p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4])
		case CubeToCmd:
			fmt.Fprintf(&sb, "C%g %g %g %g %g %g", p.d[i+1], p.d[i+2], p.d[i+3], p.d[i+4], p.d[i+5], p.d[i+6])
		case ArcToCmd:
			rot := p.d[i+3] * 180.0 / math.Pi
			large, sweep := toArcFlags(p.d[i+4])
			sLarge := "0"
			if large {
				sLarge = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			fmt.Fprintf(&sb, "A%g %g %g %s %s %g %g", p.d[i+1], p.d[i+2], rot, sLarge, sSweep, p.d[i+5], p.d[i+6])
		case CloseCmd:
			fmt.Fprintf(&sb, "z")
		}
		i += cmdLen(cmd)
	}
	return sb.String()
}

// ToSVG returns a string that represents the path in the SVG path data format with minification.
func (p *Path) ToSVG() string {
	if p.Empty() {
		return ""
	}

	sb := strings.Builder{}
	var x, y float64
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, "M%v %v", num(x), num(y))
		case LineToCmd:
			xStart, yStart := x, y
			x, y = p.d[i+1], p.d[i+2]
			if Equal(x, xStart) && Equal(y, yStart) {
				// nothing
			} else if Equal(x, xStart) {
				fmt.Fprintf(&sb, "V%v", num(y))
			} else if Equal(y, yStart) {
				fmt.Fprintf(&sb, "H%v", num(x))
			} else {
				fmt.Fprintf(&sb, "L%v %v", num(x), num(y))
			}
		case QuadToCmd:
			x, y = p.d[i+3], p.d[i+4]
			fmt.Fprintf(&sb, "Q%v %v %v %v", num(p.d[i+1]), num(p.d[i+2]), num(x), num(y))
		case CubeToCmd:
			x, y = p.d[i+5], p.d[i+6]
			fmt.Fprintf(&sb, "C%v %v %v %v %v %v", num(p.d[i+1]), num(p.d[i+2]), num(p.d[i+3]), num(p.d[i+4]), num(x), num(y))
		case ArcToCmd:
			rx, ry := p.d[i+1], p.d[i+2]
			rot := p.d[i+3] * 180.0 / math.Pi
			large, sweep := toArcFlags(p.d[i+4])
			x, y = p.d[i+5], p.d[i+6]
			sLarge := "0"
			if large {
				sLarge = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			if 90.0 <= rot {
				rx, ry = ry, rx
				rot -= 90.0
			}
			fmt.Fprintf(&sb, "A%v %v %v %s%s%v %v", num(rx), num(ry), num(rot), sLarge, sSweep, num(p.d[i+5]), num(p.d[i+6]))
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, "z")
		}
		i += cmdLen(cmd)
	}
	return sb.String()
}

// ToPS returns a string that represents the path in the PostScript data format.
func (p *Path) ToPS() string {
	if p.Empty() {
		return ""
	}

	sb := strings.Builder{}
	var x, y float64
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %v %v moveto", dec(x), dec(y))
		case LineToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %v %v lineto", dec(x), dec(y))
		case QuadToCmd, CubeToCmd:
			var start, cp1, cp2 Point
			start = Point{x, y}
			if cmd == QuadToCmd {
				x, y = p.d[i+3], p.d[i+4]
				cp1, cp2 = quadraticToCubicBezier(start, Point{p.d[i+1], p.d[i+2]}, Point{x, y})
			} else {
				cp1 = Point{p.d[i+1], p.d[i+2]}
				cp2 = Point{p.d[i+3], p.d[i+4]}
				x, y = p.d[i+5], p.d[i+6]
			}
			fmt.Fprintf(&sb, " %v %v %v %v %v %v curveto", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(x), dec(y))
		case ArcToCmd:
			x0, y0 := x, y
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			x, y = p.d[i+5], p.d[i+6]

			cx, cy, theta0, theta1 := ellipseToCenter(x0, y0, rx, ry, phi, large, sweep, x, y)
			theta0 = theta0 * 180.0 / math.Pi
			theta1 = theta1 * 180.0 / math.Pi
			rot := phi * 180.0 / math.Pi

			fmt.Fprintf(&sb, " %v %v %v %v %v %v %v ellipse", dec(cx), dec(cy), dec(rx), dec(ry), dec(theta0), dec(theta1), dec(rot))
			if !sweep {
				fmt.Fprintf(&sb, "n")
			}
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " closepath")
		}
		i += cmdLen(cmd)
	}
	return sb.String()[1:] // remove the first space
}

// ToPDF returns a string that represents the path in the PDF data format.
func (p *Path) ToPDF() string {
	if p.Empty() {
		return ""
	}
	p = p.ReplaceArcs()

	sb := strings.Builder{}
	var x, y float64
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %v %v m", dec(x), dec(y))
		case LineToCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " %v %v l", dec(x), dec(y))
		case QuadToCmd, CubeToCmd:
			var start, cp1, cp2 Point
			start = Point{x, y}
			if cmd == QuadToCmd {
				x, y = p.d[i+3], p.d[i+4]
				cp1, cp2 = quadraticToCubicBezier(start, Point{p.d[i+1], p.d[i+2]}, Point{x, y})
			} else {
				cp1 = Point{p.d[i+1], p.d[i+2]}
				cp2 = Point{p.d[i+3], p.d[i+4]}
				x, y = p.d[i+5], p.d[i+6]
			}
			fmt.Fprintf(&sb, " %v %v %v %v %v %v c", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(x), dec(y))
		case ArcToCmd:
			panic("arcs should have been replaced")
		case CloseCmd:
			x, y = p.d[i+1], p.d[i+2]
			fmt.Fprintf(&sb, " h")
		}
		i += cmdLen(cmd)
	}
	return sb.String()[1:] // remove the first space
}

// ToRasterizer rasterizes the path using the given rasterizer and resolution.
func (p *Path) ToRasterizer(ras *vector.Rasterizer, resolution Resolution) {
	p = p.ReplaceArcs()

	dpmm := resolution.DPMM()
	dy := float64(ras.Bounds().Size().Y)
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			ras.MoveTo(float32(p.d[i+1]*dpmm), float32(dy-p.d[i+2]*dpmm))
		case LineToCmd:
			ras.LineTo(float32(p.d[i+1]*dpmm), float32(dy-p.d[i+2]*dpmm))
		case QuadToCmd:
			ras.QuadTo(float32(p.d[i+1]*dpmm), float32(dy-p.d[i+2]*dpmm), float32(p.d[i+3]*dpmm), float32(dy-p.d[i+4]*dpmm))
		case CubeToCmd:
			ras.CubeTo(float32(p.d[i+1]*dpmm), float32(dy-p.d[i+2]*dpmm), float32(p.d[i+3]*dpmm), float32(dy-p.d[i+4]*dpmm), float32(p.d[i+5]*dpmm), float32(dy-p.d[i+6]*dpmm))
		case ArcToCmd:
			panic("arcs should have been replaced")
		case CloseCmd:
			ras.ClosePath()
		}
		i += cmdLen(cmd)
	}
	if 0 < len(p.d) && p.d[len(p.d)-1] != CloseCmd {
		// implicitly close path
		ras.ClosePath()
	}
}
