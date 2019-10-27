package canvas

import (
	"github.com/ByteArena/poly2tri-go"
)

func (P *Path) Tessellate() ([][3]Point, [][3]Point) {
	P = P.replace(nil, nil, ellipseToBeziers)

	simpleTriangles := [][3]Point{}
	quadTriangles := [][3]Point{}
	//for _, p := range P.Split() {
	p := P
	contour := []*poly2tri.Point{}
	var start, end Point
	for i := 0; i < len(p.d); {
		cmd := p.d[i]
		switch cmd {
		case moveToCmd, lineToCmd:
			end = Point{p.d[i+1], p.d[i+2]}
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
		case quadToCmd:
			cp := Point{p.d[i+1], p.d[i+2]}
			end = Point{p.d[i+3], p.d[i+4]}
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
			quadTriangles = append(quadTriangles, [3]Point{start, cp, end})
		case cubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			for _, quad := range cubicToQuadraticBeziers(start, cp1, cp2, end) {
				contour = append(contour, poly2tri.NewPoint(quad[2].X, quad[2].Y))
				quadTriangles = append(quadTriangles, quad)
			}
		case arcToCmd:
			panic("arcs should have been replaced")
		}
		i += cmdLen(cmd)
		start = end
	}

	swctx := poly2tri.NewSweepContext(contour, false)
	swctx.Triangulate()

	for _, tr := range swctx.GetTriangles() {
		p0 := Point{tr.Points[0].X, tr.Points[0].Y}
		p1 := Point{tr.Points[1].X, tr.Points[1].Y}
		p2 := Point{tr.Points[2].X, tr.Points[2].Y}
		simpleTriangles = append(simpleTriangles, [3]Point{p0, p1, p2})
	}
	//}
	return simpleTriangles, quadTriangles
}
