package main

import (
	"image/color"
	"math"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

func main() {
	fontLatin := canvas.NewFontFamily("latin")
	if err := fontLatin.LoadSystemFont("Nimbus Sans Bold, sans", canvas.FontRegular); err != nil {
		panic(err)
	}
	face := fontLatin.Face(2.0)

	W := 26.0
	H := 26.0
	c := canvas.New(W, H)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(W, H))

	var clip *canvas.Path

	// Floret tiling
	clip = canvas.RoundedRectangle(10.0, 10.0, 1.0).Translate(2.0, 2.0)
	floretTiling(ctx, clip)
	ctx.DrawText(7.0, 1.0, canvas.NewTextLine(face, "Floret tiling", canvas.Middle))

	// Rhombitrihexagonal tiling
	clip = canvas.RoundedRectangle(10.0, 10.0, 1.0).Translate(14.0, 2.0)
	rhombitrihexagonalTiling(ctx, clip)
	ctx.DrawText(19.0, 1.0, canvas.NewTextLine(face, "Rhombitrihexagonal tiling", canvas.Middle))

	// Cairo tiling
	clip = canvas.RoundedRectangle(10.0, 10.0, 1.0).Translate(2.0, 14.0)
	cairoTiling(ctx, clip)
	ctx.DrawText(7.0, 13.0, canvas.NewTextLine(face, "Cairo tiling", canvas.Middle))

	// Pythagorean tiling
	clip = canvas.RoundedRectangle(10.0, 10.0, 1.0).Translate(14.0, 14.0)
	pythagoreanTiling(ctx, clip)
	ctx.DrawText(19.0, 13.0, canvas.NewTextLine(face, "Pythagorean tiling", canvas.Middle))

	renderers.Write("tiling.png", c, canvas.DPMM(20.0))
}

func floretTiling(ctx *canvas.Context, clip *canvas.Path) {
	dx, dy := math.Sincos(30.0 * math.Pi / 180.0)
	p := &canvas.Path{}
	p.LineTo(dx, dy)
	p.LineTo(0.25, 1.5*dy)
	p.LineTo(-0.25, 1.5*dy)
	p.LineTo(-dx, dy)
	p.Close()

	Dx := 2.0 + 0.5*dx
	Dy := 0.5 * dy
	D := math.Sqrt(Dx*Dx + Dy*Dy)
	cell := canvas.PrimitiveCell(canvas.Point{math.Sqrt(0.75) * D, -0.5 * D}, canvas.Point{math.Sqrt(0.75) * D, 0.5 * D})

	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.02)
	colors := []color.RGBA{
		canvas.Hex("#c5f0e9"),
		canvas.Hex("#c5f0e9"),
		canvas.Hex("#c5f0e9"),
		canvas.Hex("#c5f0e9"),
		canvas.Hex("#c5f0e9"),
		canvas.Hex("#c5f0e9"),
	}
	theta0 := 30.0 + math.Atan(Dy/Dx)*180.0/math.Pi
	for i := 0; i < 6; i++ {
		ctx.SetFillColor(colors[i])
		q := p.Transform(canvas.Identity.Rotate(theta0 + 60.0*float64(i)))
		ctx.DrawPath(0.0, 0.0, q.Tile(clip, cell))
	}

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(0.0, 0.0, clip)
}

func rhombitrihexagonalTiling(ctx *canvas.Context, clip *canvas.Path) {
	d := math.Cos(30.0 * math.Pi / 180.0)
	hex := canvas.RegularPolygon(6, 1.0, false)

	tri := &canvas.Path{}
	tri.MoveTo(-1.0, 0.0)
	tri.LineTo(-1.0-d, 0.5)
	tri.LineTo(-1.0-d, -0.5)
	tri.Close()
	tri.MoveTo(1.0, 0.0)
	tri.LineTo(1.0+d, -0.5)
	tri.LineTo(1.0+d, 0.5)
	tri.Close()

	rect := canvas.Rectangle(1.0, 1.0)
	squ := rect.Translate(-0.5, -1.0-d)
	squ = squ.Append(rect.Transform(canvas.Identity.Translate(-1.0, 0.0).Rotate(210.0)))
	squ = squ.Append(rect.Transform(canvas.Identity.Translate(1.0, 0.0).Rotate(240.0)))

	cell := canvas.PrimitiveCell(canvas.Point{1.5 + d, -0.5 - d}, canvas.Point{1.5 + d, 0.5 + d})
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.SetFillColor(canvas.Saddlebrown)
	ctx.DrawPath(0.0, 0.0, hex.Tile(clip, cell))
	ctx.SetFillColor(canvas.Darkslategrey)
	ctx.DrawPath(0.0, 0.0, tri.Tile(clip, cell))
	ctx.SetFillColor(canvas.Ivory)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.02)
	ctx.DrawPath(0.0, 0.0, squ.Tile(clip, cell))

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(0.0, 0.0, clip)
}

func cairoTiling(ctx *canvas.Context, clip *canvas.Path) {
	d := 0.5 * math.Tan(30.0/180.0*math.Pi)
	p := &canvas.Path{}
	p.MoveTo(0.5-d, 0.0)
	p.LineTo(0.5, 0.5)
	p.LineTo(0.0, 0.5+d)
	p.LineTo(-0.5, 0.5)
	p.LineTo(-0.5+d, 0.0)
	p.Close()

	cell := canvas.PrimitiveCell(canvas.Point{1.0, -1.0}, canvas.Point{1.0, 1.0})
	ms := []canvas.Matrix{
		canvas.Identity,
		canvas.Identity.RotateAbout(90.0, 0.5, 0.5),
		canvas.Identity.RotateAbout(180.0, 0.5, 0.5),
		canvas.Identity.RotateAbout(270.0, 0.5, 0.5),
	}

	ctx.SetStrokeColor(canvas.Transparent)
	colors := []color.RGBA{canvas.Sandybrown, canvas.Steelblue, canvas.Lightskyblue, canvas.Peachpuff}
	for i, m := range ms {
		pi := p.Transform(m).Tile(clip, cell)
		ctx.SetFillColor(colors[i])
		ctx.DrawPath(0.0, 0.0, pi)
	}

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(0.0, 0.0, clip)
}

func pythagoreanTiling(ctx *canvas.Context, clip *canvas.Path) {
	a := 1.0
	b := 0.4
	p1 := canvas.Rectangle(a, a)
	p2 := canvas.Rectangle(b, b).Translate(a, 0.0)

	cell := canvas.PrimitiveCell(canvas.Point{b, -a}, canvas.Point{a, b})
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.SetFillColor(canvas.Cornsilk)
	ctx.DrawPath(0.0, 0.0, p1.Tile(clip, cell))
	ctx.SetFillColor(canvas.Black)
	ctx.DrawPath(0.0, 0.0, p1.Append(canvas.Rectangle(a-0.01, a-0.01).Translate(0.01, 0.01).Reverse()).Tile(clip, cell))

	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.02)
	ctx.SetFillColor(canvas.Purple)
	ctx.DrawPath(0.0, 0.0, p2.Tile(clip, cell))

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(0.0, 0.0, clip)
}
