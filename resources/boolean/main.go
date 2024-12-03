package main

import (
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

var font *canvas.FontFamily

// Shapes from http://assets.paperjs.org/boolean/
func main() {
	font = canvas.NewFontFamily("latin")
	if err := font.LoadSystemFont("DejaVu Serif", canvas.FontRegular); err != nil {
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

	a := canvas.Rectangle(3.0, 8.0).Translate(5.0, 1.0)
	p0 := canvas.Circle(3.0).Translate(6.0, 5.0)
	draw(ctx, -14.0, a, p0)
	draw(ctx, -26.0, a, p0.Reverse())
	draw(ctx, -38.0, a.Reverse(), p0)
	draw(ctx, -50.0, a.Reverse(), p0.Reverse())

	p1 := canvas.Circle(1.5).Reverse().Translate(8.0, 7.0)
	draw(ctx, -74.0, a, p0)
	draw(ctx, -86.0, a, p1)
	draw(ctx, -98.0, a, p0.Append(p1))
	draw2(ctx, -110.0, a, p0.Append(p1))

	draw(ctx, -134.0, p0, a)
	draw(ctx, -146.0, p1, a)
	draw(ctx, -158.0, p0.Append(p1), a)
	draw2(ctx, -170.0, p0.Append(p1), a)

	a = canvas.Rectangle(2.0, 4.0).Translate(6.0, 5.0)
	b := canvas.Rectangle(4.0, 2.0).Translate(5.0, 6.0)
	b = b.Append(canvas.Circle(1.0).Reverse().Translate(8.0, 8.0))
	draw(ctx, -194.0, a, b)

	c.Fit(1.0)
	c.SetZIndex(-1)
	ctx.SetFillColor(canvas.White)
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	renderers.Write("boolean.png", c, canvas.DPMM(20.0))
}

func draw2(ctx *canvas.Context, y float64, p, q *canvas.Path) {
	p = p.Flatten(0.01)
	q = q.Flatten(0.01)

	ctx.SetFillColor(canvas.Hex("#CC08"))
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(3.0, y, p)
	ctx.SetFillColor(canvas.Hex("#C0C8"))
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

func draw(ctx *canvas.Context, y float64, p, q *canvas.Path) {
	p = p.Flatten(0.01)
	q = q.Flatten(0.01)

	ctx.SetFillColor(canvas.Hex("#CC08"))
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(3.0, y, p)
	ctx.SetFillColor(canvas.Hex("#C0C8"))
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

	for i, R := range []*canvas.Path{
		p.And(q),
		p.Or(q),
		p.Xor(q),
		p.Not(q),
		q.Not(p),
	} {
		for _, r := range R.Split() {
			if r.CCW() {
				ctx.SetFillColor(canvas.Hex("#0C08"))
			} else {
				ctx.SetFillColor(canvas.Hex("#C008"))
			}
			ctx.DrawPath(15+12*float64(i), y, r)
		}
	}
}
