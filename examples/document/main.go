package main

import (
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/rasterizer"
)

var fontFamily *canvas.FontFamily

func main() {
	fontFamily = canvas.NewFontFamily("times")
	fontFamily.Use(canvas.CommonLigatures)
	if err := fontFamily.LoadLocalFont("NimbusRoman-Regular", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(200, 230)
	ctx := canvas.NewContext(c)
	draw(ctx)
	c.WriteFile("out.png", rasterizer.PNGWriter(5.0))
}

var lorem = []string{
	`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nulla malesuada fringilla libero vel ultricies. Phasellus eu lobortis lorem. Phasellus eu cursus mi. Sed enim ex, ornare et velit vitae, sollicitudin volutpat dolor. Sed aliquam sit amet nisi id sodales. Aliquam erat volutpat. In hac habitasse platea dictumst. Pellentesque luctus varius nibh sit amet porta. Vivamus tempus, enim ut sodales aliquet, magna massa viverra eros, nec gravida risus ipsum a erat. Etiam dapibus sem augue, at porta nisi dictum non. Vestibulum quis urna ut ligula dapibus mollis eu vel nisl. Vestibulum lorem dolor, eleifend lacinia fringilla eu, pulvinar vitae metus.`,
	`Morbi dapibus purus vel erat auctor, vehicula tempus leo maximus. Aenean feugiat vel quam sit amet iaculis. Fusce et justo nec arcu maximus porttitor. Cras sed aliquam ipsum. Sed molestie mauris nec dui interdum sollicitudin. Nulla id egestas massa. Fusce congue ante. Interdum et malesuada fames ac ante ipsum primis in faucibus. Praesent faucibus tellus eu viverra blandit. Vivamus mi massa, hendrerit in commodo et, luctus vitae felis.`,
	`Quisque semper aliquet augue, in dignissim eros cursus eu. Pellentesque suscipit consequat nibh, sit amet ultricies risus. Suspendisse blandit interdum tortor, consectetur tristique magna aliquet eu. Aliquam sollicitudin eleifend sapien, in pretium nisi. Sed tempor eleifend velit quis vulputate. Donec condimentum, lectus vel viverra pharetra, ex enim cursus metus, quis luctus est urna ut purus. Donec tempus gravida pharetra. Sed leo nibh, cursus at hendrerit at, ultricies a dui. Maecenas eget elit magna. Quisque sollicitudin odio erat, sed consequat libero tincidunt in. Nullam imperdiet, neque quis consequat pellentesque, metus nisl consectetur eros, ut vehicula dui augue sed tellus.`,
	//` Vivamus varius ex sed nisi vestibulum, sit amet tincidunt ante vestibulum. Nullam et augue blandit dolor accumsan tempus. Quisque at dictum elit, id ullamcorper dolor. Nullam feugiat mauris eu aliquam accumsan.`,
}

var y = 205.0

func drawText(c *canvas.Context, x float64, text *canvas.Text) {
	h := text.Bounds().H
	c.DrawText(x, y, text)
	y -= h + 10.0
}

func draw(c *canvas.Context) {
	c.SetFillColor(canvas.Black)

	headerFace := fontFamily.Face(28.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	textFace := fontFamily.Face(12.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)

	drawText(c, 30.0, canvas.NewTextBox(headerFace, "Document Example", 0.0, 0.0, canvas.Left, canvas.Top, 0.0, 0.0))
	drawText(c, 30.0, canvas.NewTextBox(textFace, lorem[0], 140.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0))

	lenna, err := os.Open("../lenna.png")
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(lenna)
	if err != nil {
		panic(err)
	}
	imgDPM := 15.0
	imgWidth := float64(img.Bounds().Max.X) / imgDPM
	imgHeight := float64(img.Bounds().Max.Y) / imgDPM
	c.DrawImage(170.0-imgWidth, y-imgHeight, img, imgDPM)

	drawText(c, 30.0, canvas.NewTextBox(textFace, lorem[1], 140.0-imgWidth-10.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0))
	drawText(c, 30.0, canvas.NewTextBox(textFace, lorem[2], 140.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0))
	//drawText(c, 30.0, canvas.NewTextBox(textFace, lorem[3], 140.0, 0.0, canvas.Justify, canvas.Top, 5.0, 0.0))
}
