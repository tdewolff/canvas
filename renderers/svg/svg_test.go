package svg

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/test"
)

func TestSVGText(t *testing.T) {
	//dejaVuSerif := NewFontFamily("dejavu-serif")
	//dejaVuSerif.LoadFontFile("font/DejaVuSerif.ttf", FontRegular)

	//ebGaramond := NewFontFamily("eb-garamond")
	//ebGaramond.LoadFontFile("font/EBGaramond12-Regular.otf", FontRegular)

	//dejaVu8 := dejaVuSerif.Face(8.0*ptPerMm, Black, FontRegular, FontNormal)
	//dejaVu12 := dejaVuSerif.Face(12.0*ptPerMm, Red, FontItalic, FontNormal, FontUnderline)
	//dejaVu12sub := dejaVuSerif.Face(12.0*ptPerMm, Black, FontRegular, FontSubscript)
	//garamond10 := ebGaramond.Face(10.0*ptPerMm, Black, FontBold, FontNormal)

	//rt := NewRichText(dejaVu12)
	//rt.WriteFace(dejaVu8, "dejaVu8")
	//rt.WriteFace(dejaVu12, " glyphspacing")
	//rt.WriteFace(dejaVu12sub, " dejaVu12sub")
	//rt.WriteFace(garamond10, " garamond10")
	//text := rt.ToText(dejaVu12.TextWidth("glyphspacing")+float64(len("glyphspacing")-1), 100.0, Justify, Top, 0.0, 0.0)

	//buf := &bytes.Buffer{}
	//svg := newSVGWriter(buf, 0.0, 0.0)
	//buf.Reset()
	//textLayer{text, Identity}.WriteSVG(svg)
	//s := regexp.MustCompile(`base64,.+'`).ReplaceAllString(buf.String(), "base64,'") // remove embedded font
	//test.String(t, s, `<style>`+"\n"+`@font-face{font-family:'dejavu-serif';src:url('data:font/truetype;base64,');}`+"\n"+`@font-face{font-family:'eb-garamond';src:url('data:font/opentype;base64,');}`+"\n"+`</style><text x="0" y="0" style="font: 12px dejavu-serif"><tspan x="0" y="7.421875" style="font:8px dejavu-serif">dejaVu8</tspan><tspan x="0" y="20.453125" letter-spacing="1" style="font-style:italic;fill:#f00">glyphspacing</tspan><tspan x="0" y="33.725625" style="font:700 6.996px dejavu-serif">dejaVu12sub</tspan><tspan x="0" y="38.5" style="font:700 10px eb-garamond">garamond10</tspan></text><path d="M0 22.703125H91.71875V21.803125H0z" fill="#f00"/>`)
}

// renderSVG draws onto a 100x100 canvas and returns the SVG output without the <svg> header.
func renderSVG(draw func(*canvas.Context)) string {
	c := canvas.New(100.0, 100.0)
	draw(canvas.NewContext(c))

	buf := &bytes.Buffer{}
	r := New(buf, 100.0, 100.0, nil)
	c.RenderTo(r)
	r.Close()

	s := buf.String()
	if i := strings.IndexByte(s, '>'); i != -1 {
		s = s[i+1:]
	}
	return strings.TrimSuffix(s, `</svg>`)
}

// SVG 1.1 does not allow rgba() where a <color> is expected, renderers that implement SVG 1.1
// rather than CSS Color 4 discard the value and fall back to black. Translucent paints must be
// written as an opaque color with the alpha in a separate *-opacity property. See issue #385.
func TestPaintOpacity(t *testing.T) {
	blue := color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 178}

	var tests = []struct {
		name string
		draw func(*canvas.Context)
		svg  string
	}{
		{"opaque fill", func(ctx *canvas.Context) {
			ctx.SetFillColor(canvas.Red)
			ctx.DrawPath(10.0, 10.0, canvas.Rectangle(80.0, 80.0))
		}, `<path d="M10 90H90V10H10z" fill="#f00"/>`},

		{"translucent fill", func(ctx *canvas.Context) {
			ctx.SetFillColor(blue)
			ctx.DrawPath(10.0, 10.0, canvas.Rectangle(80.0, 80.0))
		}, `<path d="M10 90H90V10H10z" fill="#3a82f7" fill-opacity=".69803922"/>`},

		{"translucent black fill", func(ctx *canvas.Context) {
			ctx.SetFillColor(canvas.RGBA(0.0, 0.0, 0.0, 0.5))
			ctx.DrawPath(10.0, 10.0, canvas.Rectangle(80.0, 80.0))
		}, `<path d="M10 90H90V10H10z" fill="#000" fill-opacity=".49803922"/>`},

		{"translucent fill with stroke", func(ctx *canvas.Context) {
			ctx.SetFillColor(blue)
			ctx.SetStrokeColor(canvas.Black)
			ctx.SetStrokeWidth(1.0)
			ctx.DrawPath(10.0, 10.0, canvas.Rectangle(80.0, 80.0))
		}, `<path d="M10 90H90V10H10z" style="fill:#3a82f7;fill-opacity:.69803922;stroke:#000"/>`},

		{"translucent stroke", func(ctx *canvas.Context) {
			ctx.SetFillColor(canvas.Transparent)
			ctx.SetStrokeColor(canvas.RGBA(1.0, 0.0, 0.0, 0.5))
			ctx.SetStrokeWidth(1.0)
			ctx.DrawPath(10.0, 10.0, canvas.Rectangle(80.0, 80.0))
		}, `<path d="M10 90H90V10H10z" style="fill:none;stroke:#f00;stroke-opacity:.49803922"/>`},

		{"translucent gradient stop", func(ctx *canvas.Context) {
			grad := canvas.NewLinearGradient(canvas.Point{X: 0.0, Y: 0.0}, canvas.Point{X: 100.0, Y: 0.0})
			grad.Add(0.0, canvas.RGBA(0.0, 1.0, 0.0, 0.25))
			grad.Add(1.0, canvas.Blue)
			ctx.SetFill(canvas.Paint{Gradient: grad})
			ctx.DrawPath(10.0, 10.0, canvas.Rectangle(80.0, 80.0))
		}, `<path d="M10 90H90V10H10z" fill="url(#d0)"/><defs><linearGradient id="d0" gradientUnits="userSpaceOnUse" x1="10" y1="90" x2="110" y2="90"><stop offset="0" stop-color="#0f0" stop-opacity=".24705882"/><stop offset="1" stop-color="#00f"/></linearGradient></defs>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := renderSVG(tt.draw)
			test.String(t, s, tt.svg)
			if strings.Contains(s, "rgba(") {
				test.Fail(t, "output must not contain rgba(), it is not valid SVG 1.1:", s)
			}
		})
	}
}

func TestSplitAlpha(t *testing.T) {
	var tests = []struct {
		col     color.RGBA
		css     string
		opacity float64
	}{
		{canvas.Red, "#f00", 1.0},
		{canvas.Black, "#000", 1.0},
		{canvas.Transparent, "#000", 0.0},
		{canvas.RGBA(1.0, 0.0, 0.0, 0.5), "#f00", 0.49803921568627452},
		{color.RGBA{R: 41, G: 91, B: 172, A: 178}, "#3a82f7", 0.69803921568627447},
	}
	for _, tt := range tests {
		col, opacity := splitAlpha(tt.col)
		test.String(t, col.String(), tt.css)
		test.Float(t, opacity, tt.opacity)
	}
}
