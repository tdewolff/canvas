package canvas

import "math/rand/v2"

func RandomPath(n int, lineOnly, closed bool) *Path {
	p := &Path{}
	if 0 < n {
		p.d = append(p.d, MoveToCmd, rand.NormFloat64(), rand.NormFloat64(), MoveToCmd)
		for i := 1; i < n; i++ {
			cmd := 0
			if !lineOnly {
				cmd = rand.IntN(4)
			}
			switch cmd {
			case 0:
				p.d = append(p.d, LineToCmd, rand.NormFloat64(), rand.NormFloat64(), LineToCmd)
			case 1:
				p.d = append(p.d, QuadToCmd, rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), QuadToCmd)
			case 2:
				p.d = append(p.d, CubeToCmd, rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), CubeToCmd)
			case 3:
				large, sweep := rand.IntN(2) == 0, rand.IntN(2) == 0
				p.d = append(p.d, ArcToCmd, rand.NormFloat64(), rand.NormFloat64(), rand.NormFloat64(), fromArcFlags(large, sweep), rand.NormFloat64(), rand.NormFloat64(), ArcToCmd)
			}
		}
		if closed {
			p.d = append(p.d, CloseCmd, p.d[1], p.d[2], CloseCmd)
		}
	}
	return p
}
