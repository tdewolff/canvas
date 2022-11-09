package main

import (
	"fmt"
	"image/color"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/pdf"
	"github.com/tdewolff/canvas/text"
)

var fontLatin *canvas.FontFamily
var fontArabic *canvas.FontFamily

//var fontDevanagari *canvas.FontFamily

func main() {
	t0 := time.Now()
	fontLatin = canvas.NewFontFamily("DejaVu Serif")
	if err := fontLatin.LoadLocalFont("DejaVuSerif", canvas.FontRegular); err != nil {
		panic(err)
	}

	fontArabic = canvas.NewFontFamily("DejaVu Sans")
	if err := fontArabic.LoadLocalFont("DejaVuSans", canvas.FontRegular); err != nil {
		panic(err)
	}

	//fontDevanagari = canvas.NewFontFamily("Devanagari")
	//if err := fontDevanagari.LoadLocalFont("NotoSerifDevanagari", canvas.FontRegular); err != nil {
	//	panic(err)
	//}

	f, err := os.Create("pdf_renderer.pdf")
	must(err)
	defer f.Close()

	p := pdf.New(f, 210, 297, nil)

	c1 := canvas.New(210, 297)
	ctx1 := canvas.NewContext(c1)
	ctx1.SetFillColor(canvas.Beige)
	ctx1.DrawPath(0, 0, canvas.Rectangle(210, 297))
	drawDocument(ctx1)
	c1.RenderTo(p)

	p.NewPage(210, 297)

	c2 := canvas.New(210, 297)
	ctx2 := canvas.NewContext(c2)
	ctx2.SetFillColor(canvas.White)
	ctx2.DrawPath(0, 0, canvas.Rectangle(210, 297))
	richDraw(ctx2)
	c2.RenderTo(p)

	must(p.Close())
	fmt.Printf("%s\n", time.Now().Sub(t0).Round(time.Millisecond))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var lorem = []string{
	`- Russian ‘Молоко и творог’ - “milk and cottage cheese”`,
	`- Welsh ‘Côf a lithr, llythyrau a geidw’ - “memory slips, letters remain”`,
	`- Danish ‘Så er den ged barberet’ - “now that goat has been shaved” (the work is done)`,
	`- Icelandic ‘Árinni kennir illur ræðari‘ - “a bad rower blames his oars”`,
	`- Greek ‘Όταν λείπει η γάτα, χορεύουν τα ποντίκια’ - “when the cat’s away, the mice dance”`,
	`# Meaningless Lorem Ipsum`,
	`Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. In arcu cursus euismod quis viverra nibh cras pulvinar. Tempor nec feugiat nisl pretium fusce id velit ut. Elementum nibh tellus molestie nunc non blandit massa enim nec. Placerat orci nulla pellentesque dignissim enim sit amet venenatis urna. Eros in cursus turpis massa tincidunt dui ut. Sit amet volutpat consequat mauris nunc congue. Curabitur vitae nunc sed velit dignissim sodales ut eu sem. Egestas fringilla phasellus faucibus scelerisque eleifend donec. Blandit libero volutpat sed cras ornare arcu dui.`,
	`Gravida neque convallis a cras semper auctor. Eget mi proin sed libero enim sed faucibus. Non odio euismod lacinia at quis risus sed. Non curabitur gravida arcu ac tortor dignissim. In mollis nunc sed id semper risus in hendrerit. Orci dapibus ultrices in iaculis nunc sed. Porta lorem mollis aliquam ut porttitor leo. A scelerisque purus semper eget duis at. Ullamcorper a lacus vestibulum sed arcu non odio euismod lacinia. Cras tincidunt lobortis feugiat vivamus at.`,
	`Ut porttitor leo a diam sollicitudin. Faucibus purus in massa tempor. Ante in nibh mauris cursus mattis molestie. In tellus integer feugiat scelerisque varius morbi. Viverra justo nec ultrices dui sapien eget mi proin. Adipiscing elit pellentesque habitant morbi tristique senectus. Nulla posuere sollicitudin aliquam ultrices sagittis orci a. Fames ac turpis egestas sed tempus urna et pharetra pharetra. Nascetur ridiculus mus mauris vitae. Feugiat nisl pretium fusce id velit. Mollis nunc sed id semper risus. Dictum fusce ut placerat orci nulla. Sit amet nulla facilisi morbi tempus iaculis. Iaculis at erat pellentesque adipiscing commodo elit at imperdiet dui. Non quam lacus suspendisse faucibus interdum posuere lorem ipsum. Vitae ultricies leo integer malesuada nunc vel risus commodo viverra. Pretium fusce id velit ut tortor pretium viverra suspendisse. Metus vulputate eu scelerisque felis imperdiet proin fermentum leo.`,
}

const lenna = "../../resources/lenna.png"

var y = 290.0

func drawTextAndMoveDown(c *canvas.Context, x float64, text *canvas.Text) {
	c.DrawText(x, y, text)
	const spacing = 5
	y -= text.Bounds().H + spacing
}

func drawDocument(c *canvas.Context) {
	y = 290.0
	c.SetFillColor(canvas.Black)

	headerFace := fontLatin.Face(24.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	text12Face := fontLatin.Face(12.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	text10Face := fontLatin.Face(10.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	boldFace := fontLatin.Face(12.0, canvas.Black, canvas.FontBold, canvas.FontNormal)

	logo, err := os.Open(lenna)
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(logo)
	if err != nil {
		panic(err)
	}
	imgDPMM := 20.0
	imgWidth := float64(img.Bounds().Max.X) / imgDPMM
	imgHeight := float64(img.Bounds().Max.Y) / imgDPMM
	c.DrawImage(190.0-imgWidth, y-imgHeight, img, canvas.DPMM(imgDPMM))

	y -= 10
	txt := canvas.NewTextBox(headerFace, "Document Example", 0.0, 0.0, canvas.Left, canvas.Top, 0.0, 0.0)
	drawTextAndMoveDown(c, 20.0, txt)

	for _, t := range lorem {
		if len(t) > 0 {
			if t[0] == '#' {
				txt = canvas.NewTextBox(boldFace, t[2:], 170.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0)
			} else if t[0] == '-' {
				txt = canvas.NewTextBox(text10Face, t[2:], 170.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0)
			} else {
				txt = canvas.NewTextBox(text12Face, t, 170.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0)
			}
			drawTextAndMoveDown(c, 20.0, txt)
		}
	}
}

func richDrawText(c *canvas.Context, x float64, face *canvas.FontFace, rich *canvas.RichText) {
	metrics := face.Metrics()
	width, height := 90.0, 32.0

	txt := rich.ToText(width, height, canvas.Justify, canvas.Top, 0.0, 0.1)
	b := txt.Bounds()

	// fill the text background
	c.SetFillColor(color.RGBA{255, 230, 200, 255})
	c.DrawPath(x, y, b.ToPath())

	// highlight the first line
	c.SetFillColor(color.RGBA{50, 50, 50, 92})
	c.DrawPath(x-1, y+1, canvas.Rectangle(width+2, -metrics.LineHeight-1))

	c.SetFillColor(canvas.Black)
	c.DrawPath(x-3, y+3, canvas.Rectangle(width+6, -height-3).Stroke(0.2, canvas.RoundCap, canvas.BevelJoin))
	c.DrawText(x, y, txt)
	y -= b.H + 10.0
}

func richDraw(c *canvas.Context) {
	y = 290.0
	// Draw a comprehensive text box
	pt := 14.0
	face := fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	rt := canvas.NewRichText(face)
	rt.Add(face, "Lorem dolor ipsum ")
	rt.Add(fontLatin.Face(pt, canvas.White, canvas.FontBold, canvas.FontNormal), "confiscator")
	rt.Add(face, " cur\u200babitur ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontItalic, canvas.FontNormal), "mattis")
	rt.Add(face, " dui ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontBold|canvas.FontItalic, canvas.FontNormal), "tellus")
	rt.Add(face, " vel. Proin ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontUnderline), "sodales")
	rt.Add(face, " eros vel ")
	rt.Add(fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal, canvas.FontSineUnderline), "nibh")
	rt.Add(face, " fringilla pellen\u200btesque eu cillum. ")

	face = fontLatin.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	face.Language = "ru"
	face.Script = text.Cyrillic
	rt.Add(face, "дёжжэнтиюнт холст ")

	face = fontArabic.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	face.Language = "ar"
	face.Script = text.Arabic
	face.Direction = text.RightToLeft
	rt.Add(face, "تسجّل يتكلّم ")

	// Devanagari font isn't working
	//face = fontDevanagari.Face(pt, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	//face.Language = "hi"
	//face.Script = text.Devanagari
	//rt.Add(face, "हालाँकि प्र ")

	richDrawText(c, 20, face, rt)

	y -= 40

	// Draw an elliptic arc being dashed
	ellipse, err := canvas.ParseSVG(fmt.Sprintf("A10 30 30 1 0 30 0z"))
	if err != nil {
		panic(err)
	}
	c.SetFillColor(canvas.Whitesmoke)
	c.DrawPath(110, y, ellipse)

	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.75)
	c.SetStrokeCapper(canvas.RoundCap)
	c.SetStrokeJoiner(canvas.RoundJoin)
	c.SetDashes(0.0, 2.0, 4.0, 2.0, 2.0, 4.0, 2.0)
	//ellipse = ellipse.Dash(0.0, 2.0, 4.0, 2.0).Stroke(0.5, canvas.RoundCap, canvas.RoundJoin)
	c.DrawPath(110, y, ellipse)
	c.SetStrokeColor(canvas.Transparent)
	c.SetDashes(0.0)

	y -= 20
	// Draw a raster image
	logo, err := os.Open(lenna)
	if err != nil {
		panic(err)
	}
	img, err := canvas.NewPNGImage(logo)
	if err != nil {
		panic(err)
	}
	c.Push()
	c.Rotate(5)
	c.DrawImage(40.0, y, img, 10)
	c.Pop()

	// Draw an closed set of points being smoothed
	polyline := &canvas.Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(30.0, 0.0)
	polyline.Add(30.0, 15.0)
	polyline.Add(0.0, 30.0)
	polyline.Add(0.0, 0.0)
	c.SetFillColor(canvas.Seagreen)
	c.FillColor.R = byte(float64(c.FillColor.R) * 0.25)
	c.FillColor.G = byte(float64(c.FillColor.G) * 0.25)
	c.FillColor.B = byte(float64(c.FillColor.B) * 0.25)
	c.FillColor.A = byte(float64(c.FillColor.A) * 0.25)
	c.SetStrokeColor(canvas.Seagreen)
	c.DrawPath(155, y, polyline.Smoothen())

	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.5)
	c.DrawPath(155, y, polyline.ToPath())
	c.SetStrokeWidth(0.75)
	for _, coord := range polyline.Coords() {
		c.DrawPath(155, y, canvas.Circle(2.0).Translate(coord.X, coord.Y))
	}

	y -= 20

	// Draw a open set of points being smoothed
	polyline = &canvas.Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(20.0, 10.0)
	polyline.Add(40.0, 30.0)
	polyline.Add(60.0, 40.0)
	polyline.Add(80.0, 20.0)
	c.SetStrokeColor(canvas.Dodgerblue)
	c.DrawPath(10, y, polyline.Smoothen())
	c.SetStrokeColor(canvas.Black)
	for _, coord := range polyline.Coords() {
		c.DrawPath(10, y, canvas.Circle(2.0).Translate(coord.X, coord.Y))
	}
}
