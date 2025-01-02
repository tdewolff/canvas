package canvas

import (
	"fmt"
	"math"
)

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

type simplifyItemVW struct {
	Point
	area       float64
	prev, next int
}

func (item simplifyItemVW) String() string {
	return fmt.Sprintf("%v %v (%v→·→%v)", item.Point, item.area, item.prev, item.next)
}

func (p *Path) SimplifyVisvalingamWhyatt(tolerance float64) *Path {
	return p.SimplifyVisvalingamWhyattFilter(tolerance, nil)
}

func (p *Path) SimplifyVisvalingamWhyattFilter(tolerance float64, filter func(Point) bool) *Path {
	area := func(a, b, c Point) float64 {
		return 0.5 * math.Abs(a.PerpDot(b)+b.PerpDot(c)+c.PerpDot(a))
	}

	// don't reuse memory since the new path may be much smaller and keep the extra capacity
	q := &Path{}
	pq := NewPriorityQueue[int](nil, 0)
	for _, pi := range p.Split() {
		if len(pi.d) <= 4 || len(pi.d) <= 4+cmdLen(pi.d[4]) {
			// must have at least 3 commands
			continue
		}

		closed := pi.Closed()
		prev, cur := Point{}, Point{pi.d[1], pi.d[2]}
		if closed {
			prev = Point{pi.d[len(pi.d)-7], pi.d[len(pi.d)-6]}
		}

		length := pi.Len()
		list := make([]simplifyItemVW, 0, length)
		pq.Reset(func(i, j int) bool {
			return list[i].area < list[j].area
		}, length)
		for i := 4; i < len(pi.d); i += cmdLen(pi.d[i]) {
			A := 0.0
			idx := len(list)
			j := i + cmdLen(pi.d[i])
			next := Point{pi.d[j-3], pi.d[j-2]}
			if (4 < i || closed) && (filter == nil || filter(cur)) {
				A = area(prev, cur, next)
				pq.Append(idx)
			}
			list = append(list, simplifyItemVW{
				Point: cur,
				area:  A,
				prev:  idx - 1,
				next:  idx + 1,
			})
			prev = cur
			cur = next
		}
		if closed {
			list[len(list)-1].next = 0
			list[0].prev = len(list) - 1
		} else {
			list = append(list, simplifyItemVW{
				Point: Point{pi.d[len(pi.d)-3], pi.d[len(pi.d)-2]},
				area:  0.0,
				prev:  len(list) - 1,
				next:  -1,
			})
		}
		pq.Init()

		first := 0
		for 0 < pq.Len() {
			idx := pq.Pop()
			cur := list[idx]
			if math.IsNaN(cur.area) {
				continue
			} else if tolerance <= cur.area {
				break
			}

			// remove current point
			list[cur.prev].next = cur.next
			list[cur.next].prev = cur.prev
			if first == idx {
				first = cur.next
			}

			// update previous point
			if prev := list[cur.prev]; prev.prev != -1 {
				idxPrev, _ := pq.Find(cur.prev)
				list[cur.prev].area = area(list[prev.prev].Point, prev.Point, list[prev.next].Point)
				pq.Fix(idxPrev)
			}

			// update next point
			if next := list[cur.next]; next.next != -1 {
				idxNext, _ := pq.Find(cur.next)
				list[cur.next].area = area(list[next.prev].Point, next.Point, list[next.next].Point)
				pq.Fix(idxNext)
			}
		}
		if closed && pq.Len() < 2 {
			// result too small
			continue
		}

		q.d = append(q.d, MoveToCmd, list[first].X, list[first].Y, MoveToCmd)
		for idx := list[first].next; idx != -1 && idx != first; idx = list[idx].next {
			q.d = append(q.d, LineToCmd, list[idx].X, list[idx].Y, LineToCmd)
		}
		if closed {
			q.d = append(q.d, CloseCmd, list[first].X, list[first].Y, CloseCmd)
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

	// don't reuse memory since the new path may be much smaller and keep the extra capacity
	q := &Path{}
	startIn := false
	pendingMoveTo := true
	first, start := Point{}, Point{}
	var moveToIndex, firstInIndex int
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
