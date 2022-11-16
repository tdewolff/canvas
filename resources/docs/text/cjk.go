package main

import (
	"image/color"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

var fontFamily, fontLatin, fontArabic, fontDevanagari, fontMongolian, fontCJK *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("times")
	if err := fontFamily.LoadLocalFont("Liberation Serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	fontLatin = canvas.NewFontFamily("latin")
	if err := fontLatin.LoadFontFile("/usr/share/fonts/TTF/DejaVuSerif.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontArabic = canvas.NewFontFamily("arabic")
	if err := fontArabic.LoadFontFile("/usr/share/fonts/noto/NotoSansArabic-Regular.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontDevanagari = canvas.NewFontFamily("devanagari")
	if err := fontDevanagari.LoadFontFile("/usr/share/fonts/noto/NotoSerifDevanagari-Regular.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontMongolian = canvas.NewFontFamily("mongolian")
	if err := fontMongolian.LoadFontFile("/usr/share/fonts/noto/NotoSansMongolian-Regular.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}
	fontCJK = canvas.NewFontFamily("cjk")
	if err := fontCJK.LoadFontFile("/usr/share/fonts/noto-cjk/NotoSerifCJK-Regular.ttc", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(140, 60)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	face := fontFamily.Face(14.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.DrawText(70.0, 54.0, canvas.NewTextLine(face, "Mixing vertical and horizontal scripts", canvas.Center))

	drawText(ctx, 5.0, 45.0, "HorizontalTB, Natural", canvas.HorizontalTB, canvas.Natural)
	drawText(ctx, 50.0, 45.0, "VerticalRL, Natural", canvas.VerticalRL, canvas.Natural)
	drawText(ctx, 95.0, 45.0, "VerticalLR, Upright", canvas.VerticalLR, canvas.Upright)

	renderers.Write("cjk.png", c, canvas.DPMM(5.0))
}

func drawText(ctx *canvas.Context, x, y float64, title string, mode canvas.WritingMode, orient canvas.TextOrientation) {
	face := fontFamily.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	ctx.DrawText(x+20.0, 47.0, canvas.NewTextLine(face, title, canvas.Center))

	faceLatin := fontLatin.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceArabic := fontArabic.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceDevanagari := fontDevanagari.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceMongolian := fontMongolian.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	faceCJK := fontCJK.Face(10.0, color.Black, canvas.FontRegular, canvas.FontNormal)

	rt := canvas.NewRichText(faceLatin)
	rt.Add(faceCJK, "中文 日語 ")
	rt.Add(faceArabic, "اللغة العربية ")
	rt.Add(faceLatin, "Latin script ")
	rt.Add(faceMongolian, "ᠮᠣᠩᠭᠣᠯ ᠪᠢᠴᠢᠭ ")
	rt.Add(faceLatin, "русский язык ")
	rt.Add(faceDevanagari, "देवनागरी")

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
