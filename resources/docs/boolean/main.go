package main

import (
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

var font *canvas.FontFamily

// Shapes from http://assets.paperjs.org/boolean/
func main() {
	font = canvas.NewFontFamily("latin")
	if err := font.LoadSystemFont("DejaVu Serif, serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(0.0, 0.0)
	ctx := canvas.NewContext(c)

	face := font.Face(4.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.DrawText(20.0, -2.0, canvas.NewTextLine(face, "A and B", canvas.Center))
	ctx.DrawText(32.0, -2.0, canvas.NewTextLine(face, "A or B", canvas.Center))
	ctx.DrawText(44.0, -2.0, canvas.NewTextLine(face, "A xor B", canvas.Center))
	ctx.DrawText(56.0, -2.0, canvas.NewTextLine(face, "A not B", canvas.Center))
	ctx.DrawText(68.0, -2.0, canvas.NewTextLine(face, "B not A", canvas.Center))

	draw(ctx, -13.0, "Overlapping circles", canvas.Circle(2.0).Translate(2.0, 5.0), canvas.Circle(4.0).Translate(6.0, 5.0))
	draw(ctx, -25.0, "Disjoint circles", canvas.Circle(2.0).Translate(2.5, 5.0), canvas.Circle(2.0).Translate(7.5, 5.0))
	draw(ctx, -37.0, "Contained circles", canvas.Circle(5.0).Translate(5.0, 5.0), canvas.Circle(2.0).Translate(6.0, 5.0))
	draw(ctx, -49.0, "Equal circles", canvas.Circle(5.0).Translate(5.0, 5.0), canvas.Circle(5.0).Translate(5.0, 5.0))
	draw(ctx, -61.0, "Polygon and square", canvas.RegularPolygon(12, 4.0, true).Translate(4.0, 5.0), canvas.Rectangle(4.0, 4.0).Translate(5.0, 2.0))
	draw(ctx, -73.0, "Circle and square 1", canvas.Circle(4.0).Translate(5.0, 5.0), canvas.Rectangle(4.0, 4.0).Translate(5.0, 1.0))
	draw(ctx, -85.0, "Circle and square 2", canvas.Circle(4.0).Translate(5.0, 5.0), canvas.Rectangle(4.0, 4.0).Translate(6.0, 0.0))
	draw(ctx, -97.0, "Square and square", canvas.RegularPolygon(4, 2.0, true).Translate(3.0, 5.0), canvas.Rectangle(4.0, 4.0).Translate(5.0, 3.0))
	draw(ctx, -109.0, "Rectangle and rectangle", canvas.Rectangle(4.0, 9.0).Translate(1.0, 1.0), canvas.Rectangle(4.0, 9.0).Translate(5.0, 0.0))
	draw(ctx, -121.0, "Overlapping stars 1", canvas.StarPolygon(10, 3.0, 0.5, false).Translate(4.5, 5.0), canvas.StarPolygon(10, 4.0, 1.0, false).Translate(6.0, 5.0))
	draw(ctx, -133.0, "Overlapping stars 2", canvas.StarPolygon(20, 4.0, 1.0, false).Translate(5.0, 5.0), canvas.StarPolygon(6, 5.0, 2.0, false).Translate(5.0, 5.0))

	bezier := canvas.MustParseSVGPath("M173,44c-86,152 -215,149 -126,49c240,-239 -155,219 126,-49z")
	bezier = bezier.Transform(canvas.Identity.Scale(0.05, -0.05).Translate(-100.0, -100.0))
	draw(ctx, -145.0, "Cubic beziers", bezier.Translate(5.0, 5.0), bezier.Scale(-1.0, 1.0).Translate(5.0, 5.0))

	var p *canvas.Path
	a, _, _ := font.Face(40.0, canvas.Black, canvas.FontRegular, canvas.FontNormal).ToPath("a")
	p = canvas.Circle(3.0).Translate(6.0, 5.0)
	p = p.Append(canvas.Circle(1.0).Translate(1.5, 6.0))
	draw(ctx, -157.0, "Holes and islands 1", a, p)

	p = canvas.Circle(5.0).Translate(5.0, 5.0)
	p = p.Append(canvas.Circle(3.0).Reverse().Translate(5.0, 5.0))
	p = p.Append(canvas.Circle(1.5).Translate(5.0, 5.0))
	draw(ctx, -169.0, "Holes and islands 2", a.Translate(0.5, 1.0), p)

	p = canvas.Rectangle(5.0, 9.0)
	p = p.Append(canvas.Rectangle(3.0, 6.0).Translate(0.5, 0.5).Reverse())
	draw(ctx, -181.0, "Holes and islands 3", canvas.Rectangle(5.0, 7.0).Translate(0.5, 0.3), p.Translate(5.0, 0.0))

	p = canvas.Rectangle(6.0, 2.0).Translate(0.0, 4.0)
	p = p.Append(canvas.Circle(1.0).Reverse().Translate(6.0, 5.0))
	p = p.Append(canvas.Circle(0.8).Translate(8.0, 7.0))
	p = p.Append(canvas.Circle(0.8).Translate(8.0, 3.0))
	q := canvas.Rectangle(4.0, 4.0).Translate(1.0, 3.0)
	q = q.Append(canvas.Circle(0.8).Translate(8.0, 7.0))
	draw(ctx, -193.0, "Holes and islands 4", q.Translate(1.0, -1.0), p.Translate(1.0, -1.0))

	p = &canvas.Path{}
	p.MoveTo(1.0, 3.0)
	p.LineTo(4.0, 3.0)
	p.LineTo(4.0, 4.0)
	p.LineTo(6.0, 6.0)
	p.LineTo(6.0, 7.0)
	p.LineTo(1.0, 7.0)
	p.Close()
	q = &canvas.Path{}
	q.MoveTo(9.0, 3.0)
	q.LineTo(4.0, 3.0)
	q.LineTo(4.0, 7.0)
	q.LineTo(9.0, 7.0)
	q.Close()
	draw(ctx, -205.0, "Parallel segments", p, q)

	c.Fit(1.0)
	c.SetZIndex(-1)
	ctx.SetFillColor(canvas.White)
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	renderers.Write("boolean.png", c, canvas.DPMM(20.0))
}

func draw(ctx *canvas.Context, y float64, title string, p, q *canvas.Path) {
	face := font.Face(2.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.Translate(2.0, y+5.0)
	ctx.Rotate(90.0)
	ctx.DrawText(0.0, 0.0, canvas.NewTextLine(face, title, canvas.Center))
	ctx.ResetView()

	ctx.SetFillColor(canvas.Hex("#CCC8"))
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(3.0, y, p)
	ctx.DrawPath(3.0, y, q)

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(3.0, y, p)
	ctx.DrawPath(3.0, y, q)

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Hex("#CCC"))
	ctx.SetStrokeWidth(0.1)
	for i := 1; i < 6; i++ {
		x := 3.0 + 12.0*float64(i)
		ctx.DrawPath(x, y, p)
		ctx.DrawPath(x, y, q)
	}

	ctx.SetFillColor(canvas.Hex("#00C8"))
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.1)
	ctx.DrawPath(15.0, y, p.And(q))
	ctx.DrawPath(27.0, y, p.Or(q))
	ctx.DrawPath(39.0, y, p.Xor(q))
	ctx.DrawPath(51.0, y, p.Not(q))
	ctx.DrawPath(63.0, y, q.Not(p))
}
