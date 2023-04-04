package canvas

import (
	"fmt"
	"math"
	"sort"
)

// Paths are cut at the intersections between P and Q. The intersections are put into a doubly linked list with paths going forward and backward over P and Q. Depending on the boolean operation we should choose the right cut. Note that there can be circular loops when choosing cuts based on a condition, so we should take care to visit all intersections. Additionally, if path P or path Q contain subpaths with a different winding, we will first combine the subpaths so to remove all subpath intersections.

func segmentPos(start Point, d []float64, t float64) Point {
	if d[0] == LineToCmd || d[0] == CloseCmd {
		return start.Interpolate(Point{d[1], d[2]}, t)
	} else if d[0] == QuadToCmd {
		cp := Point{d[1], d[2]}
		end := Point{d[3], d[4]}
		return quadraticBezierPos(start, cp, end, t)
	} else if d[0] == CubeToCmd {
		cp1 := Point{d[1], d[2]}
		cp2 := Point{d[3], d[4]}
		end := Point{d[5], d[6]}
		return cubicBezierPos(start, cp1, cp2, end, t)
	} else if d[0] == ArcToCmd {
		rx, ry, phi := d[1], d[2], d[3]
		large, sweep := toArcFlags(d[4])
		cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, d[5], d[6])
		return EllipsePos(rx, ry, phi, cx, cy, theta0+t*(theta1-theta0))
	}
	return Point{}
}

// returns true if p is inside q or equivalent to q, paths may not intersect
// p should not have subpaths
func (p *Path) inside(q *Path) bool {
	// if p does not fill with the EvenOdd rule, it is inside q
	p = p.Append(q)
	return !p.Filling(EvenOdd)[0]
}

// ContainsPath returns true if path q is contained within path p, i.e. path q is inside path p and both paths have no intersections (but may touch). Paths must have been settled to remove self-intersections.
func (p *Path) ContainsPath(q *Path) bool {
	ps, qs := p.Split(), q.Split()
	for _, qi := range qs {
		inside := false
		for _, pi := range ps {
			if qi.inside(pi) && len(collisions([]*Path{pi}, []*Path{qi}, false)) == 0 {
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

// Settle combines the path p with itself, including all subpaths, removing all self-intersections and overlapping parts. It returns subpaths with counter clockwise directions.
func (p *Path) Settle() *Path {
	// TODO: remove self-intersections too
	if p.Empty() {
		return p
	}

	ps := p.Split()
	p = ps[0]
	for _, q := range ps[1:] {
		p = boolean(p, pathOpSettle, q)
	}

	// make all filling paths go CCW
	r := &Path{}
	ps = p.Split()
	for i := range ps {
		if ps[i].Empty() || !ps[i].Closed() {
			r = r.Append(ps[i])
			continue
		}
		if ps[i].CCW() == ps[i].inside(p) {
			r = r.Append(ps[i])
		} else {
			r = r.Append(ps[i].Reverse())
		}
	}
	return r
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
	pathOpSettle
)

type subpathIndexer []int // index from segment to subpath

func newSubpathIndexer(ps []*Path) subpathIndexer {
	idx := make(subpathIndexer, len(ps)+1)
	idx[0] = 0
	for i, pi := range ps {
		idx[i+1] = idx[i] + pi.Len()
	}
	return idx
}

func (idx subpathIndexer) get(seg int) int {
	for i, n := range idx[1:] {
		if seg < n {
			return i
		}
	}
	panic("bug: segment doesn't exist on path")
}

// path p can be open or closed paths (we handle them separately), path q is closed implicitly
func boolean(p *Path, op pathOp, q *Path) *Path {
	if op != pathOpSettle {
		// remove self-intersections within each path and direct them all CCW
		p = p.Settle()
		q = q.Settle()
	}

	// return in case of one path is empty
	if q.Empty() {
		if op != pathOpAnd {
			return p
		}
		return &Path{}
	}
	if p.Empty() {
		if op == pathOpOr || op == pathOpXor || op == pathOpSettle {
			return q
		}
		return &Path{}
	}

	// we can only handle line-line, line-quad, line-cube, and line-arc intersections
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	ccwA, ccwB := true, true // by default true after Settle, except when operation is Settle
	ps, qs := p.Split(), q.Split()
	if op == pathOpSettle {
		// implicitly close all subpaths of path q
		// given the if above, this will close q for all boolean operations (once)
		for i := range ps {
			if ps[i].Closed() {
				ccwA = ps[i].CCW()
				break
			}
		}
		for i := range qs {
			if !qs[i].Closed() {
				qs[i].Close()
			}
			if i == 0 {
				ccwB = qs[i].CCW()
			}
		}
	}

	// find all intersections (non-tangent) between p and q
	Zs := collisions(ps, qs, false)

	// handle open subpaths on path p and remove from Zs
	Ropen := &Path{}
	p, q = &Path{}, &Path{}
	for i := range qs {
		q = q.Append(qs[i])
	}
	j, segOffsetA, d := 0, 0, 0 // j is index into Zs, d is number of removed segments
	for i := 0; i < len(ps); i++ {
		lenA := ps[i].Len()
		closed := ps[i].Closed()
		n := 0
		for ; j+n < len(Zs) && Zs[j+n].SegA < segOffsetA+lenA; n++ {
			if closed {
				Zs[j+n].SegA -= d
			} else {
				Zs[j+n].SegA -= segOffsetA
			}
		}
		segOffsetA += lenA

		if !closed {
			zs := Zs[j : j+n]
			if len(zs) == 0 {
				p0 := ps[i].StartPos()
				// determine if path is filling by checking the number of windings at the starting point of the subpath (considered to be on the exterior of the subpath)
				n, boundary := q.Windings(p0.X, p0.Y)
				inside := n != 0 // FillRule: NonZero
				for k := 4; k < len(ps[i].d) && boundary; {
					p0 = segmentPos(Point{ps[i].d[k-3], ps[i].d[k-2]}, ps[i].d[k:], 0.5) // TODO: is this correct?
					n, boundary = q.Windings(p0.X, p0.Y)
					inside = n != 0 // FillRule: NonZero
					k += cmdLen(ps[i].d[k])
				}
				if op == pathOpOr || op == pathOpSettle || inside && op == pathOpAnd || !inside && !boundary && (op == pathOpXor || op == pathOpNot) {
					Ropen = Ropen.Append(ps[i])
				}
			} else {
				pss := cut(zs, ps[i])
				inside := zs[0].Kind == BintoA
				if op == pathOpOr || op == pathOpSettle || inside && op == pathOpAnd || !inside && (op == pathOpXor || op == pathOpNot) {
					Ropen = Ropen.Append(pss[0])
				}
				for k := 1; k < len(pss); k++ {
					if zs[k-1].Parallel != Parallel && zs[k-1].Parallel != AParallel {
						inside := zs[k-1].Kind == AintoB
						if op == pathOpOr || op == pathOpSettle || inside && op == pathOpAnd || !inside && (op == pathOpXor || op == pathOpNot) {
							Ropen = Ropen.Append(pss[k])
						}
					}
				}
			}
			Zs = append(Zs[:j], Zs[j+n:]...)
			ps = append(ps[:i], ps[i+1:]...)
			d += lenA
			i--
		} else {
			p = p.Append(ps[i])
		}
		j += n
	}

	K := 1 // number of time to run from each intersection
	startInwards := []bool{false, false}
	invertA := []bool{false, false}
	invertB := []bool{false, false}
	if op == pathOpAnd {
		startInwards[0], invertA[0] = true, true
	} else if op == pathOpOr || op == pathOpSettle && ccwA == ccwB {
		invertB[0] = true
	} else if op == pathOpXor || op == pathOpSettle && ccwA != ccwB {
		K = 2
		invertA[1] = true
		invertB[1] = true
	} else if op == pathOpNot {
	} else if op == pathOpDivide {
		// run as NOT and then as AND
		K = 2
		startInwards[1] = true
		invertA[1] = true
	}

	R := &Path{}
	zs := intersectionNodes(Zs, p, q)
	visited := map[int]map[int]bool{} // per direction
	for k := 0; k < K; k++ {
		visited[k] = map[int]bool{}
	}
	for _, z0 := range zs {
		for k := 0; k < K; k++ {
			if visited[k][z0.i] {
				continue
			}
			r := &Path{}
			var forwardA, forwardB bool
			gotoB := startInwards[k] == (ccwB == (z0.kind == BintoA)) // ensure result is CCW
			if gotoB {
				forwardB = invertB[k] != (ccwA == (z0.kind == BintoA))
			} else {
				forwardA = invertA[k] != (ccwB == (z0.kind == BintoA))
			}
			// parallel lines for touching intersections
			tangentStart := z0.tangentStart(gotoB, forwardA, forwardB)
			if tangentStart {
				continue
			}
			for z := z0; ; {
				visited[k][z.i] = true
				if z.i != z0.i && (forwardA == forwardB) == (z.parallel == Parallel) {
					// parallel lines for crossing intersections
					if forwardA {
						r = r.Join(z.c)
					} else {
						r = r.Join(z.c.Reverse())
					}
				}
				if gotoB {
					if forwardB {
						r = r.Join(z.b)
						z = z.nextB
					} else {
						r = r.Join(z.prevB.b.Reverse())
						z = z.prevB
					}
				} else {
					if forwardA {
						r = r.Join(z.a)
						z = z.nextA
					} else {
						r = r.Join(z.prevA.a.Reverse())
						z = z.prevA
					}
				}
				gotoB = !gotoB
				if z.i == z0.i {
					break
				} else if !tangentStart {
					if gotoB {
						forwardB = invertB[k] != (ccwA == (z.kind == BintoA))
					} else {
						forwardA = invertA[k] != (ccwB == (z.kind == BintoA))
					}
				}
				tangentStart = z.tangentStart(gotoB, forwardA, forwardB)
			}
			r.Close()
			R = R.Append(r)
		}
	}

	// handle the remaining subpaths that are non-intersecting but possibly overlapping, either one containing the other or by being equal
	pIndex, qIndex := newSubpathIndexer(ps), newSubpathIndexer(qs)
	pHandled, qHandled := make([]bool, len(ps)), make([]bool, len(qs))
	for _, z := range Zs {
		pHandled[pIndex.get(z.SegA)] = true
		qHandled[qIndex.get(z.SegB)] = true
	}

	// equal polygons
	for i, pi := range ps {
		if !pHandled[i] {
			for j, qi := range qs {
				if !qHandled[j] {
					if pi.Same(qi) {
						if op == pathOpAnd || op == pathOpOr || op == pathOpSettle {
							R = R.Append(pi)
						}
						pHandled[i] = true
						qHandled[j] = true
					} else if pi.inside(qi) && qi.inside(pi) {
						// happens when each coordinates are on each other's boundaries
						// TODO: check whichone has largest area
						if op == pathOpAnd || op == pathOpOr || op == pathOpSettle {
							R = R.Append(pi)
						}
						pHandled[i] = true
						qHandled[j] = true
					}
				}
			}
		}
	}

	// contained (non-touching) polygons
	for i, pi := range ps {
		if !pHandled[i] && pi.inside(q) {
			if op == pathOpAnd || op == pathOpDivide || op == pathOpSettle && ccwA != ccwB {
				R = R.Append(pi)
			} else if op == pathOpXor {
				R = R.Append(pi.Reverse())
			}
			pHandled[i] = true
		}
	}
	// polygons with no overlap
	if op != pathOpAnd {
		for i, pi := range ps {
			if !pHandled[i] {
				R = R.Append(pi)
			}
		}
	}

	// contained (non-touching) polygons
	for i, qi := range qs {
		if !qHandled[i] && qi.inside(p) {
			if op == pathOpAnd || op == pathOpDivide || op == pathOpSettle && ccwA != ccwB {
				R = R.Append(qi)
			} else if op == pathOpXor || op == pathOpNot {
				R = R.Append(qi.Reverse())
			}
			qHandled[i] = true
		}
	}
	// polygons with no overlap
	if op == pathOpOr || op == pathOpXor || op == pathOpSettle {
		for i, qi := range qs {
			if !qHandled[i] {
				R = R.Append(qi)
			}
		}
	}
	return R.Append(Ropen) // add the open paths
}

// Cut cuts path p by path q and returns the parts.
func (p *Path) Cut(q *Path) []*Path {
	return cut(p.Intersections(q), p)
}

func cut(Zs Intersections, p *Path) []*Path {
	if len(Zs) == 0 {
		return []*Path{p}
	}

	// cut path segments for path P
	j := 0   // index into intersections
	k := 0   // index into ps
	seg := 0 // index into path segments
	ps := []*Path{}
	var first, cur []float64
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		if cmd == MoveToCmd {
			closed := 3 < i && p.d[i-1] == CloseCmd
			if first != nil {
				// there were intersections in the last subpath
				if closed {
					cur = append(cur, first[4:]...) // last subpath was closed
					ps = append(ps, &Path{cur})
					cur = nil
				} else {
					ps = append(ps[:k], append([]*Path{{first}}, ps[k:]...)...)
				}
			} else if closed {
				cur[len(cur)-1] = CloseCmd
				cur[len(cur)-4] = CloseCmd
			}
			first = nil
			k = len(ps)
		} else if cmd == CloseCmd {
			p.d[i], p.d[i+3] = LineToCmd, LineToCmd
		}
		if j < len(Zs) && seg == Zs[j].SegA {
			// segment has an intersection, cut it up and append first part to prev intersection
			p0, p1 := cutPathSegment(Point{p.d[i-3], p.d[i-2]}, p.d[i:i+cmdLen(cmd)], Zs[j].TA)
			if !p0.Empty() {
				cur = append(cur, p0.d[4:]...)
			}

			for j+1 < len(Zs) && seg == Zs[j+1].SegA {
				// next cut is on the same segment, find new t after the first cut and set path
				if first == nil {
					first = cur // take aside the path to the first intersection to later append it
				} else {
					ps = append(ps, &Path{cur})
				}
				j++
				t := (Zs[j].TA - Zs[j-1].TA) / (1.0 - Zs[j-1].TA)
				if !p1.Empty() {
					p0, p1 = cutPathSegment(Point{p1.d[1], p1.d[2]}, p1.d[4:], t)
				} else {
					p0 = p1
				}
				cur = p0.d
			}
			if first == nil {
				first = cur // take aside the path to the first intersection to later append it
			} else {
				ps = append(ps, &Path{cur})
			}
			cur = p1.d
			j++
		} else {
			// segment has no intersection
			if len(cur) == 0 || cmd != CloseCmd || p.d[i+1] != cur[len(cur)-3] || p.d[i+2] != cur[len(cur)-2] {
				cur = append(cur, p.d[i:i+cmdLen(cmd)]...)
			}
		}
		if cmd == CloseCmd {
			p.d[i], p.d[i+3] = CloseCmd, CloseCmd // keep the original unchanged
		}
		i += cmdLen(cmd)
		seg++
	}
	closed := 3 < len(p.d) && p.d[len(p.d)-1] == CloseCmd
	if first != nil {
		// there were intersections in the last subpath
		if closed {
			cur = append(cur, first[4:]...) // last subpath was closed
		} else {
			ps = append(ps[:k], append([]*Path{{first}}, ps[k:]...)...)
		}
	} else if closed {
		cur[len(cur)-1] = CloseCmd
		cur[len(cur)-4] = CloseCmd
	}
	ps = append(ps, &Path{cur})
	return ps
}

type intersectionNode struct {
	i            int // intersection index in path A
	prevA, nextA *intersectionNode
	prevB, nextB *intersectionNode

	kind     intersectionKind
	parallel intersectionParallel
	tangent  bool
	a, b     *Path // towards next intersection
	c        *Path // common (parallel) along A
}

func (z *intersectionNode) String() string {
	tangent := ""
	if z.tangent {
		tangent = " Tangent"
	}
	return fmt.Sprintf("(%v A=[%v→,→%v] B=[%v→,→%v] %v %v%v)", z.i, z.prevA.i, z.nextA.i, z.prevB.i, z.nextB.i, z.kind, z.parallel, tangent)
}

func (z *intersectionNode) tangentStart(gotoB, forwardA, forwardB bool) bool {
	return z.tangent && (gotoB &&
		(forwardB && (z.parallel == Parallel || z.parallel == BParallel) ||
			!forwardB && (z.prevB.parallel == Parallel || z.prevB.parallel == BParallel)) ||
		!gotoB && (forwardA && (z.parallel == Parallel || z.parallel == AParallel) ||
			!forwardA && (z.prevA.parallel == Parallel || z.prevA.parallel == AParallel)))
}

// get intersections for paths p and q sorted for both, both paths must be closed
func intersectionNodes(Zs Intersections, p, q *Path) []*intersectionNode {
	if len(Zs) == 0 {
		return nil
	} else if len(Zs)%2 != 0 {
		panic("bug: number of intersections must be even")
	}

	zs := make([]*intersectionNode, len(Zs))
	for i, z := range Zs {
		zs[i] = &intersectionNode{
			i:    i,
			kind: z.Kind,
			a:    &Path{},
			b:    &Path{},
			c:    &Path{},
		}
	}

	// cut path segments for path P
	seg := 0      // index into path segments
	j, j0 := 0, 0 // index into intersections
	var first, cur []float64
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		if cmd == MoveToCmd {
			if first != nil {
				// there were intersections in the last subpath
				zs[j-1].a.d = append(cur, first[4:]...)
				zs[j-1].nextA = zs[j0]
				zs[j0].prevA = zs[j-1]
			}
			first, cur = nil, nil
			j0 = j
		} else if cmd == CloseCmd {
			p.d[i], p.d[i+3] = LineToCmd, LineToCmd
		}
		if j < len(Zs) && seg == Zs[j].SegA {
			// segment has an intersection, cut it up and append first part to prev intersection
			p0, p1 := cutPathSegment(Point{p.d[i-3], p.d[i-2]}, p.d[i:i+cmdLen(cmd)], Zs[j].TA)
			if !p0.Empty() {
				cur = append(cur, p0.d[4:]...)
			}

			for j+1 < len(Zs) && seg == Zs[j+1].SegA {
				// next cut is on the same segment, find new t after the first cut and set path
				if first == nil {
					first = cur // take aside the path to the first intersection to later append it
				} else {
					zs[j-1].a.d = cur
					zs[j-1].nextA = zs[j]
					zs[j].prevA = zs[j-1]
				}
				j++
				t := (Zs[j].TA - Zs[j-1].TA) / (1.0 - Zs[j-1].TA)
				if !p1.Empty() {
					p0, p1 = cutPathSegment(Point{p1.d[1], p1.d[2]}, p1.d[4:], t)
				} else {
					p0 = p1
				}
				cur = p0.d
			}
			if first == nil {
				first = cur // take aside the path to the first intersection to later append it
			} else {
				zs[j-1].a.d = cur
				zs[j-1].nextA = zs[j]
				zs[j].prevA = zs[j-1]
			}
			cur = p1.d
			j++
		} else {
			// segment has no intersection
			if len(cur) == 0 || cmd != CloseCmd || p.d[i+1] != cur[len(cur)-3] || p.d[i+2] != cur[len(cur)-2] {
				cur = append(cur, p.d[i:i+cmdLen(cmd)]...)
			}
		}
		if cmd == CloseCmd {
			p.d[i], p.d[i+3] = CloseCmd, CloseCmd // keep the original unchanged
		}
		i += cmdLen(cmd)
		seg++
	}
	if first != nil {
		zs[len(zs)-1].a.d = append(cur, first[4:]...)
		zs[len(zs)-1].nextA = zs[j0]
		zs[j0].prevA = zs[len(zs)-1]
	}

	// build index map for intersections on Q to P (zs is sorted for P)
	idxs := Zs.ArgBSort() // sorted indices for intersections of q by p

	// cut path segments for path Q
	seg = 0      // index into path segments
	j, j0 = 0, 0 // index into intersections
	first, cur = nil, nil
	for i := 0; i < len(q.d); {
		cmd := q.d[i]
		if cmd == MoveToCmd {
			if first != nil {
				// there were intersections in the last subpath
				zs[idxs[j-1]].b.d = append(cur, first[4:]...)
				zs[idxs[j-1]].nextB = zs[idxs[j0]]
				zs[idxs[j0]].prevB = zs[idxs[j-1]]
			}
			first, cur = nil, nil
			j0 = j
		} else if cmd == CloseCmd {
			q.d[i], q.d[i+3] = LineToCmd, LineToCmd
		}
		if j < len(Zs) && seg == Zs[idxs[j]].SegB {
			// segment has an intersection, cut it up and append first part to prev intersection
			p0, p1 := cutPathSegment(Point{q.d[i-3], q.d[i-2]}, q.d[i:i+cmdLen(cmd)], Zs[idxs[j]].TB)
			if !p0.Empty() {
				cur = append(cur, p0.d[4:]...)
			}

			for j+1 < len(Zs) && seg == Zs[idxs[j+1]].SegB {
				// next cut is on the same segment, find new t after the first cut and set path
				if first == nil {
					first = cur // take aside the path to the first intersection to later append it
				} else {
					zs[idxs[j-1]].b.d = cur
					zs[idxs[j-1]].nextB = zs[idxs[j]]
					zs[idxs[j]].prevB = zs[idxs[j-1]]
				}
				j++
				t := (Zs[idxs[j]].TB - Zs[idxs[j-1]].TB) / (1.0 - Zs[idxs[j-1]].TB)
				if !p1.Empty() {
					p0, p1 = cutPathSegment(Point{p1.d[1], p1.d[2]}, p1.d[4:], t)
				} else {
					p0 = p1
				}
				cur = p0.d
			}
			if first == nil {
				first = cur // take aside the path to the first intersection to later append it
			} else {
				zs[idxs[j-1]].b.d = cur
				zs[idxs[j-1]].nextB = zs[idxs[j]]
				zs[idxs[j]].prevB = zs[idxs[j-1]]
			}
			cur = p1.d
			j++
		} else {
			// segment has no intersection
			if len(cur) == 0 || cmd != CloseCmd || q.d[i+1] != cur[len(cur)-3] || q.d[i+2] != cur[len(cur)-2] {
				cur = append(cur, q.d[i:i+cmdLen(cmd)]...)
			}
		}
		if cmd == CloseCmd {
			q.d[i], q.d[i+3] = CloseCmd, CloseCmd // keep the original unchanged
		}
		i += cmdLen(cmd)
		seg++
	}
	if first != nil {
		zs[idxs[len(zs)-1]].b.d = append(cur, first[4:]...)
		zs[idxs[len(zs)-1]].nextB = zs[idxs[j0]]
		zs[idxs[j0]].prevB = zs[idxs[len(zs)-1]]
	}

	// collapse nodes for parallel lines, except when tangent
	for j := len(Zs) - 1; 0 <= j; j-- {
		if Zs[j].Tangent {
			zs[j].parallel = Zs[j].Parallel
			zs[j].tangent = true
		} else if Zs[j].Parallel == Parallel || Zs[j].Parallel == AParallel {
			// remove node at j and join with next
			zs[j].nextA.c = zs[j].a
			if Zs[j].Parallel == AParallel {
				zs[j].prevB.b = zs[j].b
			}
			zs[j].nextA.parallel = Zs[j].Parallel

			zs[j].prevA.nextA = zs[j].nextA
			zs[j].nextA.prevA = zs[j].prevA
			zs[j].prevB.nextB = zs[j].nextB
			zs[j].nextB.prevB = zs[j].prevB
			zs = append(zs[:j], zs[j+1:]...)
		}
	}
	return zs
}

func cutPathSegment(start Point, d []float64, t float64) (*Path, *Path) {
	p0, p1 := &Path{}, &Path{}
	if Equal(t, 0.0) {
		p0.MoveTo(start.X, start.Y)
		p1.MoveTo(start.X, start.Y)
		p1.d = append(p1.d, d...)
		return p0, p1
	} else if Equal(t, 1.0) {
		p0.MoveTo(start.X, start.Y)
		p0.d = append(p0.d, d...)
		p1.MoveTo(d[len(d)-3], d[len(d)-2])
		return p0, p1
	}
	if d[0] == LineToCmd {
		c := start.Interpolate(Point{d[len(d)-3], d[len(d)-2]}, t)
		p0.MoveTo(start.X, start.Y)
		p0.LineTo(c.X, c.Y)
		p1.MoveTo(c.X, c.Y)
		p1.LineTo(d[len(d)-3], d[len(d)-2])
	} else if d[0] == QuadToCmd {
		r0, r1, r2, q0, q1, q2 := quadraticBezierSplit(start, Point{d[1], d[2]}, Point{d[3], d[4]}, t)
		p0.MoveTo(r0.X, r0.Y)
		p0.QuadTo(r1.X, r1.Y, r2.X, r2.Y)
		p1.MoveTo(q0.X, q0.Y)
		p1.QuadTo(q1.X, q1.Y, q2.X, q2.Y)
	} else if d[0] == CubeToCmd {
		r0, r1, r2, r3, q0, q1, q2, q3 := cubicBezierSplit(start, Point{d[1], d[2]}, Point{d[3], d[4]}, Point{d[5], d[6]}, t)
		p0.MoveTo(r0.X, r0.Y)
		p0.CubeTo(r1.X, r1.Y, r2.X, r2.Y, r3.X, r3.Y)
		p1.MoveTo(q0.X, q0.Y)
		p1.CubeTo(q1.X, q1.Y, q2.X, q2.Y, q3.X, q3.Y)
	} else if d[0] == ArcToCmd {
		large, sweep := toArcFlags(d[4])
		cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, d[1], d[2], d[3], large, sweep, d[5], d[6])
		theta := theta0 + (theta1-theta0)*t
		c, large0, large1, ok := ellipseSplit(d[1], d[2], d[3], cx, cy, theta0, theta1, theta)
		if !ok {
			// should never happen
			panic("theta not in elliptic arc range for splitting")
		}
		p0.MoveTo(start.X, start.Y)
		p0.ArcTo(d[1], d[2], d[3]*180.0/math.Pi, large0, sweep, c.X, c.Y)
		p1.MoveTo(c.X, c.Y)
		p1.ArcTo(d[1], d[2], d[3]*180.0/math.Pi, large1, sweep, d[len(d)-3], d[len(d)-2])
	}
	return p0, p1
}

// Intersects returns true if path p and path q intersect.
func (p *Path) Intersects(q *Path) bool {
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	zs := collisions(p.Split(), q.Split(), false)
	return 0 < len(zs)
}

// Intersections for path p by path q, sorted for path p.
func (p *Path) Intersections(q *Path) Intersections {
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	return collisions(p.Split(), q.Split(), false)
}

// Touches returns true if path p and path q touch or intersect.
func (p *Path) Touches(q *Path) bool {
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	zs := collisions(p.Split(), q.Split(), true)
	return 0 < len(zs)
}

// Collisions (secants/intersections and tangents/touches) for path p by path q, sorted for path p.
func (p *Path) Collisions(q *Path) Intersections {
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}
	return collisions(p.Split(), q.Split(), true)
}

func collisions(ps, qs []*Path, keepTangents bool) Intersections {
	zs := Intersections{}
	segOffsetA := 0
	for _, p := range ps {
		closedA, lenA := p.Closed(), p.Len()
		segOffsetB := 0
		for _, q := range qs {
			closedB, lenB := q.Closed(), q.Len()

			// TODO: uses O(N^2), try sweep line or bently-ottman to reduce to O((N+K) log N)
			Zs := Intersections{}
			segA := segOffsetA + 1
			for i := 4; i < len(p.d); {
				pn := cmdLen(p.d[i])
				segB := segOffsetB + 1
				for j := 4; j < len(q.d); {
					qn := cmdLen(q.d[j])
					p0, q0 := Point{p.d[i-3], p.d[i-2]}, Point{q.d[j-3], q.d[j-2]}
					Zs = Zs.appendSegment(segA, p0, p.d[i:i+pn], segB, q0, q.d[j:j+qn])
					j += qn
					segB++
				}
				i += pn
				segA++
			}
			if len(Zs) == 0 {
				segOffsetB += lenB
				continue
			}

			// sort by position on P and secondary on Q
			// wrap intersections at the very end of the path towards the beginning, note that we must ignore a final but zero distance close command
			pointCloseA, pointCloseB := 0, 0
			if closedA && 6 < len(p.d) && Equal(p.d[len(p.d)-7], p.d[len(p.d)-3]) && Equal(p.d[len(p.d)-6], p.d[len(p.d)-2]) {
				pointCloseA = 1
			}
			if closedB && 6 < len(q.d) && Equal(q.d[len(q.d)-7], q.d[len(q.d)-3]) && Equal(q.d[len(q.d)-6], q.d[len(q.d)-2]) {
				pointCloseB = 1
			}
			Zs.sortAndWrapEnd(segOffsetA, segOffsetB, lenA-pointCloseA, lenB-pointCloseB)

			// remove duplicate tangent collisions at segment endpoints: either 4 degenerate collisions
			// when for both path p and path q the endpoints coincide, or 2 degenerate collisions when
			// an endpoint collides within a segment, for each parallel segment in between an additional 2 degenerate collisions are created
			// note that collisions between segments of the same path are never generated
			for i := 0; i < len(Zs); i++ {
				z := Zs[i]
				if !z.Tangent {
					// regular intersection
					zs = append(zs, z)
				} else if !Equal(z.TA, 0.0) && !Equal(z.TB, 0.0) && !Equal(z.TA, 1.0) && !Equal(z.TB, 1.0) {
					// regular tangent that is not at segment end point, does not intersect
					if keepTangents {
						zs = append(zs, z)
					}
				} else if !closedA && (z.SegA == segOffsetA+1 && Equal(z.TA, 0.0) || z.SegA == segOffsetA+lenA-1 && Equal(z.TA, 1.0)) || !closedB && (z.SegB == segOffsetB+1 && Equal(z.TB, 0.0) || z.SegB == segOffsetB+lenB-1 && Equal(z.TB, 1.0)) {
					// tangent at start/end of path p or path q, not intersecting as paths are open
					if keepTangents {
						zs = append(zs, z)
					}
				} else {
					i0 := i
					var parallel, reverse bool // reverse is set when parallel and in reverse order
				Next:
					// tangent at segment end point: we either have a regular (mid-mid),
					// 2-degenerate (mid-end), or 4-degenerate (end-end) intersection
					m := 1
					zi := Zs[i%len(Zs)] // incoming intersection
					if Equal(zi.TA, 1.0) {
						m *= 2
					}
					if Equal(zi.TB, 0.0) || Equal(zi.TB, 1.0) {
						m *= 2
					}
					zo := Zs[(i+m-1)%len(Zs)] // outgoing intersection

					// skip if incoming is parallel since we're in the middle of a series of parallel segmentes, and we need to be at the start
					if !parallel && (angleEqual(zi.DirA, zi.DirB) || angleEqual(zi.DirA, zo.DirB+math.Pi)) {
						// when the whole path is equal (i.e. all parallels), this will skip all
						i += m - 1
						continue
					}
					i += m

					// ends in parallel segment, follow until we reach a non-parallel segment
					if !reverse && angleEqual(zo.DirA, zo.DirB) {
						// parallel
						parallel = true
						goto Next
					} else if (!parallel || reverse) && angleEqual(zo.DirA, zi.DirB+math.Pi) {
						// reverse and parallel
						reverse = true
						parallel = true
						goto Next
					}

					// choose both angles of A of the first and second intersection
					i1, i2, i3 := i0+1, (i-2)%len(Zs), (i-1)%len(Zs)
					if Equal(Zs[i1].TA, 1.0) {
						i1 += 2 // first intersection at endpoint of A, select the outgoing angle
					}
					if Equal(Zs[i1].TB, 1.0) {
						i1-- // prefer TA=TB=0 to append to intersections
					}
					if Equal(Zs[i2].TA, 0.0) {
						i2 -= 2 // second, intersection at endpoint of A, select incoming angle
					}
					z0, z1, z2, z3 := Zs[i0], Zs[i1], Zs[i2], Zs[i3]
					// first intersection is LHS of A when between (theta0,theta1)
					// second intersection is LHS of A when between (theta2,theta3)
					alpha0 := angleNorm(z1.DirA)
					alpha1 := alpha0 + angleNorm(z0.DirA+math.Pi-alpha0)
					alpha2 := angleNorm(z3.DirA)
					alpha3 := alpha2 + angleNorm(z2.DirA+math.Pi-alpha2)

					// check whether the incoming and outgoing angle of B is (going) LHS of A
					var beta1, beta2 float64
					if !reverse {
						beta0 := angleNorm(z1.DirB)
						beta1 = beta0 + angleNorm(z0.DirB+math.Pi-beta0)
						beta2 = angleNorm(z3.DirB)
					} else {
						beta0 := angleNorm(z0.DirB + math.Pi)
						beta1 = beta0 + angleNorm(z1.DirB-beta0)
						beta2 = angleNorm(z2.DirB + math.Pi)
					}
					bi := angleBetweenExclusive(beta1, alpha0, alpha1)
					bo := angleBetweenExclusive(beta2, alpha2, alpha3)

					if !parallel && bi != bo {
						// no parallels in between, add one intersection
						if bo != reverse {
							z3.Kind = BintoA
						} else {
							z3.Kind = AintoB
						}
						z3.Parallel = NoParallel
						z3.Tangent = false
						zs = append(zs, z3)
					} else if parallel {
						// parallels in between, add an intersection at the start and end
						z0 = Zs[i1] // get intersection at t=0 for B
						if bi != bo {
							if bo != reverse {
								z0.Kind = BintoA
								z3.Kind = BintoA
							} else {
								z0.Kind = AintoB
								z3.Kind = AintoB
							}
							z0.Tangent = false
							z3.Tangent = false
						} else {
							// parallel touches, but we add them as if they intersect
							if bo != reverse {
								z0.Kind = AintoB
								z3.Kind = BintoA
							} else {
								z0.Kind = BintoA
								z3.Kind = AintoB
							}
						}
						if !reverse {
							z0.Parallel = Parallel
							z3.Parallel = NoParallel
						} else {
							z0.Parallel = AParallel
							z3.Parallel = BParallel
						}
						zs = append(zs, z0, z3)
					}
					i--
				}
			}
			segOffsetB += lenB
		}
		segOffsetA += lenA
	}
	zs.ASort()
	return zs
}

// intersections of path with ray starting at (x,y) to (∞,y)
func (p *Path) rayIntersections(x, y float64) Intersections {
	j, seg := 0, 0
	var start, end Point
	zs := Intersections{}
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
				zs = zs.LineLine(Point{x, y}, Point{xmax + 1.0, y}, start, end)
			}
		case QuadToCmd:
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			ymin := math.Min(math.Min(start.Y, end.Y), cp.Y)
			ymax := math.Max(math.Max(start.Y, end.Y), cp.Y)
			xmax := math.Max(math.Max(start.X, end.X), cp.X)
			if Interval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = zs.LineQuad(Point{x, y}, Point{xmax + 1.0, y}, start, cp, end)
			}
		case CubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			ymin := math.Min(math.Min(start.Y, end.Y), math.Min(cp1.Y, cp2.Y))
			ymax := math.Max(math.Max(start.Y, end.Y), math.Max(cp1.Y, cp2.Y))
			xmax := math.Max(math.Max(start.X, end.X), math.Max(cp1.X, cp2.X))
			if Interval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = zs.LineCube(Point{x, y}, Point{xmax + 1.0, y}, start, cp1, cp2, end)
			}
		case ArcToCmd:
			rx, ry, phi := p.d[i+1], p.d[i+2], p.d[i+3]
			large, sweep := toArcFlags(p.d[i+4])
			end = Point{p.d[i+5], p.d[i+6]}
			cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			if Interval(y, cy-math.Max(rx, ry), cy+math.Max(rx, ry)) && x <= cx+math.Max(rx, ry)+Epsilon {
				zs = zs.LineEllipse(Point{x, y}, Point{cx + rx + 1.0, y}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			}
		}
		for j < len(zs) {
			if !Equal(zs[j].TA, 0.0) {
				zs[j].TA = math.NaN()
			}
			zs[j].SegB = seg
			j++
		}
		i += cmdLen(cmd)
		start = end
		seg++
	}
	sort.SliceStable(zs, func(i, j int) bool {
		return zs[i].X < zs[j].X
	})
	return zs
}
