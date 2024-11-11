package canvas

import (
	"math"
)

func snap(v, d float64) float64 {
	return math.Round(v/d) * d
}

// Gridsnap snaps all vertices to a grid with the given spacing. This will significantly reduce numerical issues e.g. for path boolean operations. This operation is in-place.
func (p *Path) Gridsnap(spacing float64) *Path {
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			p.d[i+1] = snap(p.d[i+1], spacing)
			p.d[i+2] = snap(p.d[i+2], spacing)
		case LineToCmd:
			p.d[i+1] = snap(p.d[i+1], spacing)
			p.d[i+2] = snap(p.d[i+2], spacing)
		case QuadToCmd:
			p.d[i+1] = snap(p.d[i+1], spacing)
			p.d[i+2] = snap(p.d[i+2], spacing)
			p.d[i+3] = snap(p.d[i+3], spacing)
			p.d[i+4] = snap(p.d[i+4], spacing)
		case CubeToCmd:
			p.d[i+1] = snap(p.d[i+1], spacing)
			p.d[i+2] = snap(p.d[i+2], spacing)
			p.d[i+3] = snap(p.d[i+3], spacing)
			p.d[i+4] = snap(p.d[i+4], spacing)
			p.d[i+5] = snap(p.d[i+5], spacing)
			p.d[i+6] = snap(p.d[i+6], spacing)
		case ArcToCmd:
			p.d[i+1] = snap(p.d[i+1], spacing)
			p.d[i+2] = snap(p.d[i+2], spacing)
			p.d[i+5] = snap(p.d[i+5], spacing)
			p.d[i+6] = snap(p.d[i+6], spacing)
		case CloseCmd:
			p.d[i+1] = snap(p.d[i+1], spacing)
			p.d[i+2] = snap(p.d[i+2], spacing)
		}
		i += cmdLen(cmd)
	}
	return p
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

// Clip removes all segments that are completely outside the given clipping rectangle. To ensure that the removal doesn't cause a segment to cross the rectangle from the outside, it keeps points that cross at least two lines to infinity along the rectangle's edges. This is much quicker (along O(n)) than using p.And(canvas.Rectangle(x1-x0, y1-y0).Translate(x0, y0)) (which is O(n log n)).
func (p *Path) Clip(x0, y0, x1, y1 float64) *Path {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	rect := Rect{x0, y0, x1, y1}

	startIn := false
	pendingMoveTo := true
	first, start := Point{}, Point{}
	var moveToIndex, firstInIndex int
	q := &Path{} //d: p.d[:0]} // q is always smaller or equal to p
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)

		end := Point{p.d[i-3], p.d[i-2]}
		endIn := rect.TouchesPoint(end)
		crossesXY := false
		if 0 < len(q.d) && !startIn && !endIn {
			// check if between out last position on Q and the end of the current segment we cross
			// lines along X0, X1, Y0, or Y1. If we cross both an X and Y boundary, the path may
			// cross into the clipping rectangle, so we must include some points on the exterior
			// of the clipping rectangle to prevent that.
			prev := Point{q.d[len(q.d)-3], q.d[len(q.d)-2]}
			crossesX0 := prev.X < rect.X0 && rect.X0 < end.X || rect.X0 < prev.X && end.X < rect.X0
			crossesX1 := prev.X < rect.X1 && rect.X1 < end.X || rect.X1 < prev.X && end.X < rect.X1
			crossesY0 := prev.Y < rect.Y0 && rect.Y0 < end.Y || rect.Y0 < prev.Y && end.Y < rect.Y0
			crossesY1 := prev.Y < rect.Y1 && rect.Y1 < end.Y || rect.Y1 < prev.Y && end.Y < rect.Y1
			crossesXY = (crossesX0 || crossesX1) && (crossesY0 || crossesY1)
		}

		if cmd == MoveToCmd {
			if endIn {
				q.d = append(q.d, MoveToCmd, end.X, end.Y, MoveToCmd)
				pendingMoveTo = false
				firstInIndex = i
				first = end
			} else {
				pendingMoveTo = true
			}
			moveToIndex = i - cmdLen(MoveToCmd)
		} else {
			if crossesXY || !startIn && endIn {
				if pendingMoveTo {
					q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
					pendingMoveTo = false
					firstInIndex = i
					first = start
				} else {
					q.d = append(q.d, LineToCmd, start.X, start.Y, LineToCmd)
				}
			}
			if cmd == LineToCmd && (startIn || endIn) {
				if pendingMoveTo {
					q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
					pendingMoveTo = false
					firstInIndex = i
					first = start
				}
				q.d = append(q.d, p.d[i-4:i]...)
			} else if cmd == QuadToCmd {
				cp := Point{p.d[i-5], p.d[i-4]}
				if startIn || endIn || rect.TouchesPoint(cp) {
					if pendingMoveTo {
						q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
						pendingMoveTo = false
						firstInIndex = i
						first = start
					}
					q.d = append(q.d, p.d[i-6:i]...)
				}
			} else if cmd == CubeToCmd {
				cp0 := Point{p.d[i-7], p.d[i-6]}
				cp1 := Point{p.d[i-5], p.d[i-4]}
				if startIn || endIn || rect.TouchesPoint(cp0) || rect.TouchesPoint(cp1) {
					if pendingMoveTo {
						q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
						pendingMoveTo = false
						firstInIndex = i
						first = start
					}
					q.d = append(q.d, p.d[i-8:i]...)
				}
			} else if cmd == ArcToCmd {
				touches := startIn || endIn
				rx, ry, phi := p.d[i-7], p.d[i-6], p.d[i-5]
				large, sweep := toArcFlags(p.d[i-4])
				if !touches {
					cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

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
					if angleBetween(thetaLeft, theta0, theta1) {
						touches = touches || rect.TouchesPoint(Point{cx - dx, cy})
					}
					if angleBetween(thetaRight, theta0, theta1) {
						touches = touches || rect.TouchesPoint(Point{cx + dx, cy})
					}
					if angleBetween(thetaBottom, theta0, theta1) {
						touches = touches || rect.TouchesPoint(Point{cx, cy - dy})
					}
					if angleBetween(thetaTop, theta0, theta1) {
						touches = touches || rect.TouchesPoint(Point{cx, cy + dy})
					}
				}
				if touches {
					if pendingMoveTo {
						q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
						pendingMoveTo = false
						firstInIndex = i
						first = start
					}
					q.d = append(q.d, p.d[i-8:i]...)
				}
			} else if cmd == CloseCmd {
				if !pendingMoveTo {
					// handle first part of the path which may cross boundaries
					start = Point{p.d[moveToIndex+1], p.d[moveToIndex+2]}
					for i := moveToIndex; ; {
						cmd := p.d[i]
						i += cmdLen(cmd)
						if firstInIndex <= i {
							break
						}

						end := Point{p.d[i-3], p.d[i-2]}
						prev := Point{q.d[len(q.d)-3], q.d[len(q.d)-2]}
						crossesX0 := prev.X < rect.X0 && rect.X0 < end.X || rect.X0 < prev.X && end.X < rect.X0
						crossesX1 := prev.X < rect.X1 && rect.X1 < end.X || rect.X1 < prev.X && end.X < rect.X1
						crossesY0 := prev.Y < rect.Y0 && rect.Y0 < end.Y || rect.Y0 < prev.Y && end.Y < rect.Y0
						crossesY1 := prev.Y < rect.Y1 && rect.Y1 < end.Y || rect.Y1 < prev.Y && end.Y < rect.Y1
						if (crossesX0 || crossesX1) && (crossesY0 || crossesY1) {
							q.d = append(q.d, LineToCmd, start.X, start.Y, LineToCmd)
						}
						start = end
					}
					q.d = append(q.d, CloseCmd, first.X, first.Y, CloseCmd)
				}
				pendingMoveTo = true
			}
		}
		start = end
		startIn = endIn
	}
	return q
}
