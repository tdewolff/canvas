package canvas

import "math"

type PathScanner struct {
	p *Path
	i int
}

func (s *PathScanner) Scan() bool {
	if s.i+1 < len(s.p.d) {
		s.i += cmdLen(s.p.d[s.i+1])
		return true
	}
	return false
}

func (s *PathScanner) Cmd() float64 {
	return s.p.d[s.i]
}

func (s *PathScanner) Values() []float64 {
	return s.p.d[s.i-cmdLen(s.p.d[s.i])+2 : s.i]
}

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

func (s *PathScanner) End() Point {
	return Point{s.p.d[s.i-2], s.p.d[s.i-1]}
}

type PathReverseScanner struct {
	p *Path
	i int
}

func (s *PathReverseScanner) Scan() bool {
	if 0 < s.i {
		s.i -= cmdLen(s.p.d[s.i-1])
		return true
	}
	return false
}

func (s *PathReverseScanner) Cmd() float64 {
	return s.p.d[s.i]
}

func (s *PathReverseScanner) Values() []float64 {
	return s.p.d[s.i+1 : s.i+cmdLen(s.p.d[s.i])-1]
}

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

func (s *PathReverseScanner) End() Point {
	i := s.i + cmdLen(s.p.d[s.i])
	return Point{s.p.d[i-3], s.p.d[i-2]}
}
