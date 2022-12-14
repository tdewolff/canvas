package canvas

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Paths are cut at the intersections between P and Q. The intersections are put into a doubly linked list with paths going forward and backward over P and Q. Depending on the boolean operation we should choose the right cut. Note that there can be circular loops when choosing cuts based on a condition, so we should take care to visit all intersections. Additionally, if path P or path Q contain subpaths with a different winding, we will first combine the subpaths so to remove all subpath intersections.

// get RHS normal vector at path segment extreme
func segmentNormal(start Point, d []float64, t float64) Point {
	if d[0] == LineToCmd || d[0] == CloseCmd {
		return Point{d[1], d[2]}.Sub(start).Rot90CW().Norm(Epsilon)
	} else if d[0] == QuadToCmd {
		cp := Point{d[1], d[2]}
		end := Point{d[3], d[4]}
		return quadraticBezierNormal(start, cp, end, t, Epsilon)
	} else if d[0] == CubeToCmd {
		cp1 := Point{d[1], d[2]}
		cp2 := Point{d[3], d[4]}
		end := Point{d[5], d[6]}
		return cubicBezierNormal(start, cp1, cp2, end, t, Epsilon)
	} else if d[0] == ArcToCmd {
		rx, ry, phi := d[1], d[2], d[3]
		large, sweep := toArcFlags(d[4])
		_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, d[5], d[6])
		return ellipseNormal(rx, ry, phi, sweep, theta0+t*(theta1-theta0), Epsilon)
	}
	return Point{}
}

// get points in the interior of p close to the first MoveTo
func (p *Path) interiorPoint() Point {
	if len(p.d) <= 4 || len(p.d) <= 4+cmdLen(p.d[4]) {
		panic("path too small or empty")
	}

	p0 := Point{p.d[1], p.d[2]}
	n0 := segmentNormal(p0, p.d[4:], 0.0)

	i := len(p.d) - 4
	p1 := Point{p.d[i-3], p.d[i-2]}
	if p0.Equals(p1) {
		// CloseCmd is an empty segment
		i -= cmdLen(p.d[i-1])
		p1 = Point{p.d[i-3], p.d[i-2]}
	}
	n1 := segmentNormal(p1, p.d[i:], 1.0)

	n := n0.Add(n1).Norm(Epsilon)
	if p.CCW() { // TODO what about subpaths
		n = n.Neg()
	}
	return p0.Add(n)
}

// returns true if p is inside q or equivalent to q, paths may not intersect
func (p *Path) inside(q *Path) bool {
	if len(p.d) <= 4 || len(p.d) <= 4+cmdLen(p.d[4]) || len(q.d) <= 4 || len(q.d) <= 4+cmdLen(q.d[4]) {
		return false
	}
	offset := p.interiorPoint()
	return q.Interior(offset.X, offset.Y, NonZero)
}

// Contains returns true if path q is contained within path p, i.e. path q is inside path p and both paths have no intersections (but may touch).
func (p *Path) Contains(q *Path) bool {
	if q.inside(p) {
		return len(intersectionNodes(p, q)) == 0
	}
	return false
}

// And returns the boolean path operation of path p and q.
func (p *Path) And(q *Path) *Path {
	// remove subpath intersections
	p = p.Combine()
	q = q.Combine()

	zs := intersectionNodes(p, q)
	if len(zs) == 0 {
		// paths are not intersecting
		if p.inside(q) {
			return p
		} else if q.inside(p) {
			return q
		}
		return &Path{} // paths have no overlap
	}
	return boolean(pathOpAnd, zs, p.CCW(), q.CCW())
}

// Or returns the boolean path operation of path p and q.
func (p *Path) Or(q *Path) *Path {
	// remove subpath intersections
	p = p.Combine()
	q = q.Combine()

	zs := intersectionNodes(p, q)
	if len(zs) == 0 {
		// paths are not intersecting
		if p.inside(q) {
			return q
		} else if q.inside(p) {
			return p
		} else {
			return p.Append(q) // paths have no overlap
		}
		return &Path{} // paths have no overlap
	}
	return boolean(pathOpOr, zs, p.CCW(), q.CCW())
}

// Xor returns the boolean path operation of path p and q.
func (p *Path) Xor(q *Path) *Path {
	// remove subpath intersections
	p = p.Combine()
	q = q.Combine()

	zs := intersectionNodes(p, q)
	if len(zs) == 0 {
		// paths are not intersecting
		pInQ := p.inside(q)
		qInP := q.inside(p)
		if pInQ && qInP {
			return &Path{} // equal
		} else if pInQ {
			return q.Append(p.Reverse())
		} else if qInP {
			return p.Append(q.Reverse())
		} else {
			return p.Append(q) // paths have no overlap
		}
		return &Path{} // paths have no overlap
	}
	return boolean(pathOpXor, zs, p.CCW(), q.CCW())
}

// Not returns the boolean path operation of path p and q.
func (p *Path) Not(q *Path) *Path {
	// remove subpath intersections
	p = p.Combine()
	q = q.Combine()

	zs := intersectionNodes(p, q)
	if len(zs) == 0 {
		// paths are not intersecting
		pInQ := p.inside(q)
		qInP := q.inside(p)
		if pInQ && qInP {
			return &Path{} // equal
		} else if pInQ {
			return &Path{}
		} else if qInP {
			return p.Append(q.Reverse())
		} else {
			return p // paths have no overlap
		}
		return &Path{} // paths have no overlap
	}
	return boolean(pathOpNot, zs, p.CCW(), q.CCW())
}

// DivideBy returns division of path p by path q at the intersections.
func (p *Path) DivideBy(q *Path) ([]*Path, []*Path) {
	// TODO
	return nil, nil
}

// Combine combines the subpaths of path p, removing all intersections between the subpaths (but leaving self-intersections).
func (p *Path) Combine() *Path {
	if p.Empty() {
		return p
	}
	ps := p.Split()
	p = ps[0]
	for _, q := range ps[1:] {
		zs := intersectionNodes(p, q)
		ccwA, ccwB := p.CCW(), q.CCW()
		if ccwA == ccwB {
			// OR
			if len(zs) == 0 {
				// paths are not intersecting
				if p.inside(q) {
					p = q
				} else if q.inside(p) {
				} else {
					p = p.Append(q) // paths have no overlap
				}
				continue
			}
			p = boolean(pathOpOr, zs, ccwA, ccwB)
		} else {
			// XOR
			if len(zs) == 0 {
				// paths are not intersecting
				pInQ := p.inside(q)
				qInP := q.inside(p)
				if pInQ && qInP {
					// equal
				} else {
					p = p.Append(q)
				}
				continue
			}
			p = boolean(pathOpXor, zs, ccwA, ccwB)
		}
	}
	return p
}

type pathOp int

const (
	pathOpAnd pathOp = iota
	pathOpOr  pathOp = iota
	pathOpXor pathOp = iota
	pathOpNot pathOp = iota
)

func boolean(op pathOp, zs []*intersectionNode, ccwA, ccwB bool) *Path {
	directions := []bool{true}
	startInwards, invertA, invertB := false, false, false
	if op == pathOpAnd {
		startInwards, invertA = true, true
	} else if op == pathOpOr {
		invertB = true
	} else if op == pathOpXor {
		directions = []bool{true, false}
	} else if op == pathOpNot {
	}

	R := &Path{}
	visited := map[bool]map[int]bool{ // per direction
		true:  map[int]bool{},
		false: map[int]bool{},
	}
	for _, z0 := range zs {
		for _, direction := range directions {
			if !visited[direction][z0.i] {
				r := &Path{}
				gotoB := startInwards == (ccwB == z0.BintoA)
				for z := z0; ; {
					visited[direction][z.i] = true
					if gotoB {
						if invertB != direction == (ccwA == z.BintoA) {
							r = r.Join(z.b)
							z = z.nextB
						} else {
							r = r.Join(z.prevB.b.Reverse())
							z = z.prevB
						}
					} else {
						if invertA != direction == (ccwB == z.BintoA) {
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
	}
	return R
}

// Cut returns the parts of path p and path q cut by the other at intersections.
func (p *Path) Cut(q *Path) ([]*Path, []*Path) {
	zs := intersectionNodes(p, q)
	if len(zs) == 0 {
		return []*Path{p}, []*Path{q}
	}

	ps, qs := []*Path{}, []*Path{}
	visited := map[int]bool{}
	for _, z0 := range zs {
		if !visited[z0.i] {
			for z := z0; ; {
				visited[z.i] = true
				ps = append(ps, z.a)
				z = z.nextA
				if z.i == z0.i {
					break
				}
			}
		}
	}
	visited = map[int]bool{}
	for _, z0 := range zs {
		if !visited[z0.i] {
			for z := z0; ; {
				visited[z.i] = true
				qs = append(qs, z.b)
				z = z.nextB
				if z.i == z0.i {
					break
				}
			}
		}
	}
	return ps, qs
}

type intersectionNode struct {
	i int // intersection index in path A
	intersection
	prevA, nextA *intersectionNode
	prevB, nextB *intersectionNode

	// towards next intersection
	a, b *Path
}

func (z *intersectionNode) String() string {
	return fmt.Sprintf("(%v %v A=[%v→,→%v] B=[%v→,→%v])", z.i, z.intersection, z.prevA.i, z.nextA.i, z.prevB.i, z.nextB.i)
}

// get intersections for paths p and q sorted for both
func intersectionNodes(p, q *Path) []*intersectionNode {
	if !p.Closed() {
		p.Close()
	}
	if !q.Closed() {
		q.Close()
	}

	Zs := p.Intersections(q)
	if len(Zs) == 0 {
		return nil
	} else if len(Zs)%2 != 0 {
		panic("number of intersections must be even")
	}

	zs := make([]*intersectionNode, len(Zs))
	for i, z := range Zs {
		zs[i] = &intersectionNode{
			i:            i,
			intersection: z,
			a:            &Path{},
			b:            &Path{},
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
		if j < len(zs) && seg == zs[j].SegA {
			// segment has an intersection, cut it up and append first part to prev intersection
			p0, p1 := cutPathSegment(Point{p.d[i-3], p.d[i-2]}, p.d[i:i+cmdLen(cmd)], zs[j].TA)
			if !p0.Empty() {
				cur = append(cur, p0.d[4:]...)
			}

			for j+1 < len(zs) && seg == zs[j+1].SegA {
				// next cut is on the same segment, find new t after the first cut and set path
				if first == nil {
					first = cur // take aside the path to the first intersection to later append it
				} else {
					zs[j-1].a.d = cur
					zs[j-1].nextA = zs[j]
					zs[j].prevA = zs[j-1]
				}
				j++
				t := (zs[j].TA - zs[j-1].TA) / (1.0 - zs[j-1].TA)
				if !p1.Empty() {
					p0, p1 = cutPathSegment(Point{p1.d[1], p1.d[2]}, p1.d[4:], t)
				} else {
					p0 = &Path{}
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
			// segment has no intersection, add to previous intersection
			cur = append(cur, p.d[i:i+cmdLen(cmd)]...)
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
		if j < len(zs) && seg == zs[idxs[j]].SegB {
			// segment has an intersection, cut it up and append first part to prev intersection
			p0, p1 := cutPathSegment(Point{q.d[i-3], q.d[i-2]}, q.d[i:i+cmdLen(cmd)], zs[idxs[j]].TB)
			if !p0.Empty() {
				cur = append(cur, p0.d[4:]...)
			}

			for j+1 < len(zs) && seg == zs[idxs[j+1]].SegB {
				// next cut is on the same segment, find new t after the first cut and set path
				if first == nil {
					first = cur // take aside the path to the first intersection to later append it
				} else {
					zs[idxs[j-1]].b.d = cur
					zs[idxs[j-1]].nextB = zs[idxs[j]]
					zs[idxs[j]].prevB = zs[idxs[j-1]]
				}
				j++
				t := (zs[idxs[j]].TB - zs[idxs[j-1]].TB) / (1.0 - zs[idxs[j-1]].TB)
				if !p1.Empty() {
					p0, p1 = cutPathSegment(Point{p1.d[1], p1.d[2]}, p1.d[4:], t)
				} else {
					p0 = &Path{}
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
			// segment has no intersection, add to previous intersection
			cur = append(cur, q.d[i:i+cmdLen(cmd)]...)
		}
		i += cmdLen(cmd)
		seg++
	}
	if first != nil {
		zs[idxs[len(zs)-1]].b.d = append(cur, first[4:]...)
		zs[idxs[len(zs)-1]].nextB = zs[idxs[j0]]
		zs[idxs[j0]].prevB = zs[idxs[len(zs)-1]]
	}
	return zs
}

func cutPathSegment(start Point, d []float64, t float64) (*Path, *Path) {
	p0, p1 := &Path{}, &Path{}
	if Equal(t, 0.0) {
		p1.MoveTo(start.X, start.Y)
		p1.d = append(p1.d, d...)
		return p0, p1
	} else if Equal(t, 1.0) {
		p0.MoveTo(start.X, start.Y)
		p0.d = append(p0.d, d...)
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
	zs := collisions(p, q, false)
	return 0 < len(zs)
}

// Intersections for path p by path q, sorted for path p.
func (p *Path) Intersections(q *Path) intersections {
	return collisions(p, q, false)
}

// Touches returns true if path p and path q touch or intersect.
func (p *Path) Touches(q *Path) bool {
	zs := collisions(p, q, true)
	return 0 < len(zs)
}

// Collisions (secants/intersections and tangents/touches) for path p by path q, sorted for path p.
func (p *Path) Collisions(q *Path) intersections {
	return collisions(p, q, true)
}

func collisions(p, q *Path, keepTangents bool) intersections {
	// TODO: collision code is ugly
	// TODO: uses O(N^2), try sweep line or bently-ottman to reduce to O((N+K) log N)
	Zs := intersections{}
	var pI, qI int
	var pStart, qStart Point
	for i := 0; i < len(p.d); {
		pLen := cmdLen(p.d[i])
		if p.d[i] != MoveToCmd {
			qI = 0
			qStart = Origin
			for j := 0; j < len(q.d); {
				qLen := cmdLen(q.d[j])
				if q.d[j] != MoveToCmd {
					Zs = Zs.appendSegment(pI, pStart, p.d[i:i+pLen], qI, qStart, q.d[j:j+qLen])
				}
				j += qLen
				qStart = Point{q.d[j-3], q.d[j-2]}
				qI++
			}
		}
		i += pLen
		pStart = Point{p.d[i-3], p.d[i-2]}
		pI++
	}
	sort.Stable(intersectionASort{Zs}) // needed when q intersects a segment in p from high to low T

	// remove duplicate tangent collisions at segment endpoints: either 4 degenerate collisions
	// when for both path p and path q the endpoints coincide, or 2 degenerate collisions when
	// an endpoint collides within a segment, for each parallel segment in between an additional 2 degenerate collisions are created
	// note that collisions between segments of the same path are never generated
	i0 := -1
	zs := intersections{}
Main:
	for i := 0; i < len(Zs); i++ {
		z0 := Zs[i]
		if !z0.Tangent {
			zs = append(zs, z0)
		} else if !Equal(z0.TA, 0.0) && !Equal(z0.TB, 0.0) && !Equal(z0.TA, 1.0) && !Equal(z0.TB, 1.0) {
			// regular tangent that is not at segment extreme, does not intersect
			if keepTangents {
				zs = append(zs, z0)
			}
		} else if !Equal(z0.TA, 0.0) {
			ends := 0
			qReverse := Equal(z0.TB, 0.0)
			if Equal(z0.TA, 1.0) {
				ends++
			}
			if qReverse || Equal(z0.TB, 1.0) {
				ends++
			}

			n := 1
			j := (i + 1) % len(Zs)
			if i0 == -1 {
				i0 = i
			}
			for ; j != i0; j = (j + 1) % len(Zs) {
				z := Zs[j]
				if ends == 0 {
					break
				} else if !z.Tangent || !Equal(z.TA, 0.0) && !Equal(z.TB, 0.0) && !Equal(z.TA, 1.0) && !Equal(z.TB, 1.0) {
					continue Main
				}
				if Equal(z.TA, 0.0) {
					ends--
				} else if Equal(z.TA, 1.0) {
					ends++
				}
				if Equal(z.TB, 0.0) {
					if qReverse {
						ends++
					} else {
						ends--
					}
				} else if Equal(z.TB, 1.0) {
					if qReverse {
						ends--
					} else {
						ends++
					}
				}
				n++
			}
			if n == 1 {
				if keepTangents {
					zs = append(zs, z0)
				}
				continue Main
			}

			z1 := Zs[(i+n-1)%len(Zs)]
			if 2 < n {
				// for intersections on endpoints we need to check incoming/outgoing angles
				theta0 := angleNorm(z0.DirA + math.Pi)
				theta1 := theta0 - angleNorm(theta0-z1.DirA)
				dirb0, dirb1 := z0.DirB, z1.DirB
				if qReverse {
					dirb0, dirb1 = z1.DirB, z0.DirB
				}
				z0.BintoA = !angleBetweenExclusive(dirb0+math.Pi, theta0, theta1)
				z1.BintoA = angleBetween(dirb1, theta0, theta1)
			}
			z1.Tangent = z0.BintoA != z1.BintoA
			if !z1.Tangent || keepTangents {
				if z1.SegA == 1 {
					// is at start of path P
					zs = append([]intersection{z1}, zs...)
				} else {
					zs = append(zs, z1)
				}
			}
			i += n - 1
		}
	}
	return zs
}

// intersect for path segments a and b, starting at a0 and b0
func (zs intersections) appendSegment(aSeg int, a0 Point, a []float64, bSeg int, b0 Point, b []float64) intersections {
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
		for i, _ := range zs[n:] {
			zs[n+i].TA, zs[n+i].TB = zs[n+i].TB, zs[n+i].TA
			zs[n+i].SegA, zs[n+i].SegB = aSeg, bSeg
		}
	} else {
		for i, _ := range zs[n:] {
			zs[n+i].SegA, zs[n+i].SegB = aSeg, bSeg
		}
	}
	return zs
}

// see https://github.com/signavio/svg-intersections
// see https://github.com/w8r/bezier-intersect
// see https://cs.nyu.edu/exact/doc/subdiv1.pdf

// Intersections amongst the combinations between line, quad, cube, elliptical arcs. We consider four cases: the curves do not cross nor touch (intersections is empty), the curves intersect (and cross), the curves intersect tangentially (touching), or the curves are identical (or parallel in the case of two lines). In the last case we say there are no intersections. As all curves are segments, it is considered a secant intersection when the segments touch but "intent to" cut at their ends (i.e. when position equals to 0 or 1 for either segment).

type intersection struct {
	Point
	SegA, SegB int
	TA, TB     float64 // position along segment in [0,1]
	DirA, DirB float64 // angle of direction along segment TODO: remove?
	BintoA     bool
	Tangent    bool // tangential, i.e. touching/non-crossing
}

func (z intersection) Equals(o intersection) bool {
	return z.Point.Equals(o.Point) && z.SegA == o.SegA && z.SegB == o.SegB && Equal(z.TA, o.TA) && Equal(z.TB, o.TB) && angleEqual(z.DirA, o.DirA) && angleEqual(z.DirB, o.DirB) && z.Tangent == o.Tangent
}

func (z intersection) String() string {
	s := fmt.Sprintf("pos={%.3g,%.3g} seg={%d,%d} t={%.3g,%.3g} dir={%g°,%g°}", z.Point.X, z.Point.Y, z.SegA, z.SegB, z.TA, z.TB, angleNorm(z.DirA)*180.0/math.Pi, angleNorm(z.DirB)*180.0/math.Pi)
	if z.BintoA {
		s += " BintoA"
	} else {
		s += " AintoB"
	}
	if z.Tangent {
		s += " tangent"
	}
	return s
}

type intersections []intersection

// There are intersections.
func (zs intersections) Has() bool {
	return 0 < len(zs)
}

// There are secants, i.e. the curves intersect and cross (they cut).
func (zs intersections) HasSecant() bool {
	for _, z := range zs {
		if !z.Tangent {
			return true
		}
	}
	return false
}

// There are tangents, i.e. the curves intersect but don't cross (they touch).
func (zs intersections) HasTangent() bool {
	for _, z := range zs {
		if z.Tangent {
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

// sort indices of intersections for curve A
type intersectionASort struct {
	zs intersections
}

func (a intersectionASort) Len() int {
	return len(a.zs)
}

func (a intersectionASort) Swap(i, j int) {
	a.zs[i], a.zs[j] = a.zs[j], a.zs[i]
}

func (a intersectionASort) Less(i, j int) bool {
	if a.zs[i].SegA == a.zs[j].SegA {
		return a.zs[i].TA < a.zs[j].TA
	}
	return a.zs[i].SegA < a.zs[j].SegA
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
	if Equal(ta, 0.0) || Equal(tb, 0.0) || Equal(ta, 1.0) || Equal(tb, 1.0) {
		tangent = true
	}
	return append(zs, intersection{
		Point:   pos,
		TA:      ta,
		TB:      tb,
		DirA:    dira,
		DirB:    dirb,
		BintoA:  angleNorm(dirb-dira) < math.Pi,
		Tangent: tangent,
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
			angle := da.Angle()
			a := a0.Rot(-angle, Point{}).X
			b := a1.Rot(-angle, Point{}).X
			c := b0.Rot(-angle, Point{}).X
			d := b1.Rot(-angle, Point{}).X
			//if c <= a && a <= d && c <= b && b <= d {
			//	// a in b or a == b
			//	mid := (a + b) / 2.0
			//	zs = zs.add(a0.Interpolate(a1, 0.5), 0.5, (mid-c)/(d-c), true)
			//} else if a < c && c < b && a < d && d < b {
			//	// b in a
			//	mid := (c + d) / 2.0
			//	zs = zs.add(b0.Interpolate(b1, 0.5), (mid-a)/(b-a), 0.5, true)
			//} else if a <= c && c <= b {
			//	// a before b
			//	mid := (c + b) / 2.0
			//	zs = zs.add(b0.Interpolate(a1, 0.5), (mid-a)/(b-a), (mid-c)/(d-c), true)
			//} else if a <= d && d <= b {
			//	// b before a
			//	mid := (a + d) / 2.0
			//	zs = zs.add(a0.Interpolate(b1, 0.5), (mid-a)/(b-a), (mid-c)/(d-c), true)
			//}
			if c < d && Equal(b, c) {
				// a before b
				zs = zs.add(a1, 1.0, 0.0, angle, angle, true)
			} else if c < d && Equal(a, d) {
				// b before a
				zs = zs.add(a0, 0.0, 1.0, angle, angle, true)
			} else if d < c && Equal(b, d) {
				// a before b (inverted)
				zs = zs.add(a1, 1.0, 1.0, angle, -angle, true)
			} else if d < c && Equal(a, c) {
				// b (inverted) before a
				zs = zs.add(a0, 0.0, 0.0, angle, -angle, true)
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
			deriv := quadraticBezierDeriv(p0, p1, p2, root)
			dirb := deriv.Angle()
			// deviate angle slightly to distinguish between BintoA/AintoB on head-on directions
			if (Equal(root, 0.0) || Equal(root, 1.0)) && (angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi)) {
				deriv2 := quadraticBezierDeriv2(p0, p1, p2)
				if 0.0 <= deriv.PerpDot(deriv2) {
					dirb -= Epsilon // CCW
				} else {
					dirb += Epsilon // CW
				}
			}

			pos := quadraticBezierPos(p0, p1, p2, root)
			dif := A.Dot(deriv)
			if horizontal {
				if l0.X <= pos.X && pos.X <= l1.X {
					zs = zs.add(pos, (pos.X-l0.X)/(l1.X-l0.X), root, dira, dirb, Equal(dif, 0.0))
				}
			} else if l0.Y <= pos.Y && pos.Y <= l1.Y {
				zs = zs.add(pos, (pos.Y-l0.Y)/(l1.Y-l0.Y), root, dira, dirb, Equal(dif, 0.0))
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
			deriv := cubicBezierDeriv(p0, p1, p2, p3, root)
			dirb := deriv.Angle()
			// deviate angle slightly to distinguish between BintoA/AintoB on head-on directions
			if (Equal(root, 0.0) || Equal(root, 1.0)) && (angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi)) {
				deriv2 := cubicBezierDeriv2(p0, p1, p2, p3, root)
				if 0.0 <= deriv.PerpDot(deriv2) {
					dirb -= Epsilon // CCW
				} else {
					dirb += Epsilon // CW
				}
			}

			pos := cubicBezierPos(p0, p1, p2, p3, root)
			dif := A.Dot(deriv)
			if horizontal {
				if l0.X <= pos.X && pos.X <= l1.X {
					zs = zs.add(pos, (pos.X-l0.X)/(l1.X-l0.X), root, dira, dirb, Equal(dif, 0.0))
				}
			} else if l0.Y <= pos.Y && pos.Y <= l1.Y {
				zs = zs.add(pos, (pos.Y-l0.Y)/(l1.Y-l0.Y), root, dira, dirb, Equal(dif, 0.0))
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
			if (Equal(t, 0.0) || Equal(t, 1.0)) && (angleEqual(dira, dirb) || angleEqual(dira, dirb+math.Pi)) {
				if theta0 <= theta1 {
					dirb -= Epsilon // CCW
				} else {
					dirb += Epsilon // CW
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
