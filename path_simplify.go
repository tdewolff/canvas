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

type itemVW struct {
	Point
	area       float64
	prev, next int32 // indices into points
}

func (item itemVW) String() string {
	return fmt.Sprintf("%v %v (%v→·→%v)", item.Point, item.area, item.prev, item.next)
}

func (p *Path) SimplifyVisvalingamWhyatt(tolerance float64) *Path {
	return p.SimplifyVisvalingamWhyattFilter(tolerance, nil)
}

func (p *Path) SimplifyVisvalingamWhyattFilter(tolerance float64, filter func(Point) bool) *Path {
	computeArea := func(a, b, c Point) float64 {
		return 0.5 * math.Abs(a.PerpDot(b)+b.PerpDot(c)+c.PerpDot(a))
	}

	// don't reuse memory since the new path may be much smaller and keep the extra capacity
	q := &Path{}
	pq := NewPriorityQueue[int32](nil, 0)
SubpathLoop:
	for _, pi := range p.Split() {
		closed := pi.Closed()
		if len(pi.d) <= 4 || closed && len(pi.d) <= 4+cmdLen(pi.d[4]) {
			// must have at least 2 commands for open paths, and 3 for closed
			continue
		}

		prev, cur := Point{}, Point{pi.d[1], pi.d[2]}
		if closed {
			prev = Point{pi.d[len(pi.d)-7], pi.d[len(pi.d)-6]}
		}

		length := pi.Len()
		if closed {
			length--
		}
		items := make([]itemVW, 0, length)
		pq.Reset(func(i, j int32) bool {
			return items[i].area < items[j].area
		}, length)
		for i := 4; i < len(pi.d); {
			j := i + cmdLen(pi.d[i])
			next := Point{pi.d[j-3], pi.d[j-2]}

			idx := int32(len(items))
			area := math.NaN()
			if (4 < i || closed) && (filter == nil || filter(cur)) {
				area = computeArea(prev, cur, next)
				pq.Append(idx)
			}

			idxPrev, idxNext := idx-1, idx+1
			if closed {
				if i == 4 {
					idxPrev = int32(length - 1)
				} else if j == len(pi.d) {
					idxNext = 0
				}
			}
			items = append(items, itemVW{
				Point: cur,
				area:  area,
				prev:  idxPrev,
				next:  idxNext,
			})
			prev = cur
			cur = next
			i = j
		}
		if !closed {
			items = append(items, itemVW{
				Point: cur,
				area:  math.NaN(),
				prev:  int32(len(items) - 1),
				next:  -1,
			})
		}

		pq.Init()

		first := int32(0)
		for 0 < pq.Len() {
			idx := pq.Pop()
			item := items[idx]
			if tolerance <= item.area {
				break
			} else if item.prev == item.next {
				// fewer than 3 points left
				continue SubpathLoop
			}

			// remove current point from linked list, this invalidates those items in the queue
			items[item.prev].next = item.next
			items[item.next].prev = item.prev
			if first == idx {
				first = item.next
			}

			// update previous point
			if prev := items[item.prev]; prev.prev != -1 && !math.IsNaN(prev.area) {
				area := computeArea(items[prev.prev].Point, prev.Point, items[prev.next].Point)
				items[item.prev].area = area
				idx, _ := pq.Find(item.prev)
				pq.Fix(idx)
			}

			// update next point
			if next := items[item.next]; next.next != -1 && !math.IsNaN(next.area) {
				area := computeArea(items[next.prev].Point, next.Point, items[next.next].Point)
				items[item.next].area = area
				idx, _ := pq.Find(item.next)
				pq.Fix(idx)
			}
		}

		q.d = append(q.d, MoveToCmd, items[first].X, items[first].Y, MoveToCmd)
		for i := items[first].next; i != -1 && i != first; i = items[i].next {
			q.d = append(q.d, LineToCmd, items[i].X, items[i].Y, LineToCmd)
		}
		if closed {
			q.d = append(q.d, CloseCmd, items[first].X, items[first].Y, CloseCmd)
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
	if len(p.d) <= 4 {
		return q
	}

	// TODO: we could check if the path is only in two external regions (left/right and top/bottom)
	//       and if no segment crosses the rectangle, it is fully outside the rectangle

	var rectSegs Rect // sum of rects of prev removed points
	var first, start, prev Point
	//crosses := false
	pendingMoveTo := true
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)

		end := Point{p.d[i-3], p.d[i-2]}
		if cmd == MoveToCmd {
			rectSegs = Rect{end.X, end.Y, end.X, end.Y}
			pendingMoveTo = true
			start = end
			continue
		}

		rectSeg := RectFromPoints(start, end)
		switch cmd {
		//case LineToCmd, CloseCmd:
		//if !crosses && rect.Touches(rectSeg) {
		//	crosses = true
		//}
		case QuadToCmd:
			rectSeg = rectSeg.AddPoint(Point{p.d[i-5], p.d[i-4]})
			//if !crosses && rect.Touches(rectSeg) {
			//	crosses = true
			//}
		case CubeToCmd:
			rectSeg = rectSeg.AddPoint(Point{p.d[i-7], p.d[i-6]})
			rectSeg = rectSeg.AddPoint(Point{p.d[i-5], p.d[i-4]})
			//if !crosses && rect.Touches(rectSeg) {
			//	crosses = true
			//}
		case ArcToCmd:
			rx, ry, phi := p.d[i-7], p.d[i-6], p.d[i-5]
			large, sweep := toArcFlags(p.d[i-4])
			cx, cy, _, _ := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			rectSeg = rectSeg.AddPoint(Point{cx - rx, cy - ry})
			rectSeg = rectSeg.AddPoint(Point{cx + rx, cy + ry})
			//if !crosses && rect.Touches(rectSeg) {
			//	crosses = true
			//}
		}

		rectSegs = rectSegs.Add(rectSeg)
		if cmd == CloseCmd {
			if !pendingMoveTo {
				if rect.Touches(rectSegs) && start != prev {
					// previous segments were skipped
					q.d = append(q.d, LineToCmd, start.X, start.Y, LineToCmd)
				}
				if end != first {
					// original moveTo was ignored, but now we need it
					q.d = append(q.d, LineToCmd, end.X, end.Y, LineToCmd)
				}
				q.d = append(q.d, CloseCmd, first.X, first.Y, CloseCmd)
				pendingMoveTo = true
			}
		} else if rect.Touches(rectSegs) {
			if pendingMoveTo {
				q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
				pendingMoveTo = false
				first = start
			} else if start != prev {
				q.d = append(q.d, LineToCmd, start.X, start.Y, LineToCmd)
			}
			q.d = append(q.d, p.d[i-cmdLen(cmd):i]...)
			rectSegs = Rect{end.X, end.Y, end.X, end.Y}
			prev = end
		}
		start = end
	}
	return q
}
