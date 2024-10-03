package canvas

import (
	"fmt"
	"math"
	"sort"
)

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

// Intersections amongst the combinations between line, quad, cube, elliptical arcs. We consider four cases: the curves do not cross nor touch (intersections is empty), the curves intersect (and cross), the curves intersect tangentially (touching), or the curves are identical (or parallel in the case of two lines). In the last case we say there are no intersections. As all curves are segments, it is considered a secant intersection when the segments touch but "intent to" cut at their ends (i.e. when position equals to 0 or 1 for either segment).

func segmentPos(start Point, d []float64, t float64) Point {
	// used for open paths in boolean
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

type subpathIndexer []int // index from segment to subpath

func newSubpathIndexer(p *Path) subpathIndexer {
	segs := 0
	var idx subpathIndexer
	for i := 0; i < len(p.d); i += cmdLen(p.d[i]) {
		if i != 0 && p.d[i] == MoveToCmd {
			idx = append(idx, segs)
		}
		segs++
	}
	idx = append(idx, segs)
	return idx
}

func newSubpathIndexerSubpaths(ps []*Path) subpathIndexer {
	segs := 0
	idx := make(subpathIndexer, len(ps))
	for i, pi := range ps {
		segs += pi.Len()
		idx[i] = segs
	}
	return idx
}

func (idx subpathIndexer) in(i, seg int) bool {
	return (i == 0 || idx[i-1] <= seg) && seg < idx[i]
}

func (idx subpathIndexer) get(seg int) int {
	for i, n := range idx {
		if seg < n {
			return i
		}
	}
	panic("bug: segment doesn't exist on path")
}

type PathIntersectionNode struct {
	i            int // intersection index
	prevP, nextP *PathIntersectionNode
	prevQ, nextQ *PathIntersectionNode

	p, q *Path // path towards next node
	x    *Path // parallel/ovarlapping path at node, can be nil

	PintoQ           bool
	Tangent          bool
	Parallel         bool
	ParallelReversed bool
}

func (z PathIntersectionNode) ParallelTangent(onP, forwardP, forwardQ bool) bool {
	return z.Tangent && (onP && (forwardP && z.Parallel || !forwardP && z.prevP.Parallel) || !onP && (forwardQ && (z.Parallel && !z.ParallelReversed || z.nextQ.Parallel && z.nextQ.ParallelReversed) || !forwardQ && (z.prevQ.Parallel && !z.prevQ.ParallelReversed || z.Parallel && z.ParallelReversed)))
}

func (z PathIntersectionNode) String() string {
	var extra string
	if z.PintoQ {
		extra = " PintoQ"
	} else {
		extra = " QintoP"
	}
	if z.Tangent {
		extra += "-Tangent"
	}
	if z.Parallel {
		extra += " Parallel"
		if z.ParallelReversed {
			extra += "-Reversed"
		}
	}
	pos := z.p.StartPos()
	return fmt.Sprintf("(%v {%v,%v} P=[%v→·→%v] Q=[%v→·→%v]%v)", z.i, numEps(pos.X), numEps(pos.Y), z.prevP.i, z.nextP.i, z.prevQ.i, z.nextQ.i, extra)
}

func pathIntersectionNodes(p, q *Path, zp, zq []PathIntersection) []PathIntersectionNode {
	// create graph of nodes between intersections over both paths
	if len(zp) == 0 {
		return nil
	}

	// count number of nodes
	n := len(zp)
	for _, z := range zp {
		if !z.Tangent && z.Overlapping {
			n--
		}
	}
	if n%2 != 0 {
		panic("bug: number of nodes must be even")
	}

	i, k := 0, 0
	ps, segs := cut(p, zp)
	idxZ := make([]int, len(zp)) // index zp to zs
	zs := make([]PathIntersectionNode, n)
	for _, seg := range segs {
		// loop over each subpath of p
		j := i
		for j < len(zp) && zp[j].Seg < seg {
			j++
		}

		i0, k0 := i, k
		for ; i < j; i++ {
			// loop over the intersections in a subpath of p
			idxZ[i] = k
			if zp[i].Overlapping {
				i1, k1 := i+1, k
				if i+1 == j {
					i1, k1 = i0, k0
				}
				reversed := zq[i1].Overlapping

				if zp[i].Tangent {
					zs[k].Parallel = true
					zs[k].ParallelReversed = reversed
				} else {
					// add parallel part to next node, skip this intersection
					idxZ[i] = k1
					zs[k1].Parallel = true
					zs[k1].ParallelReversed = reversed
					zs[k1].x = ps[i]
					continue
				}
			}

			zs[k].i = k
			zs[k].p = ps[i]
			zs[k].PintoQ = zp[i].Into
			zs[k].Tangent = zp[i].Tangent

			if 0 < k {
				// we overwrite the first (for non-first subpath) at the end
				zs[k].prevP = &zs[k-1]
			}
			if k+1 < len(zs) {
				// we overwrite the last (for non-first subpath) at the end
				zs[k].nextP = &zs[k+1]
			}
			k++
		}
		if k0 < k {
			zs[k0].prevP = &zs[k-1]
			zs[k-1].nextP = &zs[k0]
		}
	}

	// sort zq and keep indices of sorted to original
	idxP := make([]int, len(zq)) // index zq to zp
	for i := range zq {
		idxP[i] = i
	}
	sort.Stable(pathIntersectionSort{zq, idxP})

	i = 0
	qs, segs := cut(q, zq)
	for _, seg := range segs {
		// loop over each subpath of q
		j := i
		for j < len(zq) && zq[j].Seg < seg {
			j++
		}

		i0 := i
		for ; i < j; i++ {
			// loop over the intersections in a subpath of q
			k := idxZ[idxP[i]] // index in zq => index in zp => index in zs
			if !zq[i].Tangent && zq[i].Overlapping {
				continue
			}

			zs[k].q = qs[i]
			if i0 < i {
				i1 := i - 1
				if !zq[i1].Tangent && zq[i1].Overlapping {
					i1--
				}
				if i0 <= i1 {
					zs[k].prevQ = &zs[idxZ[idxP[i1]]]
				}
			}
			if i+1 < j {
				zs[k].nextQ = &zs[idxZ[idxP[i+1]]]
			}
		}
		if i0 < i {
			i1 := i - 1
			if !zq[i1].Tangent && zq[i1].Overlapping {
				i1--
			}
			zs[idxZ[idxP[i0]]].prevQ = &zs[idxZ[idxP[i1]]]
			zs[idxZ[idxP[i1]]].nextQ = &zs[idxZ[idxP[i0]]]
		}
	}
	return zs
}

func cut(p *Path, zs []PathIntersection) ([]*Path, subpathIndexer) {
	// zs must be sorted
	if len(zs) == 0 {
		return []*Path{p}, newSubpathIndexer(p)
	}

	j := 0   // index into zs
	k := 0   // index into ps
	seg := 0 // segment count
	var ps []*Path
	var first, cur []float64
	segs := subpathIndexer{}
	for i := 0; i < len(p.d); i += cmdLen(p.d[i]) {
		cmd := p.d[i]
		if 0 < i && cmd == MoveToCmd {
			closed := p.d[i-1] == CloseCmd
			if first != nil {
				// there were intersections in the last subpath
				if closed {
					ps = append(ps, &Path{append(cur, first[4:]...)})
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
			segs = append(segs, seg)
		}
		if j < len(zs) && seg == zs[j].Seg {
			// segment has an intersection, cut it up and append first part to prev intersection
			p0, p1 := cutSegment(Point{p.d[i-3], p.d[i-2]}, p.d[i:i+cmdLen(cmd)], zs[j].T)
			if !p0.Empty() {
				cur = append(cur, p0.d[4:]...)
			}

			for j+1 < len(zs) && seg == zs[j+1].Seg {
				// next cut is on the same segment, find new t after the first cut and set path
				if first == nil {
					first = cur // take aside the path to the first intersection to later append it
				} else {
					ps = append(ps, &Path{cur})
				}
				j++
				t := (zs[j].T - zs[j-1].T) / (1.0 - zs[j-1].T)
				if !p1.Empty() {
					p0, p1 = cutSegment(Point{p1.d[1], p1.d[2]}, p1.d[4:], t)
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
				// don't append point-close command
				cur = append(cur, p.d[i:i+cmdLen(cmd)]...)
				if cmd == CloseCmd {
					cur[len(cur)-1] = LineToCmd
					cur[len(cur)-cmdLen(CloseCmd)] = LineToCmd
				}
			}
		}
		seg++
	}
	closed := 0 < len(p.d) && p.d[len(p.d)-1] == CloseCmd
	if first != nil {
		// there were intersections in the last subpath
		if closed {
			cur = append(cur, first[4:]...)
		} else {
			ps = append(ps[:k], append([]*Path{{first}}, ps[k:]...)...)
		}
	} else if closed {
		cur[len(cur)-1] = CloseCmd
		cur[len(cur)-4] = CloseCmd
	}
	ps = append(ps, &Path{cur})
	segs = append(segs, seg)
	return ps, segs
}

func cutSegment(start Point, d []float64, t float64) (*Path, *Path) {
	p0, p1 := &Path{}, &Path{}
	if Equal(t, 0.0) {
		p0.MoveTo(start.X, start.Y)
		p1.MoveTo(start.X, start.Y)
		p1.d = append(p1.d, d...)
		if p1.d[cmdLen(MoveToCmd)] == CloseCmd {
			p1.d[cmdLen(MoveToCmd)] = LineToCmd
			p1.d[len(p1.d)-1] = LineToCmd
		}
		return p0, p1
	} else if Equal(t, 1.0) {
		p0.MoveTo(start.X, start.Y)
		p0.d = append(p0.d, d...)
		if p0.d[cmdLen(MoveToCmd)] == CloseCmd {
			p0.d[cmdLen(MoveToCmd)] = LineToCmd
			p0.d[len(p0.d)-1] = LineToCmd
		}
		p1.MoveTo(d[len(d)-3], d[len(d)-2])
		return p0, p1
	}
	if cmd := d[0]; cmd == LineToCmd || cmd == CloseCmd {
		c := start.Interpolate(Point{d[len(d)-3], d[len(d)-2]}, t)
		p0.MoveTo(start.X, start.Y)
		p0.LineTo(c.X, c.Y)
		p1.MoveTo(c.X, c.Y)
		p1.LineTo(d[len(d)-3], d[len(d)-2])
	} else if cmd == QuadToCmd {
		r0, r1, r2, q0, q1, q2 := quadraticBezierSplit(start, Point{d[1], d[2]}, Point{d[3], d[4]}, t)
		p0.MoveTo(r0.X, r0.Y)
		p0.QuadTo(r1.X, r1.Y, r2.X, r2.Y)
		p1.MoveTo(q0.X, q0.Y)
		p1.QuadTo(q1.X, q1.Y, q2.X, q2.Y)
	} else if cmd == CubeToCmd {
		r0, r1, r2, r3, q0, q1, q2, q3 := cubicBezierSplit(start, Point{d[1], d[2]}, Point{d[3], d[4]}, Point{d[5], d[6]}, t)
		p0.MoveTo(r0.X, r0.Y)
		p0.CubeTo(r1.X, r1.Y, r2.X, r2.Y, r3.X, r3.Y)
		p1.MoveTo(q0.X, q0.Y)
		p1.CubeTo(q1.X, q1.Y, q2.X, q2.Y, q3.X, q3.Y)
	} else if cmd == ArcToCmd {
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

// PathIntersection is an intersection of a path.
// Intersection is either tangent or secant. Tangent intersections may be Overlapping. Secant intersections either go into the other path (Into is set) or the other path goes into this path (Into is not set).
// Possible types of intersections:
//   - Crossing anywhere:   Tangent=false, Overlapping=false
//   - Touching anywhere:   Tangent=true, Overlapping=false, Into is invalid
//   - Overlapping onwards: Tangent=false, Overlapping=true, Into is invalid
//
// NB: Tangent may also be true for non-closing paths when touching its endpoints
type PathIntersection struct {
	Point         // coordinate of intersection
	Seg   int     // segment index
	T     float64 // position along segment [0,1]
	Dir   float64 // direction at intersection

	Into        bool // going forward, path goes to LHS of other path
	Tangent     bool // intersection is tangent (touches) instead of secant (crosses)
	Overlapping bool // going forward, paths are overlapping
}

func (z PathIntersection) Less(o PathIntersection) bool {
	ti := float64(z.Seg) + z.T
	tj := float64(o.Seg) + o.T
	// TODO: this generates panics
	//if Equal(ti, tj) {
	//	// Q crosses P twice at the same point, Q must be at a tangent intersections, since
	//	// all secant and parallel tangent intersections have been removed with Settle.
	//	// Choose the parallel-end first and then the parallel-start
	//	return !z.Parallel
	//}
	return z.Seg < o.Seg || ti < tj
}

func (z PathIntersection) Equals(o PathIntersection) bool {
	return z.Point.Equals(o.Point) && z.Seg == o.Seg && Equal(z.T, o.T) && angleEqual(z.Dir, o.Dir) && z.Into == o.Into && z.Tangent == o.Tangent && z.Overlapping == o.Overlapping
}

func (z PathIntersection) String() string {
	extra := ""
	if z.Into {
		extra += " Into"
	}
	if z.Tangent {
		extra += " Tangent"
	}
	if z.Overlapping {
		extra += " Overlapping"
	}
	return fmt.Sprintf("({%v,%v} seg=%d t=%v dir=%v°%v)", numEps(z.Point.X), numEps(z.Point.Y), z.Seg, numEps(z.T), numEps(angleNorm(z.Dir)*180.0/math.Pi), extra)
}

type pathIntersectionSort struct {
	zs  []PathIntersection
	idx []int
}

func (a pathIntersectionSort) Len() int {
	return len(a.zs)
}

func (a pathIntersectionSort) Swap(i, j int) {
	a.zs[i], a.zs[j] = a.zs[j], a.zs[i]
	a.idx[i], a.idx[j] = a.idx[j], a.idx[i]
}

func (a pathIntersectionSort) Less(i, j int) bool {
	return a.zs[i].Less(a.zs[j])
}

type pathIntersectionsSort struct {
	zp []PathIntersection
	zq []PathIntersection
}

func (a pathIntersectionsSort) Len() int {
	return len(a.zp)
}

func (a pathIntersectionsSort) Swap(i, j int) {
	a.zp[i], a.zp[j] = a.zp[j], a.zp[i]
	if a.zq != nil {
		a.zq[i], a.zq[j] = a.zq[j], a.zq[i]
	}
}

func (a pathIntersectionsSort) Less(i, j int) bool {
	return a.zp[i].Less(a.zp[j])
}

// pathIntersections converts segment intersections into path intersections, resolving tangency at segment endpoints, collapsing runs of parallel/overlapping segments
func pathIntersections(p, q *Path, withTangents, withParallelTangents bool) ([]PathIntersection, []PathIntersection) {
	self := q == nil

	// TODO: pass []*Path?
	var ps, qs []*Path
	ps = p.Split()
	if self {
		q = p
		qs = ps
	} else {
		qs = q.Split()
	}

	lenQs := make([]int, len(qs))
	closedQs := make([]bool, len(qs))
	pointClosedQs := make([]bool, len(qs))
	for i := range qs {
		lenQs[i] = qs[i].Len()
		closedQs[i] = qs[i].Closed()
		pointClosedQs[i] = qs[i].PointClosed()
	}

	offsetP := 0
	var zp, zq []PathIntersection
	for i := range ps {
		offsetQ := 0
		lenP := ps[i].Len()

		j := 0
		if self {
			j = i
			offsetQ = offsetP
		}
		for j < len(qs) {
			// register segment indices [1,len), or [1,len-1) when closed by zero-length close command, we add segOffset when adding to PathIntersection
			qsj := qs[j]
			if self && i == j {
				qsj = nil
			}

			zs, segsP, segsQ := intersectionPath(ps[i], qsj)
			if 0 < len(zs) {
				// omit close command with zero length
				lenP, closedP := ps[i].Len(), ps[i].Closed()
				if closedP && ps[i].PointClosed() {
					lenP--
				}
				lenQ, closedQ := lenQs[j], closedQs[j]
				if pointClosedQs[j] {
					lenQ--
				}

				// sort by intersections on P and secondary on Q
				// move degenerate intersections at the end of the path to the start
				sort.Stable(intersectionPathSort{
					zs:      zs,
					pSegs:   segsP,
					qSegs:   segsQ,
					pLen:    lenP,
					qLen:    lenQ,
					pClosed: closedP,
					qClosed: closedQ,
				})

				// Remove degenerate tangent intersections at segment endpoint:
				// - Intersection at endpoints for P and Q: 4 degenerate intersections
				// - Intersection at endpoints for P or Q: 2 degenerate intersections
				// - Parallel/overlapping sections: 4 degenerate + 2N intersections
				var n int
				for i := 0; i < len(zs); i += n {
					n = 1
					z := zs[i]
					startP, startQ := Equal(z.T[0], 0.0), Equal(z.T[1], 0.0)
					endP, endQ := Equal(z.T[0], 1.0), Equal(z.T[1], 1.0)
					endpointP, endpointQ := startP || endP, startQ || endQ

					if !z.Tangent {
						// crossing intersection in the middle of both segments
						PintoQ := z.Into()
						zp = append(zp, PathIntersection{
							Point: z.Point,
							Seg:   offsetP + segsP[i],
							T:     z.T[0],
							Dir:   z.Dir[0],
							Into:  PintoQ,
						})
						zq = append(zq, PathIntersection{
							Point: z.Point,
							Seg:   offsetQ + segsQ[i],
							T:     z.T[1],
							Dir:   z.Dir[1],
							Into:  !PintoQ,
						})
					} else if !endpointP && !endpointQ || !closedP && (segsP[i] == 1 && startP || segsP[i] == lenP-1 && endP) || !closedQ && (segsQ[i] == 1 && startQ || segsQ[i] == lenQ-1 && endQ) {
						// touching intersection in the middle of both segments
						// or touching at the start/end of an open path
						if withTangents {
							zp = append(zp, PathIntersection{
								Point:   z.Point,
								Seg:     offsetP + segsP[i],
								T:       z.T[0],
								Dir:     z.Dir[0],
								Tangent: true,
							})
							zq = append(zq, PathIntersection{
								Point:   z.Point,
								Seg:     offsetQ + segsQ[i],
								T:       z.T[1],
								Dir:     z.Dir[1],
								Tangent: true,
							})
						}
					} else {
						if endpointP && endpointQ {
							n = 4
						} else if endpointP || endpointQ {
							n = 2
						}

						if parallelEnding := z.Aligned() || endQ && zs[i+1].AntiAligned() || !endQ && z.AntiAligned() || self && startP; parallelEnding {
							// found end of parallel as it wraps around path end, skip until start
							if self && startP {
								n /= 2
							}
							continue
						}

						reversedParallel := endQ && zs[(i+n-2)%len(zs)].AntiAligned() || !endQ && zs[(i+n-1)%len(zs)].AntiAligned()
						parallelStart := zs[(i+n-1)%len(zs)].Aligned() || reversedParallel
						if !parallelStart || self {
							// intersection at segment endpoint of one or both paths
							// (angleQi,angleQo) is the LHS angle range for Q
							// PintoQ is the incoming and outgoing direction of P into LHS of Q
							j := (i + n - 1) % len(zs)
							zi, zo := z, zs[j]
							if self && j == 1 {
								// wrap over path end, swap P and Q
								zi = zs[i+1]
							}
							angleQo := zo.Dir[1]
							angleQi := angleQo + angleNorm(zi.Dir[1]+math.Pi-angleQo)
							PinQi := angleBetweenExclusive(zi.Dir[0]+math.Pi, angleQo, angleQi)
							PinQo := angleBetweenExclusive(zo.Dir[0], angleQo, angleQi)
							if tangent := PinQi == PinQo; withTangents || !tangent {
								zp = append(zp, PathIntersection{
									Point:   zo.Point,
									Seg:     offsetP + segsP[j],
									T:       zo.T[0],
									Dir:     zo.Dir[0],
									Into:    !tangent && PinQo,
									Tangent: tangent,
								})
								zq = append(zq, PathIntersection{
									Point:   zo.Point,
									Seg:     offsetQ + segsQ[j],
									T:       zo.T[1],
									Dir:     zo.Dir[1],
									Into:    !tangent && !PinQo,
									Tangent: tangent,
								})
							}
						} else {
							// intersection is parallel
							m := 0
							for {
								// find parallel end, skipping all parallel sections in between
								z := zs[(i+n+m)%len(zs)]
								if (Equal(z.T[0], 0.0) || Equal(z.T[0], 1.0)) && (Equal(z.T[1], 0.0) || Equal(z.T[1], 1.0)) {
									m += 4
								} else {
									m += 2
								}
								endQ := Equal(z.T[1], 1.0)
								if parallelStart := zs[(i+n+m-1)%len(zs)].Aligned() || endQ && zs[(i+n+m-2)%len(zs)].AntiAligned() || !endQ && zs[(i+n+m-1)%len(zs)].AntiAligned(); !parallelStart {
									// found end of parallel run
									break
								}
							}

							j0, j1 := i, i+1
							j2, j3 := (i+n+m-2)%len(zs), (i+n+m-1)%len(zs) // may wrap path end
							z0, z1, z2, z3 := zs[j0], zs[j1], zs[j2], zs[j3]

							// dangle is the turn angle following P over the parallel segments
							angleQo := angleNorm(z1.Dir[1])
							angleQi := angleQo + angleNorm(z0.Dir[1]+math.Pi-angleQo)
							PinQi := angleBetweenExclusive(z0.Dir[0]+math.Pi, angleQo, angleQi)

							dangle := zs[(i+n)%len(zs)].Dir[0] - zs[i+n-1].Dir[0]
							angleQo = angleNorm(z3.Dir[1])
							angleQi = angleQo + angleNorm(z2.Dir[1]+math.Pi-angleQo)
							PinQo := angleBetweenExclusive(z3.Dir[0]-dangle, angleQo-dangle, angleQi-dangle)
							if tangent := PinQi == PinQo; withParallelTangents || withTangents || !tangent {
								ji, jo := i+n-1, (i+n+m-1)%len(zs)
								zi, zo := zs[ji], zs[jo]
								zp = append(zp, PathIntersection{
									Point:       zi.Point,
									Seg:         offsetP + segsP[ji],
									T:           zi.T[0],
									Dir:         zi.Dir[0],
									Into:        withParallelTangents && !PinQo,
									Overlapping: true,
									Tangent:     withParallelTangents && tangent,
								}, PathIntersection{
									Point:   zo.Point,
									Seg:     offsetP + segsP[jo],
									T:       zo.T[0],
									Dir:     zo.Dir[0],
									Into:    (withParallelTangents || !tangent) && PinQo,
									Tangent: tangent,
								})
								if !reversedParallel {
									zq = append(zq, PathIntersection{
										Point:       zi.Point,
										Seg:         offsetQ + segsQ[ji],
										T:           zi.T[1],
										Dir:         zi.Dir[1],
										Into:        withParallelTangents && PinQo,
										Overlapping: true,
										Tangent:     withParallelTangents && tangent,
									}, PathIntersection{
										Point:   zo.Point,
										Seg:     offsetQ + segsQ[jo],
										T:       zo.T[1],
										Dir:     zo.Dir[1],
										Into:    (withParallelTangents || !tangent) && !PinQo,
										Tangent: tangent,
									})
								} else {
									zq = append(zq, PathIntersection{
										Point:   zi.Point,
										Seg:     offsetQ + segsQ[ji],
										T:       zi.T[1],
										Dir:     zi.Dir[1],
										Into:    (withParallelTangents || !tangent) && !PinQo,
										Tangent: tangent,
									}, PathIntersection{
										Point:       zo.Point,
										Seg:         offsetQ + segsQ[jo],
										T:           zo.T[1],
										Dir:         zo.Dir[1],
										Into:        withParallelTangents && PinQo,
										Overlapping: true,
										Tangent:     withParallelTangents && tangent,
									})
								}
							}
							i += m // skip parallel mid and end (here) and start (in for)
						}
					}
				}
			}
			offsetQ += lenQs[j]
			j++
		}
		offsetP += lenP
	}

	if self {
		zp = append(zp, zq...)
		zq = append(zq, zp[:len(zq)]...)
	}
	sort.Stable(pathIntersectionsSort{zp, zq})

	if self {
		q = nil
	}
	zp2, zq2 := intersectionPath2(p, q, withTangents, withParallelTangents)
	if !sort.IsSorted(pathIntersectionsSort{zp2, zq2}) {
		fmt.Println("NOT SORTED path")
	}
	if self {
		n := len(zp2)
		zp2 = append(zp2, zq2...)
		zq2 = append(zq2, zp2[:len(zq2)]...)
		sort.Stable(pathIntersectionsSort{zp2[n:], zq2[n:]})
	}
	return zp2, zq2
	return zp, zq
}

type intersectionPathSort struct {
	zs               Intersections
	pSegs, qSegs     []int
	pLen, qLen       int
	pClosed, qClosed bool
	dontWrap         bool
}

func (a intersectionPathSort) pos(seg int, t float64, length int, closed bool) (float64, bool) {
	end := Equal(t, 1.0)
	if !a.dontWrap && end && closed && seg == length-1 {
		seg = 0 // intersection at path's end into first segment (MoveTo)
	}
	return float64(seg) + t, end
}

func (a intersectionPathSort) Len() int {
	return len(a.zs)
}

func (a intersectionPathSort) Swap(i, j int) {
	a.zs[i], a.zs[j] = a.zs[j], a.zs[i]
	a.pSegs[i], a.pSegs[j] = a.pSegs[j], a.pSegs[i]
	a.qSegs[i], a.qSegs[j] = a.qSegs[j], a.qSegs[i]
}

func (a intersectionPathSort) Less(i, j int) bool {
	posPi, endPi := a.pos(a.pSegs[i], a.zs[i].T[0], a.pLen, a.pClosed)
	posPj, endPj := a.pos(a.pSegs[j], a.zs[j].T[0], a.pLen, a.pClosed)
	if Equal(posPi, posPj) {
		if endPi && !endPj {
			return true
		} else if !endPi && endPj {
			return false
		}

		posQi, endQi := a.pos(a.qSegs[i], a.zs[i].T[1], a.qLen, a.qClosed)
		posQj, endQj := a.pos(a.qSegs[j], a.zs[j].T[1], a.qLen, a.qClosed)
		if Equal(posQi, posQj) {
			if endQi && !endQj {
				return true
			} else if !endQi && endQj {
				return false
			}
			return false
		}
		return posQi < posQj
	}
	return posPi < posPj
}

type intersectionPathSort2 struct {
	zs               Intersections
	pSegs, qSegs     []int
	pLen, qLen       int
	pClosed, qClosed bool
	self             bool
}

func (a intersectionPathSort2) pos(seg int, t float64, length int, closed bool) float64 {
	if closed && seg == length-1 && t == 1.0 {
		return 0.0 // wrap to start
	}
	return float64(seg) + t
}

func (a intersectionPathSort2) Len() int {
	return len(a.zs)
}

func (a intersectionPathSort2) Swap(i, j int) {
	a.zs[i], a.zs[j] = a.zs[j], a.zs[i]
	a.pSegs[i], a.pSegs[j] = a.pSegs[j], a.pSegs[i]
	a.qSegs[i], a.qSegs[j] = a.qSegs[j], a.qSegs[i]
}

func (a intersectionPathSort2) Less(i, j int) bool {
	// Sort primarily by P, then by Q.
	if a.self && a.qSegs[i] == a.qLen-1 && a.zs[i].T[1] == 1.0 {
		// special case when at path's start/end for self-intersection
		return true
	}

	pi := a.pos(a.pSegs[i], a.zs[i].T[0], a.pLen, a.pClosed)
	pj := a.pos(a.pSegs[j], a.zs[j].T[0], a.pLen, a.pClosed)
	if Equal(pi, pj) {
		if a.zs[i].T[0] == 1.0 && a.zs[j].T[0] != 1.0 {
			return true
		} else if a.zs[i].T[0] != 1.0 && a.zs[j].T[0] == 1.0 {
			return false
		}

		qi := a.pos(a.qSegs[i], a.zs[i].T[1], a.qLen, a.qClosed)
		qj := a.pos(a.qSegs[j], a.zs[j].T[1], a.qLen, a.qClosed)
		if Equal(qi, qj) {
			if a.zs[i].T[1] == 1.0 && a.zs[j].T[1] != 1.0 {
				return true
			} else if a.zs[i].T[1] != 1.0 && a.zs[j].T[1] == 1.0 {
				return false
			}
			return false
		}
		return qi < qj
	}
	return pi < pj
}

// intersectionPath returns all intersections along a path including the path segments associated.
// If q is nil, it returns all intersections (non-tangent) within the same path (faster).
// All intersections are sorted by path P and then by path Q. P and Q must not have subpaths.
func intersectionPath2(p, q *Path, withTangents, withOverlappingTangents bool) ([]PathIntersection, []PathIntersection) {
	self := q == nil

	// TODO: pass []*Path?
	var ps, qs []*Path
	ps = p.Split()
	if self {
		q = p
		qs = ps
	} else {
		qs = q.Split()
	}

	// pre-compute subpath lengths and if closed
	// if path is point-closed (zero-length close command), don't count towards length
	pLens := make([]int, len(ps))
	pCloseds := make([]bool, len(ps))
	for i := range ps {
		pLens[i] = ps[i].Len()
		pCloseds[i] = ps[i].Closed()
		if ps[i].PointClosed() {
			pLens[i]--
		}
	}
	qLens := pLens
	qCloseds := pCloseds
	if !self {
		qLens = make([]int, len(qs))
		qCloseds = make([]bool, len(qs))
		for i := range qs {
			qLens[i] = qs[i].Len()
			qCloseds[i] = qs[i].Closed()
			if qs[i].PointClosed() {
				qLens[i]--
			}
		}
	}

	// iterate over sub paths of P and Q
	pOffset := 0
	var zp, zq []PathIntersection
	for i, p := range ps {
		j := 0
		qOffset := 0
		if self {
			// skip already checked subpath combinations
			qOffset = pOffset
			j = i
		}
		for j < len(qs) {
			q := qs[j]
			if self && i == j {
				q = nil
			}

			zp, zq = intersectionSubpath2(zp, zq, p, q, pOffset, qOffset, pLens[i], qLens[j], pCloseds[i], qCloseds[j], withTangents, withOverlappingTangents)

			qOffset += qLens[j]
			j++
		}
		pOffset += pLens[i]
	}
	return zp, zq
}

// intersectionPath returns all intersections along a path including the path segments associated.
// If q is nil, it returns all intersections (non-tangent) within the same path (faster).
// All intersections are sorted by path P and then by path Q. P and Q must not have subpaths.
func intersectionSubpath2(zp, zq []PathIntersection, p, q *Path, pOffset, qOffset, pLen, qLen int, pClosed, qClosed bool, withTangents, withOverlappingTangents bool) ([]PathIntersection, []PathIntersection) {
	// Find intersections between two segments, with possible types of intersections:
	// - Not at the end point of either segment: 1-fold degenerate intersection
	// - At the end point of one segment: 2-fold degenerate intersection
	// - At the end point of both segment: 4-fold degenerate intersection
	//
	// Furthermore, parts of both paths may be equal where the segments are parallel.
	// - Start/end can be either a 2-fold or 4-fold degenerate intersection, where one of
	//   the outgoing/incoming halves has equal direction for both paths P and Q.
	// - Middle intersections are 4-fold degenerate where both incoming/outgoing halves
	//   have equal direction for both paths P and Q.
	// Note that directions may be reversed when paths are equal but of opposite direction.			   // Also note that we may already be half-way in a parallel section if path equality
	// extends over the path's start/end position.
	//
	// It may occur that only part of a 2-fold/4-fold degenerate intersection was found due
	// to floating point precision errors. This is fixed to improve stability.
	//
	// When finding self-intersections (P==Q) we have to skip all intersections with
	// adjacent segments.

	// rebase zero'th segment to second command (past MoveTo)
	pLen--
	qLen--

	// stack
	var zs []Intersection
	var pSegs []int
	var qSegs []int

	fmt.Println()
	fmt.Println(p)
	fmt.Println(q)

	self := q == nil
	if self {
		q = p
	}

	// TODO: uses O(N^2), try sweep line or bently-ottman to reduce to O((N+K) log N) (or better yet https://dl.acm.org/doi/10.1145/147508.147511)
	// see https://www.webcitation.org/6ahkPQIsN        Bentley-Ottmann
	// iterate over path segments for both paths, skipping point-closes
	pSeg, qSeg := 0, 0
	for i := 4; i < len(p.d) && pSeg < pLen; {
		pn := cmdLen(p.d[i])
		p0 := Point{p.d[i-3], p.d[i-2]}
		if self && p.d[i] == CubeToCmd {
			// TODO: find intersections in Cube after we support non-flat paths
		}

		j := 4
		qSeg = 0
		if self {
			// skip already checked segment combinations
			qSeg = pSeg + 1
			j = i + pn
		}
		for j < len(q.d) && qSeg < qLen {
			qn := cmdLen(q.d[j])
			q0 := Point{q.d[j-3], q.d[j-2]}

			n := len(zs)
			zs = intersectionSegment(zs, p0, p.d[i:i+pn], q0, q.d[j:j+qn])

			// remove duplicate cross-segment intersections
			// this reduces the 4-fold degenerate intersections to 2-fold degenerate
			for k := len(zs) - 1; n <= k; k-- {
				z := zs[k]
				// remove all cross-segment intersections, unless we're at the start/end of
				// an open path and withTangents is true
				pStart := z.T[0] == 0.0 && (pClosed || !withTangents || pSeg != 0)
				qStart := z.T[1] == 0.0 && (qClosed || !withTangents || qSeg != 0)
				pEnd := z.T[0] == 1.0 && (pClosed || !withTangents || pSeg != pLen-1)
				qEnd := z.T[1] == 1.0 && (qClosed || !withTangents || qSeg != qLen-1)
				if pStart && qEnd || pEnd && qStart {
					zs = append(zs[:k], zs[k+1:]...)
				}
			}

			for k := n; k < len(zs); k++ {
				pSegs = append(pSegs, pSeg)
				qSegs = append(qSegs, qSeg)
			}

			j += qn
			qSeg++
		}
		i += pn
		pSeg++
	}

	// sort by intersections on P and secondary on Q
	// paths may be unsorted when they have multiple intersections in a single segment.
	sort.Sort(intersectionPathSort2{zs, pSegs, qSegs, pLen, qLen, pClosed, qClosed, self})
	for i := range zs {
		fmt.Println(i, pSegs[i]+1, qSegs[i]+1, zs[i])
	}

	// state of overlapping paths
	var overlapping, backwards bool
	var zo0 Intersection   // initial intersection
	var pSego0, qSego0 int // initial segment indices
	var PintoQi, QintoPo bool

	kk := len(zp)
	KMax := len(zs)
	for K := 0; K < KMax; K++ {
		k := K % len(zs)

		z := zs[k]
		pDegenerate := (z.T[0] == 0.0 || z.T[0] == 1.0) && (pClosed || (z.T[0] != 0.0 || pSegs[k] != 0) && (z.T[0] != 1.0 || pSegs[k] != pLen-1))
		qDegenerate := (z.T[1] == 0.0 || z.T[1] == 1.0) && (qClosed || (z.T[1] != 0.0 || qSegs[k] != 0) && (z.T[1] != 1.0 || qSegs[k] != qLen-1))
		if !pDegenerate && !qDegenerate && !overlapping {
			// 1-fold degenerate
			// middle of both segments, or at the start/end of an open part
			if !z.Tangent || withTangents {
				PintoQ := z.Into()
				zp = append(zp, PathIntersection{
					Point:   z.Point,
					Seg:     pOffset + pSegs[k] + 1,
					T:       z.T[0],
					Dir:     z.Dir[0],
					Into:    PintoQ, //!z.Tangent && PintoQ,
					Tangent: z.Tangent,
				})
				zq = append(zq, PathIntersection{
					Point:   z.Point,
					Seg:     qOffset + qSegs[k] + 1,
					T:       z.T[1],
					Dir:     z.Dir[1],
					Into:    !z.Tangent && !PintoQ,
					Tangent: z.Tangent,
				})
			}
		} else {
			// 2-fold degenerate on P and/or Q (4-folds already reduced to 2-fold)

			// state of current intersection
			var zi, zo Intersection
			var pSegi, pSego, qSegi, qSego int

			// find pair
			// selfPair occurs for self-intersections at the start/end
			if !pDegenerate && !qDegenerate && overlapping {
				// this may happen when the overlapping sections ends the open path
				zi, zo = zs[k], zs[k]
				pSegi, qSegi = pSegs[k], qSegs[k]
				pSego, qSego = pSegs[k], qSegs[k]
			} else if z.T[0] == 1.0 || z.T[1] == 1.0 {
				ki, ko := k, (k+1)%len(zs)
				pPair := pDegenerate && pSegs[ko] == (pSegs[ki]+1)%pLen && zs[ko].T[0] == 0.0 || !pDegenerate && pSegs[ko] == pSegs[ki] && Equal(zs[ko].T[0], z.T[0])
				qPair := qDegenerate && qSegs[ko] == (qSegs[ki]+1)%qLen && zs[ko].T[1] == 0.0 || !qDegenerate && qSegs[ko] == qSegs[ki] && Equal(zs[ko].T[1], z.T[1])
				selfPair := self && pDegenerate && qDegenerate && pSegs[ko] == (qSegs[ki]+1)%qLen && qSegs[ko] == (pSegs[ki]+1)%pLen && zs[ko].T[0] == 0.0 && zs[ko].T[1] == 0.0
				if pPair && qPair || selfPair {
					zi, zo = zs[ki], zs[ko]
					pSegi, qSegi = pSegs[ki], qSegs[ki]
					pSego, qSego = pSegs[ko], qSegs[ko]
					K++
				} else {
					// repair numerical error when only one halve of the pair was found
					fmt.Println("MISSING OUTGOING")
					zi, zo = zs[k], zs[k]
					pSegi, qSegi = pSegs[k], qSegs[k]
					if zi.T[0] == 1.0 {
						pSego = (pSegi + 1) % pLen
						zo.T[0] = 0.0
						zo.Dir[0] = p.Direction(pSego, 0.0).Angle()
					}
					if zi.T[1] == 1.0 {
						qSego = (qSegi + 1) % qLen
						zo.T[1] = 0.0
						zo.Dir[1] = q.Direction(qSego, 0.0).Angle()
					}
				}
			} else {
				// z.T[0] == 0.0 || z.T[1] == 0.0

				// since P is sorted, we should have encountered the incoming intersection first
				// repair numerical error when only one halve of the pair was found
				fmt.Println("MISSING INCOMING")
				zi, zo = zs[k], zs[k]
				pSego, qSego = pSegs[k], qSegs[k]
				if zo.T[0] == 0.0 {
					pSegi = pSego - 1
					if pSegi < 1 {
						pSegi = pLen - 1
					}
					zi.T[0] = 1.0
					zi.Dir[0] = p.Direction(pSegi, 1.0).Angle()
				}
				if zo.T[1] == 0.0 {
					qSegi = qSego - 1
					if qSegi < 1 {
						qSegi = qLen - 1
					}
					zi.T[1] = 1.0
					zi.Dir[1] = q.Direction(qSegi, 1.0).Angle()
				}
			}

			pEnd := !pClosed && z.T[0] == 1.0 && pSegs[k] == pLen-1
			qEnd := !qClosed && z.T[1] == 1.0 && qSegs[k] == qLen-1

			// handle overlapping paths and calculate incoming angles
			// aligned is when angle over P and Q is equal
			// anti-aligned is when angle over P is contrary to the _other_ angle over Q
			//   e.g. outgoing over P is contrary to incoming over Q, or vice versa
			if overlapping {
				// already in overlapping path, find the end
				if pEnd || qEnd {
					// end of open path
					// continue below to add intersection to zp/zq
				} else if !backwards && !zo.Aligned() || backwards && !angleEqual(zo.Dir[0], zi.Dir[1]+math.Pi) {
					// end of overlapping path
					// outgoing over P is not aligned nor anti-aligned with Q
					// continue below to add intersection to zp/zq
				} else {
					//if (!backwards && zo.Aligned() || backwards && angleEqual(zo.Dir[0], zi.Dir[1]+math.Pi)) && (pDegenerate || qDegenerate) {
					// middle of overlapping path, continue until end of overlap
					// if pDegenerate and qDegenerate are false we're at the end of an open path
					// outgoing over P is aligned or anti-aligned with Q
					continue
				}
			} else if anti := angleEqual(zi.Dir[0], zo.Dir[1]+math.Pi); zi.Aligned() || anti {
				// middle/end of overlapping path, handled when we find the start
				// incoming over P is aligned or anti-aligned with Q
				if anti := angleEqual(zo.Dir[0], zi.Dir[1]+math.Pi); zo.Aligned() || anti {
					// middle of overlapping path
				} else {
					// end of overlapping path
					// allow to wrap around end and keep going until end of overlap
					KMax = len(zs) + k + 1
				}
				continue
			} else if anti := angleEqual(zo.Dir[0], zi.Dir[1]+math.Pi); zo.Aligned() || anti {
				// start of overlapping path
				// outgoing over P is aligned or anti-aligned with Q
				overlapping, backwards = true, anti
				zo0, pSego0, qSego0 = zo, pSego, qSego

				// calculate angles and "into" state, see below for more information
				angleQo := angleNorm(zo.Dir[1])
				angleQi := angleQo + angleNorm(zi.Dir[1]+math.Pi-angleQo)
				PintoQi = angleBetweenExclusive(zi.Dir[0]+math.Pi, angleQo, angleQi)
				if anti {
					anglePo := angleNorm(zo.Dir[0])
					anglePi := anglePo + angleNorm(zi.Dir[0]+math.Pi-anglePo)
					QintoPo = angleBetweenExclusive(zo.Dir[1], anglePo, anglePi)
				}
				continue
			} else {
				// regular, non-overlapping part
				// calculate angles and "into" state, see below for more information
				angleQo := angleNorm(zo.Dir[1])
				angleQi := angleQo + angleNorm(zi.Dir[1]+math.Pi-angleQo)
				PintoQi = angleBetweenExclusive(zi.Dir[0]+math.Pi, angleQo, angleQi)
			}

			// we're either at a single intersection or at the end of an overlapping section
			// if we're at the end of an overlapping section, then zo0, pSego0, and qSego0 are from
			// the start of the overlapping section, otherwise it is zi

			// calculate angles and "into" state
			// PintoQi means that going _backwards_ over the incoming P, P goes into the LHS of Q
			// PintoQo means that going forwards over the outgoing P, P goes into the LHS of Q
			// if both are true or both are false, this is a tangent intersection
			var PintoQo bool
			if pEnd || qEnd {
				// PintoQ or QintoP is always false at the end of an open path
				PintoQi, QintoPo = false, false
			} else {
				angleQo := angleNorm(zo.Dir[1])
				angleQi := angleQo + angleNorm(zi.Dir[1]+math.Pi-angleQo)
				PintoQo = angleBetweenExclusive(zo.Dir[0], angleQo, angleQi)
				if !overlapping || !backwards {
					// otherwise, already set
					anglePo := angleNorm(zo.Dir[0])
					anglePi := anglePo + angleNorm(zi.Dir[0]+math.Pi-anglePo)
					QintoPo = angleBetweenExclusive(zo.Dir[1], anglePo, anglePi)
				}
			}

			if self && pSego == 0 {
				// swap P and Q for self-intersection wrapped around end
				// this is because P "becomes" Q as we wrap around
				PintoQi = !PintoQi
				PintoQo = !PintoQo
				QintoPo = !QintoPo
			}

			tangent := PintoQi == PintoQo
			if !tangent || withTangents || overlapping && withOverlappingTangents {
				if overlapping {
					zp = append(zp, PathIntersection{
						Point:       zo0.Point,
						Seg:         pOffset + pSego0 + 1,
						T:           zo0.T[0],
						Dir:         zo0.Dir[0],
						Into:        PintoQo,
						Tangent:     tangent,
						Overlapping: true,
					})
					zq = append(zq, PathIntersection{
						Point:       zo0.Point,
						Seg:         qOffset + qSego0 + 1,
						T:           zo0.T[1],
						Dir:         zo0.Dir[1],
						Into:        QintoPo,
						Tangent:     tangent,
						Overlapping: overlapping && !backwards,
					})
				}
				zp = append(zp, PathIntersection{
					Point:   zo.Point,
					Seg:     pOffset + pSego + 1,
					T:       zo.T[0],
					Dir:     zo.Dir[0],
					Into:    PintoQo,
					Tangent: tangent,
				})
				zq = append(zq, PathIntersection{
					Point:       zo.Point,
					Seg:         qOffset + qSego + 1,
					T:           zo.T[1],
					Dir:         zo.Dir[1],
					Into:        QintoPo,
					Tangent:     tangent,
					Overlapping: overlapping && backwards,
				})
			}
			overlapping = false
		}
	}
	for i := range zp[kk:] {
		fmt.Println(i, zp[kk+i], zq[kk+i])
	}
	fmt.Println("---")

	return zp, zq
}

// intersectionPath returns all intersections along a path including the path segments associated.
// If q is nil, it returns all intersections (non-tangent) within the same path (faster).
// All intersections are sorted by path P and then by path Q. P and Q must not have subpaths.
func intersectionPath(p, q *Path) (Intersections, []int, []int) {
	var zs Intersections
	var segsP, segsQ []int

	self := q == nil
	if self {
		q = p
	}

	// TODO: uses O(N^2), try sweep line or bently-ottman to reduce to O((N+K) log N) (or better yet https://dl.acm.org/doi/10.1145/147508.147511)
	// see https://www.webcitation.org/6ahkPQIsN        Bentley-Ottmann
	segP, segQ := 1, 1
	for i := 4; i < len(p.d); {
		pn := cmdLen(p.d[i])
		p0 := Point{p.d[i-3], p.d[i-2]}
		if p.d[i] == CloseCmd && p0.Equals(Point{p.d[i+1], p.d[i+2]}) {
			// point-closed
			i += pn
			segP++
			continue
		} else if self && p.d[i] == CubeToCmd {
			// TODO: find intersections in Cube after we support non-flat paths
		}

		j := 4
		segQ = 1
		if self {
			segQ = segP + 1
			j = i + pn
		}
		for j < len(q.d) {
			qn := cmdLen(q.d[j])
			q0 := Point{q.d[j-3], q.d[j-2]}
			if q.d[j] == CloseCmd && q0.Equals(Point{q.d[j+1], q.d[j+2]}) {
				// point-closed
				j += qn
				segQ++
				continue
			}

			k := len(zs)
			zs = intersectionSegment(zs, p0, p.d[i:i+pn], q0, q.d[j:j+qn])
			if self && (i+pn == j || i == 4) {
				// remove tangent intersections for adjacent segments on the same subpath
				for k1 := len(zs) - 1; k <= k1; k1-- {
					if !zs[k1].Tangent {
						continue
					}

					// segments are joined if either j comes after i, or if i is first and j is last (or before last if last is point-closed)
					joined := i+pn == j && Equal(zs[k1].T[0], 1.0) && Equal(zs[k1].T[1], 0.0) ||
						i == 4 && Equal(zs[k1].T[0], 0.0) && Equal(zs[k1].T[1], 1.0) &&
							(q.d[j] == CloseCmd || j+qn < len(q.d) && q.d[j+qn] == CloseCmd &&
								Point{q.d[j+qn-3], q.d[j+qn-2]}.Equals(Point{q.d[j+qn+1], q.d[j+qn+2]}))
					if joined {
						zs = append(zs[:k1], zs[k1+1:]...)
					}
				}
			}
			for ; k < len(zs); k++ {
				segsP = append(segsP, segP)
				segsQ = append(segsQ, segQ)
			}

			j += qn
			segQ++
		}
		i += pn
		segP++
	}
	return zs, segsP, segsQ
}

// intersect for path segments a and b, starting at a0 and b0
func intersectionSegment(zs Intersections, a0 Point, a []float64, b0 Point, b []float64) Intersections {
	n := len(zs)
	swapCurves := false
	if a[0] == LineToCmd || a[0] == CloseCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = intersectionLineLine(zs, a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]})
		} else if b[0] == QuadToCmd {
			zs = intersectionLineQuad(zs, a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]})
		} else if b[0] == CubeToCmd {
			zs = intersectionLineCube(zs, a0, Point{a[1], a[2]}, b0, Point{b[1], b[2]}, Point{b[3], b[4]}, Point{b[5], b[6]})
		} else if b[0] == ArcToCmd {
			rx := b[1]
			ry := b[2]
			phi := b[3] * math.Pi / 180.0
			large, sweep := toArcFlags(b[4])
			cx, cy, theta0, theta1 := ellipseToCenter(b0.X, b0.Y, rx, ry, phi, large, sweep, b[5], b[6])
			zs = intersectionLineEllipse(zs, a0, Point{a[1], a[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
		}
	} else if a[0] == QuadToCmd {
		if b[0] == LineToCmd || b[0] == CloseCmd {
			zs = intersectionLineQuad(zs, b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]})
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
			zs = intersectionLineCube(zs, b0, Point{b[1], b[2]}, a0, Point{a[1], a[2]}, Point{a[3], a[4]}, Point{a[5], a[6]})
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
			zs = intersectionLineEllipse(zs, b0, Point{b[1], b[2]}, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1)
			swapCurves = true
		} else if b[0] == QuadToCmd {
			panic("unsupported intersection for arc-quad")
		} else if b[0] == CubeToCmd {
			panic("unsupported intersection for arc-cube")
		} else if b[0] == ArcToCmd {
			rx2 := b[1]
			ry2 := b[2]
			phi2 := b[3] * math.Pi / 180.0
			large2, sweep2 := toArcFlags(b[4])
			cx2, cy2, theta20, theta21 := ellipseToCenter(b0.X, b0.Y, rx2, ry2, phi2, large2, sweep2, b[5], b[6])
			zs = intersectionEllipseEllipse(zs, Point{cx, cy}, Point{rx, ry}, phi, theta0, theta1, Point{cx2, cy2}, Point{rx2, ry2}, phi2, theta20, theta21)
		}
	}

	// swap A and B in the intersection found to match segments A and B of this function
	if swapCurves {
		for i := n; i < len(zs); i++ {
			zs[i].T[0], zs[i].T[1] = zs[i].T[1], zs[i].T[0]
			zs[i].Dir[0], zs[i].Dir[1] = zs[i].Dir[1], zs[i].Dir[0]
		}
	}
	return zs
}

// Intersection is an intersection between two path segments, e.g. Line x Line.
// Note that intersection is tangent also when it is one of the endpoints, in which case it may be tangent for this segment but we should double check when converting to a PathIntersection as it may or may not cross depending on the adjacent segment(s). Also, the Into value at tangent intersections at endpoints should be interpreted as if the paths were extended and the path would go into the left-hand side of the other path.
// Possible types of intersections:
//   - Crossing not at endpoint: Tangent=false, Aligned=false
//   - Touching not at endpoint: Tangent=true,  Aligned=true, Into is invalid
//   - Touching at endpoint:     Tangent=true,  may be aligned for (partly) overlapping paths
//
// NB: for quad/cube/ellipse aligned angles at the endpoint for non-overlapping curves are deviated slightly to correctly calculate the value for Into, and will thus not be aligned
type Intersection struct {
	Point              // coordinate of intersection
	T       [2]float64 // position along segment [0,1]
	Dir     [2]float64 // direction at intersection [0,2*pi)
	Tangent bool       // intersection is tangent (touches) instead of secant (crosses)
}

// Aligned is true when both paths are aligned at the intersection (angles are equal).
func (z Intersection) Aligned() bool {
	return angleEqual(z.Dir[0], z.Dir[1])
}

// AntiAligned is true when both paths are anti-aligned at the intersection (angles are opposite).
func (z Intersection) AntiAligned() bool {
	return angleEqual(z.Dir[0], z.Dir[1]+math.Pi)
}

// Into returns true if first path goes into the left-hand side of the second path,
// i.e. the second path goes to the right-hand side of the first path.
func (z Intersection) Into() bool {
	// TODO: test that direction is either aligned, or Into is true when in [pi,2*pi]
	return angleBetweenExclusive(z.Dir[1]-z.Dir[0], math.Pi, 2.0*math.Pi)
}

func (z Intersection) Equals(o Intersection) bool {
	return z.Point.Equals(o.Point) && Equal(z.T[0], o.T[0]) && Equal(z.T[1], o.T[1]) && angleEqual(z.Dir[0], o.Dir[0]) && angleEqual(z.Dir[1], o.Dir[1]) && z.Tangent == o.Tangent
}

func (z Intersection) String() string {
	tangent := ""
	if z.Tangent {
		tangent = " Tangent"
	}
	return fmt.Sprintf("({%v,%v} t={%v,%v} dir={%v°,%v°}%v)", numEps(z.Point.X), numEps(z.Point.Y), numEps(z.T[0]), numEps(z.T[1]), numEps(angleNorm(z.Dir[0])*180.0/math.Pi), numEps(angleNorm(z.Dir[1])*180.0/math.Pi), tangent)
}

type Intersections []Intersection

// Has returns true if there are secant/tangent intersections.
func (zs Intersections) Has() bool {
	return 0 < len(zs)
}

// HasSecant returns true when there are secant intersections, i.e. the curves intersect and cross (they cut).
func (zs Intersections) HasSecant() bool {
	for _, z := range zs {
		if !z.Tangent {
			return true
		}
	}
	return false
}

// HasTangent returns true when there are tangent intersections, i.e. the curves intersect but don't cross (they touch).
func (zs Intersections) HasTangent() bool {
	for _, z := range zs {
		if z.Tangent {
			return true
		}
	}
	return false
}

func (zs Intersections) add(pos Point, ta, tb, dira, dirb float64, tangent bool) Intersections {
	ta = math.Max(0.0, math.Min(1.0, ta))
	tb = math.Max(0.0, math.Min(1.0, tb))
	if Equal(ta, 0.0) {
		ta = 0.0
	} else if Equal(ta, 1.0) {
		ta = 1.0
	}
	if Equal(tb, 0.0) {
		tb = 0.0
	} else if Equal(tb, 1.0) {
		tb = 1.0
	}
	return append(zs, Intersection{pos, [2]float64{ta, tb}, [2]float64{dira, dirb}, tangent})
}

// https://www.geometrictools.com/GTE/Mathematics/IntrLine2Line2.h
func intersectionLineLine(zs Intersections, a0, a1, b0, b1 Point) Intersections {
	if a0.Equals(a1) || b0.Equals(b1) {
		return zs // zero-length Close
	}

	da := a1.Sub(a0)
	db := b1.Sub(b0)
	angle0 := da.Angle()
	angle1 := db.Angle()
	if angleEqual(angle0, angle1) || angleEqual(angle0, angle1+math.Pi) {
		// parallel
		if Equal(da.PerpDot(b1.Sub(a0)), 0.0) {
			// aligned, rotate to x-axis
			a := a0.Rot(-angle0, Point{}).X
			b := a1.Rot(-angle0, Point{}).X
			c := b0.Rot(-angle0, Point{}).X
			d := b1.Rot(-angle0, Point{}).X
			if Interval(a, c, d) && Interval(b, c, d) || Interval(a, d, c) && Interval(b, d, c) {
				// a-b in c-d or a-b == c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), angle0, angle1, true)
				zs = zs.add(a1, 1.0, (b-c)/(d-c), angle0, angle1, true)
			} else if Interval(c, a, b) && Interval(d, a, b) {
				// c-d in a-b
				zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
			} else if Interval(a, c, d) || Interval(a, d, c) {
				// a in c-d
				zs = zs.add(a0, 0.0, (a-c)/(d-c), angle0, angle1, true)
				if a < d-Epsilon {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
				} else if a < c-Epsilon {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				}
			} else if Interval(b, c, d) || Interval(b, d, c) {
				// b in c-d
				if c < b-Epsilon {
					zs = zs.add(b0, (c-a)/(b-a), 0.0, angle0, angle1, true)
				} else if d < b-Epsilon {
					zs = zs.add(b1, (d-a)/(b-a), 1.0, angle0, angle1, true)
				}
				zs = zs.add(a1, 1.0, (b-c)/(d-c), angle0, angle1, true)
			}
		}
		return zs
	} else if a1.Equals(b0) {
		// handle common cases with endpoints to avoid numerical issues
		zs = zs.add(a1, 1.0, 0.0, angle0, angle1, true)
		return zs
	} else if a0.Equals(b1) {
		// handle common cases with endpoints to avoid numerical issues
		zs = zs.add(a0, 0.0, 1.0, angle0, angle1, true)
		return zs
	}

	div := da.PerpDot(db)
	ta := db.PerpDot(a0.Sub(b0)) / div
	tb := da.PerpDot(a0.Sub(b0)) / div
	if Interval(ta, 0.0, 1.0) && Interval(tb, 0.0, 1.0) {
		tangent := Equal(ta, 0.0) || Equal(ta, 1.0) || Equal(tb, 0.0) || Equal(tb, 1.0)
		zs = zs.add(a0.Interpolate(a1, ta), ta, tb, da.Angle(), db.Angle(), tangent)
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineQuad(zs Intersections, l0, l1, p0, p1, p2 Point) Intersections {
	if l0.Equals(l1) {
		return zs // zero-length Close
	}

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
				endpoint := Equal(root, 0.0) || Equal(root, 1.0) || Equal(s, 0.0) || Equal(s, 1.0)
				if endpoint {
					// deviate angle slightly at endpoint when aligned to properly set Into
					deriv2 := quadraticBezierDeriv2(p0, p1, p2)
					if (0.0 <= deriv.PerpDot(deriv2)) == (Equal(root, 0.0) || !Equal(root, 1.0) && Equal(s, 0.0)) {
						dirb += Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
					}
					dirb = angleNorm(dirb)
				}
				zs = zs.add(pos, s, root, dira, dirb, endpoint || Equal(A.Dot(deriv), 0.0))
			}
		}
	}
	return zs
}

// https://www.particleincell.com/2013/cubic-line-intersection/
func intersectionLineCube(zs Intersections, l0, l1, p0, p1, p2, p3 Point) Intersections {
	if l0.Equals(l1) {
		return zs // zero-length Close
	}

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
				tangent := Equal(A.Dot(deriv), 0.0)
				endpoint := Equal(root, 0.0) || Equal(root, 1.0) || Equal(s, 0.0) || Equal(s, 1.0)
				if endpoint {
					// deviate angle slightly at endpoint when aligned to properly set Into
					deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
					if (0.0 <= deriv.PerpDot(deriv2)) == (Equal(root, 0.0) || !Equal(root, 1.0) && Equal(s, 0.0)) {
						dirb += Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
					} else {
						dirb -= Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
					}
				} else if angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi) {
					// directions are parallel but the paths do cross (inflection point)
					// TODO: test better
					deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
					if Equal(deriv2.X, 0.0) && Equal(deriv2.Y, 0.0) {
						deriv3 := cubicBezierDeriv3(p0, p1, p2, p3, root)
						if 0.0 < deriv.PerpDot(deriv3) {
							dirb += Epsilon * 2.0
						} else {
							dirb -= Epsilon * 2.0
						}
						dirb = angleNorm(dirb)
						tangent = false
					}
				}
				zs = zs.add(pos, s, root, dira, dirb, endpoint || tangent)
			}
		}
	}
	return zs
}

// handle line-arc intersections and their peculiarities regarding angles
func addLineArcIntersection(zs Intersections, pos Point, dira, dirb, t, t0, t1, angle, theta0, theta1 float64, tangent bool) Intersections {
	if theta0 <= theta1 {
		angle = theta0 - Epsilon + angleNorm(angle-theta0+Epsilon)
	} else {
		angle = theta1 - Epsilon + angleNorm(angle-theta1+Epsilon)
	}
	endpoint := Equal(t, t0) || Equal(t, t1) || Equal(angle, theta0) || Equal(angle, theta1)
	if endpoint {
		// deviate angle slightly at endpoint when aligned to properly set Into
		if (theta0 <= theta1) == (Equal(angle, theta0) || !Equal(angle, theta1) && Equal(t, t0)) {
			dirb += Epsilon * 2.0 // t=0 and CCW, or t=1 and CW
		} else {
			dirb -= Epsilon * 2.0 // t=0 and CW, or t=1 and CCW
		}
		dirb = angleNorm(dirb)
	}

	// snap segment parameters to 0.0 and 1.0 to avoid numerical issues
	var s float64
	if Equal(t, t0) {
		t = 0.0
	} else if Equal(t, t1) {
		t = 1.0
	} else {
		t = (t - t0) / (t1 - t0)
	}
	if Equal(angle, theta0) {
		s = 0.0
	} else if Equal(angle, theta1) {
		s = 1.0
	} else {
		s = (angle - theta0) / (theta1 - theta0)
	}
	return zs.add(pos, t, s, dira, dirb, endpoint || tangent)
}

// https://www.geometrictools.com/GTE/Mathematics/IntrLine2Circle2.h
func intersectionLineCircle(zs Intersections, l0, l1, center Point, radius, theta0, theta1 float64) Intersections {
	if l0.Equals(l1) {
		return zs // zero-length Close
	}

	// solve l0 + t*(l1-l0) = P + t*D = X  (line equation)
	// and |X - center| = |X - C| = R = radius  (circle equation)
	// by substitution and squaring: |P + t*D - C|^2 = R^2
	// giving: D^2 t^2 + 2D(P-C) t + (P-C)^2-R^2 = 0
	dir := l1.Sub(l0)
	diff := l0.Sub(center) // P-C
	length := dir.Length()
	D := dir.Div(length)

	// we normalise D to be of length 1, so that the roots are in [0,length]
	a := 1.0
	b := 2.0 * D.Dot(diff)
	c := diff.Dot(diff) - radius*radius

	// find solutions for t ∈ [0,1], the parameter along the line's path
	roots := []float64{}
	r0, r1 := solveQuadraticFormula(a, b, c)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) && !Equal(r0, r1) {
			roots = append(roots, r1)
		}
	}

	// handle common cases with endpoints to avoid numerical issues
	// snap closest root to path's start or end
	if 0 < len(roots) {
		if pos := l0.Sub(center); Equal(pos.Length(), radius) {
			if len(roots) == 1 || math.Abs(roots[0]) < math.Abs(roots[1]) {
				roots[0] = 0.0
			} else {
				roots[1] = 0.0
			}
		}
		if pos := l1.Sub(center); Equal(pos.Length(), radius) {
			if len(roots) == 1 || math.Abs(roots[0]-length) < math.Abs(roots[1]-length) {
				roots[0] = length
			} else {
				roots[1] = length
			}
		}
	}

	// add intersections
	dira := dir.Angle()
	tangent := len(roots) == 1
	for _, root := range roots {
		pos := diff.Add(dir.Mul(root / length))
		angle := math.Atan2(pos.Y*radius, pos.X*radius)
		if Interval(root, 0.0, length) && angleBetween(angle, theta0, theta1) {
			pos = center.Add(pos)
			dirb := ellipseDeriv(radius, radius, 0.0, theta0 <= theta1, angle).Angle()
			zs = addLineArcIntersection(zs, pos, dira, dirb, root, 0.0, length, angle, theta0, theta1, tangent)
		}
	}
	return zs
}

func intersectionLineEllipse(zs Intersections, l0, l1, center, radius Point, phi, theta0, theta1 float64) Intersections {
	if Equal(radius.X, radius.Y) {
		return intersectionLineCircle(zs, l0, l1, center, radius.X, theta0, theta1)
	} else if l0.Equals(l1) {
		return zs // zero-length Close
	}

	// TODO: needs more testing
	// TODO: intersection inconsistency due to numerical stability in finding tangent collisions for subsequent paht segments (line -> ellipse), or due to the endpoint of a line not touching with another arc, but the subsequent segment does touch with its starting point
	dira := l1.Sub(l0).Angle()

	// we take the ellipse center as the origin and counter-rotate by phi
	l0 = l0.Sub(center).Rot(-phi, Origin)
	l1 = l1.Sub(center).Rot(-phi, Origin)

	// line: cx + dy + e = 0
	c := l0.Y - l1.Y
	d := l1.X - l0.X
	e := l0.PerpDot(l1)

	// follow different code paths when line is mostly horizontal or vertical
	horizontal := math.Abs(c) <= math.Abs(d)

	// ellipse: x^2/a + y^2/b = 1
	a := radius.X * radius.X
	b := radius.Y * radius.Y

	// rewrite as a polynomial by substituting x or y to obtain:
	// At^2 + Bt + C = 0, with t either x (horizontal) or y (!horizontal)
	var A, B, C float64
	A = a*c*c + b*d*d
	if horizontal {
		B = 2.0 * a * c * e
		C = a*e*e - a*b*d*d
	} else {
		B = 2.0 * b * d * e
		C = b*e*e - a*b*c*c
	}

	// find solutions
	roots := []float64{}
	r0, r1 := solveQuadraticFormula(A, B, C)
	if !math.IsNaN(r0) {
		roots = append(roots, r0)
		if !math.IsNaN(r1) && !Equal(r0, r1) {
			roots = append(roots, r1)
		}
	}

	for _, root := range roots {
		// get intersection position with center as origin
		var x, y, t0, t1 float64
		if horizontal {
			x = root
			y = -e/d - c*root/d
			t0 = l0.X
			t1 = l1.X
		} else {
			x = -e/c - d*root/c
			y = root
			t0 = l0.Y
			t1 = l1.Y
		}

		tangent := Equal(root, 0.0)
		angle := math.Atan2(y*radius.X, x*radius.Y)
		if Interval(root, t0, t1) && angleBetween(angle, theta0, theta1) {
			pos := Point{x, y}.Rot(phi, Origin).Add(center)
			dirb := ellipseDeriv(radius.X, radius.Y, phi, theta0 <= theta1, angle).Angle()
			zs = addLineArcIntersection(zs, pos, dira, dirb, root, t0, t1, angle, theta0, theta1, tangent)
		}
	}
	return zs
}

func intersectionEllipseEllipse(zs Intersections, c0, r0 Point, phi0, thetaStart0, thetaEnd0 float64, c1, r1 Point, phi1, thetaStart1, thetaEnd1 float64) Intersections {
	// TODO: needs more testing
	if !Equal(r0.X, r0.Y) || !Equal(r1.X, r1.Y) {
		panic("not handled") // ellipses
	}

	arcAngle := func(theta float64, sweep bool) float64 {
		theta += math.Pi / 2.0
		if !sweep {
			theta -= math.Pi
		}
		return angleNorm(theta)
	}

	dtheta0 := thetaEnd0 - thetaStart0
	thetaStart0 = angleNorm(thetaStart0 + phi0)
	thetaEnd0 = thetaStart0 + dtheta0

	dtheta1 := thetaEnd1 - thetaStart1
	thetaStart1 = angleNorm(thetaStart1 + phi1)
	thetaEnd1 = thetaStart1 + dtheta1

	if c0.Equals(c1) && r0.Equals(r1) {
		// parallel
		tOffset1 := 0.0
		dirOffset1 := 0.0
		if (0.0 <= dtheta0) != (0.0 <= dtheta1) {
			thetaStart1, thetaEnd1 = thetaEnd1, thetaStart1 // keep order on first arc
			dirOffset1 = math.Pi
			tOffset1 = 1.0
		}

		// will add either 1 (when touching) or 2 (when overlapping) intersections
		if t := angleTime(thetaStart0, thetaStart1, thetaEnd1); Interval(t, 0.0, 1.0) {
			// ellipse0 starts within/on border of ellipse1
			dir := arcAngle(thetaStart0, 0.0 <= dtheta0)
			pos := EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaStart0)
			zs = zs.add(pos, 0.0, math.Abs(t-tOffset1), dir, angleNorm(dir+dirOffset1), true)
		}
		if t := angleTime(thetaStart1, thetaStart0, thetaEnd0); IntervalExclusive(t, 0.0, 1.0) {
			// ellipse1 starts within ellipse0
			dir := arcAngle(thetaStart1, 0.0 <= dtheta0)
			pos := EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaStart1)
			zs = zs.add(pos, t, tOffset1, dir, angleNorm(dir+dirOffset1), true)
		}
		if t := angleTime(thetaEnd1, thetaStart0, thetaEnd0); IntervalExclusive(t, 0.0, 1.0) {
			// ellipse1 ends within ellipse0
			dir := arcAngle(thetaEnd1, 0.0 <= dtheta0)
			pos := EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaEnd1)
			zs = zs.add(pos, t, 1.0-tOffset1, dir, angleNorm(dir+dirOffset1), true)
		}
		if t := angleTime(thetaEnd0, thetaStart1, thetaEnd1); Interval(t, 0.0, 1.0) {
			// ellipse0 ends within/on border of ellipse1
			dir := arcAngle(thetaEnd0, 0.0 <= dtheta0)
			pos := EllipsePos(r0.X, r0.Y, 0.0, c0.X, c0.Y, thetaEnd0)
			zs = zs.add(pos, 1.0, math.Abs(t-tOffset1), dir, angleNorm(dir+dirOffset1), true)
		}
		return zs
	}

	// https://math.stackexchange.com/questions/256100/how-can-i-find-the-points-at-which-two-circles-intersect
	// https://gist.github.com/jupdike/bfe5eb23d1c395d8a0a1a4ddd94882ac
	R := c0.Sub(c1).Length()
	if R < math.Abs(r0.X-r1.X) || r0.X+r1.X < R {
		return zs
	}
	R2 := R * R

	k := r0.X*r0.X - r1.X*r1.X
	a := 0.5
	b := 0.5 * k / R2
	c := 0.5 * math.Sqrt(2.0*(r0.X*r0.X+r1.X*r1.X)/R2-k*k/(R2*R2)-1.0)

	mid := c1.Sub(c0).Mul(a + b)
	dev := Point{c1.Y - c0.Y, c0.X - c1.X}.Mul(c)

	tangent := dev.Equals(Point{})
	anglea0 := mid.Add(dev).Angle()
	anglea1 := c0.Sub(c1).Add(mid).Add(dev).Angle()
	ta0 := angleTime(anglea0, thetaStart0, thetaEnd0)
	ta1 := angleTime(anglea1, thetaStart1, thetaEnd1)
	if Interval(ta0, 0.0, 1.0) && Interval(ta1, 0.0, 1.0) {
		dir0 := arcAngle(anglea0, 0.0 <= dtheta0)
		dir1 := arcAngle(anglea1, 0.0 <= dtheta1)
		endpoint := Equal(ta0, 0.0) || Equal(ta0, 1.0) || Equal(ta1, 0.0) || Equal(ta1, 1.0)
		zs = zs.add(c0.Add(mid).Add(dev), ta0, ta1, dir0, dir1, tangent || endpoint)
	}

	if !tangent {
		angleb0 := mid.Sub(dev).Angle()
		angleb1 := c0.Sub(c1).Add(mid).Sub(dev).Angle()
		tb0 := angleTime(angleb0, thetaStart0, thetaEnd0)
		tb1 := angleTime(angleb1, thetaStart1, thetaEnd1)
		if Interval(tb0, 0.0, 1.0) && Interval(tb1, 0.0, 1.0) {
			dir0 := arcAngle(angleb0, 0.0 <= dtheta0)
			dir1 := arcAngle(angleb1, 0.0 <= dtheta1)
			endpoint := Equal(tb0, 0.0) || Equal(tb0, 1.0) || Equal(tb1, 0.0) || Equal(tb1, 1.0)
			zs = zs.add(c0.Add(mid).Sub(dev), tb0, tb1, dir0, dir1, endpoint)
		}
	}
	return zs
}

// TODO: bezier-bezier intersection
// TODO: bezier-ellipse intersection

// For Bézier-Bézier interesections:
// see T.W. Sederberg, "Computer Aided Geometric Design", 2012
// see T.W. Sederberg and T. Nishita, "Curve intersection using Bézier clipping", 1990
// see T.W. Sederberg and S.R. Parry, "Comparison of three curve intersection algorithms", 1986

func intersectionRayLine(a0, a1, b0, b1 Point) (Point, bool) {
	da := a1.Sub(a0)
	db := b1.Sub(b0)
	div := da.PerpDot(db)
	if Equal(div, 0.0) {
		// parallel
		return Point{}, false
	}

	tb := da.PerpDot(a0.Sub(b0)) / div
	if Interval(tb, 0.0, 1.0) {
		return b0.Interpolate(b1, tb), true
	}
	return Point{}, false
}

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
