package main

import (
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

var font *canvas.FontFamily

func main() {
	font = canvas.NewFontFamily("latin")
	if err := font.LoadSystemFont("DejaVu Serif, serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	canvas.Tolerance = 0.001
	c := canvas.New(0.0, 0.0)
	ctx := canvas.NewContext(c)

	face := font.Face(4.5, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.Rotate(45)
	ctx.SetCoordView(canvas.Identity.Rotate(-45))
	ctx.DrawText(18.0, -2.0, canvas.NewTextLine(face, "Disjoint", canvas.Left))
	ctx.DrawText(23.0, -2.0, canvas.NewTextLine(face, "Intersects", canvas.Left))
	ctx.DrawText(28.0, -2.0, canvas.NewTextLine(face, "Touches", canvas.Left))
	ctx.DrawText(33.0, -2.0, canvas.NewTextLine(face, "Overlaps", canvas.Left))
	ctx.DrawText(38.0, -2.0, canvas.NewTextLine(face, "Contains", canvas.Left))
	ctx.DrawText(43.0, -2.0, canvas.NewTextLine(face, "Within", canvas.Left))
	ctx.DrawText(48.0, -2.0, canvas.NewTextLine(face, "Equals", canvas.Left))
	ctx.ResetView()
	ctx.SetCoordView(canvas.Identity)

	draw(ctx, -13.0, "Disjoint circles", canvas.Circle(2.0).Translate(2.5, 5.0), canvas.Circle(2.0).Translate(7.5, 5.0))
	draw(ctx, -25.0, "Touching circles", canvas.Circle(2.5).Translate(2.5, 5.0), canvas.Circle(2.5).Translate(7.5, 5.0))
	draw(ctx, -37.0, "Overlapping circles", canvas.Circle(2.0).Translate(2.0, 5.0), canvas.Circle(4.0).Translate(6.0, 5.0))
	draw(ctx, -49.0, "Contained circles", canvas.Circle(5.0).Translate(5.0, 5.0), canvas.Circle(2.0).Translate(6.0, 5.0))
	draw(ctx, -61.0, "Contained circles", canvas.Circle(2.0).Translate(8.0, 5.0), canvas.Circle(5.0).Translate(5.0, 5.0))
	draw(ctx, -73.0, "Equal circles", canvas.Circle(5.0).Translate(5.0, 5.0), canvas.Circle(5.0).Translate(5.0, 5.0))

	c.Fit(1.0)
	c.SetZIndex(-1)
	ctx.SetFillColor(canvas.White)
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	renderers.Write("relation.png", c, canvas.DPMM(20.0))
}

func draw(ctx *canvas.Context, y float64, title string, p, q *canvas.Path) {
	face := font.Face(3.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.Translate(2.0, y+5.0)
	ctx.Rotate(90.0)
	ctx.DrawText(0.0, 0.0, canvas.NewTextLine(face, title, canvas.Center))
	ctx.ResetView()

	ctx.SetFillColor(canvas.Hex("#13468688"))
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.1)
	ctx.DrawPath(3.0, y, p)
	ctx.SetFillColor(canvas.Hex("#FEB12A88"))
	ctx.DrawPath(3.0, y, q)

	check := canvas.MustParseSVGPath("M-1 -0L-0.333 -0.75L1 0.75")
	xmark := canvas.MustParseSVGPath("M-1 -1L1 1M-1 1L1 -1")

	rel, zs := p.Relate(q)
	for _, z := range zs {
		ctx.SetFillColor(canvas.Hex("#BF0000"))
		ctx.SetStrokeColor(canvas.Transparent)
		ctx.SetStrokeCapper(canvas.RoundCap)
		ctx.DrawPath(3.0+z.X, y+z.Y, canvas.Circle(0.30))
	}

	y += 5.0
	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeWidth(0.25)
	if rel.Disjoint() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(18.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(18.0, y, xmark)
	}
	if rel.Intersects() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(23.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(23.0, y, xmark)
	}
	if rel.Touches() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(28.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(28.0, y, xmark)
	}
	if rel.Overlaps() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(33.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(33.0, y, xmark)
	}
	if rel.Contains() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(38.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(38.0, y, xmark)
	}
	if rel.Within() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(43.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(43.0, y, xmark)
	}
	if rel.Equals() {
		ctx.SetStrokeColor(canvas.Green)
		ctx.DrawPath(48.0, y, check)
	} else {
		ctx.SetStrokeColor(canvas.Red)
		ctx.DrawPath(48.0, y, xmark)
	}
}
