package canvas

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Seanld/canvas/text"
)

func loadFont(name string, style FontStyle) ([]byte, error) {
	filename, ok := FindSystemFont(name, style)
	if !ok {
		return nil, fmt.Errorf("failed to find font '%s'", name)
	}
	return os.ReadFile(filename)
}

// DrawPreview draws the canvas's preview to a Context.
func DrawPreview(ctx *Context) error {
	root := os.Getenv("GOPATH")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), "go")
	}
	root = filepath.Join(root, "src/github.com/tdewolff/canvas")

	latin, err := loadFont("DejaVu Serif, serif", FontRegular)
	if err != nil {
		return err
	}
	arabic, err := loadFont("DejaVu Sans, sans", FontRegular)
	if err != nil {
		return err
	}
	devanagari, err := loadFont("Noto Serif Devanagari", FontRegular)
	if err != nil {
		return err
	}
	lenna, err := os.ReadFile(filepath.Join(root, "resources/lenna.png"))
	if err != nil {
		return err
	}
	return DrawPreviewWithAssets(ctx, latin, arabic, devanagari, lenna)
}

// DrawPreviewWithAssets draws the canvas's preview to a Context with assets preloaded.
func DrawPreviewWithAssets(ctx *Context, latin, arabic, devanagari, lenna []byte) error {
	fontLatin := NewFontFamily("latin")
	if err := fontLatin.LoadFont(latin, 0, FontRegular); err != nil {
		return err
	}

	fontArabic := NewFontFamily("arabic")
	if err := fontArabic.LoadFont(arabic, 0, FontRegular); err != nil {
		return err
	}

	fontDevanagari := NewFontFamily("devanagari")
	if err := fontDevanagari.LoadFont(devanagari, 0, FontRegular); err != nil {
		return err
	}

	W, H := ctx.Size()
	ctx.SetFillColor(White)
	ctx.DrawPath(0, 0, Rectangle(W, H))

	// Draw a comprehensive text box
	pt := 14.0
	face := fontLatin.Face(pt)
	rt := NewRichText(face)
	rt.WriteString("Lorem dolor ipsum ")
	rt.WriteFace(fontLatin.Face(pt, White, FontBold), "confiscator")
	rt.WriteString(" cur\u00ADabitur ")
	rt.WriteFace(fontLatin.Face(pt, FontItalic), "mattis")
	rt.WriteString(" dui ")
	rt.WriteFace(fontLatin.Face(pt, FontBold|FontItalic), "tellus")
	rt.WriteString(" vel. Proin ")
	rt.WriteFace(fontLatin.Face(pt, FontUnderline), "sodales")
	rt.WriteString(" eros vel ")
	rt.WriteFace(fontLatin.Face(pt, FontSineUnderline), "nibh")
	rt.WriteString(" fringilla pellen\u00ADtesque eu ")

	// Smiley face
	c2 := New(6.144, 6.144)
	ctx2 := NewContext(c2)
	ctx2.SetView(Identity.Translate(0.0, 6.144).Scale(0.05, -0.05))
	// face
	ctx2.SetFillColor(Hex("#fbd433"))
	ctx2.DrawPath(0.0, 0.0, MustParseSVGPath("M45.54,2.11A61.42,61.42,0,1,1,2.11,77.34,61.42,61.42,0,0,1,45.54,2.11Z"))
	// eyes
	ctx2.SetFillColor(Hex("#141518"))
	ctx2.DrawPath(0.0, 0.0, MustParseSVGPath("M45.78,32.27c4.3,0,7.79,5,7.79,11.27s-3.49,11.27-7.79,11.27S38,49.77,38,43.54s3.48-11.27,7.78-11.27Z"))
	ctx2.DrawPath(0.0, 0.0, MustParseSVGPath("M77.1,32.27c4.3,0,7.78,5,7.78,11.27S81.4,54.81,77.1,54.81s-7.79-5-7.79-11.27S72.8,32.27,77.1,32.27Z"))
	// mouth
	ctx2.DrawPath(0.0, 0.0, MustParseSVGPath("M28.8,70.82a39.65,39.65,0,0,0,8.83,8.41,42.72,42.72,0,0,0,25,7.53,40.44,40.44,0,0,0,24.12-8.12,35.75,35.75,0,0,0,7.49-7.87.22.22,0,0,1,.31,0L97,73.14a.21.21,0,0,1,0,.29A45.87,45.87,0,0,1,82.89,88.58,37.67,37.67,0,0,1,62.83,95a39,39,0,0,1-20.68-5.55A50.52,50.52,0,0,1,25.9,73.57a.23.23,0,0,1,0-.28l2.52-2.5a.22.22,0,0,1,.32,0l0,0Z"))
	rt.WriteCanvas(c2, FontMiddle)
	rt.WriteString(" cillum. ")

	face = fontLatin.Face(pt)
	face.Language = "ru"
	face.Script = text.Cyrillic
	rt.WriteFace(face, "дёжжэнтиюнт холст ")

	face = fontArabic.Face(pt)
	face.Language = "ar"
	face.Script = text.Arabic
	rt.WriteFace(face, "تسجّل يتكلّم ")

	face = fontDevanagari.Face(pt)
	face.Language = "hi"
	face.Script = text.Devanagari
	rt.WriteFace(face, "हालाँकि प्र ")

	x := 5.0
	y := 95.0
	metrics := face.Metrics()
	width, height := 90.0, 32.0
	text := rt.ToText(width, height, Justify, Top, 0.0, 0.0)
	ctx.SetFillColor(color.RGBA{192, 0, 64, 255})
	ctx.DrawPath(x, y, text.Bounds().ToPath())
	ctx.SetFillColor(color.RGBA{51, 51, 51, 51})
	ctx.DrawPath(x, y, Rectangle(width, -metrics.LineHeight))
	ctx.SetFillColor(color.RGBA{0, 0, 0, 51})
	ctx.DrawPath(x, y+metrics.CapHeight-metrics.Ascent, Rectangle(width, -metrics.CapHeight-metrics.Descent))
	ctx.DrawPath(x, y+metrics.XHeight-metrics.Ascent, Rectangle(width, -metrics.XHeight))
	ctx.SetFillColor(Black)
	ctx.DrawPath(x, y, Rectangle(width, -height).Stroke(0.2, RoundCap, RoundJoin, Tolerance))
	ctx.DrawText(x, y, text)

	// Draw the word Stroke being stroked
	gradient := NewRadialGradient(Point{120.0, 5.0}, 0.0, Point{140.0, 5.0}, 60.0)
	gradient.Add(0.0, Turquoise)
	gradient.Add(1.0, Red)
	ctx.SetFillGradient(gradient)

	face = fontLatin.Face(80.0)
	p, _, _ := face.ToPath("Stroke")
	ctx.DrawPath(100, 5, p.Stroke(0.75, RoundCap, RoundJoin, Tolerance))

	// Draw a LaTeX formula
	if runtime.GOOS != "js" {
		latex, err := ParseLaTeX(`$y_i = \frac{\sin(x)}{2} e^{\frac{a*b}{c}}$`)
		if err != nil {
			return err
		}
		latex = latex.Transform(Identity.Rotate(-30).Scale(2.0, 2.0))
		ctx.SetFillColor(Black)
		ctx.DrawPath(150, 90, latex)
	}

	// Draw an elliptic arc being dashed
	ellipse, err := ParseSVGPath("A10 30 30 1 0 30 0z")
	if err != nil {
		return err
	}
	ctx.SetFillColor(Whitesmoke)
	ctx.DrawPath(110, 60, ellipse)

	ctx.SetFillColor(Transparent)
	ctx.SetStrokeColor(Black)
	ctx.SetStrokeWidth(0.75)
	ctx.SetStrokeCapper(RoundCap)
	ctx.SetStrokeJoiner(RoundJoin)
	ctx.SetDashes(0.0, 2.0, 4.0, 2.0, 2.0, 4.0, 2.0)
	ctx.DrawPath(110, 60, ellipse)
	ctx.SetStrokeColor(Transparent)
	ctx.SetDashes(0.0)

	// Draw a raster image
	img, err := NewPNGImage(bytes.NewReader(lenna))
	if err != nil {
		return err
	}
	ctx.Push()
	ctx.Rotate(5)
	ctx.DrawImage(50.0, 10.0, img, 15)
	ctx.Pop()

	// Draw an closed set of points being smoothed
	polyline := &Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(30.0, 0.0)
	polyline.Add(30.0, 15.0)
	polyline.Add(0.0, 30.0)
	polyline.Add(0.0, 0.0)
	ctx.SetFillColor(Seagreen)
	col := Seagreen
	col.R = byte(float64(col.R) * 0.25)
	col.G = byte(float64(col.G) * 0.25)
	col.B = byte(float64(col.B) * 0.25)
	col.A = byte(float64(col.A) * 0.25)
	ctx.SetFillColor(col)
	ctx.SetStrokeColor(Seagreen)
	ctx.DrawPath(155, 35, polyline.Smoothen())

	ctx.SetFillColor(Transparent)
	ctx.SetStrokeColor(Black)
	ctx.SetStrokeWidth(0.5)
	ctx.DrawPath(155, 35, polyline.ToPath())
	for _, coord := range polyline.Coords() {
		ctx.DrawPath(155, 35, Circle(1.5).Translate(coord.X, coord.Y))
	}

	// Draw a open set of points being smoothed
	polyline = &Polyline{}
	polyline.Add(0.0, 0.0)
	polyline.Add(10.0, 5.0)
	polyline.Add(20.0, 15.0)
	polyline.Add(30.0, 20.0)
	polyline.Add(40.0, 10.0)
	ctx.SetStrokeColor(Dodgerblue)
	ctx.DrawPath(95, 30, polyline.Smoothen())
	ctx.SetStrokeColor(Black)
	for _, coord := range polyline.Coords() {
		ctx.DrawPath(95, 30, Circle(1.5).Translate(coord.X, coord.Y))
	}

	// Draw path boolean operations
	a := Circle(5.0)
	b := Circle(5.0).Translate(5.0, 0.0)
	a = a.Flatten(Tolerance)
	b = b.Flatten(Tolerance)
	ctx.SetFillColor(Transparent)
	ctx.SetStrokeColor(Black)
	ctx.SetStrokeWidth(0.1)
	face = fontLatin.Face(8.0)
	titles := []string{"A and B", "A or B", "A xor B", "A not B", "B not A"}
	for i := 0; i < 5; i++ {
		y := 56.0 - 12.0*float64(i)
		ctx.DrawText(15.0, y, NewTextBox(face, titles[i], 0.0, 0.0, Right, Middle, 0.0, 0.0))
		ctx.DrawPath(25.0, y, a)
		ctx.DrawPath(25.0, y, b)
	}
	ctx.SetFillColor(Hex("#00C8"))
	ctx.SetStrokeColor(Black)
	ctx.SetStrokeWidth(0.1)
	ctx.DrawPath(25.0, 56.0, a.And(b))
	ctx.DrawPath(25.0, 44.0, a.Or(b))
	ctx.DrawPath(25.0, 32.0, a.Xor(b))
	ctx.DrawPath(25.0, 20.0, a.Not(b))
	ctx.DrawPath(25.0, 8.0, b.Not(a))
	return nil
}
