package canvas

import "math"

// Scanner returns a path scanner.
func (p *Path) Scanner() *PathScanner {
	return &PathScanner{p, -1}
}

// ReverseScanner returns a path scanner in reverse order.
func (p *Path) ReverseScanner() *PathReverseScanner {
	return &PathReverseScanner{p, len(p.d)}
}

// PathScanner scans the path.
type PathScanner struct {
	p *Path
	i int
}

// Scan scans a new path segment and should be called before the other methods.
func (s *PathScanner) Scan() bool {
	if s.i+1 < len(s.p.d) {
		s.i += cmdLen(s.p.d[s.i+1])
		return true
	}
	return false
}

// Cmd returns the current path segment command.
func (s *PathScanner) Cmd() float64 {
	return s.p.d[s.i]
}

// Values returns the current path segment values.
func (s *PathScanner) Values() []float64 {
	return s.p.d[s.i-cmdLen(s.p.d[s.i])+2 : s.i]
}

// Start returns the current path segment start position.
func (s *PathScanner) Start() Point {
	i := s.i - cmdLen(s.p.d[s.i])
	if i == -1 {
		return Point{}
	}
	return Point{s.p.d[i-2], s.p.d[i-1]}
}

// CP1 returns the first control point for quadratic and cubic Béziers.
func (s *PathScanner) CP1() Point {
	if s.p.d[s.i] != QuadToCmd && s.p.d[s.i] != CubeToCmd {
		panic("must be quadratic or cubic Bézier")
	}
	i := s.i - cmdLen(s.p.d[s.i]) + 1
	return Point{s.p.d[i+1], s.p.d[i+2]}
}

// CP2 returns the second control point for cubic Béziers.
func (s *PathScanner) CP2() Point {
	if s.p.d[s.i] != CubeToCmd {
		panic("must be cubic Bézier")
	}
	i := s.i - cmdLen(s.p.d[s.i]) + 1
	return Point{s.p.d[i+3], s.p.d[i+4]}
}

// Arc returns the arguments for arcs (rx,ry,rot,large,sweep).
func (s *PathScanner) Arc() (float64, float64, float64, bool, bool) {
	if s.p.d[s.i] != ArcToCmd {
		panic("must be arc")
	}
	i := s.i - cmdLen(s.p.d[s.i]) + 1
	large, sweep := toArcFlags(s.p.d[i+4])
	return s.p.d[i+1], s.p.d[i+2], s.p.d[i+3] * 180.0 / math.Pi, large, sweep
}

// End returns the current path segment end position.
func (s *PathScanner) End() Point {
	return Point{s.p.d[s.i-2], s.p.d[s.i-1]}
}

// Path returns the current path segment.
func (s *PathScanner) Path() *Path {
	p := &Path{}
	p.MoveTo(s.Start().X, s.Start().Y)
	switch s.Cmd() {
	case LineToCmd:
		p.LineTo(s.End().X, s.End().Y)
	case QuadToCmd:
		p.QuadTo(s.CP1().X, s.CP1().Y, s.End().X, s.End().Y)
	case CubeToCmd:
		p.CubeTo(s.CP1().X, s.CP1().Y, s.CP2().X, s.CP2().Y, s.End().X, s.End().Y)
	case ArcToCmd:
		rx, ry, rot, large, sweep := s.Arc()
		p.ArcTo(rx, ry, rot, large, sweep, s.End().X, s.End().Y)
	}
	return p
}

// PathReverseScanner scans the path in reverse order.
type PathReverseScanner struct {
	p *Path
	i int
}

// Scan scans a new path segment and should be called before the other methods.
func (s *PathReverseScanner) Scan() bool {
	if 0 < s.i {
		s.i -= cmdLen(s.p.d[s.i-1])
		return true
	}
	return false
}

// Cmd returns the current path segment command.
func (s *PathReverseScanner) Cmd() float64 {
	return s.p.d[s.i]
}

// Values returns the current path segment values.
func (s *PathReverseScanner) Values() []float64 {
	return s.p.d[s.i+1 : s.i+cmdLen(s.p.d[s.i])-1]
}

// Start returns the current path segment start position.
func (s *PathReverseScanner) Start() Point {
	if s.i == 0 {
		return Point{}
	}
	return Point{s.p.d[s.i-3], s.p.d[s.i-2]}
}

// CP1 returns the first control point for quadratic and cubic Béziers.
func (s *PathReverseScanner) CP1() Point {
	if s.p.d[s.i] != QuadToCmd && s.p.d[s.i] != CubeToCmd {
		panic("must be quadratic or cubic Bézier")
	}
	return Point{s.p.d[s.i+1], s.p.d[s.i+2]}
}

// CP2 returns the second control point for cubic Béziers.
func (s *PathReverseScanner) CP2() Point {
	if s.p.d[s.i] != CubeToCmd {
		panic("must be cubic Bézier")
	}
	return Point{s.p.d[s.i+3], s.p.d[s.i+4]}
}

// Arc returns the arguments for arcs (rx,ry,rot,large,sweep).
func (s *PathReverseScanner) Arc() (float64, float64, float64, bool, bool) {
	if s.p.d[s.i] != ArcToCmd {
		panic("must be arc")
	}
	large, sweep := toArcFlags(s.p.d[s.i+4])
	return s.p.d[s.i+1], s.p.d[s.i+2], s.p.d[s.i+3] * 180.0 / math.Pi, large, sweep
}

// End returns the current path segment end position.
func (s *PathReverseScanner) End() Point {
	i := s.i + cmdLen(s.p.d[s.i])
	return Point{s.p.d[i-3], s.p.d[i-2]}
}

// Path returns the current path segment.
func (s *PathReverseScanner) Path() *Path {
	p := &Path{}
	p.MoveTo(s.Start().X, s.Start().Y)
	switch s.Cmd() {
	case LineToCmd:
		p.LineTo(s.End().X, s.End().Y)
	case QuadToCmd:
		p.QuadTo(s.CP1().X, s.CP1().Y, s.End().X, s.End().Y)
	case CubeToCmd:
		p.CubeTo(s.CP1().X, s.CP1().Y, s.CP2().X, s.CP2().Y, s.End().X, s.End().Y)
	case ArcToCmd:
		rx, ry, rot, large, sweep := s.Arc()
		p.ArcTo(rx, ry, rot, large, sweep, s.End().X, s.End().Y)
	}
	return p
}
