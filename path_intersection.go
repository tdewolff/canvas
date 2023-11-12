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
			zp, _ := pathIntersections([]*Path{pi}, []*Path{qi}, false)
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

// Settle combines the path p with itself, including all subpaths, removing all self-intersections and overlapping parts. It returns subpaths with counter clockwise directions when filling, and clockwise directions for holes.
func (p *Path) Settle() *Path {
	// TODO: settle and self-settle for fillrule == EvenOdd
	// TODO: optimize, is very slow for many paths, maybe not use boolean for each subpath, but process in one go?
	if p.Empty() {
		return p
	}
	return p

	//ps := p.Split()
	//p = ps[0].selfSettle()
	//for _, q := range ps[1:] {
	//	q = q.selfSettle()
	//	p = boolean(p, pathOpSettle, q)
	//}

	//// make all filling paths go CCW
	//r := &Path{}
	//ps = p.Split()
	//filling := p.Filling(NonZero)
	//for i := range ps {
	//	if ps[i].Empty() || !ps[i].Closed() {
	//		r = r.Append(ps[i])
	//		continue
	//	}
	//	if ps[i].CCW() == filling[i] {
	//		r = r.Append(ps[i])
	//	} else {
	//		r = r.Append(ps[i].Reverse())
	//	}
	//}
	//return r
}

//func (p *Path) Settle2() *Path {
//	if p.Empty() {
//		return p
//	}
//
//	Zs := selfCollisions(p)
//	//fmt.Println(Zs)
//
//	// duplicate intersections for intersectionNodes
//	Zs2 := make(Intersections, len(Zs)*2)
//	for i, z := range Zs {
//		Zs2[2*i+0] = z
//		z.SegA, z.SegB = z.SegB, z.SegA
//		z.TA, z.TB = z.TB, z.TA
//		z.DirA, z.DirB = z.DirB, z.DirA
//		if z.Kind == AintoB {
//			z.Kind = BintoA
//		} else if z.Kind == BintoA {
//			z.Kind = AintoB
//		}
//		Zs2[2*i+1] = z
//	}
//	idx := Zs2.ArgASort()
//	Zs2.ASort()
//	fmt.Println(Zs2)
//
//	zs2 := intersectionNodes(Zs2, p, p) // TODO: don't calculate twice for p
//	fmt.Println(zs2)
//	for i, z := range zs2 {
//		fmt.Println("", i, z.a, z.b)
//	}
//
//	// reverse intersection duplication for z.i
//	zs := make([]*intersectionNode, 0, len(zs2)/2)
//	handled := make([]bool, len(zs2))
//	for i := range zs2 {
//		if handled[i] {
//			continue
//		}
//
//		j := 0
//		for ; j < len(zs2); j++ {
//			if i != j && idx[zs2[i].i]/2 == idx[zs2[j].i]/2 {
//				break
//			}
//		}
//		handled[i] = true
//		handled[j] = true
//
//		zs2[j].prevA.nextA = zs2[i]
//		zs2[j].nextA.prevA = zs2[i]
//		zs2[j].prevB.nextB = zs2[i]
//		zs2[j].nextB.prevB = zs2[i]
//		zs = append(zs, zs2[i])
//	}
//	fmt.Println(zs)
//	for i, z := range zs {
//		fmt.Println("", i, z.a, z.b)
//	}
//
//	R := &Path{}
//	for _, z0 := range zs {
//		r := &Path{}
//		gotoB := z0.kind == BintoA
//		forward := !gotoB
//		for z := z0; ; {
//			if gotoB {
//				if forward {
//					r = r.Join(z.b)
//					z = z.nextB
//				} else {
//					r = r.Join(z.b.Reverse())
//					z = z.prevB
//				}
//			} else {
//				if forward {
//					r = r.Join(z.a)
//					z = z.nextA
//				} else {
//					r = r.Join(z.a.Reverse())
//					z = z.prevA
//				}
//			}
//			if z.i == z0.i {
//				break
//			}
//			forward = z.kind == BintoA
//			gotoB = !gotoB
//		}
//		r.Close()
//		//r.optimizeClose()
//		R = R.Append(r)
//	}
//	return R
//}
//
//func (p *Path) selfSettle() *Path {
//	// p is non-complex
//	if p.Empty() || !p.Closed() {
//		return p
//	}
//	q := p.Flatten(Tolerance)
//	Zs := collisions([]*Path{q}, []*Path{q}, false)
//	if len(Zs) == 0 {
//		return p
//	}
//
//	// TODO: implement parallel lines in selfCollisions, which is more efficient than collisions
//	//Zs := selfCollisions(q)
//
//	// duplicate intersections
//	//Zs2 := make(Intersections, len(Zs)*2)
//	//for i, z := range Zs {
//	//	Zs2[2*i+0] = z
//	//	z.SegA, z.SegB = z.SegB, z.SegA
//	//	z.TA, z.TB = z.TB, z.TA
//	//	z.DirA, z.DirB = z.DirB, z.DirA
//	//	if z.Kind == AintoB {
//	//		z.Kind = BintoA
//	//	} else if z.Kind == BintoA {
//	//		z.Kind = AintoB
//	//	}
//	//	Zs2[2*i+1] = z
//	//}
//	//Zs2.ASort()
//
//	ccw := q.CCW()
//	return booleanIntersections(pathOpNot, Zs, q, q, ccw, ccw) // TODO: not sure why NOT works
//}

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
	// remove self-intersections within each path and direct them all CCW
	p = p.Settle()
	q = q.Settle()

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

	// we can only handle line-line, line-quad, line-cube, and line-arc intersections
	if !p.Flat() {
		q = q.Flatten(Tolerance)
	}

	ps, qs := p.Split(), q.Split()
	for i := range ps {
		if ps[i].Closed() && !ps[i].CCW() {
			fmt.Println("warn: pi != ccw")
			ps[i] = ps[i].Reverse()
		}
	}
	for i := range qs {
		if qs[i].Closed() && !qs[i].CCW() {
			fmt.Println("warn: qi != ccw")
			qs[i] = qs[i].Reverse()
		}
	}

	// implicitly close all subpaths of path q
	p, q = &Path{}, &Path{} // collect all closed paths
	for i := range qs {
		if !qs[i].Closed() {
			qs[i].Close()
		}
		q = q.Append(qs[i])
	}

	// find all intersections (non-tangent) between p and q
	zp, zq := pathIntersections(ps, qs, false)

	// handle open subpaths
	j, d := 0, 0 // j is index into zp/zq, d is number of removed segments
	segOffset := 0
	Ropen := &Path{}
	for i := 0; i < len(ps); i++ {
		length, closed := ps[i].Len(), ps[i].Closed()

		n := 0
		j0 := j
		for ; j < len(zp) && zp[j].Seg < segOffset+length; j++ {
			if closed {
				zp[j].Seg -= d
			} else {
				zp[j].Seg -= segOffset
			}
			if !zp[j].Tangent {
				n++
			}
		}
		segOffset += length

		if closed {
			p = p.Append(ps[i])
		} else {
			if n == 0 {
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
				// parts on the boundary are removed
				zpj := zp[j0:j]
				psi, _ := cut(ps[i], zpj)
				inside := !zpj[0].Into
				if op == pathOpOr || inside && op == pathOpAnd || !inside && (op == pathOpXor || op == pathOpNot) {
					Ropen = Ropen.Append(psi[0])
				}
				for k := 1; k < len(psi); k++ {
					inside := zpj[k-1].Into
					if !zpj[k-1].Parallel && (op == pathOpOr || inside && op == pathOpAnd || !inside && (op == pathOpXor || op == pathOpNot)) {
						Ropen = Ropen.Append(psi[k])
					}
				}
			}
			zp = append(zp[:j0], zp[j:]...)
			zq = append(zq[:j0], zq[j:]...)
			ps = append(ps[:i], ps[i+1:]...)
			d += length
			i--
		}
	}

	// handle intersecting subpaths
	zs := pathIntersectionNodes(p, q, zp, zq)
	R := booleanIntersections(op, zs)

	// handle the remaining subpaths that are non-intersecting but possibly overlapping, either one containing the other or by being equal
	pIndex, qIndex := newSubpathIndexer(ps), newSubpathIndexer(qs)
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
	visited := [][]bool{} // per direction
	for k := 0; k < K; k++ {
		visited = append(visited, make([]bool, len(zs)))
	}
	for _, z0 := range zs {
		for k := 0; k < K; k++ {
			if visited[k][z0.i] {
				continue
			}

			r := &Path{}
			var forwardP, forwardQ bool
			onQ := startInwards[k] != z0.PintoQ // ensure result is CCW
			if onQ {
				forwardQ = invertQ[k] == z0.PintoQ
			} else {
				forwardP = invertP[k] == z0.PintoQ
			}

			for z := &z0; ; {
				visited[k][z.i] = true
				if z.i != z0.i && z.Parallel && (forwardP == forwardQ) != z.Reversed {
					// parallel lines for crossing intersections
					// only show when not changing forwardness, or when parallel in reverse order
					if forwardP {
						r = r.Join(z.x)
					} else {
						r = r.Join(z.x.Reverse())
					}
				}
				if onQ {
					if forwardQ {
						r = r.Join(z.q)
						z = z.nextQ
					} else {
						r = r.Join(z.prevQ.q.Reverse())
						z = z.prevQ
					}
				} else {
					if forwardP {
						r = r.Join(z.p)
						z = z.nextP
					} else {
						r = r.Join(z.prevP.p.Reverse())
						z = z.prevP
					}
				}

				if z.i == z0.i {
					break
				}
				onQ = !onQ
				if onQ {
					forwardQ = invertQ[k] == z.PintoQ
				} else {
					forwardP = invertP[k] == z.PintoQ
				}
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
	return pathIntersections(p.Split(), q.Split(), false)
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
	return pathIntersections(p.Split(), q.Split(), true)
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
	seg := 0
	var start, end Point
	var zs []Intersection
	Zs := []PathIntersection{}
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		zs = zs[:0]
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
		for _, z := range zs {
			Zs = append(Zs, PathIntersection{
				Point:    z.Point,
				Seg:      seg,
				T:        z.T[1],
				Tangent:  Equal(z.T[0], 0.0),
				Parallel: z.Aligned() || z.AntiAligned(),
				Into:     z.Into(),
			})
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
