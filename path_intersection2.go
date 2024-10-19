package canvas

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

const (
	opAND pathOp = iota
	opOR
	opNOT
	opXOR
)

// And returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) And2(q *Path) *Path {
	return bentleyOttmann(p, q, opAND, NonZero)
}

// Or returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Or2(q *Path) *Path {
	return bentleyOttmann(p, q, opOR, NonZero)
}

// Xor returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Xor2(q *Path) *Path {
	return bentleyOttmann(p, q, opXOR, NonZero)
}

// Not returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Not2(q *Path) *Path {
	return bentleyOttmann(p, q, opNOT, NonZero)
}

type SweepPoint struct {
	Point
	left       bool        // point is left-end of segment
	increasing bool        // segment goes left to right (or bottom to top for vertical segments)
	vertical   bool        // segment is vertical
	other      *SweepPoint // pointer to the other endpoint of the segment
	node       *SweepNode  // used for fast accessing node in O(1) (instead of Find in O(log n))

	clipping  bool
	windings  int // windings of the other polygon
	index     int // index into result array
	processed bool
}

func (s SweepPoint) Left() Point {
	if s.left {
		return s.Point
	}
	return s.other.Point
}

func (s SweepPoint) Right() Point {
	if s.left {
		return s.other.Point
	}
	return s.Point
}

func (s SweepPoint) Start() Point {
	if s.left == s.increasing {
		return s.Point
	}
	return s.other.Point
}

func (s SweepPoint) End() Point {
	if s.left == s.increasing {
		return s.other.Point
	}
	return s.Point
}

func (s SweepPoint) String() string {
	return fmt.Sprintf("%v−%v", s.Point, s.other.Point)
}

// SweepEvents is a heap priority queue of sweep events.
type SweepEvents []*SweepPoint

func (q SweepEvents) Less(i, j int) bool {
	return q[i].LessH(q[j])
}

func (q SweepEvents) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *SweepEvents) AddPathEndpoints(p *Path, clipping bool) Rect {
	seg := 0
	var bounds Rect
	for i := 4; i < len(p.d); {
		n := cmdLen(p.d[i])
		start := Point{p.d[i-3], p.d[i-2]}
		end := Point{p.d[i+n-3], p.d[i+n-2]}

		// update bounding box
		if i == 4 {
			bounds = Rect{start.X, start.Y, 0.0, 0.0}
		} else {
			bounds = bounds.AddPoint(start)
		}
		bounds = bounds.AddPoint(end)

		if p.d[i] == MoveToCmd {
			continue
		} else if p.d[i] == CloseCmd && start.Equals(end) {
			// skip zero-length close command
			continue
		} else if p.d[i] != LineToCmd && p.d[i] != CloseCmd {
			panic("non-flat paths not supported")
		}

		increasing := start.X < end.X
		if Equal(start.X, end.X) {
			increasing = start.Y < start.Y
		}
		vertical := Equal(start.X, end.X)
		a := &SweepPoint{
			Point:      start,
			left:       increasing,
			increasing: increasing,
			vertical:   vertical,
			clipping:   clipping,
		}
		b := &SweepPoint{
			Point:      end,
			left:       !increasing,
			increasing: increasing,
			vertical:   vertical,
			clipping:   clipping,
		}
		a.other = b
		b.other = a
		*q = append(*q, a, b)

		i += n
		seg++
	}
	return bounds
}

func (q SweepEvents) Init() {
	n := len(q)
	for i := n/2 - 1; 0 <= i; i-- {
		q.down(i, n)
	}
}

func (q *SweepEvents) Push(item *SweepPoint) {
	*q = append(*q, item)
	q.up(len(*q) - 1)
}

func (q *SweepEvents) Pop() []*SweepPoint {
	n := len(*q) - 1
	q.Swap(0, n)
	q.down(0, n)
	//for 0 < n && !q.Less(n, 0) {
	//	n--
	//	q.Swap(0, n)
	//	q.down(0, n)
	//}

	items := (*q)[n:]
	*q = (*q)[:n]
	return items
}

// from container/heap
func (q SweepEvents) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !q.Less(j, i) {
			break
		}
		q.Swap(i, j)
		j = i
	}
}

func (q SweepEvents) down(i0, n int) {
	i := i0
	for {
		j1 := 2*i + 1
		if n <= j1 || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && q.Less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !q.Less(j, i) {
			break
		}
		q.Swap(i, j)
		i = j
	}
}

func (q SweepEvents) Print(w io.Writer) {
	q2 := make(SweepEvents, len(q))
	copy(q2, q)
	q = q2

	n := len(q) - 1
	for 0 < n {
		q.Swap(0, n)
		q.down(0, n)
		n--
	}
	for k := len(q) - 1; 0 <= k; k-- {
		fmt.Fprintln(w, len(q)-1-k, q[k])
	}
	return
}

func (q SweepEvents) String() string {
	sb := strings.Builder{}
	q.Print(&sb)
	str := sb.String()
	if 0 < len(str) {
		str = str[:len(str)-1]
	}
	return str
}

type SweepNode struct {
	parent, left, right *SweepNode
	height              int

	*SweepPoint
}

func (n *SweepNode) Prev() *SweepNode {
	// go left
	if n.left != nil {
		n = n.left
		for n.right != nil {
			n = n.right // find the right-most of current subtree
		}
		return n
	}

	for n.parent != nil && n.parent.left == n {
		n = n.parent // find first parent for which we're right
	}
	return n.parent // can be nil
}

func (n *SweepNode) Next() *SweepNode {
	// go right
	if n.right != nil {
		n = n.right
		for n.left != nil {
			n = n.left // find the left-most of current subtree
		}
		return n
	}

	for n.parent != nil && n.parent.right == n {
		n = n.parent // find first parent for which we're left
	}
	return n.parent // can be nil
}

func (n *SweepNode) balance() int {
	r := 0
	if n.left != nil {
		r -= n.left.height
	}
	if n.right != nil {
		r += n.right.height
	}
	return r
}

func (n *SweepNode) updateHeight() {
	n.height = 0
	if n.left != nil {
		n.height = n.left.height
	}
	if n.right != nil && n.height < n.right.height {
		n.height = n.right.height
	}
	n.height++
}

func (n *SweepNode) swapChild(a, b *SweepNode) {
	if n.right == a {
		n.right = b
	} else {
		n.left = b
	}
	if b != nil {
		b.parent = n
	}
}

func (a *SweepNode) rotateLeft() *SweepNode {
	b := a.right
	if a.parent != nil {
		a.parent.swapChild(a, b)
	} else {
		b.parent = nil
	}
	a.parent = b
	if a.right = b.left; a.right != nil {
		a.right.parent = a
	}
	b.left = a
	return b
}

func (a *SweepNode) rotateRight() *SweepNode {
	b := a.left
	if a.parent != nil {
		a.parent.swapChild(a, b)
	} else {
		b.parent = nil
	}
	a.parent = b
	if a.left = b.right; a.left != nil {
		a.left.parent = a
	}
	b.right = a
	return b
}

func (n *SweepNode) Print(w io.Writer, indent int) {
	if n.right != nil {
		n.right.Print(w, indent+1)
	} else if n.left != nil {
		fmt.Fprintf(w, "%vnil\n", strings.Repeat("  ", indent+1))
	}
	fmt.Fprintf(w, "%v%v\n", strings.Repeat("  ", indent), n.SweepPoint)
	if n.left != nil {
		n.left.Print(w, indent+1)
	} else if n.right != nil {
		fmt.Fprintf(w, "%vnil\n", strings.Repeat("  ", indent+1))
	}
}

type SweepStatus struct {
	root *SweepNode
	pool *sync.Pool
}

func NewSweepStatus() *SweepStatus {
	return &SweepStatus{
		pool: &sync.Pool{New: func() any { return &SweepNode{} }},
	}
}

func (s *SweepStatus) newNode(item *SweepPoint) *SweepNode {
	n := s.pool.Get().(*SweepNode)
	n.parent = nil
	n.left = nil
	n.right = nil
	n.height = 1
	n.SweepPoint = item
	n.SweepPoint.node = n
	return n
}

func (s *SweepStatus) returnNode(n *SweepNode) {
	n.SweepPoint.node = nil
	n.SweepPoint = nil // help the GC
	s.pool.Put(n)
}

func (s *SweepStatus) find(item *SweepPoint) (*SweepNode, int) {
	n := s.root
	for n != nil {
		cmp := item.CompareV(n.SweepPoint)
		if cmp < 0 {
			if n.left == nil {
				return n, -1
			}
			n = n.left
		} else if 0 < cmp {
			if n.right == nil {
				return n, 1
			}
			n = n.right
		} else {
			break
		}
	}
	return n, 0
}

func (s *SweepStatus) rebalance(n *SweepNode) {
	for {
		oheight := n.height
		if balance := n.balance(); balance == 2 {
			// Tree is excessively right-heavy, rotate it to the left.
			if n.right != nil && n.right.balance() < 0 {
				// Right tree is left-heavy, which would cause the next rotation to result in
				// overall left-heaviness. Rotate the right tree to the right to counteract this.
				n.right = n.right.rotateRight()
				n.right.right.updateHeight()
			}
			n = n.rotateLeft()
			n.left.updateHeight()
		} else if balance == -2 {
			// Tree is excessively left-heavy, rotate it to the right
			if n.left != nil && n.left.balance() > 0 {
				// The left tree is right-heavy, which would cause the next rotation to result in
				// overall right-heaviness. Rotate the left tree to the left to compensate.
				n.left = n.left.rotateLeft()
				n.left.left.updateHeight()
			}
			n = n.rotateRight()
			n.right.updateHeight()
		} else if balance < -2 || 2 < balance {
			panic("Tree too far out of shape!")
		}

		n.updateHeight()
		if n.parent == nil {
			s.root = n
			return
		}
		if oheight == n.height {
			return
		}
		n = n.parent
	}
}

func (s *SweepStatus) String() string {
	if s.root == nil {
		return "nil"
	}

	sb := strings.Builder{}
	s.root.Print(&sb, 0)
	str := sb.String()
	if 0 < len(str) {
		str = str[:len(str)-1]
	}
	return str
}

func (s *SweepStatus) First() *SweepNode {
	if s.root == nil {
		return nil
	}
	n := s.root
	for n.left != nil {
		n = n.left
	}
	return n
}

func (s *SweepStatus) Last() *SweepNode {
	if s.root == nil {
		return nil
	}
	n := s.root
	for n.right != nil {
		n = n.right
	}
	return n
}

// Find returns the node equal to item. May return nil.
func (s *SweepStatus) Find(item *SweepPoint) *SweepNode {
	n, cmp := s.find(item)
	if cmp == 0 {
		return n
	}
	return nil
}

func (s *SweepStatus) Insert(item *SweepPoint) *SweepNode {
	if s.root == nil {
		s.root = s.newNode(item)
		return s.root
	} else {
		rebalance := false
		n, cmp := s.find(item)
		if cmp < 0 {
			// lower
			n.left = s.newNode(item)
			n.left.parent = n
			rebalance = n.right == nil
			n = n.left
		} else if 0 < cmp {
			// higher
			n.right = s.newNode(item)
			n.right.parent = n
			rebalance = n.left == nil
			n = n.right
		} else {
			// equal, replace
			fmt.Println("REPLACE!")
			n.SweepPoint.node = nil
			n.SweepPoint = item
			n.SweepPoint.node = n
			return n
		}

		if rebalance {
			n.height++
			if n.parent != nil {
				s.rebalance(n.parent)
			}
		}
		return n
	}
}

func (s *SweepStatus) Remove(n *SweepNode) {
	var o *SweepNode
	for {
		if n.height == 1 {
			o = n.parent
			if o != nil {
				o.swapChild(n, nil)
				s.rebalance(o)
			} else {
				s.root = nil
			}
			s.returnNode(n)
			return
		} else if n.right != nil {
			o = n.right
			for o.left != nil {
				o = o.left
			}
		} else if n.left != nil {
			o = n.left
			for o.right != nil {
				o = o.right
			}
		} else {
			panic("Impossible")
		}
		n.SweepPoint, o.SweepPoint = o.SweepPoint, n.SweepPoint
		n.SweepPoint.node, o.SweepPoint.node = n, o
		n = o
	}
}

func (a *SweepPoint) LessH(b *SweepPoint) bool {
	// used for sweep queue
	if !Equal(a.X, b.X) {
		return a.X < b.X // sort left to right
	} else if !Equal(a.Y, b.Y) {
		return a.Y < b.Y // then bottom to top
	} else if a.left != b.left {
		return b.left // handle right-endpoints before left-endpoints
	} else if a.compareTangentsV(b) < 0 {
		return true // sort upwards, this ensures CCW orientation of result
	}
	return false
}

func (a *SweepPoint) compareTangentsV(b *SweepPoint) int {
	// compare segments vertically at a.X, b.X <= a.X, and a and b coincide at (a.X,a.Y)
	aRight := a.Right()
	bRight := b.Right()
	if aRight.X < bRight.X {
		t := (aRight.X - b.X) / (bRight.X - b.X)
		by := b.Point.Interpolate(bRight, t).Y // b's y at a's right
		if Equal(aRight.Y, by) {
			return 0 // overlapping  TODO: is right?
		} else if aRight.Y < by {
			return -1
		} else {
			return 1
		}
	} else {
		t := (bRight.X - a.X) / (aRight.X - a.X)
		ay := a.Point.Interpolate(aRight, t).Y // a's y at b's right
		if Equal(ay, bRight.Y) {
			return 0 // overlapping  TODO: is right?
		} else if ay < bRight.Y {
			return -1
		} else {
			return 1
		}
	}
}

func (a *SweepPoint) compareV(b *SweepPoint) int {
	// compare segments vertically at a.X and b.X <= a.X
	bRight := b.Right()
	t := (a.X - b.X) / (bRight.X - b.X)
	by := b.Point.Interpolate(bRight, t).Y // b's y at a's left
	if Equal(a.Y, by) {
		return a.compareTangentsV(b)
	} else if a.Y < by {
		return -1
	} else {
		return 1
	}
}

func (a *SweepPoint) CompareV(b *SweepPoint) int {
	// used for sweep status, a is the point to be inserted / found
	if Equal(a.X, b.X) {
		// left-point at same X
		if Equal(a.Y, b.Y) {
			// left-point the same
			if a.vertical {
				// a is vertical
				if b.vertical {
					// a and b are vertical
					if Equal(a.Y, b.Y) {
						return 0
					} else if a.Y < b.Y {
						return -1
					} else {
						return 1
					}
				}
				return 1
			} else if b.vertical {
				// b is vertical
				return -1
			}
			return a.compareTangentsV(b)
		} else if a.Y < b.Y {
			return -1
		} else {
			return 1
		}
	} else if a.X < b.X {
		// a starts to the left of b
		return -b.compareV(a)
	} else {
		// a starts to the right of b
		return a.compareV(b)
	}
}

type SweepPointPair struct {
	a, b *SweepPoint
}

func addIntersections(queue *SweepEvents, handled map[SweepPointPair]bool, zs Intersections, a, b *SweepPoint) {
	if !handled[SweepPointPair{a, b}] && !handled[SweepPointPair{b, a}] {
		// TODO: support multiple intersections, be wary of order along B (for quad/cube/arc)
		zs = intersectionLineLine(zs[:0], a.Start(), a.End(), b.Start(), b.End())
		if len(zs) == 1 && !zs[0].Tangent {
			z := zs[0]

			// create new events
			a1Right_, b1Right_ := *a.other, *b.other
			a1Right, b1Right := &a1Right_, &b1Right_
			a1Left, b1Left := a, b

			a2Right, b2Right := a.other, b.other
			a2Left := &SweepPoint{
				Point:      z.Point,
				left:       true,
				increasing: a.increasing,
				vertical:   a.vertical,
				clipping:   a.clipping,
			}
			b2Left := &SweepPoint{
				Point:      z.Point,
				left:       true,
				increasing: b.increasing,
				vertical:   b.vertical,
				clipping:   b.clipping,
			}
			a1Right.Point, b1Right.Point = z.Point, z.Point

			// update references
			a1Left.other, a1Right.other = a1Right, a1Left
			a2Left.other, a2Right.other = a2Right, a2Left
			b1Left.other, b1Right.other = b1Right, b1Left
			b2Left.other, b2Right.other = b2Right, b2Left

			// push new items
			queue.Push(a1Right)
			queue.Push(b1Right)
			queue.Push(a2Left)
			queue.Push(b2Left)
		}
		handled[SweepPointPair{a, b}] = true
	}
}

func (n *SweepNode) computeSweepFields() {
	cur := n
	for {
		n = n.Prev()
		if n == nil {
			break
		} else if n.vertical {
			continue
		}

		if cur.clipping != n.clipping {
			if n.increasing {
				cur.windings++
			} else {
				cur.windings--
			}
		}
	}
	cur.other.windings = cur.windings
}

func (p *SweepPoint) InResult(op pathOp, fillRule FillRule) bool {
	switch op {
	case opAND:
		// all edges inside the other
		return fillRule.Fills(p.windings)
	case opOR:
		// all edges outside the other
		return !fillRule.Fills(p.windings)
	case opNOT:
		// all edges outside the clipping and inside the subject
		return p.clipping == fillRule.Fills(p.windings)
	case opXOR:
		// all edges
		return true
	}
	return false
}

func bentleyOttmann(p, q *Path, op pathOp, fillRule FillRule) *Path {
	// Implementation of the Bentley-Ottmann algorithm by reducing the complexity of finding
	// intersections to O((n + k) log n), with n the number of segments and k the number of
	// intersections. All special cases are handled by use of:
	// - M. de Berg, et al. "Computational Geometry", Chapter 2, DOI: 10.1007/978-3-540-77974-2
	// - F. Martínez, et al. "A simple algorithm for Boolean operations on polygons", Advances in
	//   Engineering Software 64, p. 11-19, 2013, DOI: 10.1016/j.advengsoft.2013.04.004

	// return in case of one path is empty
	if q.Empty() {
		if op == opAND {
			return &Path{}
		}
		return p
	}
	if p.Empty() {
		if op == opOR || op == opXOR {
			return q
		}
		return &Path{}
	}

	// ensure that X-monotone property holds for Béziers and arcs by breaking them up at their
	// extremes along X (ie. their inflection points along X)
	// TODO: handle Béziers and arc segments
	//p = p.XMonotone()
	//q = q.XMonotone()
	p = p.Flatten(Tolerance)
	q = q.Flatten(Tolerance)

	// implicitly close all subpaths on Q
	qs := q.Split()
	q = &Path{} // collect all closed paths
	for i := range qs {
		if !qs[i].Closed() {
			qs[i].Close()
		}
		q = q.Append(qs[i])
	}

	// TODO: handle open paths

	// construct the priority queue of sweep events
	queue := &SweepEvents{}
	pBounds := queue.AddPathEndpoints(p, false)
	qBounds := queue.AddPathEndpoints(q, true)
	if !pBounds.Overlaps(qBounds) {
		// path bounding boxes do not overlap, thus no intersections
		if op == opOR || op == opXOR {
			return p.Append(q)
		} else if op == opNOT {
			return p
		}
		return &Path{}
	}
	queue.Init() // sort from left to right

	// construct sweep line status structure
	var zs Intersections // reusable buffer
	var result []*SweepPoint
	status := NewSweepStatus()
	handled := map[SweepPointPair]bool{} // prevent testing for intersections more than once
	for 0 < len(*queue) {
		// pop the next left-most endpoint from the queue
		events := queue.Pop()
		for _, event := range events {
			// TODO: skip or stop depending on operation if we're to the left/right of subject/clipping polygon
			if event.left {
				// add segment to sweep status
				n := status.Insert(event)
				n.computeSweepFields()
				if prev := n.Prev(); prev != nil {
					addIntersections(queue, handled, zs, prev.SweepPoint, n.SweepPoint)
				}
				if next := n.Next(); next != nil {
					addIntersections(queue, handled, zs, n.SweepPoint, next.SweepPoint)
				}
			} else {
				// remove segment from sweep status
				n := event.other.node
				if n == nil {
					fmt.Println("COULD NOT FIND RIGHT END!")
					continue
				}
				prev := n.Prev()
				next := n.Next()
				if prev != nil && next != nil {
					addIntersections(queue, handled, zs, prev.SweepPoint, next.SweepPoint)
				}
				status.Remove(n) // TODO: this shouldn't touch SweepPoint inside the nodes
			}

			if event.InResult(op, fillRule) {
				result = append(result, event)
			}
		}
	}

	for idx := range result {
		result[idx].index = idx
	}

	r := &Path{}
	if 0 < len(result) {
		for _, cur := range result {
			if cur.processed {
				continue
			}

			r.MoveTo(cur.X, cur.Y)
			cur.processed = true
			cur.other.processed = true
			for {
				cur = cur.other
				if cur.index%2 == 0 {
					cur = result[cur.index+1]
				} else {
					cur = result[cur.index-1]
				}
				if cur.processed {
					break
				}
				r.LineTo(cur.X, cur.Y)
				cur.processed = true
				cur.other.processed = true
			}
			r.Close()
		}
	}
	return r
}
