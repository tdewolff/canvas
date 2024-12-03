package main

import (
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers/pdf"
)

var fontLatin *canvas.FontFamily

func main() {
	t0 := time.Now()
	fontLatin = canvas.NewFontFamily("latin")
	if err := fontLatin.LoadSystemFont("serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	f, err := os.Create("document.pdf")
	if err != nil {
		panic(err)
	}
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
	ctx2.SetView(canvas.Identity.Translate(0.0, 197.0))
	if err := canvas.DrawPreview(ctx2); err != nil {
		panic(err)
	}
	c2.RenderTo(p)

	if err := p.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", time.Now().Sub(t0).Round(time.Millisecond))
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
	y -= text.Bounds().H() + spacing
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
