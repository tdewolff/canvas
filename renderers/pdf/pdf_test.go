package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tdewolff/canvas"
	cimage "github.com/tdewolff/canvas/image"
	"github.com/tdewolff/test"
)

func TestPDF(t *testing.T) {
	c := canvas.New(10, 10)
	c.RenderPath(canvas.MustParseSVGPath("L10 0"), canvas.DefaultStyle, canvas.Identity)

	//	pdfCompress = false
	//	buf := &bytes.Buffer{}
	//	c.WritePDF(buf)
	//	test.T(t, buf.String(), `%PDF-1.7
	//1 0 obj
	//<< /Length 14 >> stream
	//0 0 m 10 0 l f
	//endstream
	//endobj
	//2 0 obj
	//<< /Type /Page /Contents 1 0 R /Group << /Type /Group /CS /DeviceRGB /I true /S /Transparency >> /MediaBox [0 0 10 10] /Parent 2 0 R /Resources << >> >>
	//endobj
	//3 0 obj
	//<< /Type /Pages /Count 1 /Kids [2 0 R] >>
	//endobj
	//4 0 obj
	//<< /Type /Catalog /Pages 3 0 R >>
	//endobj
	//xref
	//0 5
	//0000000000 65535 f
	//0000000009 00000 n
	//0000000073 00000 n
	//0000000241 00000 n
	//0000000298 00000 n
	//trailer
	//<< /Root 4 0 R /Size 4 >>
	//starxref
	//347
	//%%EOF`)
}

func TestPDFPath(t *testing.T) {
	buf := &bytes.Buffer{}
	pdf := newPDFWriter(buf).NewPage(210.0, 297.0)
	pdf.SetAlpha(0.5)
	pdf.SetFill(canvas.Paint{Color: canvas.Red}, canvas.Identity)
	pdf.SetStroke(canvas.Paint{Color: canvas.Blue}, canvas.Identity)
	pdf.SetLineWidth(5.0)
	pdf.SetLineCap(canvas.RoundCap)
	pdf.SetLineJoin(canvas.RoundJoin)
	pdf.SetDashes(2.0, []float64{1.0, 2.0, 3.0})
	test.String(t, pdf.String(), " 2.8346457 0 0 2.8346457 0 0 cm /A0 gs 1 0 0 rg /A1 gs 0 0 1 RG 5 w 1 J 1 j [1 2 3 1 2 3] 2 d")
}

const fontDir = "../../resources/"

func TestPDFText(t *testing.T) {
	t.Run("without_subset", func(t *testing.T) {
		doTestPDFText(t, false, 506000, "TestPDFText_no_subset.pdf")
	})
	t.Run("with_subset", func(t *testing.T) {
		doTestPDFText(t, true, 9500, "TestPDFText_subset_fonts.pdf")
	})
}

func doTestPDFText(t *testing.T, subsetFonts bool, expectedSize int, filename string) {
	dejaVuSerif := canvas.NewFontFamily("dejavu-serif")
	err := dejaVuSerif.LoadFontFile(fontDir+"DejaVuSerif.ttf", canvas.FontRegular)
	test.Error(t, err)

	ebGaramond := canvas.NewFontFamily("eb-garamond")
	err = ebGaramond.LoadFontFile(fontDir+"EBGaramond12-Regular.otf", canvas.FontRegular)
	test.Error(t, err)

	dejaVu8 := dejaVuSerif.Face(8, canvas.Black, canvas.FontRegular, canvas.FontNormal)
	dejaVu12 := dejaVuSerif.Face(12, canvas.Red, canvas.FontRegular, canvas.FontNormal, canvas.FontUnderline)
	dejaVu12sub := dejaVuSerif.Face(12, canvas.Black, canvas.FontRegular, canvas.FontSubscript)
	garamond10 := ebGaramond.Face(10, canvas.Black, canvas.FontBold, canvas.FontNormal)

	rt := canvas.NewRichText(dejaVu12)
	rt.WriteFace(dejaVu8, "dejaVu8")
	rt.WriteFace(dejaVu12, " glyphspacing")
	rt.WriteFace(dejaVu12sub, " dejaVu12sub")
	rt.WriteFace(garamond10, " garamond10")
	text := rt.ToText(180, 20.0, canvas.Justify, canvas.Top, nil)

	buf := &bytes.Buffer{}
	var w io.Writer = buf
	if testing.Verbose() {
		f, _ := os.Create(filename) // for manual inspection
		defer f.Close()
		w = io.MultiWriter(buf, f)
	}

	pdf := New(w, 210, 297, &Options{Compress: false, SubsetFonts: subsetFonts})

	pdf.RenderText(text, canvas.Identity.Translate(15, 250))

	pdf.Close()

	written := len(buf.Bytes()) // expecting around 506K
	test.That(t, expectedSize-1000 < written && written < expectedSize+1000, "Unexpected rendering result length")
}

func TestPDFImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))

	buf := &bytes.Buffer{}
	pdf := newPDFWriter(buf).NewPage(210.0, 297.0)
	pdf.DrawImage(img, cimage.Lossless, canvas.Identity)
	test.String(t, pdf.String(), " 2.8346457 0 0 2.8346457 0 0 cm q 0 0 2 2 re W n 0 0 m 0 2 l 2 2 l 2 0 l h W n 2 0 0 2 0 0 cm /Im0 Do Q")
}

func TestPDFMultipage(t *testing.T) {
	buf := &bytes.Buffer{}
	pdf := New(buf, 210, 297, nil)
	pdf.NewPage(210, 297)
	err := pdf.Close()
	test.Error(t, err)
	out := buf.String()

	test.That(t, strings.Contains(out, "/Type/Pages/Count 2"), `could not find "/Type /Pages /Count 2" in output`)

	nbPages := strings.Count(out, "/Type/Page/")
	test.That(t, nbPages == 2, "expected 2 pages, got", nbPages)
}

func TestPDFMetadata(t *testing.T) {
	buf := &bytes.Buffer{}
	pdf := New(buf, 210, 297, nil)
	pdf.NewPage(210, 297)
	pdf.SetInfo("a1", "b2", "c3", "d4", "e5")
	err := pdf.Close()
	test.Error(t, err)
	out := buf.String()

	test.That(t, strings.Contains(out, "/Title(a1)"), `could not find "/Title (a1)" in output`)
	test.That(t, strings.Contains(out, "/Subject(b2)"), `could not find "/Subject (b2)" in output`)
	test.That(t, strings.Contains(out, "/Keywords(c3)"), `could not find "/Keywords (c3)" in output`)
	test.That(t, strings.Contains(out, "/Author(d4)"), `could not find "/Author (d4)" in output`)
	test.That(t, strings.Contains(out, "/Creator(e5)"), `could not find "/Creator (e5)" in output`)
}

// TestPDFTransparentPaintNoNaN pins the guard in SetFill / SetStroke.
//
// Paint.Color is premultiplied, and both writers un-premultiply by
// dividing each component by alpha. At alpha 0 the components are zero
// too, so the division is 0/0 and Go prints the result as "NaN" — not a
// PDF number. Viewers parse it as an operator and give up on the rest of
// the content stream, so every object drawn after that point on the page
// disappears: `color: transparent` on one paragraph used to blank the
// whole page, text extraction included.
//
// R == G == B on a transparent paint also sends it down the grayscale
// branch, which is why the symptom was the single-token " NaN g" rather
// than " NaN NaN NaN rg".
func TestPDFTransparentPaintNoNaN(t *testing.T) {
	for _, tt := range []struct {
		name  string
		color color.RGBA
	}{
		{"transparent black", color.RGBA{0, 0, 0, 0}},
		{"transparent red", color.RGBA{255, 0, 0, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			pdf := newPDFWriter(buf).NewPage(210.0, 297.0)
			pdf.SetFill(canvas.Paint{Color: tt.color}, canvas.Identity)
			pdf.SetStroke(canvas.Paint{Color: tt.color}, canvas.Identity)
			out := pdf.String()

			test.That(t, !strings.Contains(out, "NaN"), "NaN in content stream:", out)
			// Black, plus the alpha-0 ExtGState that makes it invisible.
			test.That(t, strings.Contains(out, " 0 g"), `expected " 0 g":`, out)
			test.That(t, strings.Contains(out, " 0 G"), `expected " 0 G":`, out)
			test.That(t, strings.Contains(out, " gs"), "expected an alpha ExtGState:", out)
		})
	}
}

// TestPDFOpaquePaintUnchanged guards the other side of the same branch:
// the fix must not disturb colors that were already being written.
func TestPDFOpaquePaintUnchanged(t *testing.T) {
	buf := &bytes.Buffer{}
	pdf := newPDFWriter(buf).NewPage(210.0, 297.0)
	pdf.SetFill(canvas.Paint{Color: color.RGBA{255, 0, 0, 255}}, canvas.Identity)
	test.That(t, strings.Contains(pdf.String(), " 1 0 0 rg"), `expected " 1 0 0 rg":`, pdf.String())

	buf2 := &bytes.Buffer{}
	pdf2 := newPDFWriter(buf2).NewPage(210.0, 297.0)
	// Premultiplied 50% gray at 50% alpha un-premultiplies back to 1.
	pdf2.SetFill(canvas.Paint{Color: color.RGBA{128, 128, 128, 128}}, canvas.Identity)
	test.That(t, !strings.Contains(pdf2.String(), "NaN"), "NaN in content stream:", pdf2.String())
	test.That(t, strings.Contains(pdf2.String(), " g"), `expected a grayscale fill:`, pdf2.String())
}

// TestPDFTransparentGradientStopNoNaN is the shading-dictionary sibling of
// TestPDFTransparentPaintNoNaN.
//
// A Type 2 function names straight colors, so patternStopFunction
// un-premultiplies each stop. A fully transparent stop divides 0/0 and used to
// write NaN into C0/C1; viewers reject the whole shading with "Illegal value in
// function C1 array". This one is reached far more easily than the fill/stroke
// case — `linear-gradient(red, transparent)` is everyday CSS.
func TestPDFTransparentGradientStopNoNaN(t *testing.T) {
	grad := canvas.NewLinearGradient(canvas.Point{0, 0}, canvas.Point{10, 0})
	grad.Add(0.0, color.RGBA{255, 0, 0, 255})
	grad.Add(1.0, color.RGBA{0, 0, 0, 0}) // transparent

	buf := &bytes.Buffer{}
	r := New(buf, 100, 100, &Options{Compress: false, SubsetFonts: false})
	style := canvas.DefaultStyle
	style.Fill = canvas.Paint{Gradient: grad}
	r.RenderPath(canvas.MustParseSVGPath("M0 0L10 0L10 10L0 10z"), style, canvas.Identity)
	test.Error(t, r.Close())

	out := buf.String()
	test.That(t, !strings.Contains(out, "NaN"), "NaN in shading function:", out)

	// The transparent stop resolves to a finite triple. (A bare "Inf" scan
	// would false-positive on the trailer's /Info key.)
	fn := patternStopFunction(
		canvas.Stop{Offset: 0.0, Color: color.RGBA{255, 0, 0, 255}},
		canvas.Stop{Offset: 1.0, Color: color.RGBA{0, 0, 0, 0}},
	)
	test.T(t, fmt.Sprint(fn["C0"]), "[1 0 0]")
	test.T(t, fmt.Sprint(fn["C1"]), "[0 0 0]")
}

// TestPDFOpaqueGradientStopsUnchanged guards the other side: an ordinary
// gradient must still un-premultiply to its straight colors.
func TestPDFOpaqueGradientStopsUnchanged(t *testing.T) {
	fn := patternStopFunction(
		canvas.Stop{Offset: 0.0, Color: color.RGBA{255, 0, 0, 255}},
		canvas.Stop{Offset: 1.0, Color: color.RGBA{0, 0, 255, 255}},
	)
	test.T(t, fmt.Sprint(fn["C0"]), "[1 0 0]")
	test.T(t, fmt.Sprint(fn["C1"]), "[0 0 1]")
}
