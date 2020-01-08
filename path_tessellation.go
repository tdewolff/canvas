package canvas

import (
	"github.com/ByteArena/poly2tri-go"
)

// Tessellate tessellates the path and returns the triangles that fill the path. WIP
func (p *Path) Tessellate() ([][3]Point, [][5]Point) {
	p = p.replace(nil, nil, nil, arcToCube)

	beziers := [][5]Point{}
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
			cp1, cp2 := quadraticToCubicBezier(start, cp, end)
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
			beziers = append(beziers, [5]Point{start, cp1, cp2, end, Point{1.0, 1.0}})
		case cubeToCmd:
			cp1 := Point{p.d[i+1], p.d[i+2]}
			cp2 := Point{p.d[i+3], p.d[i+4]}
			end = Point{p.d[i+5], p.d[i+6]}
			contour = append(contour, poly2tri.NewPoint(end.X, end.Y))
			beziers = append(beziers, [5]Point{start, cp1, cp2, end, Point{1.0, 1.0}})
		case arcToCmd:
			panic("arcs should have been replaced")
		}
		i += cmdLen(cmd)
		start = end
	}

	swctx := poly2tri.NewSweepContext(contour, false)
	swctx.Triangulate()

	triangles := [][3]Point{}
	for _, tr := range swctx.GetTriangles() {
		p0 := Point{tr.Points[0].X, tr.Points[0].Y}
		p1 := Point{tr.Points[1].X, tr.Points[1].Y}
		p2 := Point{tr.Points[2].X, tr.Points[2].Y}
		triangles = append(triangles, [3]Point{p0, p1, p2})
	}
	return triangles, beziers
}
