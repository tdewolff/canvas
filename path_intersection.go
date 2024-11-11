package canvas

import (
	"fmt"
	"io"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
)

// RayIntersections returns the intersections of a path with a ray starting at (x,y) to (∞,y).
// An intersection is tangent only when it is at (x,y), i.e. the start of the ray. Intersections
// are sorted along the ray. This function runs in O(n) with n the number of path segments.
func (p *Path) RayIntersections(x, y float64) []Intersection {
	var start, end Point
	var zs []Intersection
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
		case LineToCmd, CloseCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			ymin := math.Min(start.Y, end.Y)
			ymax := math.Max(start.Y, end.Y)
			xmax := math.Max(start.X, end.X)
			if Interval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = intersectionLineLine(zs, Point{x, y}, Point{xmax + 1.0, y}, start, end)
			}
		case QuadToCmd:
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			ymin := math.Min(math.Min(start.Y, end.Y), cp.Y)
			ymax := math.Max(math.Max(start.Y, end.Y), cp.Y)
			xmax := math.Max(math.Max(start.X, end.X), cp.X)
			if Interval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = intersectionLineQuad(zs, Point{x, y}, Point{xmax + 1.0, y}, start, cp, end)
			}
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			ymin := math.Min(math.Min(start.Y, end.Y), math.Min(cp1.Y, cp2.Y))
			ymax := math.Max(math.Max(start.Y, end.Y), math.Max(cp1.Y, cp2.Y))
			xmax := math.Max(math.Max(start.X, end.X), math.Max(cp1.X, cp2.X))
			if Interval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = intersectionLineCube(zs, Point{x, y}, Point{xmax + 1.0, y}, start, cp1, cp2, end)
			}
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			if Interval(y, cy-math.Max(rx, ry), cy+math.Max(rx, ry)) && x <= cx+math.Max(rx, ry)+Epsilon {
				zs = intersectionLineEllipse(zs, Point{x, y}, Point{cx + rx + 1.0, y}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			}
		}
		i += cmdLen(cmd)
		start = end
	}
	for i := range zs {
		if zs[i].T[0] != 0.0 {
			zs[i].T[0] = math.NaN()
		}
	}
	sort.SliceStable(zs, func(i, j int) bool {
		if Equal(zs[i].X, zs[j].X) {
			return false
		}
		return zs[i].X < zs[j].X
	})
	return zs
}

type pathOp int

const (
	opSettle pathOp = iota
	opAND
	opOR
	opNOT
	opXOR
)

var boPointPool *sync.Pool
var boNodePool *sync.Pool
var boInitPoolsOnce = sync.OnceFunc(func() {
	boPointPool = &sync.Pool{New: func() any { return &SweepPoint{} }}
	boNodePool = &sync.Pool{New: func() any { return &SweepNode{} }}
})

// Settle returns the "settled" path. It removes all self-intersections, orients all filling paths
// CCW and all holes CW, and tries to split into subpaths if possible. Note that path p is
// flattened unless q is already flat. Path q is implicitly closed. It runs in O((n + k) log n),
// with n the sum of the number of segments, and k the number of intersections.
func (p *Path) Settle(fillRule FillRule) *Path {
	return bentleyOttmann(p.Split(), nil, opSettle, fillRule)
}

func (ps Paths) Settle(fillRule FillRule) *Path {
	return bentleyOttmann(ps, nil, opSettle, fillRule)
}

// And returns the boolean path operation of path p AND q, i.e. the intersection of both. It
// removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p *Path) And(q *Path) *Path {
	return bentleyOttmann(p.Split(), q.Split(), opAND, NonZero)
}

func (ps Paths) And(qs Paths) *Path {
	return bentleyOttmann(ps, qs, opAND, NonZero)
}

// Or returns the boolean path operation of path p OR q, i.e. the union of both. It
// removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p *Path) Or(q *Path) *Path {
	return bentleyOttmann(p.Split(), q.Split(), opOR, NonZero)
}

// Xor returns the boolean path operation of path p XOR q, i.e. the symmetric difference of both.
// It removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p *Path) Xor(q *Path) *Path {
	return bentleyOttmann(p.Split(), q.Split(), opXOR, NonZero)
}

// Not returns the boolean path operation of path p NOT q, i.e. the difference of both.
// It removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p *Path) Not(q *Path) *Path {
	return bentleyOttmann(p.Split(), q.Split(), opNOT, NonZero)
}

type SweepPoint struct {
	// initial data
	Point                    // position of this endpoint
	other        *SweepPoint // pointer to the other endpoint of the segment
	clipping     bool        // is clipping polygon (otherwise is subject polygon)
	segment      int         // segment index to distinguish self-overlapping segments
	left         bool        // point is left-end of segment
	selfWindings int         // positive if segment goes left-right (or bottom-top when vertical)
	vertical     bool        // segment is vertical

	// processing the queue
	node              *SweepNode // used for fast accessing btree node in O(1) (instead of Find in O(log n))
	otherSelfWindings int        // used when merging overlapping segments

	// computing sweep fields
	windings      int         // windings of the same polygon (excluding this segment)
	otherWindings int         // windings of the other polygon
	inResult      bool        // in the final result polygon
	prevInResult  *SweepPoint // previous (downwards) segment that is in the final result polygon

	// building the polygon
	index          int // index into result array
	processed      bool
	resultWindings int // windings of the resulting polygon
}

func (s SweepPoint) Increasing() bool {
	return 0 < s.selfWindings
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
	if s.left == s.Increasing() {
		return s.Point
	}
	return s.other.Point
}

func (s SweepPoint) End() Point {
	if s.left == s.Increasing() {
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
	// TODO: change this if we allow non-flat paths
	// allocate all memory at once to prevent multiple allocations/memmoves below
	n := len(p.d) / 4
	if cap(*q) < len(*q)+n {
		q2 := make(SweepEvents, len(*q), len(*q)+n)
		copy(q2, *q)
		*q = q2
	}
	for i := 4; i < len(p.d); {
		if p.d[i] != LineToCmd && p.d[i] != CloseCmd {
			panic("non-flat paths not supported")
		}

		n := cmdLen(p.d[i])
		start := Point{p.d[i-3], p.d[i-2]}
		end := Point{p.d[i+n-3], p.d[i+n-2]}
		i += n
		seg++

		if start.Equals(end) {
			// skip zero-length lineTo or close command
			continue
		}

		vertical := Equal(start.X, end.X)
		increasing := start.X < end.X
		if vertical {
			increasing = start.Y < end.Y
		}
		selfWindings := 1
		if !increasing {
			selfWindings = -1
		}
		a := boPointPool.Get().(*SweepPoint)
		b := boPointPool.Get().(*SweepPoint)
		*a = SweepPoint{
			Point:        start,
			clipping:     clipping,
			segment:      seg,
			left:         increasing,
			selfWindings: selfWindings,
			vertical:     vertical,
		}
		*b = SweepPoint{
			Point:        end,
			clipping:     clipping,
			segment:      seg,
			left:         !increasing,
			selfWindings: selfWindings,
			vertical:     vertical,
		}
		a.other = b
		b.other = a
		*q = append(*q, a, b)
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

func (q *SweepEvents) Top() *SweepPoint {
	return (*q)[0]
}

func (q *SweepEvents) Pop() *SweepPoint {
	n := len(*q) - 1
	q.Swap(0, n)
	q.down(0, n)

	items := (*q)[n]
	*q = (*q)[:n]
	return items
}

func (q *SweepEvents) Remove(item *SweepPoint) {
	// TODO: make O(log n)
	index := -1
	for i := range *q {
		if (*q)[i] == item {
			index = i
			break
		}
	}

	n := len(*q) - 1
	if index == -1 {
		panic("Item not in queue")
	} else if index < n {
		q.Swap(index, n)
		q.down(index, n)
	}
	*q = (*q)[:n]
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
}

func (s *SweepStatus) newNode(item *SweepPoint) *SweepNode {
	n := boNodePool.Get().(*SweepNode)
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
	boNodePool.Put(n)
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

func (s *SweepStatus) FindPrevNext(item *SweepPoint) (*SweepNode, *SweepNode) {
	if s.root == nil {
		return nil, nil
	}

	n, cmp := s.find(item)
	if cmp < 0 {
		return n.Prev(), n
	} else if 0 < cmp {
		return n, n.Next()
	} else {
		return n.Prev(), n.Next()
	}
}

func (s *SweepStatus) Insert(item *SweepPoint) *SweepNode {
	if s.root == nil {
		s.root = s.newNode(item)
		return s.root
	}

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

func (s *SweepStatus) InsertAfter(n *SweepNode, item *SweepPoint) *SweepNode {
	rebalance := false
	if n == nil {
		if s.root == nil {
			s.root = s.newNode(item)
			return s.root
		}

		// insert as left-most node in tree
		n = s.root
		for n.left != nil {
			n = n.left
		}
		n.left = s.newNode(item)
		n.left.parent = n
		rebalance = n.right == nil
		n = n.left
	} else if n.right == nil {
		// insert directly to the right of n
		n.right = s.newNode(item)
		n.right.parent = n
		rebalance = n.left == nil
		n = n.right
	} else {
		// insert next to n at a deeper level
		n = n.right
		for n.left != nil {
			n = n.left
		}
		n.left = s.newNode(item)
		n.left.parent = n
		rebalance = n.right == nil
		n = n.left
	}

	if rebalance {
		n.height++
		if n.parent != nil {
			s.rebalance(n.parent)
		}
	}
	return n
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

func (s *SweepStatus) Clear() {
	n := s.First()
	for n != nil {
		cur := n
		n = n.Next()
		boNodePool.Put(cur)
	}
	s.root = nil
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
		return true // sort upwards, this ensures CCW orientation order of result
	}
	return false
}

func (a *SweepPoint) compareOverlapsV(b *SweepPoint) int {
	// compare segments vertically that overlap (ie. are the same)
	if a.clipping != b.clipping {
		// for equal segments, clipping path is virtually to the top-right of subject path
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
	// note that a.left==b.left, we may be comparing right-endpoints
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

	sign := 1
	if !a.left {
		sign = -1
	}
	if a.left && a.other.X < b.other.X || !a.left && b.other.X < a.other.X {
		t := (a.other.X - b.X) / (b.other.X - b.X)
		by := b.Interpolate(b.other.Point, t).Y // b's y at a's other
		if Equal(a.other.Y, by) {
			return sign * a.compareOverlapsV(b)
		} else if a.other.Y < by {
			return sign * -1
		} else {
			return sign * 1
		}
	} else {
		t := (b.other.X - a.X) / (a.other.X - a.X)
		ay := a.Interpolate(a.other.Point, t).Y // a's y at b's other
		if Equal(ay, b.other.Y) {
			return sign * a.compareOverlapsV(b)
		} else if ay < b.other.Y {
			return sign * -1
		} else {
			return sign * 1
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

type SweepPointPair [2]*SweepPoint

func compareIntersections(a, b Intersection) int {
	if Equal(a.X, b.X) {
		if Equal(a.Y, b.Y) {
			return 0
		} else if a.Y < b.Y {
			return -1
		} else {
			return 1
		}
	} else if a.X < b.X {
		return -1
	} else {
		return 1
	}
}

func addIntersections(queue *SweepEvents, handled map[SweepPointPair]struct{}, zs Intersections, a, b *SweepPoint) bool {
	// a and b are always left-endpoints and a is below b
	if _, ok := handled[SweepPointPair{a, b}]; ok {
		return false
	} else if _, ok := handled[SweepPointPair{b, a}]; ok {
		return false
	}

	// find all intersections between segment pair
	// this returns either no intersections, or one or more secant/tangent intersections,
	// or exactly two "same" intersections which occurs when the segments overlap.
	zs = intersectionLineLine(zs[:0], a.Start(), a.End(), b.Start(), b.End())

	// clean up intersections outside one of the segments, this may happen for nearly parallel
	// lines for example
	for i := 0; i < len(zs); i++ {
		zs[i].Point = zs[i].Point.Gridsnap(2.0 * Epsilon) // prevent numerical issues

		if z := zs[i]; !a.vertical && !Interval(z.X, a.X, a.other.X) || a.vertical && !Interval(z.Y, a.Y, a.other.Y) || !b.vertical && !Interval(z.X, b.X, b.other.X) || b.vertical && !Interval(z.Y, b.Y, b.other.Y) { //z.X < a.X || z.X < b.X || a.other.X < z.X || b.other.X < z.X {
			fmt.Println("WARNING: removing intersection", zs[i], "between", a, b)
			zs = append(zs[:i], zs[i+1:]...)
			i--
		}
	}

	// no (valid) intersections
	if len(zs) == 0 {
		handled[SweepPointPair{a, b}] = struct{}{}
		return false
	}

	// sort intersections from left to right
	slices.SortFunc(zs, compareIntersections)

	// handle a
	aLefts := []*SweepPoint{a}
	aPrevLeft, aLastRight := a, a.other
	for i, z := range zs {
		if z.T[0] == 0.0 || z.T[0] == 1.0 {
			// ignore tangent intersections at the endpoints
			continue
		} else if aPrevLeft.Point.Equals(z.Point) || i == len(zs)-1 && z.Point.Equals(aLastRight.Point) {
			// ignore tangent intersections at the endpoints
			continue
		}

		// split segment at intersection
		aRight, aLeft := *a.other, *a
		aRight.Point = z.Point
		aLeft.Point = z.Point

		// update references
		aPrevLeft.other, aRight.other = &aRight, aPrevLeft

		// add to queue
		queue.Push(&aRight)
		aLefts = append(aLefts, &aLeft)
		aPrevLeft = &aLeft
	}
	aPrevLeft.other, aLastRight.other = aLastRight, aPrevLeft
	for _, aLeft := range aLefts[1:] {
		// add to queue
		queue.Push(aLeft)
	}

	// handle b
	bLefts := []*SweepPoint{b}
	bPrevLeft, bLastRight := b, b.other
	for i, z := range zs {
		if z.T[1] == 0.0 || z.T[1] == 1.0 {
			// ignore tangent intersections at the endpoints
			continue
		} else if bPrevLeft.Point.Equals(z.Point) || i == len(zs)-1 && z.Point.Equals(bLastRight.Point) {
			// ignore tangent intersections at the endpoints
			continue
		}

		// split segment at intersection
		bRight, bLeft := *b.other, *b
		bRight.Point = z.Point
		bLeft.Point = z.Point

		// update references
		bPrevLeft.other, bRight.other = &bRight, bPrevLeft

		// add to queue
		queue.Push(&bRight)
		bLefts = append(bLefts, &bLeft)
		bPrevLeft = &bLeft
	}
	bPrevLeft.other, bLastRight.other = bLastRight, bPrevLeft
	for _, bLeft := range bLefts[1:] {
		// add to queue
		queue.Push(bLeft)
	}

	if zs[0].Same {
		// Handle overlapping paths. Since we just split both segments above, we first find the
		// segments that overlap. We then transfer all selfWindings and otherSelfWindings to the
		// segment above and remove the segment below from the result.
		overlapBelow, overlapAbove := aLefts[0], bLefts[0]
		if zs[0].T[0] != 0.0 && zs[0].T[0] != 1.0 {
			// b starts to the right of a
			overlapBelow = aLefts[1]
		} else if zs[0].T[1] != 0.0 && zs[0].T[1] != 1.0 {
			// b starts to the left of a
			overlapAbove = bLefts[1]
		}
		if a.clipping == b.clipping {
			overlapAbove.selfWindings += overlapBelow.selfWindings
			overlapAbove.otherSelfWindings += overlapBelow.otherSelfWindings
		} else {
			overlapAbove.selfWindings += overlapBelow.otherSelfWindings
			overlapAbove.otherSelfWindings += overlapBelow.selfWindings
		}
		overlapBelow.selfWindings, overlapBelow.otherSelfWindings = 0, 0
		overlapBelow.inResult, overlapBelow.other.inResult = false, false
	}

	for _, a := range aLefts {
		for _, b := range bLefts {
			handled[SweepPointPair{a, b}] = struct{}{}
		}
	}
	return 0 < len(aLefts) || 0 < len(bLefts)
}

func (a *SweepPoint) Overlaps(b *SweepPoint) bool {
	// a is "CompareV==-1" to b (ie. below b) and crosses the vertical line at b.X
	if a.vertical && b.vertical {
		return true
	}
	return a.Point.Equals(b.Point) && a.other.Point.Equals(b.other.Point)
}

func (cur *SweepNode) computeSweepFields(op pathOp, fillRule FillRule) {
	// may have been copied when intersected
	cur.windings, cur.otherWindings = 0, 0
	cur.inResult, cur.prevInResult = false, nil

	// cur is left-endpoint
	if prev := cur.Prev(); prev != nil {
		// skip vertical segments
		if !cur.vertical {
			for prev.vertical {
				prev = prev.Prev()
				if prev == nil {
					break
				}
			}
		}
		if prev != nil {
			if cur.clipping == prev.clipping {
				cur.windings = prev.windings + prev.selfWindings
				cur.otherWindings = prev.otherWindings + prev.otherSelfWindings
			} else {
				cur.windings = prev.otherWindings + prev.otherSelfWindings
				cur.otherWindings = prev.windings + prev.selfWindings
			}

			cur.prevInResult = prev.SweepPoint
			if !prev.inResult || prev.vertical {
				cur.prevInResult = prev.prevInResult
			}
		}
	}
	cur.inResult = cur.InResult(op, fillRule)
	cur.other.inResult = cur.inResult
}

func (s *SweepPoint) InResult(op pathOp, fillRule FillRule) bool {
	lowerWindings, lowerOtherWindings := s.windings, s.otherWindings
	upperWindings, upperOtherWindings := s.windings+s.selfWindings, s.otherWindings+s.otherSelfWindings

	// lower/upper windings refers to subject path, otherWindings to clipping path
	var belowFills, aboveFills bool
	switch op {
	case opSettle:
		belowFills = fillRule.Fills(lowerWindings)
		aboveFills = fillRule.Fills(upperWindings)
	case opAND:
		belowFills = fillRule.Fills(lowerWindings) && fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) && fillRule.Fills(upperOtherWindings)
	case opOR:
		belowFills = fillRule.Fills(lowerWindings) || fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) || fillRule.Fills(upperOtherWindings)
	case opNOT:
		belowFills = fillRule.Fills(lowerWindings) != s.clipping && fillRule.Fills(lowerOtherWindings) == s.clipping
		aboveFills = fillRule.Fills(upperWindings) != s.clipping && fillRule.Fills(upperOtherWindings) == s.clipping
	case opXOR:
		belowFills = fillRule.Fills(lowerWindings) != fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) != fillRule.Fills(upperOtherWindings)
	}

	// only keep edge if there is a change in filling between both sides
	return belowFills != aboveFills
}

func bentleyOttmann(ps, qs Paths, op pathOp, fillRule FillRule) *Path {
	// Implementation of the Bentley-Ottmann algorithm by reducing the complexity of finding
	// intersections to O((n + k) log n), with n the number of segments and k the number of
	// intersections. All special cases are handled by use of:
	// - M. de Berg, et al. "Computational Geometry", Chapter 2, DOI: 10.1007/978-3-540-77974-2
	// - F. Martínez, et al. "A simple algorithm for Boolean operations on polygons", Advances in
	//   Engineering Software 64, p. 11-19, 2013, DOI: 10.1016/j.advengsoft.2013.04.004
	// - https://github.com/verven/contourklip

	boInitPoolsOnce() // use pools for SweepPoint and SweepNode to amortize repeated calls to BO

	// return in case of one path is empty
	if op == opSettle {
		qs = nil
	} else if qs.Empty() {
		if op == opAND {
			return &Path{}
		}
		return ps.Settle(fillRule)
	}
	if ps.Empty() {
		if qs != nil && (op == opOR || op == opXOR) {
			return qs.Settle(fillRule)
		}
		return &Path{}
	}

	// ensure that X-monotone property holds for Béziers and arcs by breaking them up at their
	// extremes along X (ie. their inflection points along X)
	// TODO: handle Béziers and arc segments
	//p = p.XMonotone()
	//q = q.XMonotone()
	for i, iMax := 0, len(ps); i < iMax; i++ {
		split := ps[i].Split()
		if 1 < len(split) {
			ps[i] = split[0]
			ps = append(ps, split[1:]...)
		}
	}
	for i := range ps {
		ps[i] = ps[i].Flatten(Tolerance)
		ps[i] = ps[i].Gridsnap(2.0 * Epsilon) // prevent numerical issues
	}
	if qs != nil {
		for i, iMax := 0, len(qs); i < iMax; i++ {
			split := qs[i].Split()
			if 1 < len(split) {
				qs[i] = split[0]
				qs = append(qs, split[1:]...)
			}
		}
		for i := range qs {
			qs[i] = qs[i].Flatten(Tolerance)
			qs[i] = qs[i].Gridsnap(2.0 * Epsilon) // prevent numerical issues
		}
	}

	// check for path bounding boxes to overlap
	R := &Path{}
	var pOverlaps, qOverlaps []bool
	if qs != nil {
		pBounds := make([]Rect, len(ps))
		qBounds := make([]Rect, len(qs))
		for i := range ps {
			pBounds[i] = ps[i].FastBounds()
		}
		for i := range qs {
			qBounds[i] = qs[i].FastBounds()
		}
		pOverlaps = make([]bool, len(ps))
		qOverlaps = make([]bool, len(qs))
		for i := range ps {
			for j := range qs {
				if pBounds[i].Touches(qBounds[j]) {
					pOverlaps[i] = true
					qOverlaps[j] = true
				}
			}
			if !pOverlaps[i] && (op == opOR || op == opXOR || op == opNOT) {
				// path bounding boxes do not overlap, thus no intersections
				R = R.Append(ps[i].Settle(fillRule))
			}
		}
		for j := range qs {
			if !qOverlaps[j] && (op == opOR || op == opXOR) {
				// path bounding boxes do not overlap, thus no intersections
				R = R.Append(qs[j].Settle(fillRule))
			}
		}
	}

	// TODO: handle open paths

	// construct the priority queue of sweep events
	pSeg, qSeg := 0, 0
	queue := &SweepEvents{}
	for i := range ps {
		if qs == nil || pOverlaps[i] {
			// implicitly close all subpaths on P
			// TODO: remove and support open paths only on P
			if !ps[i].Closed() {
				ps[i].Close()
			}
			pSeg = queue.AddPathEndpoints(ps[i], pSeg, false)
		}
	}
	if qs != nil {
		for i := range qs {
			if qOverlaps[i] {
				// implicitly close all subpaths on Q
				if !qs[i].Closed() {
					qs[i].Close()
				}
				qSeg = queue.AddPathEndpoints(qs[i], qSeg, true)
			}
		}
	}
	queue.Init() // sort from left to right

	// construct sweep line status structure
	zs_ := [2]Intersection{}
	zs := zs_[:]                                                // reusable buffer
	var preResult []*SweepPoint                                 // TODO: remove in favor of keeping points in queue, implement queue.Clear() to put back all points in pool
	status := &SweepStatus{}                                    // contains only left events
	handled := make(map[SweepPointPair]struct{}, len(*queue)*2) // prevent testing for intersections more than once, allocation length is an approximation
	for 0 < len(*queue) {
		// TODO: skip or stop depending on operation if we're to the left/right of subject/clipping polygon

		// We slightly divert from the original Bentley-Ottmann and paper implementation. First
		// we find the top element in queue but do not pop it off yet. If it is a right-event, pop
		// from queue and proceed as usual, but if it's a left-event we first check (and add) all
		// surrounding intersections to the queue. This may change the order from which we should
		// pop off the queue, since intersections may create right-events, or new left-events that
		// are lower (by CompareV). If no intersections are found, pop off the queue and proceed
		// as usual.

		// get the next left-most endpoint from the queue
		event := queue.Top()
		if event.left {
			// add intersections to queue
			intersects := false
			prev, next := status.FindPrevNext(event)
			if prev != nil {
				intersects = addIntersections(queue, handled, zs, prev.SweepPoint, event)
			}
			if next != nil {
				intersects = intersects || addIntersections(queue, handled, zs, event, next.SweepPoint)
			}
			if intersects {
				// check if the queue order was changed, note that if the order wasn't changed, we
				// won't find new intersections since we prevent double checking with `handled`.
				continue
			}
			queue.Pop()

			// add event to sweep status
			n := status.InsertAfter(prev, event)

			// compute sweep fields after adding intersections as that may create overlapping
			// segments. Also note that events may have already been computed if it was intersected
			n.computeSweepFields(op, fillRule)
		} else {
			queue.Pop()

			// remove segment from sweep status
			n := event.other.node
			if n == nil {
				continue
			} else if n.SweepPoint == nil {
				// this may happen if the left-endpoint is to the right of the right-endpoint for some reason
				// usually due to a bug in the segment intersection code
				fmt.Println("WARNING: other endpoint already removed, probably buggy intersection code")
			}
			prev := n.Prev()
			next := n.Next()
			if prev != nil && next != nil {
				addIntersections(queue, handled, zs, prev.SweepPoint, next.SweepPoint)
			}
			status.Remove(n) // TODO: this shouldn't touch SweepPoint inside the nodes
		}
		preResult = append(preResult, event)
	}
	status.Clear()

	// reorder preResult, this may be required when addIntersections adds new intersections
	// that may need to ordered before the event causing the intersection (e.g. a right-endpoint).
	// TODO: surely this could be improved by detecting when this happens and only sort a limited
	//       set of events?
	sort.Slice(preResult, SweepEvents(preResult).Less)

	// build result array
	var result [][]*SweepPoint // put segments at the same position together
	for _, event := range preResult {
		// store in result array, set index, and store same position at same index
		if event.inResult {
			// inResult may be set false afterwards due to overlapping edges, we check again
			// when building the polygon
			if len(result) == 0 || !event.Point.Equals(result[len(result)-1][0].Point) {
				if 0 < len(result) && len(result[len(result)-1]) == 1 {
					fmt.Println("WARNING: single segment endpoint at position", result[len(result)-1])
				}
				result = append(result, []*SweepPoint{event})
			} else {
				result[len(result)-1] = append(result[len(result)-1], event)
			}
			event.index = len(result) - 1
		}
	}

	// build resulting polygons
	for _, nodes := range result {
		for _, cur := range nodes {
			if !cur.inResult || cur.processed {
				continue
			}

			windings := 0
			if cur.prevInResult != nil {
				windings = cur.prevInResult.resultWindings
			}

			first := cur
			indexR := len(R.d)
			R.MoveTo(cur.X, cur.Y)
			cur.resultWindings = windings + 1 // always increasing
			cur.other.resultWindings = cur.resultWindings
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
				var next *SweepPoint
				for i := i0 - 1; ; i-- {
					if i < 0 {
						i += len(nodes)
					}
					if i == i0 {
						break
					}
					if nodes[i].inResult && !nodes[i].processed {
						next = nodes[i]
						break
					}
				}
				if next == nil {
					break // contour is done
				} else if next == first {
					first.processed, first.other.processed = true, true
					break
				}
				cur = next

				R.LineTo(cur.X, cur.Y)
				cur.resultWindings = windings
				if cur.left {
					// we go to the right/top
					cur.resultWindings++
				}
				cur.other.resultWindings = cur.resultWindings
				cur.processed, cur.other.processed = true, true
			}
			R.Close()

			if windings%2 != 0 {
				// orient holes clockwise
				hole := (&Path{R.d[indexR:]}).Reverse()
				R.d = append(R.d[:indexR], hole.d...)
			}
		}
	}

	for _, event := range preResult {
		boPointPool.Put(event)
	}
	return R
}
