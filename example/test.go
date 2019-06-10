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

	c := canvas.New(200, 80)
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
	c.SetFillColor(canvas.Black)
	p, err := canvas.ParseSVG(path)
	if err != nil {
		panic(err)
	}
	c.DrawPath(x, y, p)

	p = p.Stroke(d, canvas.ButtCapper, canvas.MiterClipJoiner(canvas.RoundJoiner, d))
	c.SetFillColor(color.RGBA{255, 0, 0, 127})
	c.DrawPath(x, y, p)

	p = p.Stroke(0.2, canvas.RoundCapper, canvas.RoundJoiner)
	c.SetFillColor(color.RGBA{255, 0, 0, 255})
	c.DrawPath(x, y, p)
}

func Draw(c *canvas.C) {
	c.SetStrokeColor(canvas.Blue)
	c.SetDashes(-1.0, 5.0)

	p, _ := canvas.ParseSVG(fmt.Sprintf("H10.0Q20.0 0.0 20.0 10.0A20.0 10.0 45.0 0 1 0.0 20.0z"))
	c.DrawPath(30.0, 30.0, p)

	//p, _ := canvas.ParseSVG(fmt.Sprintf("M10 0V10H-10V-10H10zM5 0V-5H-5V5H5z"))
	//p = p.Offset(1.0)

	//ellipse, _ := canvas.ParseSVG(fmt.Sprintf("A10 20 0 0 0 20 0z"))
	//c.SetColor(canvas.Red)
	//c.DrawPath(50.0, 40.0, 0.0, ellipse)
	//ps := ellipse.SplitAt(10.0)
	//ellipse = ps[0]
	//fmt.Println(ellipse)
	//ellipse = ellipse.Stroke(2.0, canvas.RoundCapper, canvas.RoundJoiner)
	//ellipse = ellipse.Flatten()
	//c.SetColor(canvas.BlackTransparent)
	//c.DrawPath(50.0, 40.0, 0.0, ellipse)

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

	//c.SetColor(canvas.LightGrey)
	////c.DrawPath(20.0, 60.0, 0.0, canvas.Rectangle(0.0, 0.0, 50.0, -20.0))
	//c.SetColor(canvas.Black)
	//rich := canvas.NewRichText()
	//rich.Add(dejaVuSerif.Face(8.0), "Lorem ")
	//rich.Add(dejaVuSerif.Face(8.0).FauxBold(), "ipsum ")
	//rich.Add(dejaVuSerif.Face(8.0).FauxItalic(), "dolor ")
	//rich.Add(dejaVuSerif.Face(8.0), "sit am\u200bet, ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.DoubleUnderline), "confiscatur")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.SineUnderline), "patria")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.SawtoothUnderline), "est")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.DottedUnderline), "gravus")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.DashedUnderline), "instantum")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.Underline), "norpe")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.Strikethrough), "targe")
	//rich.Add(dejaVuSerif.Face(8.0), " ")
	//rich.Add(dejaVuSerif.Face(8.0).Decoration(canvas.Overline), "yatum")
	//text := rich.ToText(50.0, 20.0, canvas.Left, canvas.Top, 0.0)
	//c.DrawPath(10, 70, 0.0, text.ToPath().Scale(2.0, 2.0))
	//c.DrawPath(10, 70, 0.0, text.ToPathDecorations().Scale(2.0, 2.0))
}
