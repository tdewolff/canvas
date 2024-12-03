package main

import (
	"image/color"
	"image/png"
	"os"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

var fontFamily, fontLatin, fontArabic, fontDevanagari, fontMongolian, fontCJK *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("times")
	if err := fontFamily.LoadSystemFont("Liberation Serif, serif", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontLatin = canvas.NewFontFamily("latin")
	if err := fontLatin.LoadSystemFont("DejaVu Serif, serif", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontArabic = canvas.NewFontFamily("arabic")
	if err := fontArabic.LoadFontFile("/usr/share/fonts/noto/NotoSansArabic-Regular.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontDevanagari = canvas.NewFontFamily("devanagari")
	if err := fontDevanagari.LoadSystemFont("Noto Serif Devanagari", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontMongolian = canvas.NewFontFamily("mongolian")
	if err := fontMongolian.LoadSystemFont("Noto Sans Mongolian", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontCJK = canvas.NewFontFamily("cjk")
	if err := fontCJK.LoadFontFile("/usr/share/fonts/noto-cjk/NotoSerifCJK-Regular.ttc", canvas.FontRegular); err != nil {
		panic(err)
	}

	faceTitle := fontFamily.Face(14.0, color.Black, canvas.FontRegular, canvas.FontNormal)

	// text
	c := canvas.New(265, 90)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))
	ctx.DrawText(132.5, 84.0, canvas.NewTextLine(faceTitle, "Different horizontal and vertical alignments with indent", canvas.Center))

	drawText(ctx, 5.0, 80.0, canvas.Left, canvas.Top)
	drawText(ctx, 70.0, 80.0, canvas.Center, canvas.Top)
	drawText(ctx, 135.0, 80.0, canvas.Right, canvas.Top)
	drawText(ctx, 200.0, 80.0, canvas.Justify, canvas.Top)
	drawText(ctx, 5.0, 40.0, canvas.Left, canvas.Top)
	drawText(ctx, 70.0, 40.0, canvas.Left, canvas.Center)
	drawText(ctx, 135.0, 40.0, canvas.Left, canvas.Bottom)
	drawText(ctx, 200.0, 40.0, canvas.Left, canvas.Justify)
	renderers.Write("text.png", c, canvas.DPMM(5.0))

	// cjk
	c = canvas.New(140, 60)
	ctx = canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))
	ctx.DrawText(70.0, 54.0, canvas.NewTextLine(faceTitle, "Mixing vertical and horizontal scripts", canvas.Center))

	drawCJK(ctx, 5.0, 45.0, "HorizontalTB, Natural", canvas.HorizontalTB, canvas.Natural)
	drawCJK(ctx, 50.0, 45.0, "VerticalRL, Natural", canvas.VerticalRL, canvas.Natural)
	drawCJK(ctx, 95.0, 45.0, "VerticalLR, Upright", canvas.VerticalLR, canvas.Upright)
	renderers.Write("cjk.png", c, canvas.DPMM(5.0))

	// obj
	c = canvas.New(80, 15)
	ctx = canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))
	ctx.DrawText(40.0, 9.0, canvas.NewTextLine(faceTitle, "Mixing text with paths and images", canvas.Center))

	p := &canvas.Path{}
	p.LineTo(2.0, 0.0)
	p.LineTo(1.0, 2.0)
	p.Close()

	lenna, err := os.Open("../../lenna.png")
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(lenna)
	if err != nil {
		panic(err)
	}

	face := fontFamily.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	rt := canvas.NewRichText(face)
	rt.WriteString("Where ")
	rt.WritePath(p, canvas.Green, canvas.Baseline)
	rt.WriteString(" and ")
	rt.WriteImage(img, canvas.DPMM(200.0), canvas.Baseline)
	rt.WriteString(" refer to foo when ")
	if err := rt.WriteLaTeX("x = \\frac{5}{2}"); err != nil {
		panic(err)
	}
	rt.WriteString(".")
	ctx.DrawText(40.0, 7.0, rt.ToText(0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0))

	renderers.Write("obj.png", c, canvas.DPMM(5.0))
}

func drawText(ctx *canvas.Context, x, y float64, halign, valign canvas.TextAlign) {
	face := fontFamily.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	phrase := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Morbi egestas, augue eget blandit laoreet, dolor lorem interdum ante, quis consectetur lorem massa vitae nulla. Sed cursus tellus id venenatis suscipit." // Nunc volutpat imperdiet ipsum vel varius. Pellentesque mattis viverra odio, ullamcorper iaculis massa tristique imperdiet. Aliquam posuere nisl tortor, in scelerisque elit eleifend sed. Suspendisse in risus aliquam leo vestibulum gravida. Sed ipsum massa, fringilla at pellentesque vitae, dictum nec libero. Morbi lorem ante, facilisis a justo vel, mollis fringilla massa. Mauris aliquet imperdiet magna, ac tempor sem fringilla sed."

	text := canvas.NewTextBox(face, phrase, 60.0, 35.0, halign, valign, 5.0, 0.0)
	ctx.SetFillColor(canvas.Whitesmoke)
	ctx.SetStrokeColor(canvas.Gray)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(x, y, canvas.Rectangle(60.0, -35.0))
	ctx.DrawText(x, y, text)
}

func drawCJK(ctx *canvas.Context, x, y float64, title string, mode canvas.WritingMode, orient canvas.TextOrientation) {
	face := fontFamily.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.DrawText(x+20.0, 47.0, canvas.NewTextLine(face, title, canvas.Center))

	faceLatin := fontLatin.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceArabic := fontArabic.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceDevanagari := fontDevanagari.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceMongolian := fontMongolian.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceCJK := fontCJK.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)

	rt := canvas.NewRichText(faceLatin)
	rt.WriteFace(faceCJK, "中文 日語 ")
	rt.WriteFace(faceArabic, "اللغة العربية ")
	rt.WriteFace(faceLatin, "Latin script ")
	rt.WriteFace(faceMongolian, "ᠮᠣᠩᠭᠣᠯ ᠪᠢᠴᠢᠭ ")
	rt.WriteFace(faceLatin, "русский язык ")
	rt.WriteFace(faceDevanagari, "देवनागरी")

	halign := canvas.Left
	if mode == canvas.VerticalRL {
		halign = canvas.Right
	}
	rt.SetWritingMode(mode)
	rt.SetTextOrientation(orient)
	text := rt.ToText(40.0, 40.0, halign, canvas.Top, 0.0, 0.0)

	ctx.SetFillColor(canvas.Whitesmoke)
	ctx.SetStrokeColor(canvas.Gray)
	ctx.SetStrokeWidth(0.05)
	ctx.DrawPath(x, y, canvas.Rectangle(40.0, -40.0))
	ctx.DrawText(x, y, text)
}
