package canvas

import (
	"fmt"
	"math"
	"sort"
)

/*
Paths are cut at the intersections between P and Q. The intersections are put into a doubly linked list with paths going forward and backward over P and Q. Depending on the boolean operation we should choose the right cut. Note that there can be circular loops when choosing cuts based on a condition, so we should take care to visit all intersections. Additionally, if path P or path Q contain subpaths with a different winding, we will first combine the subpaths so to remove all subpath intersections.

Functions:
 - LineLine, LineQuad, LineCube, LineEllipse: find intersections between segments (line is A, the other is B) and record coordinate, position along segment A and B in the range of [0,1], direction of segment A and B at intersection, and whether the intersection is secant (crossing) or tangent (touching).
 - appendSegment, rayIntersections, selfCollisions, collisions: find intersections between paths and record segment index, and for collisions also record kind (AintoB or BintoA), parallel (No-/A-/B-/ABParallel).
 - cutPathSegment: cut segment at position [0,1]
 - intersectionNodes: cut path at the intersections and connect as two doubly-linked lists, one along path A and one along path B, recording the path from one node to the other. Handles parallel parts as well.
 - cut: cut path at the intersections
 - booleanIntersections: build up path from intersections according to boolean operation
 - boolean: boolean operation on path
*/

// ContainsPath returns true if path q is contained within path p, i.e. path q is inside path p and both paths have no intersections (but may touch). Paths must have been settled to remove self-intersections.
func (p *Path) ContainsPath(q *Path) bool {
	ps, qs := p.Split(), q.Split()
	for _, qi := range qs {
		inside := false
		for _, pi := range ps {
			zp, _ := pathIntersections(pi, qi, false, false)
			if len(zp) == 0 && qi.inside(pi) {
				inside = true
				break
			}
		}
		if !inside {
			return false
		}
	}
	return true
}

func leftmostWindings(subpath int, ps []*Path, x, y float64) int {
	// Count windings on left-most coordinate, taking into account tangent intersections which may
	// be tangent on the outside or inside of the current subpath.
	// Secant intersections are not counted.
	zs := ps[subpath].RayIntersections(x, y)

	//ccw := true
	var angle0, angle1 float64 // as this is the left-most point, must be in [-0.5*PI,0.5*PI]
	angle0 = angleNorm(zs[0].Dir + math.Pi)
	if Equal(zs[0].T, 1.0) {
		angle1 = zs[1].Dir
	} else {
		angle1 = zs[0].Dir
	}
	if angle1 < angle0 {
		// angle turns right, ie. part of a clock-wise oriented path
		angle0, angle1 = angle1, angle0
		//ccw = false
	}
	angle0 = angle1 + angleNorm(angle0-angle1)

	var n int
	for i := range ps {
		if i == subpath {
			continue
		}

		zs := ps[i].RayIntersections(x, y)
		ni, tangenti := windings(zs)
		if !tangenti {
			n += ni
		} else {
			//var in0, in1 bool
			in0 := angleBetweenExclusive(zs[0].Dir+math.Pi, angle1, angle0)
			//if Equal(zs[0].T, 1.0) {
			//	in1 = angleBetweenExclusive(zs[1].Dir, angle1, angle0)
			//} else {
			//	in1 = angleBetweenExclusive(zs[0].Dir, angle1, angle0)
			//}

			//fmt.Println(in0, in1, ccw)
			if !in0 { //&& !in1|| {
				// following this subpath would go inside the subpath of interest
				// count non-boundary windings of this path
				n += ni
			}
		}
	}
	return n
}

// pseudoVertex is an self-intersection node on the path with indices (A is further along the original path, B is over the crossing path) going further along the path in either direction. We keep information of the direction of crossing, either A goes left of B (AintoB) or B goes left of A (BintoA). TODO: tangents?
type pseudoVertex struct {
	k     int   // pseudo vertex pair index
	Point       // position
	p     *Path // path to next pseudo vertex
	next  int   // index to next vertex in array (over the intersecting path)

	AintoB  bool // A goes into the LHS of B
	Tangent bool
}

func (v pseudoVertex) String() string {
	var extra string
	if v.AintoB {
		extra += " AintoB"
	} else {
		extra += " BintoA"
	}
	if v.Tangent {
		extra += " Tangent"
	}
	return fmt.Sprintf("(%d {%g,%g} ·→%d%s)", v.k, v.Point.X, v.Point.Y, v.next, extra)
}

type nodeItem struct {
	i              int // index in nodes
	parentWindings int // winding number of parent (outer) ring
	winding        int // winding of current ring (+1 or -1)
}

// Settle simplifies a path by removing all self-intersections and overlapping parts. Open paths are not handled and returned as-is. The returned subpaths are oriented counter clock-wise when filled and clock-wise for holes. This means that the result is agnostic to the winding rule used for drawing. The result will only contain point-tangent intersections, but not parallel-tangent intersections or regular intersections.
// See L. Subramaniam, "Partition of a non-simple polygon into simple pologons", 2003
func (p *Path) Settle(fillRule FillRule) *Path {
	// TODO: handle tangent intersections, which should divide into inner/disjoint rings
	// TODO: handle and remove parallel parts
	// TODO: for EvenOdd, output filled polygons only, not fill-rings and hole-rings
	if p.Empty() {
		return p
	}

	// split open and closed paths since we only handle closed paths
	ps := p.Split()
	p = &Path{}
	open := &Path{}
	for i := 0; i < len(ps); i++ {
		if !ps[i].Closed() {
			open = open.Append(ps[i])
			ps = append(ps[:i], ps[i+1:]...)
			i--
		} else {
			p = p.Append(ps[i])
		}
	}
	if p.Empty() {
		return open
	}

	// Flatten Bézier segments, this is justified since the primary usage of Settle is after
	// applying stroke, which already flattened all Bézier curves. The other usages of Settle, such
	// as for path boolean operations, suffer from the loss of precision (though this is very common
	// among path manipulation libraries).
	// Additionally, this ensures that the path X-monotone (which should be preserved if in the
	// future we add support for quad/cube intersections).
	// TODO: can we delay XMonotone until after pathIntersections to avoid processing longer paths?
	// TODO: can we undo XMonotone at the end so we don't end up with longer paths?
	quad := func(p0, p1, p2 Point) *Path {
		return flattenQuadraticBezier(p0, p1, p2, Tolerance)
	}
	cube := func(p0, p1, p2, p3 Point) *Path {
		return flattenCubicBezier(p0, p1, p2, p3, Tolerance)
	}
	arc := func(start Point, rx, ry, phi float64, large, sweep bool, end Point) *Path {
		if !Equal(rx, ry) {
			return flattenEllipticArc(start, rx, ry, phi, large, sweep, end, Tolerance)
		}
		return xmonotoneEllipticArc(start, rx, ry, phi, large, sweep, end)
	}
	p = p.replace(nil, quad, cube, arc)

	// zp is an list of even length where each ith intersections is a pseudo-vertex of the ith+len/2
	zp, zq := pathIntersections(p, nil, false, false)
	psOrig := ps
	ps = p.Split()
	//for i, z := range zp {
	//	fmt.Println(i, z)
	//}
	//for i := range zp {
	//	fmt.Println(i, zp[i])
	//}

	// sort zq and keep indices between pseudo-vertices
	pair := make([]int, len(zp))
	for i := range zp {
		pair[i] = i
	}
	sort.Stable(pathIntersectionSort{zq, pair})
	//for i := range pair {
	//	fmt.Println(i, pair[i])
	//}

	// build up map from 2-degenerate intersections to nodes
	k := 0
	idxK := make([]int, len(zp))
	for i, _ := range zp {
		if i < pair[i] {
			idxK[i] = k
			idxK[pair[i]] = k
			k++
		}
	}
	n := k

	// cut path at intersections
	paths, segs := cut(p, zp)
	//for i, z := range zp[1:] {
	//	if z.Seg < zp[i].Seg || z.Seg == zp[i].Seg && !Equal(z.T, zp[i].T) && z.T < zp[i].T {
	//		fmt.Println(i, "bad", zp[i], z, zp[i].Less(z))
	//	}
	//}
	//fmt.Println(len(paths), len(zp))

	// build up linked nodes between the intersections
	// reverse direction for clock-wise path to ensure one of both paths goes outwards
	// the next index is always along the "other" path, ie. the path intersecting the current
	i0, subpathIndex := 0, 0
	nodes := make([]pseudoVertex, len(zp))
	for i, _ := range zp {
		nodes[i].k = idxK[i]
		nodes[i].Point = zp[i].Point
		nodes[i].p = paths[i]

		// the next node on the end of a subpath should be its starting point
		i1 := i + 1
		subpathIndex = segs.get(zp[i].Seg)
		if i1 == len(zp) || segs.get(zp[i1].Seg) != subpathIndex {
			i0, i1 = i1, i0
		}
		nodes[i].next = pair[i1] // next node on the intersecting (not current) path

		nodes[i].AintoB = zp[i].Into
		nodes[i].Tangent = zp[i].Tangent
	}
	//for i := range nodes {
	//	fmt.Println(i, nodes[i], nodes[i].p)
	//}
	//fmt.Println()

	// split simple (non-self-intersecting) paths from intersecting paths
	// find the starting intersections and their winding for intersecting paths
	j0 := 0
	simple := &Path{}
	var xs []float64
	var queue []nodeItem
	for i, pi := range ps {
		if len(pi.d) <= 8 {
			continue // consists solely of MoveTo and Close
		}

		j1 := j0
		for _, z := range zp[j0:] {
			if segs[i] <= z.Seg {
				break
			}
			j1++
		}
		hasIntersections := j0 != j1

		if !hasIntersections {
			// use unflattened path when subpath has no intersections, but make sure it is XMonotone
			pi = psOrig[i].XMonotone()
		}

		// Find the left-most path segment coordinate for each subpath, we thus know that to the
		// right of the coordinate the path is filled and that it is an outer ring. If the path
		// runs counter clock-wise at the coordinate, the LHS is filled, otherwise it runs
		// clock-wise and has the RHS filled.
		// Follow the path until the first intersection and add to the queue.
		var pos Point
		j, seg, curSeg := 0, 0, 0
		for i := 4; i < len(pi.d); {
			if curSeg == 0 || pi.d[i-3] < pos.X {
				pos = Point{pi.d[i-3], pi.d[i-2]}
				seg = curSeg + 1
				j = i
			}
			i += cmdLen(pi.d[i])
			curSeg++
		}

		// get directions to and from the leftmost point
		next := pi.direction(j, 0.0)
		if j == 4 {
			j = len(pi.d)
		}
		prev := pi.direction(j-cmdLen(pi.d[j-1]), 1.0)

		// find if leftmost point turns to the left (CCW) or to the right (CW)
		var ccw bool
		if dir := prev.PerpDot(next); Equal(dir, 0.0) {
			// both segment go straight up or down
			ccw = next.Y < 0.0 // goes down
		} else {
			ccw = 0.0 < dir
		}

		parentWindings := leftmostWindings(i, ps, pos.X, pos.Y)

		// find the intersections for the subpath
		if !hasIntersections {
			// subpath has no intersections
			winding := 1
			if !ccw {
				winding = -1
			}

			windings := parentWindings + winding
			fills := fillRule.Fills(windings)
			if fills != fillRule.Fills(parentWindings) {
				if ccw != fills {
					// orient filling paths CCW
					pi = pi.Reverse()
				}
				simple = simple.Append(pi)
			}
		} else {
			// find the first intersection that follows
			next := j0
			for j := j0; j < j1; j++ {
				if seg <= zp[j].Seg {
					next = j
					break
				}
			}

			// get the previous intersection to find the whole path segment including the left-most
			// vertex, this will allow us to determine if this part runs counter clock-wise
			prev := next - 1
			if prev < j0 {
				prev = j1 - 1
			}
			winding := 1
			if !ccw {
				winding = -1
			}

			// from the intersection, we will follow the "other" path first
			next = pair[next]

			// insert into queueDisjoint reverse sorted by X
			// if the path goes ccw, then the LHS is filled when arriving at the intersection
			k := len(queue)
			for 0 < k && xs[k-1] <= pos.X {
				k--
			}

			item := nodeItem{next, parentWindings, winding}
			queue = append(queue[:k], append([]nodeItem{item}, queue[k:]...)...)
			xs = append(xs[:k], append([]float64{pos.X}, xs[k:]...)...)
		}
		j0 = j1
	}

	R := &Path{}
	ring := 0
	visits := make([]int, n) // visits per intersections, we will visit each twice
	for 0 < len(queue) {
		cur := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		j := len(queue)

		// process all nodes connected on the outside (another outer ring) or inside (inner ring)
		if 2 <= visits[nodes[cur.i].k] {
			continue // already processed
		}

		i0 := cur.i
		windings := cur.parentWindings + cur.winding
		fills := fillRule.Fills(windings)
		use := fills != fillRule.Fills(cur.parentWindings)

		i := i0
		r := &Path{}
		for {
			visits[nodes[i].k]++
			if visits[nodes[i].k] < 2 {
				var item nodeItem
				if !nodes[i].Tangent && (cur.winding == 1) != nodes[i].AintoB {
					// inner ring
					//fmt.Println(ring, i, "inner", nodes[i])
					item = nodeItem{pair[i], windings, cur.winding}
				} else {
					// disjoint ring
					winding := -cur.winding
					if nodes[i].Tangent {
						winding = cur.winding
					}
					//fmt.Println(ring, i, "disjoint", nodes[i])
					item = nodeItem{pair[i], cur.parentWindings, winding}
				}
				queue = append(queue[:j], append([]nodeItem{item}, queue[j:]...)...)
			}

			if use {
				r = r.Join(nodes[i].p)
			}
			i = nodes[i].next
			if i == i0 {
				break
			}
		}

		if use {
			if (cur.winding == 1) != fills {
				r = r.Reverse() // orient all filling paths CCW
			}
			r.Close()
			r.optimizeClose()
			R = R.Append(r)
		}
		ring++
	}
	// TODO: undo X-Monotone to give a more optimal path?
	return R.Append(simple).Append(open)
}

// And returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) And(q *Path) *Path {
	return boolean(p, pathOpAnd, q)
}

// Or returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Or(q *Path) *Path {
	return boolean(p, pathOpOr, q)
}

// Xor returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Xor(q *Path) *Path {
	return boolean(p, pathOpXor, q)
}

// Not returns the boolean path operation of path p and q. Path q is implicitly closed.
func (p *Path) Not(q *Path) *Path {
	return boolean(p, pathOpNot, q)
}

// DivideBy returns the division of path p by path q at intersections.
func (p *Path) DivideBy(q *Path) *Path {
	return boolean(p, pathOpDivide, q)
}

type pathOp int

const (
	pathOpAnd pathOp = iota
	pathOpOr
	pathOpXor
	pathOpNot
	pathOpDivide
)

// path p can be open or closed paths (we handle them separately), path q is closed implicitly
func boolean(p *Path, op pathOp, q *Path) *Path {
	// return in case of one path is empty
	if q.Empty() {
		if op != pathOpAnd {
			return p
		}
		return &Path{}
	}
	if p.Empty() {
		if op == pathOpOr || op == pathOpXor {
			return q
		}
		return &Path{}
	}

	// remove self-intersections within each path and make filling paths CCW
	p = p.Settle(NonZero) // TODO: where to get fillrule from?
	q = q.Settle(NonZero)

	ps, qs := p.Split(), q.Split()

	// implicitly close all subpaths of path q
	q = &Path{} // collect all closed paths
	lenQs := make([]int, len(qs))
	for i := range qs {
		lenQs[i] = qs[i].Len()
		if !qs[i].Closed() {
			qs[i].Close()
		}
		q = q.Append(qs[i])
	}

	// find all intersections (incl. parallel-tangent but not point-tangent) between p and q
	zp, zq := pathIntersections(p, q, false, true)

	// split open subpaths from p
	j := 0      // index into zp
	p = &Path{} // collect all closed paths
	offset, shift := 0, 0
	Ropen := &Path{}
	for i := 0; i < len(ps); i++ {
		n := 0
		length, closed := ps[i].Len(), ps[i].Closed()
		for ; j+n < len(zp) && zp[j+n].Seg < offset+length; n++ {
			if closed {
				zp[j+n].Seg -= shift
			} else {
				zp[j+n].Seg -= offset
			}
		}
		offset += length

		if closed {
			p = p.Append(ps[i])
			j += n
		} else {
			// open subpath on P
			hasIntersections := false
			for _, z := range zp[j : j+n] {
				if !z.Tangent {
					hasIntersections = true
					break
				}
			}

			if !hasIntersections {
				// either the path is outside, inside, or fully on the boundary
				p0 := ps[i].StartPos()
				n, boundary := q.Windings(p0.X, p0.Y)
				for k := 4; k < len(ps[i].d) && boundary; {
					// check along path in case parts are parallel/on the boundary
					p0 = segmentPos(Point{ps[i].d[k-3], ps[i].d[k-2]}, ps[i].d[k:], 0.5)
					n, boundary = q.Windings(p0.X, p0.Y)
					k += cmdLen(ps[i].d[k])
				}
				inside := n != 0 // NonZero
				if op == pathOpOr || inside && op == pathOpAnd || !inside && !boundary && (op == pathOpXor || op == pathOpNot) {
					Ropen = Ropen.Append(ps[i])
				}
			} else {
				// paths cross, select the parts outside/inside depending on the operation
				pss, _ := cut(ps[i], zp[j:j+n])
				inside := !zp[j].Into
				if op == pathOpOr || inside && op == pathOpAnd || !inside && (op == pathOpXor || op == pathOpNot) {
					Ropen = Ropen.Append(pss[0])
				}
				for k := 1; k < len(pss); k++ {
					inside := zp[j+k-1].Into
					if !zp[j+k-1].Parallel && (op == pathOpOr || inside && op == pathOpAnd || !inside && (op == pathOpXor || op == pathOpNot)) {
						Ropen = Ropen.Append(pss[k])
					}
				}
			}
			ps = append(ps[:i], ps[i+1:]...)
			zp = append(zp[:j], zp[j+n:]...)
			zq = append(zq[:j], zq[j+n:]...)
			shift += length
			i--
		}
	}

	// handle intersecting subpaths
	zs := pathIntersectionNodes(p, q, zp, zq)
	R := booleanIntersections(op, zs)

	// handle the remaining subpaths that are non-intersecting but possibly overlapping, either one containing the other or by being equal
	pIndex, qIndex := newSubpathIndexerSubpaths(ps), newSubpathIndexerSubpaths(qs)
	pHandled, qHandled := make([]bool, len(ps)), make([]bool, len(qs))
	for i := range zp {
		pHandled[pIndex.get(zp[i].Seg)] = true
		qHandled[qIndex.get(zq[i].Seg)] = true
	}

	// equal paths
	for i, pi := range ps {
		if !pHandled[i] {
			for j, qi := range qs {
				if !qHandled[j] {
					if pi.Same(qi) {
						if op == pathOpAnd || op == pathOpOr {
							R = R.Append(pi)
						}
						pHandled[i] = true
						qHandled[j] = true
					}
				}
			}
		}
	}

	// contained paths
	for i, pi := range ps {
		if !pHandled[i] && pi.inside(q) {
			if op == pathOpAnd || op == pathOpDivide {
				R = R.Append(pi)
			} else if op == pathOpXor {
				R = R.Append(pi.Reverse())
			}
			pHandled[i] = true
		}
	}
	// non-overlapping paths
	if op != pathOpAnd {
		for i, pi := range ps {
			if !pHandled[i] {
				R = R.Append(pi)
			}
		}
	}

	// contained paths
	for i, qi := range qs {
		if !qHandled[i] && qi.inside(p) {
			if op == pathOpAnd || op == pathOpDivide {
				R = R.Append(qi)
			} else if op == pathOpXor || op == pathOpNot {
				R = R.Append(qi.Reverse())
			}
			qHandled[i] = true
		}
	}
	// non-overlapping paths
	if op == pathOpOr || op == pathOpXor {
		for i, qi := range qs {
			if !qHandled[i] {
				R = R.Append(qi)
			}
		}
	}
	return R.Append(Ropen) // add the open paths
}

func booleanIntersections(op pathOp, zs []PathIntersectionNode) *Path {
	K := 1 // number of time to run from each intersection
	startInwards := []bool{false, false}
	invertP := []bool{false, false}
	invertQ := []bool{false, false}
	if op == pathOpAnd {
		startInwards[0] = true
		invertP[0] = true
	} else if op == pathOpOr {
		invertQ[0] = true
	} else if op == pathOpXor {
		// run as (p NOT q) and then as (q NOT p)
		K = 2
		invertP[1] = true
		invertQ[1] = true
	} else if op == pathOpDivide {
		// run as (p NOT q) and then as (p AND q)
		K = 2
		startInwards[1] = true
		invertP[1] = true
	}

	R := &Path{}
	visited := make([][2]bool, len(zs)) // per direction
	for _, z0 := range zs {
		for k := 0; k < K; k++ {
			if visited[z0.i][k] {
				continue
			}

			r := &Path{}
			var forwardP, forwardQ bool
			onP := startInwards[k] == z0.PintoQ // ensure result is CCW
			if onP {
				forwardP = invertP[k] == z0.PintoQ
			} else {
				forwardQ = invertQ[k] == z0.PintoQ
			}

			// don't start on parallel tangent intersection (ie. not crossing)
			parallelTangent := z0.ParallelTangent(onP, forwardP, forwardQ)
			if parallelTangent {
				continue
			}

			for z := &z0; ; {
				visited[z.i][k] = true
				if z.i != z0.i && z.x != nil && (forwardP == forwardQ) != z.ParallelReversed {
					// parallel lines for crossing intersections
					// only show when not changing forwardness, or when parallel in reverse order
					if forwardP {
						r = r.Join(z.x)
					} else {
						r = r.Join(z.x.Reverse())
					}
				}

				if onP {
					if forwardP {
						r = r.Join(z.p)
						z = z.nextP
					} else {
						r = r.Join(z.prevP.p.Reverse())
						z = z.prevP
					}
				} else {
					if forwardQ {
						r = r.Join(z.q)
						z = z.nextQ
					} else {
						r = r.Join(z.prevQ.q.Reverse())
						z = z.prevQ
					}
				}

				if z.i == z0.i {
					break
				}
				onP = !onP
				if parallelTangent {
					// no-op
				} else if onP {
					forwardP = invertP[k] == z.PintoQ
				} else {
					forwardQ = invertQ[k] == z.PintoQ
				}
				parallelTangent = z.ParallelTangent(onP, forwardP, forwardQ)
			}
			r.Close()
			r.optimizeClose()
			R = R.Append(r)
		}
	}
	return R
}

// Cut cuts path p by path q and returns the parts.
func (p *Path) Cut(q *Path) []*Path {
	zs, _ := p.Intersections(q)
	pi, _ := cut(p, zs)
	return pi
}

// Intersects returns true if path p and path q intersect.
func (p *Path) Intersects(q *Path) bool {
	zs, _ := p.Intersections(q)
	return 0 < len(zs)
}

// Intersections for path p by path q, sorted for path p.
func (p *Path) Intersections(q *Path) ([]PathIntersection, []PathIntersection) {
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	return pathIntersections(p, q, false, false)
}

// Touches returns true if path p and path q touch or intersect.
func (p *Path) Touches(q *Path) bool {
	zs, _ := p.Collisions(q)
	return 0 < len(zs)
}

// Collisions (secants/intersections and tangents/touches) for path p by path q, sorted for path p.
func (p *Path) Collisions(q *Path) ([]PathIntersection, []PathIntersection) {
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	return pathIntersections(p, q, true, false)
}

// SelfIntersects returns true if path p self-intersect.
//func (p *Path) SelfIntersects() bool {
//	return 0 < len(p.SelfIntersections())
//}
//
//// SelfIntersections for path p.
//func (p *Path) SelfIntersections() []PathIntersection {
//	if !p.Flat() {
//		p = p.Flatten(Tolerance)
//	}
//	return selfCollisions(p)
//}

// RayIntersections returns the intersections of a path with a ray starting at (x,y) to (∞,y).
// An intersection is tangent only when it is at (x,y), i.e. the start of the ray.
// Intersections are sorted along the ray.
func (p *Path) RayIntersections(x, y float64) []PathIntersection {
	seg, k0 := 0, 0
	var start, end Point
	var zs []Intersection
	Zs := []PathIntersection{}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		zs = zs[:0]
		switch cmd {
		case MoveToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			k0 = len(Zs)
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
		for _, z := range zs {
			Z := PathIntersection{
				Point:    z.Point,
				Seg:      seg,
				T:        z.T[1],
				Dir:      z.Dir[1],
				Tangent:  Equal(z.T[0], 0.0),
				Parallel: z.Aligned() || z.AntiAligned(),
				Into:     z.Into(),
			}

			if cmd == CloseCmd && Equal(z.T[1], 1.0) {
				// sort at subpath's end as first
				Zs = append(Zs[:k0], append([]PathIntersection{Z}, Zs[k0:]...)...)
			} else {
				Zs = append(Zs, Z)
			}
		}
		i += cmdLen(cmd)
		start = end
		seg++
	}
	sort.SliceStable(Zs, func(i, j int) bool {
		return Zs[i].X < Zs[j].X
	})
	return Zs
}
