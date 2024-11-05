package canvas

import (
	"math"
)

func snap(v, d float64) float64 {
	f := math.Mod(v+0.5*d, d)
	return v - f
}

// Gridsnap snaps all vertices to a grid with the given spacing. This will significantly reduce numerical issues e.g. for path boolean operations.
func (p *Path) Gridsnap(spacing float64) *Path {
	q := &Path{d: make([]float64, 0, len(p.d))}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			q.MoveTo(snap(p.d[i+1], spacing), snap(p.d[i+2], spacing))
		case LineToCmd:
			q.LineTo(snap(p.d[i+1], spacing), snap(p.d[i+2], spacing))
		case QuadToCmd:
			q.QuadTo(snap(p.d[i+1], spacing), snap(p.d[i+2], spacing), snap(p.d[i+3], spacing), snap(p.d[i+4], spacing))
		case CubeToCmd:
			q.CubeTo(snap(p.d[i+1], spacing), snap(p.d[i+2], spacing), snap(p.d[i+3], spacing), snap(p.d[i+4], spacing), snap(p.d[i+5], spacing), snap(p.d[i+6], spacing))
		case ArcToCmd:
			large, sweep := toArcFlags(p.d[i+4])
			q.ArcTo(p.d[i+1], p.d[i+2], p.d[i+3], large, sweep, snap(p.d[i+5], spacing), snap(p.d[i+6], spacing))
		case CloseCmd:
			q.d = append(q.d, CloseCmd, snap(p.d[i+1], spacing), snap(p.d[i+2], spacing), CloseCmd)
		}
		i += cmdLen(cmd)
	}
	return q
}

// Decimate decimates the path using the Visvalingam-Whyatt algorithm. Assuming path is flat and has no subpaths.
func (p *Path) Decimate(tolerance float64) *Path {
	q := &Path{}
Loop:
	for _, pi := range p.Split() {
		// indices are always one past the current point with -1 the command and [-3,-2] the endpoint
		var is []int // stack of coordinate indices
		closed := pi.d[len(pi.d)-1] == CloseCmd
		if closed {
			// put before-close command first
			is = append(is, len(pi.d)-cmdLen(CloseCmd))
		}

		i := 0
		for len(is) < 3 {
			if len(pi.d) <= i {
				q = q.Append(pi)
				continue Loop
			}
			i += cmdLen(pi.d[i])
			is = append(is, i)
		}

		// find indices of triangles with an area superior or equal to tolerance
		for {
			iPrev, iCur, iNext := is[len(is)-3], is[len(is)-2], is[len(is)-1]
			prev := Point{pi.d[iPrev-3], pi.d[iPrev-2]}
			cur := Point{pi.d[iCur-3], pi.d[iCur-2]}
			next := Point{pi.d[iNext-3], pi.d[iNext-2]}
			area := 0.5 * math.Abs(prev.X*cur.Y+cur.X*next.Y+next.X*prev.Y-prev.X*next.Y-cur.X*prev.Y-next.X*cur.Y)
			if area < tolerance {
				// remove point
				is[len(is)-2] = is[len(is)-1] // cur = next
				is = is[:len(is)-1]
			}
			if tolerance <= area || len(is) < 3 {
				// advance to next triangle
				if len(pi.d) <= i {
					// end of path
					break
				} else if closed && i == is[0] {
					if iNext < iCur || len(is) < 3 {
						// past the end, no point is removed, so we're done
						break
					}

					// end of closed path, move first index to the end
					is = append(is, is[0])
					is = is[1:]
					i = is[0]
				} else {
					i += cmdLen(pi.d[i])
					is = append(is, i)
				}
			}
		}

		// build the new path
		if len(is) < 2 || closed && len(is) < 3 {
			continue
		}
		if closed {
			q.MoveTo(pi.d[is[len(is)-1]-3], pi.d[is[len(is)-1]-2])
			is = is[:len(is)-1]
		} else {
			q.MoveTo(pi.d[is[0]-3], pi.d[is[0]-2])
			is = is[1:]
		}
		for _, i := range is {
			q.d = append(q.d, pi.d[i-cmdLen(pi.d[i-1]):i]...)
		}
		if closed {
			q.Close()
		}
	}
	return q
}
