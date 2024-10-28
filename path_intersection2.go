package canvas

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
)

func (p *Path) Settle(fillRule FillRule) *Path {
	// TODO
	return p
}

type pathOp int

const (
	opAND pathOp = iota
	opOR
	opNOT
	opXOR
)

// And returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) And(q *Path) *Path {
	return bentleyOttmann(p, q, opAND, NonZero)
}

// Or returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Or(q *Path) *Path {
	return bentleyOttmann(p, q, opOR, NonZero)
}

// Xor returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Xor(q *Path) *Path {
	return bentleyOttmann(p, q, opXOR, NonZero)
}

// Not returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Not(q *Path) *Path {
	return bentleyOttmann(p, q, opNOT, NonZero)
}

type SweepPoint struct {
	// initial data
	Point                  // position of this endpoint
	other      *SweepPoint // pointer to the other endpoint of the segment
	clipping   bool        // is clipping polygon (otherwise is subject polygon)
	segment    int         // segment index to distinguish self-overlapping segments
	left       bool        // point is left-end of segment
	increasing bool        // segment goes left to right (or bottom to top for vertical segments)
	vertical   bool        // segment is vertical

	// processing the queue
	node *SweepNode // used for fast accessing btree node in O(1) (instead of Find in O(log n))

	// computing sweep fields
	//equal         *SweepPoint // pointer to other segment that is equal (overlapping), can be nil
	windings      int         // windings of the same polygon (excluding this segment)
	otherWindings int         // windings of the other polygon
	inResult      bool        // in the final result polygon
	prevInResult  *SweepPoint // previous (downwards) segment that is in the final result polygon

	// building the polygon
	index     int // index into result array
	depth     int // contour depth (even = filling, odd = hole)
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
	path := "P"
	if s.clipping {
		path = "Q"
	}
	return fmt.Sprintf("%s(%v−%v)", path, s.Point, s.other.Point)
}

// SweepEvents is a heap priority queue of sweep events.
type SweepEvents []*SweepPoint

func (q SweepEvents) Less(i, j int) bool {
	return q[i].LessH(q[j])
}

func (q SweepEvents) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *SweepEvents) AddPathEndpoints(p *Path, seg int, clipping bool) int {
	//open := !pi.Closed()
	for i := 4; i < len(p.d); {
		n := cmdLen(p.d[i])
		start := Point{p.d[i-3], p.d[i-2]}
		end := Point{p.d[i+n-3], p.d[i+n-2]}

		if p.d[i] != LineToCmd && p.d[i] != CloseCmd {
			panic("non-flat paths not supported")
		} else if start.Equals(end) {
			// skip zero-length lineTo or close command
			continue
		}

		vertical := Equal(start.X, end.X)
		increasing := start.X < end.X
		if vertical {
			increasing = start.Y < end.Y
		}
		a := &SweepPoint{
			Point:      start,
			clipping:   clipping,
			segment:    seg,
			left:       increasing,
			increasing: increasing,
			vertical:   vertical,
		}
		b := &SweepPoint{
			Point:      end,
			clipping:   clipping,
			segment:    seg,
			left:       !increasing,
			increasing: increasing,
			vertical:   vertical,
		}
		a.other = b
		b.other = a
		*q = append(*q, a, b)

		i += n
		seg++
	}
	return seg
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

// TODO: use AB tree with A=2 and B=16 instead of AVL, according to LEDA (S. Naber. Comparison of search-tree data structures in LEDA. Personal communication.) this was faster.
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
	} else if !a.left {
		// right-endpoints
		if 0 < a.compareTangentsV(b) {
			return true // sort upwards, this ensures CCW orientation order of result
		}
	} else {
		// left-endpoints
		if a.compareTangentsV(b) < 0 {
			return true // sort upwards, this ensures CCW orientation order of result
		}
	}
	return false
}

func (a *SweepPoint) compareOverlapsV(b *SweepPoint) int {
	// compare segments vertically that overlap (ie. are the same)
	if a.clipping != b.clipping {
		if b.clipping {
			return -1
		} else {
			return 1
		}
	}

	// equal segment on same path, sort by segment index
	if a.segment < b.segment {
		return -1
	} else {
		return 1
	}
}

func (a *SweepPoint) compareTangentsV(b *SweepPoint) int {
	// compare segments vertically at a.X, b.X <= a.X, and a and b coincide at (a.X,a.Y)
	if a.vertical {
		// a is vertical
		if b.vertical {
			// a and b are vertical
			if Equal(a.Y, b.Y) {
				return a.compareOverlapsV(b)
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

	aLeft, aRight := a.Left(), a.Right()
	bLeft, bRight := b.Left(), b.Right()
	if aRight.X < bRight.X {
		t := (aRight.X - bLeft.X) / (bRight.X - bLeft.X)
		by := bLeft.Interpolate(bRight, t).Y // b's y at a's right
		if Equal(aRight.Y, by) {
			return a.compareOverlapsV(b)
		} else if aRight.Y < by {
			return -1
		} else {
			return 1
		}
	} else {
		t := (bRight.X - aLeft.X) / (aRight.X - aLeft.X)
		ay := aLeft.Interpolate(aRight, t).Y // a's y at b's right
		if Equal(ay, bRight.Y) {
			return a.compareOverlapsV(b)
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
	// a and b are always left-endpoints
	if !handled[SweepPointPair{a, b}] && !handled[SweepPointPair{b, a}] {

		// find all intersections between segment pair
		zs = intersectionLineLine(zs[:0], a.Start(), a.End(), b.Start(), b.End())
		if len(zs) == 0 {
			handled[SweepPointPair{a, b}] = true
			return
		}

		// sort intersections from left to right and add to queue
		// handle a
		aSign := 1
		if !a.increasing {
			aSign = -1
		}
		slices.SortFunc(zs, func(a, b Intersection) int {
			if a.T[0] < b.T[0] {
				return -aSign
			} else if b.T[0] < a.T[0] {
				return aSign
			}
			return 0
		})
		aLefts := []*SweepPoint{a}
		aPrevLeft, aLastRight := a, a.other
		for _, z := range zs {
			if z.T[0] == 0.0 || z.T[0] == 1.0 {
				// ignore tangent intersections at the endpoints
				continue
			}

			// split segment at intersection
			aRight, aLeft := *a.other, *a
			aRight.Point = z.Point
			aLeft.Point = z.Point

			// update references
			aPrevLeft.other, aRight.other = &aRight, aPrevLeft
			aPrevLeft = &aLeft

			// add to queue
			queue.Push(&aRight)
			queue.Push(&aLeft)
			aLefts = append(aLefts, &aLeft)
		}
		aPrevLeft.other, aLastRight.other = aLastRight, aPrevLeft

		// handle b
		bSign := 1
		if !b.increasing {
			bSign = -1
		}
		slices.SortFunc(zs, func(a, b Intersection) int {
			if a.T[1] < b.T[1] {
				return -bSign
			} else if b.T[1] < a.T[1] {
				return bSign
			}
			return 0
		})
		bLefts := []*SweepPoint{b}
		bPrevLeft, bLastRight := b, b.other
		for _, z := range zs {
			if z.T[1] == 0.0 || z.T[1] == 1.0 {
				// ignore tangent intersections at the endpoints
				continue
			}

			// split segment at intersection
			bRight, bLeft := *b.other, *b
			bRight.Point = z.Point
			bLeft.Point = z.Point

			// update references
			bPrevLeft.other, bRight.other = &bRight, bPrevLeft
			bPrevLeft = &bLeft

			// add to queue
			queue.Push(&bRight)
			queue.Push(&bLeft)
			bLefts = append(bLefts, &bLeft)
		}
		bPrevLeft.other, bLastRight.other = bLastRight, bPrevLeft

		for _, a := range aLefts {
			for _, b := range bLefts {
				handled[SweepPointPair{a, b}] = true
			}
		}
	}
}

func (cur *SweepNode) computeSweepFields(op pathOp, fillRule FillRule) {
	// cur is left-endpoint
	var overlapping *SweepPoint
	if !cur.clipping {
		if next := cur.Next(); next != nil && next.clipping && cur.Point.Equals(next.Point) && cur.other.Point.Equals(next.other.Point) {
			// equal segment, this happens when P starts to the left of Q, and thus the intersection
			// of P at the start of Q was inserted into the queue while handling the parallel
			// segment of Q, and is thus the only instance where we check for an equal segment in Q
			// using P, thus we need to look up/Next (not down/Prev).
			//overlapping = next.SweepPoint
			fmt.Println("OVERLAPS NEXT")
		}
	}

	// may have been copied when intersected
	cur.windings, cur.otherWindings = 0, 0
	cur.inResult, cur.prevInResult = false, nil

	fmt.Println("  cur", cur, cur.vertical)
	if prev := cur.Prev(); prev != nil {
		if cur.clipping && cur.Point.Equals(prev.Point) && cur.other.Point.Equals(prev.other.Point) {
			// equal segment
			overlapping = prev.SweepPoint
		}
		//for prev != nil && prev.vertical { // || cur.vertical && Equal(cur.X, prev.X)) {
		//	prev = prev.Prev()
		//}
		if prev != nil {
			fmt.Println("  prev", prev, prev.vertical, "windings", prev.windings, prev.otherWindings)
			if cur.clipping == prev.clipping {
				cur.windings = prev.windings
				cur.otherWindings = prev.otherWindings
				if prev.increasing {
					cur.windings++
				} else {
					cur.windings--
				}
			} else {
				cur.windings = prev.otherWindings
				cur.otherWindings = prev.windings
				if prev.increasing {
					cur.otherWindings++
				} else {
					cur.otherWindings--
				}
			}

			cur.prevInResult = prev.SweepPoint
			if !prev.inResult || prev.vertical {
				cur.prevInResult = prev.prevInResult
			}
		} else if cur.windings != 0 {
			fmt.Println("WINDINGS NOT ZERO")
		}
	}

	// prevent duplicate edges when overlapping
	if overlapping == nil {
		cur.inResult = cur.InResult(op, fillRule)
	} else {
		subj, clip := overlapping, cur.SweepPoint
		if !cur.clipping {
			subj, clip = cur.SweepPoint, overlapping
		}

		fmt.Println("subj", subj, subj.otherWindings, "clip", clip, clip.otherWindings)
		//if !cur.vertical {
		// note that clipping polygon is ordered as "higher", correct windings for both edges
		// there are four cases:
		// - clipping is outside and against the top of subject:
		//     clip.windings=even, subj.windings=even
		//     no-op
		// - clipping is outside and against the bottom of subject:
		//     clip.windings=odd, subj.windings=odd
		//     add clip's winding to subj.windings, remove subj's winding from clip.windings
		// - clipping is inside and against the top of subject:
		//     clip.windings=even, subj.windings=odd
		//     remove subj's winding from clip.windings
		// - clipping is inside and against the bottom of subject:
		//     clip.windings=odd, subj.windings=even
		//     add clip's winding to subj.windings
		addToSubj, subFromClip := clip.otherWindings%2 != 0, subj.otherWindings%2 != 0
		if addToSubj {
			// add clip's winding to subj
			if clip.increasing {
				subj.otherWindings++
			} else {
				subj.otherWindings--
			}
			subj.other.otherWindings = subj.otherWindings
		}
		if subFromClip {
			// remove subj's winding from clip
			if subj.increasing {
				clip.otherWindings--
			} else {
				clip.otherWindings++
			}
		}
		//	}

		fmt.Println("  overlap windings", subj.otherWindings, clip.otherWindings, "clipping", subj.clipping, clip.clipping)
		//
		switch op {
		case opAND, opOR:
			// include edges when overlap on interior of polygons
			subj.inResult = fillRule.Fills(subj.otherWindings)
		case opNOT:
			// include edges when overlap on exterior of polygons
			subj.inResult = !fillRule.Fills(subj.otherWindings)
		case opXOR:
			subj.inResult = false
		}
		subj.other.inResult = subj.inResult
		// clip.inResult is false to prevent duplicate edge
	}

	fmt.Println(" ", cur.otherWindings, cur.inResult)
	//cur.other.windings = cur.windings
	//cur.other.otherWindings = cur.otherWindings
	cur.other.inResult = cur.inResult
	//cur.other.prevInResult = cur.prevInResult
	return

	//first, firstForDepth, equalWithNext := true, true, false
	//cur := n
	//cur.windings = 0 // might have been set previously if split up
	//if nNext := n.Next(); nNext != nil && !cur.clipping && nNext.clipping && cur.Point.Equals(nNext.Point) && cur.other.Point.Equals(nNext.other.Point) {
	//	// equal segment, this happens when P starts to the left of Q, and thus the intersection
	//	// of P at the start of Q was inserted into the queue while handling the parallel
	//	// segment of Q, and is thus the only instance where we check for an equal segment in Q
	//	// using P, thus we need to look up/Next (not down/Prev).
	//	cur.equal = nNext.SweepPoint
	//	cur.equal.equal = cur.SweepPoint
	//	cur.other.equal = nNext.other
	//	cur.other.equal.equal = cur.other
	//	equalWithNext = true
	//	first = false
	//}

	//for {
	//	n = n.Prev()
	//	if n == nil {
	//		break
	//	} else if cur.clipping == n.clipping {
	//		continue
	//	} else if first && cur.clipping && cur.Point.Equals(n.Point) && cur.other.Point.Equals(n.other.Point) {
	//		// equal segment
	//		cur.equal = n.SweepPoint
	//		cur.equal.equal = cur.SweepPoint
	//		cur.other.equal = n.other
	//		cur.other.equal.equal = cur.other
	//	} else if n.vertical {
	//		continue
	//	} else if firstForDepth {
	//		cur.depth = n.depth + 1
	//		firstForDepth = false
	//	}

	//	if n.increasing {
	//		cur.windings++
	//	} else {
	//		cur.windings--
	//	}
	//	first = false
	//}

	//// note that clipping polygon is ordered as "higher", correct windings for both edges
	//// there are four cases:
	//// - clipping is outside and against the top of subject:
	////     clip.windings=even, subj.windings=even, selfWindings=even
	////     no-op
	//// - clipping is outside and against the bottom of subject:
	////     clip.windings=odd, subj.windings=odd, selfWindings=odd
	////     add clip's winding to subj.windings, remove subj's winding from clip.windings
	//// - clipping is inside and against the top of subject:
	////     clip.windings=even, subj.windings=odd, selfWindings=odd
	////     remove subj's winding from clip.windings
	//// - clipping is inside and against the bottom of subject:
	////     clip.windings=odd, subj.windings=even, selfWindings=even
	////     add clip's winding to subj.windings
	//if cur.equal != nil && !equalWithNext {
	//	clip, subj := cur.SweepPoint, cur.equal
	//	if !cur.clipping {
	//		clip, subj = subj, clip
	//	}

	//	//outside := clip.windings%2 == subj.windings%2
	//	addToSubj, subFromClip := clip.windings%2 != 0, subj.windings%2 != 0
	//	if addToSubj {
	//		//if outside && cur.selfWindings%2 != 0 || !outside && subj.windings%2 == 0 {
	//		// add clip's winding to subj
	//		if clip.increasing {
	//			subj.windings++
	//		} else {
	//			subj.windings--
	//		}
	//		subj.other.windings = subj.windings
	//	}
	//	//if outside && cur.selfWindings%2 != 0 || !outside && clip.windings%2 == 0 {
	//	if subFromClip {
	//		// remove subj's winding from clip
	//		if subj.increasing {
	//			clip.windings--
	//		} else {
	//			clip.windings++
	//		}
	//	}
	//}
	//cur.other.windings = cur.windings
}

func (s *SweepPoint) InResult(op pathOp, fillRule FillRule) bool {
	//if p.equal != nil {
	//	// handle touching (but not overlapping) polygons
	//	// remove so that OR and XOR will merge, and AND and NOT will ignore the clipping polygon
	//	// either both segments are from the same path, or both are not inside the other path
	//	if p.clipping == p.equal.clipping {
	//		// overlapping segment on same path
	//		return true
	//	}

	//	switch op {
	//	case opAND, opOR:
	//		return fillRule.Fills(p.otherWindings) // != fillRule.Fills(p.equal.windings)
	//	case opNOT:
	//		return fillRule.Fills(p.otherWindings) == fillRule.Fills(p.equal.otherWindings)
	//	}
	//	return false
	//}

	switch op {
	case opAND:
		// all edges inside the other
		return fillRule.Fills(s.otherWindings)
	case opOR:
		// all edges outside the other
		return !fillRule.Fills(s.otherWindings)
	case opNOT:
		// all edges outside the clipping and inside the subject
		return s.clipping == fillRule.Fills(s.otherWindings)
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

	// check for path bounding boxes to overlap
	R := &Path{}
	ps, qs := p.Split(), q.Split()
	pBounds := make([]Rect, len(ps))
	qBounds := make([]Rect, len(qs))
	for i := range ps {
		pBounds[i] = ps[i].FastBounds()
	}
	for i := range qs {
		qBounds[i] = qs[i].FastBounds()
	}
	pOverlaps := make([]bool, len(ps))
	qOverlaps := make([]bool, len(qs))
	for i := range ps {
		for j := range qs {
			if pBounds[i].Touches(qBounds[j]) {
				pOverlaps[i] = true
				qOverlaps[j] = true
			}
		}
		if !pOverlaps[i] && (op == opOR || op == opXOR || op == opNOT) {
			// path bounding boxes do not overlap, thus no intersections
			R = R.Append(p)
		}
	}
	for j := range qs {
		if !qOverlaps[j] && (op == opOR || op == opXOR) {
			// path bounding boxes do not overlap, thus no intersections
			R = R.Append(q)
		}
	}

	// TODO: handle open paths

	// construct the priority queue of sweep events
	pSeg, qSeg := 0, 0
	queue := &SweepEvents{}
	for i := range ps {
		if pOverlaps[i] {
			// implicitly close all subpaths on P
			// TODO: remove and support open paths only on P
			if !ps[i].Closed() {
				ps[i].Close()
			}
			pSeg = queue.AddPathEndpoints(ps[i], pSeg, false)
		}
	}
	for i := range qs {
		if qOverlaps[i] {
			// implicitly close all subpaths on Q
			if !qs[i].Closed() {
				qs[i].Close()
			}
			qSeg = queue.AddPathEndpoints(qs[i], qSeg, true)
		}
	}
	queue.Init() // sort from left to right

	// construct sweep line status structure
	var zs Intersections // reusable buffer
	var preResult []*SweepPoint
	status := NewSweepStatus()           // contains only left events
	handled := map[SweepPointPair]bool{} // prevent testing for intersections more than once
	for 0 < len(*queue) {
		// pop the next left-most endpoint from the queue
		events := queue.Pop()
		for _, event := range events {
			// TODO: skip or stop depending on operation if we're to the left/right of subject/clipping polygon
			fmt.Println("---", event)
			if event.left {
				// add segment to sweep status
				n := status.Insert(event)
				if prev := n.Prev(); prev != nil {
					addIntersections(queue, handled, zs, prev.SweepPoint, n.SweepPoint)
				}
				if next := n.Next(); next != nil {
					addIntersections(queue, handled, zs, n.SweepPoint, next.SweepPoint)
				}

				// compute fields after addIntersections as it may make segments equal
				n.computeSweepFields(op, fillRule)
			} else {
				// remove segment from sweep status
				n := event.other.node
				if n == nil {
					continue
				}
				prev := n.Prev()
				next := n.Next()
				if prev != nil && next != nil {
					addIntersections(queue, handled, zs, prev.SweepPoint, next.SweepPoint)
				}
				status.Remove(n) // TODO: this shouldn't touch SweepPoint inside the nodes
			}

			//if event.left && !event.clipping && event.equal != nil {
			//	// correct order for when P starts to the left of Q, and both are overlapping
			//	// over Q. In this case, P is split while handling Q and is handled AFTER Q.
			//	preResult = append(preResult[:len(preResult)-2], preResult[len(preResult)-1], event, preResult[len(preResult)-2])
			//} else {
			//preResult = append(preResult, event)
			//}

			preResult = append(preResult, event)
		}
	}

	// build result array
	var result [][]*SweepPoint // put segments at the same position together
	for _, event := range preResult {
		// store in result array, set index, and store same position at same index
		if event.inResult {
			// inResult may be set false afterwards due to overlapping edges, we check again
			// when building the polygon
			fmt.Println("  add", event, "windings", event.otherWindings, "other", event.other, event.other.inResult)
			if len(result) == 0 || !event.Point.Equals(result[len(result)-1][0].Point) {
				result = append(result, []*SweepPoint{event})
			} else {
				result[len(result)-1] = append(result[len(result)-1], event)
			}
			event.index = len(result) - 1
		} else {
			fmt.Println("  rem", event, "windings", event.otherWindings)
		}
	}

	// build resulting polygons
	for _, nodes := range result {
		for _, cur := range nodes {
			if !cur.inResult || cur.processed {
				continue
			}

			depth := 0
			if cur.prevInResult != nil {
				depth = cur.prevInResult.depth + 1
			}

			r := &Path{}
			r.MoveTo(cur.X, cur.Y)
			cur.processed, cur.other.processed = true, true
			cur.depth, cur.other.depth = depth, depth
			//if cur.equal != nil {
			//	cur.equal.processed, cur.equal.other.processed = true, true
			//	cur.equal.depth, cur.equal.other.depth = depth, depth
			//}
			for {
				// find segments starting from other endpoint, find the other segment amongst
				// them, the next segment should be the next going CCW
				i0 := 0
				nodes := result[cur.other.index]
				for i := range nodes {
					if nodes[i] == cur.other {
						i0 = i
						break
					}
				}
				// find the next segment in CW order, this will make smaller subpaths
				// instead one large path when multiple segments end at the same position
				// TODO: we should break two polygons that have one point in common, instead
				//       of a single outline for both
				var next *SweepPoint
				for i := (i0 + 1) % len(nodes); i != i0; i = (i + 1) % len(nodes) {
					if nodes[i].inResult && !nodes[i].processed {
						next = nodes[i]
						break
					}
				}
				if next == nil {
					break // contour is done
				}
				cur = next

				r.LineTo(cur.X, cur.Y)
				cur.processed, cur.other.processed = true, true
				cur.depth, cur.other.depth = depth, depth
				//	if cur.equal != nil {
				//		cur.equal.processed, cur.equal.other.processed = true, true
				//		cur.equal.depth, cur.equal.other.depth = depth, depth
				//	}
			}
			r.Close()
			fmt.Println("depth", depth, r)

			if depth%2 == 1 {
				// orient holes clockwise
				r = r.Reverse()
			}
			R = R.Append(r)
		}
	}
	return R
}
