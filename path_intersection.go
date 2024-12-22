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

// BentleyOttmannEpsilon is the snap rounding grid used by the Bentley-Ottmann algorithm.
// This prevents numerical issues. It must be larger than Epsilon since we use that to calculate
// intersections between segments. It is the number of binary digits to keep.
var BentleyOttmannEpsilon = 1e2 * Epsilon

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
	//opDivide // TODO
)

var nn0, nn1 int
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
	Point               // position of this endpoint
	other   *SweepPoint // pointer to the other endpoint of the segment
	segment int         // segment index to distinguish self-overlapping segments

	// processing the queue
	node *SweepNode // used for fast accessing btree node in O(1) (instead of Find in O(log n))

	// computing sweep fields
	windings          int         // windings of the same polygon (excluding this segment)
	otherWindings     int         // windings of the other polygon
	selfWindings      int         // positive if segment goes left-right (or bottom-top when vertical)
	otherSelfWindings int         // used when merging overlapping segments
	prev              *SweepPoint // segment below

	// building the polygon
	index          int // index into result array
	resultWindings int // windings of the resulting polygon

	// bools at the end to optimize memory layout of struct
	clipping   bool // is clipping polygon (otherwise is subject polygon)
	left       bool // point is left-end of segment
	vertical   bool // segment is vertical
	increasing bool // original direction is left-right (or bottom-top)
	inResult   bool // in final result polygon
	processed  bool // written to final path
}

func (s *SweepPoint) InterpolateY(x float64) float64 {
	t := (x - s.X) / (s.other.X - s.X)
	return s.Interpolate(s.other.Point, t).Y
}

// ToleranceEdgeY returns the y-value of the SweepPoint at the tolerance edges given by xLeft and
// xRight, or at the endpoints of the SweepPoint, whichever comes first.
func (s *SweepPoint) ToleranceEdgeY(xLeft, xRight float64) (float64, float64) {
	if !s.left {
		s = s.other
	}

	y0 := s.Y
	if s.X < xLeft {
		y0 = s.InterpolateY(xLeft)
	}
	y1 := s.other.Y
	if xRight <= s.other.X {
		y1 = s.InterpolateY(xRight)
	}
	return y0, y1
}

func (s *SweepPoint) Reverse() {
	s.left, s.other.left = !s.left, s.left
	s.increasing, s.other.increasing = !s.increasing, !s.increasing
}

func (s *SweepPoint) String() string {
	path := "P"
	if s.clipping {
		path = "Q"
	}
	arrow := "→"
	if !s.left {
		arrow = "←"
	}
	return fmt.Sprintf("%s(%v%v%v)", path, s.Point, arrow, s.other.Point)
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
	if len(p.d) == 0 {
		return 0
	}

	// TODO: change this if we allow non-flat paths
	// allocate all memory at once to prevent multiple allocations/memmoves below
	n := len(p.d) / 4
	if cap(*q) < len(*q)+n {
		q2 := make(SweepEvents, len(*q), len(*q)+n)
		copy(q2, *q)
		*q = q2
	}

	start := Point{p.d[1], p.d[2]}
	for i := 4; i < len(p.d); {
		if p.d[i] != LineToCmd && p.d[i] != CloseCmd {
			panic("non-flat paths not supported")
		}

		n := cmdLen(p.d[i])
		end := Point{p.d[i+n-3], p.d[i+n-2]}
		i += n
		seg++

		if start == end {
			// skip zero-length lineTo or close command
			start = end
			continue
		}

		vertical := start.X == end.X
		increasing := start.X < end.X
		if vertical {
			increasing = start.Y < end.Y
		}
		nn0 += 2
		a := boPointPool.Get().(*SweepPoint)
		b := boPointPool.Get().(*SweepPoint)
		*a = SweepPoint{
			Point:      start,
			clipping:   clipping,
			segment:    seg,
			left:       increasing,
			increasing: increasing,
			vertical:   vertical,
		}
		*b = SweepPoint{
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
		start = end
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
	width := int(math.Log10(float64(len(q)-1))) + 1
	for k := len(q) - 1; 0 <= k; k-- {
		fmt.Fprintf(w, "%*d %v\n", width, len(q)-1-k, q[k])
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
		fmt.Fprintf(w, "%v ↱", strings.Repeat("  ", indent))
		n.right.Print(w, indent+1)
	} else if n.left != nil {
		fmt.Fprintf(w, "%v ↱nil\n", strings.Repeat("  ", indent))
	}
	fmt.Fprintf(w, "%v\n", n.SweepPoint)
	if n.left != nil {
		fmt.Fprintf(w, "%v ↳", strings.Repeat("  ", indent))
		n.left.Print(w, indent+1)
	} else if n.right != nil {
		fmt.Fprintf(w, "%v ↳nil\n", strings.Repeat("  ", indent))
	}
}

// TODO: test performance versus (2,4)-tree (current LEDA implementation), (2,16)-tree (as proposed by S. Naber/Näher in "Comparison of search-tree data structures in LEDA. Personal communication" apparently), RB-tree (likely a good candidate), and an AA-tree (simpler implementation may be faster). Perhaps an unbalanced (e.g. Treap) works well due to the high number of insertions/deletions.
type SweepStatus struct {
	root *SweepNode
}

func (s *SweepStatus) newNode(item *SweepPoint) *SweepNode {
	nn1++
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
	nn1--
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
			if n.left != nil && 0 < n.left.balance() {
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
	} else if 0 < cmp {
		// higher
		n.right = s.newNode(item)
		n.right.parent = n
		rebalance = n.left == nil
	} else {
		// equal, replace
		n.SweepPoint.node = nil
		n.SweepPoint = item
		n.SweepPoint.node = n
		return n
	}

	if rebalance && n.parent != nil {
		n.height++
		s.rebalance(n.parent)
	}

	if cmp < 0 {
		return n.left
	} else {
		return n.right
	}
}

func (s *SweepStatus) InsertAfter(n *SweepNode, item *SweepPoint) *SweepNode {
	var cur *SweepNode
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
		cur = n.left
	} else if n.right == nil {
		// insert directly to the right of n
		n.right = s.newNode(item)
		n.right.parent = n
		rebalance = n.left == nil
		cur = n.right
	} else {
		// insert next to n at a deeper level
		n = n.right
		for n.left != nil {
			n = n.left
		}
		n.left = s.newNode(item)
		n.left.parent = n
		rebalance = n.right == nil
		cur = n.left
	}

	if rebalance && n.parent != nil {
		n.height++
		s.rebalance(n.parent)
	}
	return cur
}

func (s *SweepStatus) Remove(n *SweepNode) {
	ancestor := n.parent
	if n.left == nil || n.right == nil {
		// no children or one child
		child := n.left
		if n.left == nil {
			child = n.right
		}
		if n.parent != nil {
			n.parent.swapChild(n, child)
		} else {
			s.root = child
		}
		if child != nil {
			child.parent = n.parent
		}
	} else {
		// two children
		succ := n.right
		for succ.left != nil {
			succ = succ.left
		}
		ancestor = succ.parent // rebalance from here
		if succ.parent == n {
			// succ is child of n
			ancestor = succ
		}
		succ.parent.swapChild(succ, succ.right)

		// swap succesor with deleted node
		succ.parent, succ.left, succ.right = n.parent, n.left, n.right
		if n.parent != nil {
			n.parent.swapChild(n, succ)
		} else {
			s.root = succ
		}
		if n.left != nil {
			n.left.parent = succ
		}
		if n.right != nil {
			n.right.parent = succ
		}
	}

	// rebalance all ancestors
	for ; ancestor != nil; ancestor = ancestor.parent {
		s.rebalance(ancestor)
	}
	s.returnNode(n)
	return
}

func (s *SweepStatus) Clear() {
	n := s.First()
	for n != nil {
		cur := n
		n = n.Next()
		s.returnNode(cur)
	}
	s.root = nil
}

func (a *SweepPoint) LessH(b *SweepPoint) bool {
	// used for sweep queue
	if a.X != b.X {
		return a.X < b.X // sort left to right
	} else if a.Y != b.Y {
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
		// for equal segments, clipping path is virtually on top (or left if vertical) of subject
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
	sign := 1
	if !a.left {
		sign = -1
	}
	if a.vertical {
		// a is vertical
		if b.vertical {
			// a and b are vertical
			if a.Y == b.Y {
				return sign * a.compareOverlapsV(b)
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

	if a.left && a.other.X < b.other.X || !a.left && b.other.X < a.other.X {
		by := b.InterpolateY(a.other.X) // b's y at a's other
		if a.other.Y == by {
			return sign * a.compareOverlapsV(b)
		} else if a.other.Y < by {
			return sign * -1
		} else {
			return sign * 1
		}
	} else {
		ay := a.InterpolateY(b.other.X) // a's y at b's other
		if ay == b.other.Y {
			return sign * a.compareOverlapsV(b)
		} else if ay < b.other.Y {
			return sign * -1
		} else {
			return sign * 1
		}
	}
}

func (a *SweepPoint) compareV(b *SweepPoint) int {
	// compare segments vertically at a.X and b.X < a.X
	// note that by may be infinite/large for fully/nearly vertical segments
	by := b.InterpolateY(a.X) // b's y at a's left
	if a.Y == by {
		return a.compareTangentsV(b)
	} else if a.Y < by {
		return -1
	} else {
		return 1
	}
}

func (a *SweepPoint) CompareV(b *SweepPoint) int {
	// used for sweep status, a is the point to be inserted / found
	if a.X == b.X {
		// left-point at same X
		if a.Y == b.Y {
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

func compareIntersections(a, b Point) int {
	if a.X == b.X {
		if a.Y == b.Y {
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

func addIntersections(queue *SweepEvents, handled map[SweepPointPair]struct{}, a, b *SweepPoint) {
	// a and b are always left-endpoints and a is below b
	if _, ok := handled[SweepPointPair{a, b}]; ok {
		return
	} else if _, ok := handled[SweepPointPair{b, a}]; ok {
		return
	}

	if !a.left || !b.left {
		fmt.Println("WARNING: a/b not left")
	}

	// find all intersections between segment pair
	// this returns either no intersections, or one or more secant/tangent intersections,
	// or exactly two "same" intersections which occurs when the segments overlap.
	zs_ := [2]Point{}
	zs := zs_[:]
	zs = intersectionLineLineBentleyOttmann(zs[:0], a.Point, a.other.Point, b.Point, b.other.Point)

	// no (valid) intersections
	if len(zs) == 0 {
		handled[SweepPointPair{a, b}] = struct{}{}
		return
	}

	// Non vertical but downward sloped segments may become vertical upon intersection due to
	// floating point rounding and limited precision. We must make sure that the first segment
	// after breaking up at the intersection remains as-is (its left-endpoint was already popped
	// off the queue), but the second segment may become vertical and thus be reversed in direction

	aMinY, aMaxY := a.Y, a.other.Y
	if aMaxY < aMinY {
		aMinY, aMaxY = aMaxY, aMinY
	}
	bMinY, bMaxY := b.Y, b.other.Y
	if bMaxY < bMinY {
		bMinY, bMaxY = bMaxY, bMinY
	}
	for _, z := range zs {
		fmt.Println("INTERSECTION", a, b, "--", z)
		if z.X < a.X {
			fmt.Println("WARNING: ax0", a, b, "at", z)
		} else if a.other.X < z.X {
			fmt.Println("WARNING: ax1", a, b, "at", z)
		} else if a.X != a.other.X && (z.X == a.X || z.X == a.other.X) && (z.Y != a.Y && z.Y != a.other.Y) {
			fmt.Println("WARNING: ay", a, b, "at", z)
		}
		if z.X < b.X {
			fmt.Println("WARNING: b", a, b, "at", z)
		} else if b.other.X < z.X {
			fmt.Println("WARNING: b.other", a, b, "at", z)
		} else if b.X != b.other.X && (z.X == b.X || z.X == b.other.X) && (z.Y != b.Y && z.Y != b.other.Y) {
			fmt.Println("WARNING: by", a, b, "at", z)
		}

		if z.Y < aMinY {
			fmt.Println("WARNING: a2", a, b, "at", z)
		} else if aMaxY < z.Y {
			fmt.Println("WARNING: a2.other", a, b, "at", z)
		}
		if z.Y < bMinY {
			fmt.Println("WARNING: b2", a, b, "at", z)
		} else if bMaxY < z.Y {
			fmt.Println("WARNING: b2.other", a, b, "at", z)
		}
	}

	// sort intersections from left to right
	if !slices.IsSortedFunc(zs, compareIntersections) {
		fmt.Println("WARNING: intersections not sorted")
		slices.SortFunc(zs, compareIntersections)
	}

	// handle a
	aLefts_ := [2]*SweepPoint{a, nil} // there is only one non-tangential intersection
	aLefts := aLefts_[:]
	aPrevLeft, aLastRight := a, a.other
	for _, z := range zs {
		if z == a.Point || z == aLastRight.Point {
			// ignore tangent intersections at the endpoints
			continue
		}

		// split segment at intersection
		nn0 += 2
		aRight := boPointPool.Get().(*SweepPoint)
		aLeft := boPointPool.Get().(*SweepPoint)
		*aRight, *aLeft = *a.other, *a
		aRight.Point = z
		aLeft.Point = z

		// update references
		aPrevLeft.other, aRight.other = aRight, aPrevLeft

		// reverse direction if necessary, see note above
		if a.X < aLastRight.X && aLastRight.Y < a.Y && z.X == aLastRight.X {
			fmt.Println("NOTE: REVERSING DIRECTION for", aLeft)
			aLeft.Reverse()
			aLeft.vertical = true
		}

		// add to queue
		queue.Push(aRight)
		queue.Push(aLeft)
		aLefts = append(aLefts, aLeft)
		aPrevLeft = aLeft
	}
	aPrevLeft.other, aLastRight.other = aLastRight, aPrevLeft

	// handle b
	for _, a := range aLefts {
		handled[SweepPointPair{a, b}] = struct{}{}
	}
	bPrevLeft, bLastRight := b, b.other
	for _, z := range zs {
		if z == b.Point || z == bLastRight.Point {
			// ignore tangent intersections at the endpoints
			continue
		}

		// split segment at intersection
		nn0 += 2
		bRight := boPointPool.Get().(*SweepPoint)
		bLeft := boPointPool.Get().(*SweepPoint)
		*bRight, *bLeft = *b.other, *b
		bRight.Point = z
		bLeft.Point = z

		// update references
		bPrevLeft.other, bRight.other = bRight, bPrevLeft

		// reverse direction if necessary, see note above
		if b.X < bLastRight.X && bLastRight.Y < b.Y && z.X == bLastRight.X {
			fmt.Println("NOTE: REVERSING DIRECTION for", bLeft)
			bLeft.Reverse()
			bLeft.vertical = true
		}

		// add to queue
		queue.Push(bRight)
		queue.Push(bLeft)
		for _, a := range aLefts {
			handled[SweepPointPair{a, bLeft}] = struct{}{}
		}
		bPrevLeft = bLeft
	}
	bPrevLeft.other, bLastRight.other = bLastRight, bPrevLeft
}

func (cur *SweepPoint) computeSweepFields(prev *SweepNode, op pathOp, fillRule FillRule) {
	// cur is left-endpoint
	cur.selfWindings = 1
	if !cur.increasing {
		cur.selfWindings = -1
	}
	if prev != nil {
		// compute windings
		if cur.clipping == prev.clipping {
			cur.windings = prev.windings + prev.selfWindings
			cur.otherWindings = prev.otherWindings + prev.otherSelfWindings
		} else {
			cur.windings = prev.otherWindings + prev.otherSelfWindings
			cur.otherWindings = prev.windings + prev.selfWindings
		}
		cur.prev = prev.SweepPoint
	} else {
		// may have been copied when intersected / broken up
		cur.windings, cur.otherWindings, cur.otherSelfWindings = 0, 0, 0
		cur.prev = nil
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

type toleranceSquare struct {
	X, Y   float64       // snapped value
	Events []*SweepPoint // all events in this square

	// reference node inside or near the square
	// after breaking up segments, this is the previous node (ie. completely below the square)
	Node *SweepNode

	// lower and upper node crossing this square
	Lower, Upper *SweepNode
}

type toleranceSquares []toleranceSquare

func (squares *toleranceSquares) find(x, y float64) (int, bool) {
	// find returns the index of the square at or above (x,y) (or len(squares) if above all)
	// the bool indicates if the square exists, otherwise insert a new square at that index
	for i := len(*squares) - 1; 0 <= i; i-- {
		if (*squares)[i].X < x || (*squares)[i].Y < y {
			return i + 1, false
		} else if (*squares)[i].Y == y {
			return i, true
		}
	}
	return 0, false
}

func (squares *toleranceSquares) Add(x float64, event *SweepPoint, refNode *SweepNode) {
	// refNode is always the node itself for left-endpoints, and otherwise the previous node (ie.
	// the node below) of a right-endpoint, or the next (ie. above) node if the previous is nil.
	// It may be inside or outside the right edge of the square. If outside, it is the first such
	// segment going upwards/downwards from the square (and not just any segment).
	y := snap(event.Y, BentleyOttmannEpsilon)
	fmt.Println("set refNode", x, y, "=>", refNode)
	if idx, ok := squares.find(x, y); !ok {
		// create new tolerance square
		square := toleranceSquare{
			X:      x,
			Y:      y,
			Node:   refNode,
			Events: []*SweepPoint{event},
		}
		*squares = append((*squares)[:idx], append(toleranceSquares{square}, (*squares)[idx:]...)...)
	} else {
		// insert into existing tolerance square
		(*squares)[idx].Node = refNode
		(*squares)[idx].Events = append((*squares)[idx].Events, event)
	}

	// (nearly) vertical segments may still be used as the reference segment for squares around
	// in that case, replace with the new reference node (above or below that segment)
	if !event.left {
		orig := event.other.node
		for i := len(*squares) - 1; 0 <= i && (*squares)[i].X == x; i-- {
			if (*squares)[i].Node == orig {
				(*squares)[i].Node = refNode
				fmt.Println("upt refNode", (*squares)[i].X, (*squares)[i].Y, "=>", refNode)
			}
		}
	}
}

func (event *SweepPoint) breakupSegment(events *[]*SweepPoint, index int, x, y float64) {
	// break up a segment in two parts and let the middle point be (x,y)
	if snap(event.X, BentleyOttmannEpsilon) == x && snap(event.Y, BentleyOttmannEpsilon) == y || snap(event.other.X, BentleyOttmannEpsilon) == x && snap(event.other.Y, BentleyOttmannEpsilon) == y {
		// segment starts or ends in tolerance square, don't break up
		return
	}
	fmt.Println("BREAKUP", event, "at", x, y)

	// original segment should be kept in-place to not alter the queue or status
	nn0 += 2
	right := boPointPool.Get().(*SweepPoint)
	left := boPointPool.Get().(*SweepPoint)
	*right, *left = *event.other, *event
	right.X, right.Y = x, y
	left.X, left.Y = x, y
	left.other, event.other.other = event.other, left
	right.other, event.other = event, right
	right.index, left.index = index, index

	if event.node != nil {
		left.node.SweepPoint = left
		event.node = nil
	}

	*events = append(*events, right, left)
}

func (squares toleranceSquares) breakupCrossingSegments(n int, x float64) {
	// find and break up all segments that cross this tolerance square
	// note that we must move up to find all upwards-sloped segments and then move down for the
	// downwards-sloped segments, since they may need to be broken up in other squares first
	x0, x1 := x-BentleyOttmannEpsilon/2.0, x+BentleyOttmannEpsilon/2.0

	// scan squares bottom to top
	for i := n; i < len(squares); i++ {
		square := &squares[i] // take address to make changes persistent

		// be aware that a tolerance square is inclusive of the left and bottom edge
		// and only the bottom-left corner
		yTop, yBottom := square.Y+BentleyOttmannEpsilon/2.0, square.Y-BentleyOttmannEpsilon/2.0

		// from reference node find the previous/lower/upper segments for this square
		// the reference node may be any of the segments that cross the right-edge of the square,
		// or the first segment below or above the right-edge of the square
		if square.Node != nil {
			if square.Node.SweepPoint == nil {
				fmt.Println("WARNING: node.SweepPoint is nil for square", i, "--", x, square.Y)
			}

			y0, y1 := square.Node.ToleranceEdgeY(x0, x1)
			below, above := y0 < yBottom && y1 <= yBottom, yTop <= y0 && yTop <= y1
			if !below && !above {
				// reference node is inside the square
				square.Lower, square.Upper = square.Node, square.Node
			}

			// find upper node
			if !above {
				for next := square.Node.Next(); next != nil; next = next.Next() {
					y0, y1 := next.ToleranceEdgeY(x0, x1)
					if yTop <= y0 && yTop <= y1 {
						break
					}
					square.Upper = next
					if square.Lower == nil {
						// this is set if the reference node is below the square
						square.Lower = next
					}
				}
			}

			// find lower node and set reference node to the node completely below the square
			if !below {
				prev := square.Node.Prev()
				for ; prev != nil; prev = prev.Prev() {
					y0, y1 := prev.ToleranceEdgeY(x0, x1)
					if y0 < yBottom && y1 <= yBottom { // exclusive for bottom-right corner
						break
					}
					square.Lower = prev
					if square.Upper == nil {
						// this is set if the reference node is above the square
						square.Upper = prev
					}
				}
				square.Node = prev
			}
		}
		fmt.Println("square", i, "--", x, square.Y, "--", square.Lower, square.Upper, "prev", square.Node)

		// find all segments that cross the tolerance square upwards or horizontally
		// first find all segments that extend to the right (they are in the sweepline status)
		if square.Lower != nil {
			for node := square.Lower; ; node = node.Next() {
				y0, y1 := node.ToleranceEdgeY(x0, x1)
				if y0 < yTop && yBottom < y1 {
					node.breakupSegment(&squares[i].Events, i, x, square.Y)
				}
				if node == square.Upper {
					break
				}
			}
		}

		// then find other segments in the square below that may cross into this square
		// these can only be right-endpoints in those squares and are downwards sloped
		if n < i {
			for _, event := range squares[i-1].Events {
				if !event.left && yBottom <= event.other.Y {
					// right-endpoint in square below with its left-endpoint above yBottom
					y0, _ := event.ToleranceEdgeY(x0, x1)
					if yBottom <= y0 {
						event.breakupSegment(&squares[i].Events, i, x, square.Y)
					}
				}
			}
		}
	}

	// scan squares top to bottom
	for i := len(squares) - 1; n <= i; i-- {
		square := &squares[i]
		yTop := square.Y + BentleyOttmannEpsilon/2.0

		// find all segments that cross the tolerance square from above
		if square.Upper != nil {
			for node := square.Upper; ; node = node.Prev() {
				y0, y1 := node.ToleranceEdgeY(x0, x1)
				if yTop <= y0 && y1 < yTop {
					node.breakupSegment(&squares[i].Events, i, x, square.Y)
				}
				if node == square.Lower {
					break
				}
			}
		}

		// then find other segments in the square above that may cross into this square
		// these can only be right-endpoints in those squares and are upwards sloped
		if i+1 < len(squares) {
			for _, event := range squares[i+1].Events {
				if !event.left && event.other.Y < yTop {
					// right-endpoint in square above with its left-endpoint below yTop
					y0, _ := event.ToleranceEdgeY(x0, x1)
					if y0 < yTop {
						event.breakupSegment(&squares[i].Events, i, x, square.Y)
					}
				}
			}
		}
	}
}

type nodeList []*SweepNode

func (a nodeList) Len() int {
	return len(a)
}

func (a nodeList) Less(i, j int) bool {
	return a[i].CompareV(a[j].SweepPoint) < 0
}

func (a nodeList) Swap(i, j int) {
	a[i].SweepPoint.node, a[j].SweepPoint.node = a[j].SweepPoint.node, a[i].SweepPoint.node
	a[i].SweepPoint, a[j].SweepPoint = a[j].SweepPoint, a[i].SweepPoint
	a[i], a[j] = a[j], a[i]
}

type eventList []*SweepPoint

func (a eventList) Len() int {
	return len(a)
}

func (a eventList) Less(i, j int) bool {
	return a[i].LessH(a[j])
}

func (a eventList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (below *SweepPoint) mergeOverlapping(above *SweepPoint, op pathOp, fillRule FillRule) {
	if below.clipping == above.clipping {
		above.selfWindings += below.selfWindings
		above.otherSelfWindings += below.otherSelfWindings
		above.windings = below.windings
		above.otherWindings = below.otherWindings
	} else {
		above.selfWindings += below.otherSelfWindings
		above.otherSelfWindings += below.selfWindings
		above.windings = below.otherWindings
		above.otherWindings = below.windings
	}
	below.inResult, below.other.inResult = false, false

	above.inResult = above.InResult(op, fillRule)
	above.other.inResult = above.inResult
}

func bentleyOttmann(ps, qs Paths, op pathOp, fillRule FillRule) *Path {
	// TODO: make public and add grid spacing argument
	// TODO: support OpDiv
	// TODO: support open paths on ps
	// TODO: support elliptical arcs
	// TODO: use a red-black tree for the sweepline status?
	// TODO: use a red-black tree for the sweepline queue?

	// Implementation of the Bentley-Ottmann algorithm by reducing the complexity of finding
	// intersections to O((n + k) log n), with n the number of segments and k the number of
	// intersections. All special cases are handled by use of:
	// - M. de Berg, et al., "Computational Geometry", Chapter 2, DOI: 10.1007/978-3-540-77974-2
	// - F. Martínez, et al., "A simple algorithm for Boolean operations on polygons", Advances in
	//   Engineering Software 64, p. 11-19, 2013, DOI: 10.1016/j.advengsoft.2013.04.004
	// - J.D. Hobby, "Practical segment intersection with ﬁnite precision output", Computational
	//   Geometry, 1997
	// - J. Hershberger, "Stable snap rounding", Computational Geometry: Theory and Applications,
	//   2013, DOI: 10.1016/j.comgeo.2012.02.011
	// - https://github.com/verven/contourklip

	// Bentley-Ottmann is the most popular algorithm to find path intersections, which is mainly
	// due to it's relative simplicity and the fact that it is (much) faster than the naive
	// approach. It however does not specify how special cases should be handled (overlapping
	// segments, multiple segment endpoints in one point, vertical segments), which is treated in
	// later works by other authors (e.g. Martínez from which this implementation draws
	// inspiration). I've made some small additions and adjustments to make it work in all cases
	// I encountered. Specifically, this implementation has the following properties:
	// - Subject and clipping paths may consist of any number of contours / subpaths.
	// - Any contour may be oriented clockwise (CW) or counter-clockwise (CCW).
	// - Any path or contour may self-intersect any number of times.
	// - Any point may be crossed multiple times by any path.
	// - Segments may overlap any number of times by any path.
	// - Segments may be vertical.
	// - The clipping path is implicitly closed, it makes no sense if it is an open path.
	// - The subject path is currently implicitly closed, but it is WIP to support open paths.
	// - Paths are currently flattened, but supporting Bézier or elliptical arcs is a WIP.

	// An unaddressed problem in those works is that of numerical accuracies. The main problem is
	// that calculating the intersections is not precise; the imprecision of the initial endpoints
	// of a path can be trivially fixed before the algorithm. Intersections however are calculated
	// during the algorithm and must be addressed. There are a few authors that propose a solution,
	// and Hobby's work inspired this implementation. The approach taken is somewhat different
	// though:
	// - Instead of integers (or rational numbers implemented using integers), floating points are
	//   used for their speed. It isn't even necessary that the grid points can be represented
	//   exactly in the floating point format, as long as all points in the tolerance square around
	//   the grid points snap to the same point. Now we can compare using == instead of an equality
	//   test.
	// - As in Martínez, we treat an intersection as a right- and left-endpoint combination and not
	//   as a third type of event. This avoids rearrangement of events in the sweep status as it is
	//   removed and reinserted into the right position, but at the cost of more delete/insert
	//   operations in the sweep status (potential to improve performance).
	// - As we run the Bentley-Ottmann algorithm, found endpoints must also be snapped to the grid.
	//   Since intersections are found in advance (ie. towards the right), we have no idea how the
	//   sweepline status will be yet, so we cannot snap those intersections to the grid yet. We
	//   must snap all endpoints/intersections when we reach them (ie. pop them off the queue).
	//   When we get to an endpoint, snap all endpoints in the tolerance square around the grid
	//   point to that point, and process all endpoints and intersections. Additionally, we should
	//   break-up all segments that pass through the square into two, and snap them to the grid
	//   point as well. These segments pass very close to another endpoint, and by snapping those
	//   to the grid we avoid the problem where we may or may not find that the segment intersects.
	// - Note that most (not all) intersections on the right are calculated with the left-endpoint
	//   already snapped, which may move the intersection to another grid point. These inaccuracies
	//   depend on the grid spacing and can be made small relative to the size of the input paths.
	//
	// The difference with Hobby's steps is that we advance Bentley-Ottmann for the entire column,
	// and only then do we calculate crossing segments. I'm not sure what reason Hobby has to do
	// this in two fases. Also, Hobby uses a shadow sweep line status structure which contains the
	// segments sorted after snapping. Instead of using two sweep status structures (the original
	// Bentley-Ottmann and the shadow with snapped segments), we sort the status after each column.
	// Additionally, we need to keep the sweep line queue structure ordered as well for the result
	// polygon (instead of the queue we gather the events for each sqaure, and sort those), and we
	// need to calculate the sweep fields for the result polygon.
	//
	// It is best to think of processing the tolerance squares, one at a time moving bottom-to-top,
	// for each column while moving the sweepline from left to right. Since all intersections
	// in this implementation are already converted to two right-endpoints and two left-endpoints,
	// we do all the snapping after each column and snapping the endpoints beforehand is not
	// necessary. We pop off all events from the queue that belong to the same column and process
	// them as we would with Bentley-Ottmann. This ensures that we find all original locations of
	// the intersections (except for intersections between segments in the sweep status structure
	// that are not yet adjacent, see note above) and may introduce new tolerance squares. For each
	// square, we find all segments that pass through and break them up and snap them to the grid.
	// Then snap all endpoints in the
	// square to the grid. We must sort the sweep line status and all events per square to account
	// for the new order after snapping. Some implementation observations:
	// - We must breakup segments that cross the square BEFORE we snap the square's endpoints,
	//   since we depend on the order of in the sweep status (from after processing the column
	//   using the original Bentley-Ottmann sweep line) for finding crossing segments.
	// - We find all original locations of intersections for adjacent segments during and after
	//   processing the column. However, if intersections become adjacent later on, the
	//   left-endpoint has already been snapped and the intersection has moved.
	// - We must be careful with overlapping segments. Since gridsnapping may introduce new
	//   overlapping segments (potentially vertical), we must check for that when processing the
	//   right-endpoints of each square.
	//
	// We thus proceed as follows:
	// - Process all events from left-to-right in a column using the regular Bentley-Ottmann.
	// - Identify all "hot" squares (those that contain endpoints / intersections).
	// - Find all segments that pass through each hot square, break them up and snap to the grid.
	//   These may be segments that start left of the column and end right of it, but also segments
	//   that start or end inside the column, or even start AND end inside the column (eg. vertical
	//   or almost vertical segments).
	// - Snap all endpoints and intersections to the grid.
	// - Compute sweep fields / windings for all new left-endpoints.
	// - Handle segments that are now overlapping for all right-endpoints.
	// Note that we must be careful with vertical segments.

	nn0, nn1 = 0, 0
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
		}
	}

	// check for path bounding boxes to overlap
	// TODO: cluster paths that overlap and treat non-overlapping clusters separately, this
	// makes the algorithm "more linear"
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

	// run sweep line left-to-right
	status := &SweepStatus{}                                    // contains only left events
	squares := toleranceSquares{}                               // sorted vertically, squares and their events
	nodes := []*SweepNode{}                                     // used for ordering status
	handled := make(map[SweepPointPair]struct{}, len(*queue)*2) // prevent testing for intersections more than once, allocation length is an approximation
	for 0 < len(*queue) {
		// TODO: skip or stop depending on operation if we're to the left/right of subject/clipping polygon

		// We slightly divert from the original Bentley-Ottmann and paper implementation. First
		// we find the top element in queue but do not pop it off yet. If it is a right-event, pop
		// from queue and proceed as usual, but if it's a left-event we first check (and add) all
		// surrounding intersections to the queue. This may change the order from which we should
		// pop off the queue, since intersections may create right-events, or new left-events that
		// are lower (by compareTangentV). If no intersections are found, pop off the queue and
		// proceed as usual.

		// process all events of the current column
		n := len(squares)
		x := snap(queue.Top().X, BentleyOttmannEpsilon)
		fmt.Println("----")
		fmt.Println("X", x)
		fmt.Println(queue)
		for 0 < len(*queue) && snap(queue.Top().X, BentleyOttmannEpsilon) == x {
			event := queue.Top()
			fmt.Println()
			fmt.Println("event:", event)
			// TODO: breaking intersections into two right and two left endpoints is not the most
			// efficient. We could keep an intersection-type event and simply swap the order of the
			// segments in status (note there can be multiple segments crossing in one point). This
			// would alleviate a 2*m*log(n) search in status to remove/add the segments (m number
			// of intersections in one point, and n number of segments in status), and instead use
			// an m/2 number of swap operations. This alleviates pressure on the CompareV method.
			if !event.left {
				queue.Pop()

				n := event.other.node
				if n == nil {
					fmt.Println("WARNING: right-endpoint not part of status, probably buggy intersection code")
					boPointPool.Put(event)
					nn0--
					continue
				} else if n.SweepPoint == nil {
					// this may happen if the left-endpoint is to the right of the right-endpoint for some reason
					// usually due to a bug in the segment intersection code
					fmt.Println("WARNING: other endpoint already removed, probably buggy intersection code")
					fmt.Println(event)
					boPointPool.Put(event)
					nn0--
					continue
				}

				// find intersections between the now adjacent segments
				prev := n.Prev()
				next := n.Next()
				if prev != nil && next != nil {
					addIntersections(queue, handled, prev.SweepPoint, next.SweepPoint)
				}

				// add event to tolerance square
				if prev != nil {
					squares.Add(x, event, prev)
				} else {
					// next can be nil
					squares.Add(x, event, next)
				}

				// remove event from sweep status
				status.Remove(n)
			} else {
				// add intersections to queue
				prev, next := status.FindPrevNext(event)
				if prev != nil {
					addIntersections(queue, handled, prev.SweepPoint, event)
				}
				if next != nil {
					addIntersections(queue, handled, event, next.SweepPoint)
				}
				if event != queue.Top() {
					// check if the queue order was changed, this happens if the current event
					// is the left-endpoint of a segment that intersects with an existing segment
					// that goes below
					continue
				}
				queue.Pop()

				// add event to sweep status
				n := status.InsertAfter(prev, event)

				// add event to tolerance square
				squares.Add(x, event, n)
			}
		}
		fmt.Println("status:")
		fmt.Println(status)

		// find all crossing segments, break them up and snap to the grid
		squares.breakupCrossingSegments(n, x)

		for j := n; j < len(squares); j++ {
			square := &squares[j] // take address to make changes persistent

			// snap events to grid
			// note that this may make segments overlapping from/to the left, we handle the former
			// but ignore the latter. This may result in overlapping segments not strictly ordered
			// also note that a downwards sloped segment may become vertical
			for i := 0; i < len(square.Events); i++ {
				event := square.Events[i]
				event.index = j
				event.X, event.Y = x, square.Y

				other := event.other.Point.Gridsnap(BentleyOttmannEpsilon)
				if event.Point == other {
					// remove collapsed segments
					boPointPool.Put(event)
					nn0--
					square.Events = append(square.Events[:i], square.Events[i+1:]...)
					i--
				} else if event.X == other.X {
					fmt.Println("VERTICAL", event)
					// segment is now vertical
					if !event.left && event.Y < other.Y {
						// downward sloped, reverse direction
						event.Reverse()
					}
					event.vertical = true
				}
			}
		}

		// we must first snap all segments in this column before sorting
		for _, square := range squares[n:] {
			// reorder sweep status and events for result polygon
			// note that the number of events/nodes is usually small
			// TODO: improve efficiency? could we sort events/status simultaneously?
			nodes = nodes[:0]
			if square.Lower != nil {
				for n := square.Lower; ; n = n.Next() {
					nodes = append(nodes, n)
					if n == square.Upper {
						break
					}
				}
			}
			sort.Sort(nodeList(nodes))
			sort.Sort(eventList(square.Events))

			// merge overlapping segments on right-endpoints
			// note that order is reversed for right-endpoints
			first := len(square.Events)
			for i := len(square.Events) - 1; 0 <= i; i-- {
				if event := square.Events[i]; event.left {
					first = i
				} else if 0 < i && !square.Events[i-1].left && event.Point == square.Events[i-1].Point && event.other.Point == square.Events[i-1].other.Point {
					event.other.mergeOverlapping(square.Events[i-1].other, op, fillRule)
				}
			}

			// compute sweep fields on left-endpoints
			var prev *SweepNode
			for i := first; i < len(square.Events); i++ {
				if event := square.Events[i]; !event.left {
					fmt.Println("WARNING: left/right endpoints badly ordered")
				} else if event.node == nil {
					// vertical
					if prev != nil {
						// against last (right-extending) left-endpoint in square
						// inside this square there are no crossing segments, they have been broken
						// up and have their left-endpoints sorted
						event.computeSweepFields(prev, op, fillRule)
					} else {
						// against first segment below square
						// square.Node may be nil
						event.computeSweepFields(square.Node, op, fillRule)
					}
				} else {
					event.computeSweepFields(event.node.Prev(), op, fillRule)
					prev = event.node
				}
			}
		}
	}
	status.Clear()

	for _, square := range squares {
		for _, event := range square.Events {
			if event.left {
				fmt.Println(event, event.inResult, "--", event.windings, event.selfWindings, "/", event.otherWindings, event.otherSelfWindings, fmt.Sprintf("%p", event))
			}
		}
	}

	//for j := 0; j < len(squares); j++ {
	//	first := true
	//	for i, event := range squares[j].Events {
	//		if event.inResult {
	//			if first {
	//				fmt.Println("square", j, squares[j].Y)
	//				first = false
	//			}
	//			fmt.Println("", i, event.index, event)
	//		}
	//	}
	//}

	// build resulting polygons
	for _, square := range squares {
		for _, cur := range square.Events {
			if !cur.inResult || cur.processed {
				continue
			}

			windings := 0
			prev := cur.prev
			for {
				if prev == nil {
					break
				} else if prev.inResult {
					windings = prev.resultWindings
					break
				}
				prev = prev.prev
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
				nodes := squares[cur.other.index].Events
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
					} else if nodes[i].inResult && !nodes[i].processed {
						next = nodes[i]
						break
					}
				}
				if next == nil {
					fmt.Println("WARNING: next node for result polygon is nil, probably buggy intersection code")
					break
				} else if next == first {
					break // contour is done
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
			first.processed, first.other.processed = true, true
			R.Close()

			if windings%2 != 0 {
				// orient holes clockwise
				hole := (&Path{R.d[indexR:]}).Reverse()
				R.d = append(R.d[:indexR], hole.d...)
			}
		}

		for _, event := range square.Events {
			boPointPool.Put(event)
			nn0--
		}
	}
	if nn0 != 0 || nn1 != 0 {
		fmt.Println("WARNING: pool count: points", nn0, "nodes", nn1)
	}
	return R
}
