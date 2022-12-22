package canvas

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Paths are cut at the intersections between P and Q. The intersections are put into a doubly linked list with paths going forward and backward over P and Q. Depending on the boolean operation we should choose the right cut. Note that there can be circular loops when choosing cuts based on a condition, so we should take care to visit all intersections. Additionally, if path P or path Q contain subpaths with a different winding, we will first combine the subpaths so to remove all subpath intersections.

// returns true if p is inside q or equivalent to q, paths may not intersect
// p and q should not have subpaths
func (p *Path) inside(q *Path) bool {
	if len(p.d) <= 4 || len(p.d) <= 4+cmdLen(p.d[4]) {
		return false
	}
	offset := p.interiorPoint()
	return q.Interior(offset.X, offset.Y, NonZero)
}

// Contains returns true if path q is contained within path p, i.e. path q is inside path p and both paths have no intersections (but may touch).
func (p *Path) Contains(q *Path) bool {
	// TODO: what about subpaths?
	if q.inside(p) {
		if !p.Flat() {
			q = q.Flatten()
		}
		return len(collisions(p.Split(), q.Split(), false)) == 0
	}
	return false
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
		pos := ps[i].interiorPoint()
		if ps[i].CCW() == p.Interior(pos.X, pos.Y, NonZero) {
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

type indexPath []int // index from segment to subpath

func (indices indexPath) get(seg int) int {
	i := 0
	for j, n := range indices {
		if seg < i+n {
			return j
		}
		i += n
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
		q = q.Flatten()
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
	pIndex, qIndex := make(indexPath, 0, len(ps)), make(indexPath, 0, len(qs))
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
			//zs := Zs[j : j+n]
			// TODO: handle open paths
			Zs = append(Zs[:j], Zs[j+n:]...)
			ps = append(ps[:i], ps[i+1:]...)
			d += lenA
			i--
		} else {
			p = p.Append(ps[i])
			pIndex = append(pIndex, lenA)
		}
		j += n
	}
	for i := range qs {
		q = q.Append(qs[i])
		qIndex = append(qIndex, qs[i].Len())
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
			for z := z0; ; {
				visited[k][z.i] = true
				if gotoB {
					forwardB = invertB[k] != (ccwA == (z.kind == BintoA))
				} else {
					forwardA = invertA[k] != (ccwB == (z.kind == BintoA))
				}
				if z.i != z0.i && (forwardA == forwardB) == (z.parallel == Parallel) {
					// parallel lines
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
				if z.i == z0.i {
					break
				}
				gotoB = !gotoB
			}
			r.Close()
			R = R.Append(r)
		}
	}

	// handle the remaining subpaths that are non-intersecting but possibly overlapping, either con containment or by being equal
	pHandled, qHandled := make([]bool, len(ps)), make([]bool, len(qs))
	for _, z := range Zs {
		pHandled[pIndex.get(z.SegA)] = true
		qHandled[qIndex.get(z.SegB)] = true
	}
	for i, pi := range ps {
		for j, qi := range qs {
			pInQ := pi.inside(qi)
			qInP := qi.inside(pi)
			if !pInQ && !qInP {
				continue
			}
			if pInQ && qInP {
				// equal
				if op == pathOpAnd || op == pathOpOr || op == pathOpDivide || op == pathOpSettle && ccwA == ccwB {
					if !pHandled[i] && !qHandled[j] {
						R = R.Append(pi)
					}
				}
			} else if pInQ {
				// p is inside q
				if op == pathOpAnd || op == pathOpDivide {
					if !pHandled[i] {
						R = R.Append(pi)
					}
				} else if op == pathOpOr || op == pathOpSettle && ccwA == ccwB {
					if !qHandled[j] {
						R = R.Append(qi)
					}
				} else if op == pathOpXor {
					if !qHandled[j] {
						R = R.Append(qi)
					}
					if !pHandled[i] {
						R = R.Append(pi.Reverse())
					}
				} else if op == pathOpSettle && ccwA != ccwB {
					if !pHandled[i] {
						R = R.Append(pi)
					}
					if !qHandled[j] {
						R = R.Append(qi)
					}
				}
			} else {
				// q is inside p
				if op == pathOpAnd {
					if !qHandled[j] {
						R = R.Append(qi)
					}
				} else if op == pathOpOr {
					if !pHandled[i] {
						R = R.Append(pi)
					}
				} else if op == pathOpXor || op == pathOpNot {
					if !pHandled[i] {
						R = R.Append(pi)
					}
					if !qHandled[j] {
						R = R.Append(qi.Reverse())
					}
				} else if op == pathOpSettle && ccwA != ccwB {
					if !pHandled[i] {
						R = R.Append(pi)
					}
					if !qHandled[j] {
						R = R.Append(qi)
					}
				} else if op == pathOpDivide {
					if !pHandled[i] {
						R = R.Append(pi)
					}
					if !qHandled[j] {
						R = R.Append(qi.Reverse())
					}
					if !qHandled[j] {
						R = R.Append(qi)
					}
				}
			}
			pHandled[i] = true
			qHandled[j] = true
		}
	}

	// no overlap
	if op != pathOpAnd {
		for i, pi := range ps {
			if !pHandled[i] {
				R = R.Append(pi)
			}
		}
	}
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

func cut(Zs intersections, p *Path) []*Path {
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
					ps = append(ps[:k], append([]*Path{&Path{first}}, ps[k:]...)...)
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
			ps = append(ps[:k], append([]*Path{&Path{first}}, ps[k:]...)...)
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
	a, b     *Path // towards next intersection
	c        *Path // common (parallel) along A
}

func (z *intersectionNode) String() string {
	return fmt.Sprintf("(%v A=[%v→,→%v] B=[%v→,→%v] %v %v)", z.i, z.prevA.i, z.nextA.i, z.prevB.i, z.nextB.i, z.kind, z.parallel)
}

// get intersections for paths p and q sorted for both, both paths must be closed
func intersectionNodes(Zs intersections, p, q *Path) []*intersectionNode {
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
	idxs := Zs.argBSort() // sorted indices for intersections of q by p

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

	// collapse nodes for parallel lines
	for j := len(Zs) - 1; 0 <= j; j-- {
		if Zs[j].Parallel == Parallel || Zs[j].Parallel == AParallel {
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
		c, large0, large1, ok := ellipseSplit(d[1], d[2], d[3], cx, cy, theta0, theta1, t)
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
		q = q.Flatten()
	}
	zs := collisions(p.Split(), q.Split(), false)
	return 0 < len(zs)
}

// Intersections for path p by path q, sorted for path p.
func (p *Path) Intersections(q *Path) intersections {
	if !p.Flat() {
		q = q.Flatten()
	}
	return collisions(p.Split(), q.Split(), false)
}

// Touches returns true if path p and path q touch or intersect.
func (p *Path) Touches(q *Path) bool {
	if !p.Flat() {
		q = q.Flatten()
	}
	zs := collisions(p.Split(), q.Split(), true)
	return 0 < len(zs)
}

// Collisions (secants/intersections and tangents/touches) for path p by path q, sorted for path p.
func (p *Path) Collisions(q *Path) intersections {
	if !p.Flat() {
		q = q.Flatten()
	}
	return collisions(p.Split(), q.Split(), true)
}

func collisions(ps, qs []*Path, keepTangents bool) intersections {
	zs := intersections{}
	segOffsetA := 0
	for _, p := range ps {
		closedA, lenA := p.Closed(), p.Len()
		segOffsetB := 0
		for _, q := range qs {
			closedB, lenB := q.Closed(), q.Len()

			// TODO: uses O(N^2), try sweep line or bently-ottman to reduce to O((N+K) log N)
			Zs := intersections{}
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
			Zs.SortAndWrapEnd(segOffsetA, segOffsetB, lenA-pointCloseA, lenB-pointCloseB)

			// remove duplicate tangent collisions at segment endpoints: either 4 degenerate collisions
			// when for both path p and path q the endpoints coincide, or 2 degenerate collisions when
			// an endpoint collides within a segment, for each parallel segment in between an additional 2 degenerate collisions are created
			// note that collisions between segments of the same path are never generated
			for i := 0; i < len(Zs); i++ {
				z := Zs[i]
				if z.Kind != Tangent {
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
					if !parallel && (angleNorm(zi.DirA) == angleNorm(zi.DirB) || angleNorm(zi.DirA) == angleNorm(zo.DirB+math.Pi)) {
						// when the whole path is equal (i.e. all parallels), this will skip all
						i += m - 1
						continue
					}
					i += m

					// ends in parallel segment, follow until we reach a non-parallel segment
					if !reverse && angleNorm(zo.DirA) == angleNorm(zo.DirB) {
						// parallel
						parallel = true
						goto Next
					} else if (!parallel || reverse) && angleNorm(zo.DirA) == angleNorm(zi.DirB+math.Pi) {
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
						i2-- // second, intersection at endpoint of A, select incoming angle
					}
					z0, z1, z2, z3 := Zs[i0], Zs[i1], Zs[i2], Zs[i3]
					// first intersection is LHS of A between (theta0,theta1)
					// second intersecton is LHS of A between (theta2,theta3)
					theta0 := angleNorm(z1.DirA)
					theta1 := theta0 + angleNorm(z0.DirA+math.Pi-theta0)
					theta2 := angleNorm(z3.DirA)
					theta3 := theta2 + angleNorm(z2.DirA+math.Pi-theta2)

					// check whether the incoming and outgoing angle of B is (going) LHS of A
					gamma0, gamma1 := Zs[i0].DirB+math.Pi, Zs[i3].DirB
					if reverse {
						gamma0, gamma1 = Zs[i2].DirB+math.Pi, Zs[i1].DirB
						theta0, theta1, theta2, theta3 = theta2, theta3, theta0, theta1
					}
					bi := angleBetweenExclusive(gamma0, theta0, theta1)
					bo := angleBetweenExclusive(gamma1, theta2, theta3)
					if bi != bo {
						// intersection is not tangent
						if !parallel {
							// no parallels in between, add one intersection
							if bo {
								z3.Kind = BintoA
							} else {
								z3.Kind = AintoB
							}
							z3.Parallel = NoParallel
							zs = append(zs, z3)
						} else {
							// parallels in between, add an intersection at the start and end
							z0 = Zs[i1]
							if bo {
								z0.Kind = BintoA
								z3.Kind = BintoA
							} else {
								z0.Kind = AintoB
								z3.Kind = AintoB
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
					}
					i--
				}
			}
			segOffsetB += lenB
		}
		segOffsetA += lenA
	}
	zs.Sort()
	return zs
}

// intersect for path segments a and b, starting at a0 and b0
func (zs intersections) appendSegment(segA int, a0 Point, a []float64, segB int, b0 Point, b []float64) intersections {
	// TODO: add fast check if bounding boxes overlap, below doesn't account for vertical/horizontal lines

	n := len(zs)
	swapCurves := false
	if a[0] == LineToCmd || a[0] == CloseCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineLine(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]})
		} else if b[0] == QuadToCmd {
			zs = zs.LineQuad(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]})
		} else if b[0] == CubeToCmd {
			zs = zs.LineCube(a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]}, Point{b[5], b[6]})
		} else if b[0] == ArcToCmd {
			rx := b[1]
			ry := b[2]
			phi := b[3] * math.Pi / 180.0
			large, sweep := toArcFlags(b[4])
			cx, cy, theta0, theta1 := ellipseToCenter(b0.X, b0.Y, rx, ry, phi, large, sweep, b[5], b[6])
			zs = zs.LineEllipse(a0, Point{a[1], a[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
		}
	} else if a[0] == QuadToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineQuad(b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]})
			swapCurves = true
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for quad-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for quad-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for quad-arc")
		}
	} else if a[0] == CubeToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineCube(b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]}, Point{a[5], a[6]})
			swapCurves = true
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for cube-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for cube-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for cube-arc")
		}
	} else if a[0] == ArcToCmd {
		rx := a[1]
		ry := a[2]
		phi := a[3] * math.Pi / 180.0
		large, sweep := toArcFlags(a[4])
		cx, cy, theta0, theta1 := ellipseToCenter(a0.X, a0.Y, rx, ry, phi, large, sweep, a[5], a[6])
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = zs.LineEllipse(b0, Point{b[1], b[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			swapCurves = true
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for arc-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for arc-cube")
		} else if b[0] == ArcToCmd {
			panic("unsupported intersection for arc-arc")
		}
	} else {
		// MoveCmd
	}

	// swap A and B in the intersection found to match segments A and B of this function
	if swapCurves {
		for i := n; i < len(zs); i++ {
			zs[i].SegA, zs[i].SegB = segA, segB
			zs[i].TA, zs[i].TB = zs[i].TB, zs[i].TA
			zs[i].DirA, zs[i].DirB = zs[i].DirB, zs[i].DirA
		}
	} else {
		for i := n; i < len(zs); i++ {
			zs[i].SegA, zs[i].SegB = segA, segB
		}
	}
	return zs
}

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

// Intersections amongst the combinations between line, quad, cube, elliptical arcs. We consider four cases: the curves do not cross nor touch (intersections is empty), the curves intersect (and cross), the curves intersect tangentially (touching), or the curves are identical (or parallel in the case of two lines). In the last case we say there are no intersections. As all curves are segments, it is considered a secant intersection when the segments touch but "intent to" cut at their ends (i.e. when position equals to 0 or 1 for either segment).

type intersectionKind int

const (
	AintoB intersectionKind = iota
	BintoA
	Tangent
)

func (v intersectionKind) String() string {
	if v == AintoB {
		return "AintoB"
	} else if v == BintoA {
		return "BintoA"
	}
	return "Tangent"
}

type intersectionParallel int

const (
	NoParallel intersectionParallel = iota
	AParallel                       // parallel along A
	BParallel                       // parallel along B
	Parallel                        // parallel along both
)

func (v intersectionParallel) String() string {
	if v == Parallel {
		return "Parallel"
	} else if v == AParallel {
		return "AParallel"
	} else if v == BParallel {
		return "BParallel"
	}
	return "NoParallel"
}

type intersection struct {
	// SegA, SegB, and Parallel are filled/specified only for path intersections, not segment
	Point
	SegA, SegB int
	TA, TB     float64 // position along segment in [0,1]
	DirA, DirB float64 // angle of direction along segment
	Kind       intersectionKind
	Parallel   intersectionParallel // 3 = parallel along A and B
}

func (z intersection) Equals(o intersection) bool {
	return z.Point.Equals(o.Point) && z.SegA == o.SegA && z.SegB == o.SegB && Equal(z.TA, o.TA) && Equal(z.TB, o.TB) && angleEqual(z.DirA, o.DirA) && angleEqual(z.DirB, o.DirB) && z.Kind == o.Kind && z.Parallel == o.Parallel
}

func (z intersection) String() string {
	return fmt.Sprintf("pos={%g,%g} seg={%d,%d} t={%g,%g} dir={%g°,%g°} %v %v", z.Point.X, z.Point.Y, z.SegA, z.SegB, z.TA, z.TB, angleNorm(z.DirA)*180.0/math.Pi, angleNorm(z.DirB)*180.0/math.Pi, z.Kind, z.Parallel)
}

type intersections []intersection

// There are intersections.
func (zs intersections) Has() bool {
	return 0 < len(zs)
}

// HasSecant returns true when there are secant intersections, i.e. the curves intersect and cross (they cut).
func (zs intersections) HasSecant() bool {
	for _, z := range zs {
		if z.Kind != Tangent {
			return true
		}
	}
	return false
}

// HasTangent returns true when there are tangent intersections, i.e. the curves intersect but don't cross (they touch).
func (zs intersections) HasTangent() bool {
	for _, z := range zs {
		if z.Kind == Tangent {
			return true
		}
	}
	return false
}

func (zs intersections) String() string {
	sb := strings.Builder{}
	for i, z := range zs {
		fmt.Fprintf(&sb, "%v %v\n", i, z)
	}
	return sb.String()
}

func (zs intersections) Sort() {
	sort.Stable(intersectionSort{zs, 0, 0, 0, 0})
}

func (zs intersections) SortAndWrapEnd(segOffsetA, segOffsetB, lenA, lenB int) {
	sort.Stable(intersectionSort{zs, segOffsetA, segOffsetB, lenA, lenB})
}

// sort indices of intersections for curve A
type intersectionSort struct {
	zs                     intersections
	segOffsetA, segOffsetB int
	lenA, lenB             int
}

func (a intersectionSort) Len() int {
	return len(a.zs)
}

func (a intersectionSort) Swap(i, j int) {
	a.zs[i], a.zs[j] = a.zs[j], a.zs[i]
}

func (a intersectionSort) pos(z intersection) (float64, float64) {
	posa := float64(z.SegA) + z.TA
	if Equal(z.TA, 1.0) {
		posa -= Epsilon
		if z.SegA == a.segOffsetA+a.lenA-1 {
			posa -= float64(a.lenA - 1) // put end into first segment (moveto)
		}
	}
	posb := float64(z.SegB) + z.TB
	if Equal(z.TB, 1.0) {
		posb -= Epsilon
		if z.SegB == a.segOffsetB+a.lenB-1 {
			posb -= float64(a.lenB - 1) // put end into first segment (moveto)
		}
	}
	return posa, posb
}

func (a intersectionSort) Less(i, j int) bool {
	// sort by P and secondary to Q. Consider a point at the very end of the curve (seg=len-1, t=1) as if it were at the beginning, since it is on the starting point of the path
	posai, posbi := a.pos(a.zs[i])
	posaj, posbj := a.pos(a.zs[j])
	if posai == posaj {
		return posbi < posbj
	}
	return posai < posaj
}

// sort indices of intersections for curve B
type intersectionArgBSort struct {
	zs  intersections
	idx []int
}

func (a intersectionArgBSort) Len() int {
	return len(a.zs)
}

func (a intersectionArgBSort) Swap(i, j int) {
	a.idx[i], a.idx[j] = a.idx[j], a.idx[i]
}

func (a intersectionArgBSort) Less(i, j int) bool {
	if a.zs[a.idx[i]].SegB == a.zs[a.idx[j]].SegB {
		return a.zs[a.idx[i]].TB < a.zs[a.idx[j]].TB
	}
	return a.zs[a.idx[i]].SegB < a.zs[a.idx[j]].SegB
}

// get indices of sorted intersections for curve B
func (zs intersections) argBSort() []int {
	idx := make([]int, len(zs))
	for i := range idx {
		idx[i] = i
	}
	sort.Stable(intersectionArgBSort{zs, idx})
	return idx
}

func (zs intersections) add(pos Point, ta, tb float64, dira, dirb float64, tangent bool) intersections {
	// the segment-segment functions check whether ta/tb are between [0.0,1.0+Epsilon], clamp
	if ta < 0.0 {
		ta = 0.0
	} else if 1.0 < ta {
		ta = 1.0
	}
	if tb < 0.0 {
		tb = 0.0
	} else if 1.0 < tb {
		tb = 1.0
	}

	var kind intersectionKind
	var parallel intersectionParallel
	if angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi) {
		parallel = Parallel
	}
	if tangent || parallel == Parallel || Equal(ta, 0.0) || Equal(tb, 0.0) || Equal(ta, 1.0) || Equal(tb, 1.0) {
		kind = Tangent
	} else if angleNorm(dirb-dira) < math.Pi {
		kind = BintoA
	} else {
		kind = AintoB
	}
	return append(zs, intersection{
		Point:    pos,
		TA:       ta,
		TB:       tb,
		DirA:     dira,
		DirB:     dirb,
		Kind:     kind,
		Parallel: parallel,
	})
}

// http://www.cs.swan.ac.uk/~cssimon/line_intersection.html
func (zs intersections) LineLine(a0, a1, b0, b1 Point) intersections {
	if a0.Equals(a1) || b0.Equals(b1) {
		return zs
	}

	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if Equal(div, 0.0) {
		// parallel
		if Equal(da.PerpDot(b1.Sub(a0)), 0.0) {
			// aligned, rotate to x-axis
			angle0 := da.Angle()
			angle1 := db.Angle()
			a := a0.Rot(-angle0, Point{}).X
			b := a1.Rot(-angle0, Point{}).X
			c := b0.Rot(-angle0, Point{}).X
			d := b1.Rot(-angle0, Point{}).X
			if (c <= a && a <= d && c <= b && b <= d) || (d <= a && a <= c && d <= b && b <= c) {
				// a-b in c-d or a-b == c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), angle0, angle1, true)
				zs = zs.add(a1, 1.0, (b-c)/(d-c), angle0, angle1, true)
			} else if a < c && c < b && a < d && d < b {
				// c-d in a-b
				zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
			} else if c <= a && a <= d || d <= a && a <= c {
				// a in c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), angle0, angle1, true)
				if a < d {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
				} else if a < c {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				}
			} else if c <= b && b <= d || d <= b && b <= c {
				// b in c-d
				if c < b {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				} else if d < b {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
				}
				zs = zs.add(a1, 1.0, (b-c)/(d-c), angle0, angle1, true)
			}
		}
		return zs
	}

	ta := db.PerpDot(a0.Sub(b0)) / div
	tb := da.PerpDot(a0.Sub(b0)) / div
	if Interval(ta, 0.0, 1.0) && Interval(tb, 0.0, 1.0) {
		zs = zs.add(a0.Interpolate(a1, ta), ta, tb, da.Angle(), db.Angle(), false)
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func (zs intersections) LineQuad(l0, l1, p0, p1, p2 Point) intersections {
	// write line as A.X = bias
	A := Point{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p0.Sub(p1.Mul(2.0)).Add(p2))
	b := A.Dot(p1.Sub(p0).Mul(2.0))
	c := A.Dot(p0) - bias

	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
		}
	}

	dira := l1.Sub(l0).Angle()
	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	for _, root := range roots {
		if Interval(root, 0.0, 1.0) {
			var s float64
			pos := quadraticBezierPos(p0, p1, p2, root)
			if horizontal {
				s = (pos.X - l0.X) / (l1.X - l0.X)
			} else {
				s = (pos.Y - l0.Y) / (l1.Y - l0.Y)
			}
			if Interval(s, 0.0, 1.0) {
				deriv := quadraticBezierDeriv(p0, p1, p2, root)
				dirb := deriv.Angle()
				// deviate angle slightly to distinguish between BintoA/AintoB on head-on collision
				if Equal(root, 0.0) || Equal(root, 1.0) || Equal(s, 0.0) || Equal(s, 1.0) {
					deriv2 := quadraticBezierDeriv2(p0, p1, p2)
					if (0.0 <= deriv.PerpDot(deriv2)) == (Equal(root, 0.0) || !Equal(root, 1.0) && Equal(s, 0.0)) {
						dirb += Epsilon / 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= Epsilon / 2.0 // t=0 and CW, or t=1 and CCW
					}
				}
				zs = zs.add(pos, s, root, dira, dirb, Equal(A.Dot(deriv), 0.0))
			}
		}
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func (zs intersections) LineCube(l0, l1, p0, p1, p2, p3 Point) intersections {
	// write line as A.X = bias
	A := Point{l1.Y - l0.Y, l0.X - l1.X}
	bias := l0.Dot(A)

	a := A.Dot(p3.Sub(p0).Add(p1.Mul(3.0)).Sub(p2.Mul(3.0)))
	b := A.Dot(p0.Mul(3.0).Sub(p1.Mul(6.0)).Add(p2.Mul(3.0)))
	c := A.Dot(p1.Mul(3.0).Sub(p0.Mul(3.0)))
	d := A.Dot(p0) - bias

	roots := []float64{}
	r0, r1, r2 := solveCubicFormula(a, b, c, d)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) {
			roots = append(roots, r1)
			if !math.IsNaN(r2) {
				roots = append(roots, r2)
			}
		}
	}

	dira := l1.Sub(l0).Angle()
	horizontal := math.Abs(l1.Y-l0.Y) <= math.Abs(l1.X-l0.X)
	for _, root := range roots {
		if Interval(root, 0.0, 1.0) {
			var s float64
			pos := cubicBezierPos(p0, p1, p2, p3, root)
			if horizontal {
				s = (pos.X - l0.X) / (l1.X - l0.X)
			} else {
				s = (pos.Y - l0.Y) / (l1.Y - l0.Y)
			}
			if Interval(s, 0.0, 1.0) {
				deriv := cubicBezierDeriv(p0, p1, p2, p3, root)
				dirb := deriv.Angle()
				// deviate angle slightly to distinguish between BintoA/AintoB on head-on collision
				if Equal(root, 0.0) || Equal(root, 1.0) || Equal(s, 0.0) || Equal(s, 1.0) {
					deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
					if (0.0 <= deriv.PerpDot(deriv2)) == (Equal(root, 0.0) || !Equal(root, 1.0) && Equal(s, 0.0)) {
						dirb += Epsilon / 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= Epsilon / 2.0 // t=0 and CW, or t=1 and CCW
					}
				}

				// deviate angle slightly to distinguish between BintoA/AintoB when the line and the cubic bezier are parallel only in the intersection, but the paths do cross
				tangent := Equal(A.Dot(deriv), 0.0)
				if !Equal(root, 0.0) && !Equal(root, 1.0) && (angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi)) {
					dirb = p3.Sub(p0).Angle()
					tangent = false
				}
				zs = zs.add(pos, s, root, dira, dirb, tangent)
			}
		}
	}
	return zs
}

func (zs intersections) LineEllipse(l0, l1, center, radius Point, phi, theta0, theta1 float64) intersections {
	dira := l1.Sub(l0).Angle()

	// we take the ellipse center as the origin and counter-rotate by phi
	l0 = l0.Sub(center).Rot(-phi, Origin)
	l1 = l1.Sub(center).Rot(-phi, Origin)

	// write ellipse as Ax^2 + By^2 = 1 and line as Cx + Dy = E
	A := 1.0 / (radius.X * radius.X)
	B := 1.0 / (radius.Y * radius.Y)
	C := l1.Y - l0.Y
	D := l0.X - l1.X
	E := l0.Dot(Point{C, D})

	// rewrite as a polynomial by substituting x or y: ax^2 + bx + c = 0
	var a, b, c float64
	horizontal := math.Abs(C) <= math.Abs(D)
	if horizontal {
		a = A*D*D + B*C*C
		b = -2.0 * B * E * C
		c = B*E*E - D*D
	} else {
		a = B*C*C + A*D*D
		b = -2.0 * A * E * D
		c = A*E*E - C*C
	}

	// find solutions
	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) && !Equal(r0, r1) {
			roots = append(roots, r1)
		}
	}

	for _, root := range roots {
		// get intersection position with center as origin
		var x, y, s float64
		if horizontal {
			x = root
			y = (E - C*x) / D
			s = (x - l0.X) / (l1.X - l0.X)
		} else {
			y = root
			x = (E - D*x) / C
			s = (y - l0.Y) / (l1.Y - l0.Y)
		}

		angle := math.Atan2(y, x)
		if Interval(s, 0.0, 1.0) && angleBetween(angle, theta0, theta1) {
			var t float64
			if theta0 <= theta1 {
				t = angleNorm(angle-theta0) / angleNorm(theta1-theta0)
			} else {
				t = 1.0 - angleNorm(angle-theta1)/angleNorm(theta0-theta1)
			}
			pos := Point{x, y}.Rot(phi, Origin).Add(center)
			dirb := ellipseDeriv(radius.X, radius.Y, phi, theta0 <= theta1, angle).Angle()
			// deviate angle slightly to distinguish between BintoA/AintoB on head-on directions
			if Equal(t, 0.0) || Equal(t, 1.0) || Equal(s, 0.0) || Equal(s, 1.0) {
				if (theta0 <= theta1) == (Equal(t, 0.0) || !Equal(t, 1.0) && Equal(s, 0.0)) {
					dirb += Epsilon / 2.0 // t=0 and CCW, or t=1 and CW
				} else {
					dirb -= Epsilon / 2.0 // t=0 and CW, or t=1 and CCW
				}
			}
			zs = zs.add(pos, s, t, dira, dirb, Equal(root, 0.0))
		}
	}
	return zs
}

// TODO: bezier-bezier intersection
// TODO: bezier-ellipse intersection
// TODO: ellipse-ellipse intersection

// For Bézier-Bézier interesections:
// see T.W. Sederberg, "Computer Aided Geometric Design", 2012
// see T.W. Sederberg and T. Nishita, "Curve intersection using Bézier clipping", 1990
// see T.W. Sederberg and S.R. Parry, "Comparison of three curve intersection algorithms", 1986

// http://mathworld.wolfram.com/Circle-LineIntersection.html
func intersectionRayCircle(l0, l1, c Point, r float64) (Point, Point, bool) {
	d := l1.Sub(l0).Norm(1.0) // along line direction, anchored in l0, its length is 1
	D := l0.Sub(c).PerpDot(d)
	discriminant := r*r - D*D
	if discriminant < 0 {
		return Point{}, Point{}, false
	}
	discriminant = math.Sqrt(discriminant)

	ax := D * d.Y
	bx := d.X * discriminant
	if d.Y < 0.0 {
		bx = -bx
	}
	ay := -D * d.X
	by := math.Abs(d.Y) * discriminant
	return c.Add(Point{ax + bx, ay + by}), c.Add(Point{ax - bx, ay - by}), true
}

// https://math.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
func intersectionCircleCircle(c0 Point, r0 float64, c1 Point, r1 float64) (Point, Point, bool) {
	R := c0.Sub(c1).Length()
	if R < math.Abs(r0-r1) || r0+r1 < R || c0.Equals(c1) {
		return Point{}, Point{}, false
	}
	R2 := R * R

	k := r0*r0 - r1*r1
	a := 0.5
	b := 0.5 * k / R2
	c := 0.5 * math.Sqrt(2.0*(r0*r0+r1*r1)/R2-k*k/(R2*R2)-1.0)

	i0 := c0.Add(c1).Mul(a)
	i1 := c1.Sub(c0).Mul(b)
	i2 := Point{c1.Y - c0.Y, c0.X - c1.X}.Mul(c)
	return i0.Add(i1).Add(i2), i0.Add(i1).Sub(i2), true
}
