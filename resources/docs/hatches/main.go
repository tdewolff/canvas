package main

import (
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

var face *canvas.FontFace

func main() {
	fontLatin := canvas.NewFontFamily("latin")
	if err := fontLatin.LoadSystemFont("Nimbus Sans Bold, sans", canvas.FontRegular); err != nil {
		panic(err)
	}
	face = fontLatin.Face(2.0)

	W := 26.0
	H := 26.0
	c := canvas.New(W, H)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(W, H))

	draw(ctx, 0, 0, "LineHatch", canvas.NewLineHatch(canvas.Black, 90.0, 0.5, 0.05))
	draw(ctx, 1, 0, "LineHatch", canvas.NewLineHatch(canvas.Black, 0.0, 0.25, 0.025))
	draw(ctx, 2, 0, "LineHatch", canvas.NewLineHatch(canvas.Black, 45.0, 0.5, 0.05))
	draw(ctx, 0, 1, "CrossHatch", canvas.NewCrossHatch(canvas.Black, 90.0, 0.0, 0.25, 0.25, 0.025))
	draw(ctx, 1, 1, "CrossHatch", canvas.NewCrossHatch(canvas.Black, 45.0, -45.0, 0.5, 0.5, 0.05))
	draw(ctx, 2, 1, "CrossHatch", canvas.NewCrossHatch(canvas.Black, 30.0, -30.0, 1.0, 0.5, 0.05))
	draw(ctx, 0, 2, "ShapeHatch", canvas.NewShapeHatch(canvas.Black, canvas.Circle(0.05), 0.5, 0.0))
	draw(ctx, 1, 2, "ShapeHatch", canvas.NewShapeHatch(canvas.Black, canvas.Circle(0.15), 0.5, 0.05))
	draw(ctx, 2, 2, "ShapeHatch", canvas.NewShapeHatch(canvas.Black, canvas.StarPolygon(5, 0.2, 0.1, true), 0.5, 0.0))

	renderers.Write("hatches.png", c, canvas.DPMM(20.0))
}

func draw(ctx *canvas.Context, i, j int, text string, hatch *canvas.HatchPattern) {
	size := 5.0
	spacing := 2.5
	x := 3.0 + float64(i)*(spacing+size)
	y := 3.0 + float64(j)*(spacing+size)

	clip := canvas.RoundedRectangle(size, size, 0.2).Translate(x, y)
	ctx.SetFill(hatch.Fill)
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(0.0, 0.0, hatch.Tile(clip))

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(0.0, 0.0, clip)

	ctx.DrawText(x+0.5*size, y-1.0, canvas.NewTextLine(face, text, canvas.Middle))
}
