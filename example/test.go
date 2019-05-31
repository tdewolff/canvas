package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/tdewolff/canvas"
)

var dejaVuSerif canvas.Font

func main() {
	var err error
	dejaVuSerif, err = canvas.LoadFontFile("DejaVuSerif", canvas.Regular, "DejaVuSerif.woff")
	if err != nil {
		panic(err)
	}

	c := canvas.New(160, 80)
	Draw(c)

	////////////////

	svgFile, err := os.Create("test.svg")
	if err != nil {
		panic(err)
	}
	defer svgFile.Close()
	c.WriteSVG(svgFile)

	////////////////

	pngFile, err := os.Create("test.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := c.WriteImage(144.0)
	err = png.Encode(pngFile, img)
	if err != nil {
		panic(err)
	}
}

func drawStrokedPath(c *canvas.C, x, y, d float64, path string) {
	fmt.Println("----------")
	c.SetColor(canvas.Black)
	p, err := canvas.ParseSVG(path)
	if err != nil {
		panic(err)
	}
	c.DrawPath(x, y, 0.0, p)

	c.SetColor(color.RGBA{255, 0, 0, 127})
	p = p.Stroke(d, canvas.ButtCapper, canvas.ArcsJoiner)
	c.DrawPath(x, y, 0.0, p)
}

func Draw(c *canvas.C) {
	//p, _ := canvas.ParseSVG("A20 20 0 0 1 -20 20V0z")
	//c.SetColor(canvas.Red)
	//c.DrawPath(50.0, 50.0, 0.0, p)
	//p = p.Flatten()
	//c.SetColor(canvas.Black)
	//c.DrawPath(50.0, 50.0, 0.0, p)

	//drawStrokedPath(c, 30, 50, 2.0, "M0 0L50 0")
	//drawStrokedPath(c, 30, 40, 2.0, "M0 0L50 0L50 -5")
	//drawStrokedPath(c, 30, 30, 2.0, "M-25 -25A25 25 0 0 1 0 0A25 25 0 0 1 25 -25z")
	//drawStrokedPath(c, 80, 30, 2.0, "M-35.35 -14.65A50 50 0 0 0 0 0A50 50 0 0 0 35.35 -14.65L-35.35 -14.65z")
	//drawStrokedPath(c, 140, 35, 2.0, "M-25 -30A50 50 0 0 1 0 0A50 50 0 0 1 25 -30L-25 -30z")
	//drawStrokedPath(c, 30, 70, 2.0, "M0 -25A25 25 0 0 1 0 0A25 25 0 0 1 0 -25z") // CCW
	//drawStrokedPath(c, 60, 70, 2.0, "M0 -25A25 25 0 0 0 0 0A25 25 0 0 0 0 -25z") // CW
	//drawStrokedPath(c, 90, 65, 2.0, "M0 -25A25 25 0 0 0 0 0A25 25 0 0 1 20 -25z")
	//drawStrokedPath(c, 140, 50, 4.0, "M0 0A20 20 0 0 0 40 0A10 10 0 0 1 20 0z")
	//drawStrokedPath(c, 170, 20, 2.0, "C10 -13.33 10 -13.33 20 0z")
	//drawStrokedPath(c, 170, 30, 2.0, "C10 13.33 10 13.33 20 0z")

	// c.SetColor(canvas.LightGrey)
	// c.DrawPath(20.0, 20.0, 0.0, canvas.Rectangle(0.0, 0.0, 160.0, 40.0))
	// c.SetColor(canvas.Black)

	// ff := dejaVuSerif.Face(8.0)
	// text := canvas.NewTextBox(ff, "Lorem ipsum dolor sid amet, confiscusar patria est gravus repara sid ipsum. Apare tu garage.", 160.0, 40.0, canvas.Justify, canvas.Justify, 140.0)
	// c.DrawText(20, 60-ff.Metrics().Ascent, 0.0, text)

	//p, _ := canvas.ParseSVG("C0 10 10 10 100 0")
	//ps := p.SplitAt(52.5)
	//c.DrawPath(110, 20, 0, ps[0])
	//c.DrawPath(110, 20, 0, ps[1])
	//c.DrawPath(110, 30, 0, p.Dash(1.0, 1.0).Stroke(1.0, canvas.ButtCapper, canvas.RoundJoiner))

	//p, _ = canvas.ParseSVG("Q0 10 100 0")
	//ps = p.SplitAt(52.5)
	//c.DrawPath(110, 50, 0, ps[0])
	//c.DrawPath(110, 50, 0, ps[1])
	//c.DrawPath(110, 60, 0, p.Dash(1.0, 1.0).Stroke(1.0, canvas.ButtCapper, canvas.RoundJoiner))

	//p, _ = canvas.ParseSVG("A100 10 0 0 1 -100 10")
	//ps = p.SplitAt(52.5)
	//c.DrawPath(105, 40, 0, ps[0])
	//c.DrawPath(105, 40, 0, ps[1])
	//c.DrawPath(105, 50, 0, p.Dash(1.0, 1.0).Stroke(1.0, canvas.ButtCapper, canvas.RoundJoiner))

	c.SetColor(canvas.LightGrey)
	c.DrawPath(20.0, 60.0, 0.0, canvas.Rectangle(0.0, 0.0, 50.0, -20.0))
	c.SetColor(canvas.Black)
	rich := canvas.NewRichText()
	rich.Add(dejaVuSerif.Face(8.0), "Lorem ipsum dolor sit am\u200bet, confiscatur patria est gravus et mas")
	c.DrawText(20, 60, 0.0, rich.ToText(50.0, 20.0, canvas.Center, canvas.Justify, 10.0))
}
