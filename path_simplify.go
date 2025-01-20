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

func (p *Path) SimplifyVisvalingamWhyatt(tolerance float64) *Path {
	return NewVisvalingamWhyatt(nil).Simplify(p.Split(), tolerance)
}

type itemVW struct {
	Point
	area       float64
	prev, next int32 // indices into items
	heapIdx    int32
}

func (item itemVW) String() string {
	return fmt.Sprintf("%v %v (%v→·→%v)", item.Point, item.area, item.prev, item.next)
}

type CoordinateFilter func(Point) bool

type VisvalingamWhyatt struct {
	heap   heapVW
	items  []itemVW
	filter CoordinateFilter
}

func NewVisvalingamWhyatt(filter CoordinateFilter) *VisvalingamWhyatt {
	return &VisvalingamWhyatt{
		filter: filter,
	}
}

func (s *VisvalingamWhyatt) Simplify(ps []*Path, tolerance float64) *Path {
	computeArea := func(a, b, c Point) float64 {
		return math.Abs(a.PerpDot(b) + b.PerpDot(c) + c.PerpDot(a))
	}
	tolerance *= 2.0 // save on 0.5 multiply in computeArea

	q := &Path{}
SubpathLoop:
	for _, pi := range ps {
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
		if cap(s.items) < length {
			s.items = make([]itemVW, 0, length)
		} else {
			s.items = s.items[:0]
		}
		s.heap.Reset(length)

		tooSmall := closed
		bounds := Rect{cur.X, cur.Y, cur.X, cur.Y}
		for i := 4; i < len(pi.d); {
			j := i + cmdLen(pi.d[i])
			next := Point{pi.d[j-3], pi.d[j-2]}
			if tooSmall {
				bounds = bounds.AddPoint(next)
				if tolerance <= bounds.Area() {
					tooSmall = false
				}
			}

			idx := int32(len(s.items))
			idxPrev, idxNext := idx-1, idx+1
			if closed {
				if i == 4 {
					idxPrev = int32(length - 1)
				} else if j == len(pi.d) {
					idxNext = 0
				}
			}

			area := math.NaN()
			add := (4 < i || closed) && (s.filter == nil || s.filter(cur))
			if add {
				area = computeArea(prev, cur, next)
			}
			s.items = append(s.items, itemVW{
				Point: cur,
				area:  area,
				prev:  idxPrev,
				next:  idxNext,
			})
			if add {
				s.heap.Append(&s.items[idx])
			}

			prev = cur
			cur = next
			i = j
		}
		if tooSmall {
			continue
		}
		if !closed {
			s.items = append(s.items, itemVW{
				Point: cur,
				area:  math.NaN(),
				prev:  int32(len(s.items) - 1),
				next:  -1,
			})
		}

		s.heap.Init()

		removed := false
		first := int32(0)
		for 0 < len(s.heap) {
			item := s.heap.Pop()
			if tolerance <= item.area {
				break
			} else if item.prev == item.next {
				// fewer than 3 points left
				continue SubpathLoop
			}

			// remove current point from linked list, this invalidates those items in the queue
			s.items[item.prev].next = item.next
			s.items[item.next].prev = item.prev
			if item == &s.items[first] {
				first = item.next
			}

			// update previous point
			if prev := &s.items[item.prev]; prev.prev != -1 && !math.IsNaN(prev.area) {
				area := computeArea(s.items[prev.prev].Point, prev.Point, s.items[prev.next].Point)
				prev.area = area
				s.heap.Fix(int(prev.heapIdx))
			}

			// update next point
			if next := &s.items[item.next]; next.next != -1 && !math.IsNaN(next.area) {
				area := computeArea(s.items[next.prev].Point, next.Point, s.items[next.next].Point)
				next.area = area
				s.heap.Fix(int(next.heapIdx))
			}
			removed = true
		}

		if first == 0 && !removed {
			q.d = append(q.d, pi.d...)
		} else {
			point := s.items[first].Point
			q.d = append(q.d, MoveToCmd, point.X, point.Y, MoveToCmd)
			for i := s.items[first].next; i != -1 && i != first; i = s.items[i].next {
				point = s.items[i].Point
				q.d = append(q.d, LineToCmd, point.X, point.Y, LineToCmd)
			}
			if closed {
				point = s.items[first].Point
				q.d = append(q.d, CloseCmd, point.X, point.Y, CloseCmd)
			}
		}
	}
	return q
}

type heapVW []*itemVW

func (q *heapVW) Reset(capacity int) {
	if cap(*q) < capacity {
		*q = heapVW(make([]*itemVW, 0, capacity))
	} else {
		*q = (*q)[:0]
	}
}

func (q heapVW) Init() {
	n := len(q)
	for i := n/2 - 1; 0 <= i; i-- {
		q.down(i, n)
	}
}

func (q *heapVW) Append(item *itemVW) {
	item.heapIdx = int32(len(*q))
	*q = append(*q, item)
}

func (q *heapVW) Push(item *itemVW) {
	q.Append(item)
	q.up(len(*q) - 1)
}

func (q *heapVW) Pop() *itemVW {
	n := len(*q) - 1
	q.swap(0, n)
	q.down(0, n)

	item := (*q)[n]
	(*q) = (*q)[:n]
	return item
}

func (q heapVW) Fix(i int) {
	if !q.down(i, len(q)) {
		q.up(i)
	}
}

func (q heapVW) less(i, j int) bool {
	return q[i].area < q[j].area
}

func (q heapVW) swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].heapIdx, q[j].heapIdx = int32(i), int32(j)
}

// from container/heap
func (q heapVW) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !q.less(j, i) {
			break
		}
		q.swap(i, j)
		j = i
	}
}

func (q heapVW) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if n <= j1 || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && q.less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !q.less(j, i) {
			break
		}
		q.swap(i, j)
		i = j
	}
	return i0 < i
}

// FastClip removes all segments that are completely outside the given clipping rectangle. To ensure that the removal doesn't cause a segment to cross the rectangle from the outside, it keeps points that cross at least two lines to infinity along the rectangle's edges. This is much quicker (along O(n)) than using p.And(canvas.Rectangle(x1-x0, y1-y0).Translate(x0, y0)) (which is O(n log n)).
func (p *Path) FastClip(x0, y0, x1, y1 float64, closed bool) *Path {
	// TODO: check if path is closed while processing instead of as parameter
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

	// Note that applying AND to multiple Cohen-Sutherland outcodes will give us whether all points are left/right and/or above/below
	// the rectangle.
	var first, start, prev Point
	var pendingMoveTo bool
	var startOutcode int
	var outcodes int // cumulative of removed segments
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)

		end := Point{p.d[i-3], p.d[i-2]}
		if cmd == MoveToCmd {
			startOutcode = cohenSutherlandOutcode(rect, end, 0.0)
			outcodes = startOutcode
			start = end
			pendingMoveTo = true
			continue
		}

		endOutcode := cohenSutherlandOutcode(rect, end, 0.0)
		outcodes &= endOutcode
		switch cmd {
		case QuadToCmd:
			outcodes &= cohenSutherlandOutcode(rect, Point{p.d[i-5], p.d[i-4]}, 0.0)
		case CubeToCmd:
			outcodes &= cohenSutherlandOutcode(rect, Point{p.d[i-7], p.d[i-6]}, 0.0)
			outcodes &= cohenSutherlandOutcode(rect, Point{p.d[i-5], p.d[i-4]}, 0.0)
		case ArcToCmd:
			rx, ry, phi := p.d[i-7], p.d[i-6], p.d[i-5]
			large, sweep := toArcFlags(p.d[i-4])
			cx, cy, _, _ := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			outcodes &= cohenSutherlandOutcode(rect, Point{cx - rx, cy - ry}, 0.0)
			outcodes &= cohenSutherlandOutcode(rect, Point{cx + rx, cy + ry}, 0.0)
		}

		// either start is inside, or entire segment is left/right or above/below
		if crosses := outcodes == 0; cmd == CloseCmd {
			if !pendingMoveTo {
				if crosses && start != prev {
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
		} else if crosses {
			if pendingMoveTo {
				q.d = append(q.d, MoveToCmd, start.X, start.Y, MoveToCmd)
				pendingMoveTo = false
				first = start
			} else if start != prev {
				// previous segments were skipped
				q.d = append(q.d, LineToCmd, start.X, start.Y, LineToCmd)
			}
			q.d = append(q.d, p.d[i-cmdLen(cmd):i]...)
			outcodes = endOutcode
			prev = end
		} else if !closed && pendingMoveTo {
			// there is no line from previous point that may move inside
			outcodes = endOutcode
		}
		startOutcode = endOutcode
		start = end
	}
	return q
}

// LineClip converts the path to line segments between all coordinates and clips those lines against the given rectangle.
func (p *Path) LineClip(x0, y0, x1, y1 float64) *Path {
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

	var in bool
	var firstMoveTo, lastMoveTo int
	var start Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		i += cmdLen(cmd)

		end := Point{p.d[i-3], p.d[i-2]}
		if cmd == MoveToCmd {
			start = end
			continue
		}

		a, b, inEntirely, inPartially := cohenSutherlandLineClip(rect, start, end, 0.0)
		if inEntirely || inPartially {
			if !in {
				lastMoveTo = len(q.d)
				q.d = append(q.d, MoveToCmd, a.X, a.Y, MoveToCmd)
				in = true
			}
			q.d = append(q.d, LineToCmd, b.X, b.Y, LineToCmd)
		} else {
			in = false
		}
		if cmd == CloseCmd {
			if in && firstMoveTo < lastMoveTo {
				// connect the last segment with the first
				if end := len(q.d) - lastMoveTo; end < lastMoveTo-firstMoveTo-4 {
					tmp := make([]float64, end)
					copy(tmp, q.d[lastMoveTo:])
					copy(q.d[firstMoveTo+end:], q.d[firstMoveTo+4:lastMoveTo])
					copy(q.d[firstMoveTo:], tmp)
				} else {
					tmp := make([]float64, lastMoveTo-firstMoveTo-4)
					copy(tmp, q.d[firstMoveTo+4:lastMoveTo])
					copy(q.d[firstMoveTo:], q.d[lastMoveTo:])
					copy(q.d[firstMoveTo+end:], tmp)
				}
				q.d = q.d[:len(q.d)-4]
			}
			firstMoveTo = len(q.d)
			lastMoveTo = firstMoveTo
			in = false
		}
		start = end
	}
	return q
}
