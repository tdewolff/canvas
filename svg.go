package canvas

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"
)

type SVG struct {
	w             io.Writer
	width, height float64
	embedFonts    bool
	fonts         map[*Font]bool
	maskID        int
	imgEnc        ImageEncoding

	classes []string
}

// NewSVG creates a scalable vector graphics renderer.
func NewSVG(w io.Writer, width, height float64) *SVG {
	fmt.Fprintf(w, `<svg version="1.1" width="%vmm" height="%vmm" viewBox="0 0 %v %v" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, dec(width), dec(height), dec(width), dec(height))
	return &SVG{
		w:          w,
		width:      width,
		height:     height,
		embedFonts: true,
		fonts:      map[*Font]bool{},
		maskID:     0,
		imgEnc:     Lossless,
		classes:    []string{},
	}
}

func (r *SVG) Close() error {
	_, err := fmt.Fprintf(r.w, "</svg>")
	return err
}

func (r *SVG) AddClass(class string) {
	if class == "" {
		return
	}
	for _, c := range r.classes {
		if c == class {
			return
		}
	}
	r.classes = append(r.classes, class)
}

func (r *SVG) RemoveClass(class string) {
	for i, c := range r.classes {
		if c == class {
			r.classes = append(r.classes[:i], r.classes[i+1:]...)
			return
		}
	}
}

func (r *SVG) writeClasses(w io.Writer) {
	if len(r.classes) != 0 {
		fmt.Fprintf(w, `" class="%s`, strings.Join(r.classes, " "))
	}
}

func (r *SVG) EmbedFonts(embedFonts bool) {
	r.embedFonts = embedFonts
}

func (r *SVG) SetImageEncoding(enc ImageEncoding) {
	r.imgEnc = enc
}

func (r *SVG) writeFonts(fonts []*Font) {
	is := []int{}
	for i, font := range fonts {
		if _, ok := r.fonts[font]; !ok {
			is = append(is, i)
			r.fonts[font] = true
		}
	}

	if 0 < len(is) {
		fmt.Fprintf(r.w, "<style>")
		for _, i := range is {
			mimetype, raw := fonts[i].Raw()
			fmt.Fprintf(r.w, "\n@font-face{font-family:'%s';src:url('data:%s;base64,", fonts[i].name, mimetype)
			encoder := base64.NewEncoder(base64.StdEncoding, r.w)
			encoder.Write(raw)
			encoder.Close()
			fmt.Fprintf(r.w, "');}")
		}
		fmt.Fprintf(r.w, "\n</style>")
	}
}

func (r *SVG) Size() (float64, float64) {
	return r.width, r.height
}

func (r *SVG) RenderPath(path *Path, style Style, m Matrix) {
	fill := style.FillColor.A != 0
	stroke := style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth

	path = path.Transform(Identity.ReflectYAbout(r.height / 2.0).Mul(m))
	fmt.Fprintf(r.w, `<path d="%s`, path.ToSVG())

	strokeUnsupported := false
	if arcs, ok := style.StrokeJoiner.(ArcsJoiner); ok && math.IsNaN(arcs.Limit) {
		strokeUnsupported = true
	} else if miter, ok := style.StrokeJoiner.(MiterJoiner); ok {
		if math.IsNaN(miter.Limit) {
			strokeUnsupported = true
		} else if _, ok := miter.GapJoiner.(BevelJoiner); !ok {
			strokeUnsupported = true
		}
	}

	if !stroke {
		if fill {
			if style.FillColor != Black {
				fmt.Fprintf(r.w, `" fill="%v`, CSSColor(style.FillColor))
			}
			if style.FillRule == EvenOdd {
				fmt.Fprintf(r.w, `" fill-rule="evenodd`)
			}
		} else {
			fmt.Fprintf(r.w, `" fill="none`)
		}
	} else {
		b := &strings.Builder{}
		if fill {
			if style.FillColor != Black {
				fmt.Fprintf(b, ";fill:%v", CSSColor(style.FillColor))
			}
			if style.FillRule == EvenOdd {
				fmt.Fprintf(b, ";fill-rule:evenodd")
			}
		} else {
			fmt.Fprintf(b, ";fill:none")
		}
		if stroke && !strokeUnsupported {
			fmt.Fprintf(b, `;stroke:%v`, CSSColor(style.StrokeColor))
			if style.StrokeWidth != 1.0 {
				fmt.Fprintf(b, ";stroke-width:%v", dec(style.StrokeWidth))
			}
			if _, ok := style.StrokeCapper.(RoundCapper); ok {
				fmt.Fprintf(b, ";stroke-linecap:round")
			} else if _, ok := style.StrokeCapper.(SquareCapper); ok {
				fmt.Fprintf(b, ";stroke-linecap:square")
			} else if _, ok := style.StrokeCapper.(ButtCapper); !ok {
				panic("SVG: line cap not support")
			}
			if _, ok := style.StrokeJoiner.(BevelJoiner); ok {
				fmt.Fprintf(b, ";stroke-linejoin:bevel")
			} else if _, ok := style.StrokeJoiner.(RoundJoiner); ok {
				fmt.Fprintf(b, ";stroke-linejoin:round")
			} else if arcs, ok := style.StrokeJoiner.(ArcsJoiner); ok && !math.IsNaN(arcs.Limit) {
				fmt.Fprintf(b, ";stroke-linejoin:arcs")
				if !equal(arcs.Limit, 4.0) {
					fmt.Fprintf(b, ";stroke-miterlimit:%v", dec(arcs.Limit))
				}
			} else if miter, ok := style.StrokeJoiner.(MiterJoiner); ok && !math.IsNaN(miter.Limit) {
				// a miter line join is the default
				if !equal(miter.Limit*2.0/style.StrokeWidth, 4.0) {
					fmt.Fprintf(b, ";stroke-miterlimit:%v", dec(miter.Limit*2.0/style.StrokeWidth))
				}
			} else {
				panic("SVG: line join not support")
			}

			if 0 < len(style.Dashes) {
				fmt.Fprintf(b, ";stroke-dasharray:%v", dec(style.Dashes[0]))
				for _, dash := range style.Dashes[1:] {
					fmt.Fprintf(b, " %v", dec(dash))
				}
				if 0.0 != style.DashOffset {
					fmt.Fprintf(b, ";stroke-dashoffset:%v", dec(style.DashOffset))
				}
			}
		}
		if 0 < b.Len() {
			fmt.Fprintf(r.w, `" style="%s`, b.String()[1:])
		}
	}
	r.writeClasses(r.w)
	fmt.Fprintf(r.w, `"/>`)

	if stroke && strokeUnsupported {
		// stroke settings unsupported by PDF, draw stroke explicitly
		if 0 < len(style.Dashes) {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)
		fmt.Fprintf(r.w, `<path d="%s`, path.ToSVG())
		if style.StrokeColor != Black {
			fmt.Fprintf(r.w, `" fill="%v`, CSSColor(style.StrokeColor))
		}
		if style.FillRule == EvenOdd {
			fmt.Fprintf(r.w, `" fill-rule="evenodd`)
		}
		r.writeClasses(r.w)
		fmt.Fprintf(r.w, `"/>`)
	}
}

func (r *SVG) writeFontStyle(ff, ffMain FontFace) {
	boldness := ff.boldness()
	differences := 0
	if ff.style&FontItalic != ffMain.style&FontItalic {
		differences++
	}
	if boldness != ffMain.boldness() {
		differences++
	}
	if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
		differences++
	}
	if ff.color != ffMain.color {
		differences++
	}
	if ff.font.name != ffMain.font.name || ff.size*ff.scale != ffMain.size || differences == 3 {
		fmt.Fprintf(r.w, `" style="font:`)

		buf := &bytes.Buffer{}
		if ff.style&FontItalic != ffMain.style&FontItalic {
			fmt.Fprintf(buf, ` italic`)
		}

		if boldness != ffMain.boldness() {
			fmt.Fprintf(buf, ` %d`, boldness)
		}

		if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
			fmt.Fprintf(buf, ` small-caps`)
		}

		fmt.Fprintf(buf, ` %vpx %s`, num(ff.size*ff.scale), ff.font.name)
		buf.ReadByte()
		buf.WriteTo(r.w)

		if ff.color != ffMain.color {
			fmt.Fprintf(r.w, `;fill:%v`, CSSColor(ff.color))
		}
	} else if differences == 1 && ff.color != ffMain.color {
		fmt.Fprintf(r.w, `" fill="%v`, CSSColor(ff.color))
	} else if 0 < differences {
		fmt.Fprintf(r.w, `" style="`)
		buf := &bytes.Buffer{}
		if ff.style&FontItalic != ffMain.style&FontItalic {
			fmt.Fprintf(buf, `;font-style:italic`)
		}
		if boldness != ffMain.boldness() {
			fmt.Fprintf(buf, `;font-weight:%d`, boldness)
		}
		if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
			fmt.Fprintf(buf, `;font-variant:small-caps`)
		}
		if ff.color != ffMain.color {
			fmt.Fprintf(buf, `;fill:%v`, CSSColor(ff.color))
		}
		buf.ReadByte()
		buf.WriteTo(r.w)
	}
}

func (r *SVG) RenderText(text *Text, m Matrix) {
	if r.embedFonts {
		r.writeFonts(text.Fonts())
	}

	if len(text.lines) == 0 || len(text.lines[0].spans) == 0 {
		return
	}

	ffMain := text.mostCommonFontFace()

	x0, y0 := 0.0, 0.0
	if m.IsTranslation() {
		x0, y0 = m.Pos()
		y0 = r.height - y0
		fmt.Fprintf(r.w, `<text x="%v" y="%v`, num(x0), num(y0))
	} else {
		fmt.Fprintf(r.w, `<text transform="%s`, m.ToSVG(r.height))
	}
	fmt.Fprintf(r.w, `" style="font:`)
	if ffMain.style&FontItalic != 0 {
		fmt.Fprintf(r.w, ` italic`)
	}
	if boldness := ffMain.boldness(); boldness != 400 {
		fmt.Fprintf(r.w, ` %d`, boldness)
	}
	if ffMain.variant&FontSmallcaps != 0 {
		fmt.Fprintf(r.w, ` small-caps`)
	}
	fmt.Fprintf(r.w, ` %vpx %s`, num(ffMain.size*ffMain.scale), ffMain.font.name)
	if ffMain.color != Black {
		fmt.Fprintf(r.w, `;fill:%v`, CSSColor(ffMain.color))
	}
	r.writeClasses(r.w)
	fmt.Fprintf(r.w, `">`)

	decoPaths := []*Path{}
	decoColors := []color.RGBA{}
	for _, line := range text.lines {
		for _, span := range line.spans {
			fmt.Fprintf(r.w, `<tspan x="%v" y="%v`, num(x0+span.dx), num(y0-line.y-span.ff.voffset))
			if span.wordSpacing > 0.0 {
				fmt.Fprintf(r.w, `" word-spacing="%v`, num(span.wordSpacing))
			}
			if span.glyphSpacing > 0.0 {
				fmt.Fprintf(r.w, `" letter-spacing="%v`, num(span.glyphSpacing))
			}
			r.writeFontStyle(span.ff, ffMain)
			s := span.text
			s = strings.ReplaceAll(s, `"`, `&quot;`)
			r.writeClasses(r.w)
			fmt.Fprintf(r.w, `">%s</tspan>`, s)
		}
		for _, deco := range line.decos {
			p := deco.ff.Decorate(deco.x1 - deco.x0)
			p = p.Transform(Identity.Mul(m).Translate(deco.x0, line.y+deco.ff.voffset))
			decoPaths = append(decoPaths, p)
			decoColors = append(decoColors, deco.ff.color)
		}
	}
	fmt.Fprintf(r.w, `</text>`)
	style := DefaultStyle
	for i := range decoPaths {
		style.FillColor = decoColors[i]
		r.RenderPath(decoPaths[i], style, Identity)
	}
}

func (r *SVG) RenderImage(img image.Image, m Matrix) {
	refMask := ""
	mimetype := "image/png"
	if r.imgEnc == Lossy {
		mimetype = "image/jpg"
		if opaqueImg, ok := img.(interface{ Opaque() bool }); !ok || !opaqueImg.Opaque() {
			hasMask := false
			size := img.Bounds().Size()
			opaque := image.NewRGBA(img.Bounds())
			mask := image.NewGray(img.Bounds())
			for y := 0; y < size.Y; y++ {
				for x := 0; x < size.X; x++ {
					R, G, B, A := img.At(x, y).RGBA()
					if A != 0 {
						r := byte((R * 65535 / A) >> 8)
						g := byte((G * 65535 / A) >> 8)
						b := byte((B * 65535 / A) >> 8)
						opaque.SetRGBA(x, y, color.RGBA{r, g, b, 255})
						mask.SetGray(x, y, color.Gray{byte(A >> 8)})
					}
					if A>>8 != 255 {
						hasMask = true
					}
				}
			}
			if hasMask {
				img = opaque
				refMask = fmt.Sprintf("m%v", r.maskID)
				r.maskID++

				fmt.Fprintf(r.w, `<mask id="%s"><image width="%d" height="%d" xlink:href="data:image/jpg;base64,`, refMask, size.X, size.Y)
				encoder := base64.NewEncoder(base64.StdEncoding, r.w)
				if err := jpeg.Encode(encoder, mask, nil); err != nil {
					panic(err)
				}
				if err := encoder.Close(); err != nil {
					panic(err)
				}
				fmt.Fprintf(r.w, `"/></mask>`)
			}
		}
	}

	m = m.Translate(0.0, float64(img.Bounds().Size().Y))
	fmt.Fprintf(r.w, `<image transform="%s" width="%d" height="%d" xlink:href="data:%s;base64,`,
		m.ToSVG(r.height), img.Bounds().Size().X, img.Bounds().Size().Y, mimetype)

	encoder := base64.NewEncoder(base64.StdEncoding, r.w)
	if mimetype == "image/jpg" {
		if err := jpeg.Encode(encoder, img, nil); err != nil {
			panic(err)
		}
	} else {
		if err := png.Encode(encoder, img); err != nil {
			panic(err)
		}
	}
	if err := encoder.Close(); err != nil {
		panic(err)
	}

	if refMask != "" {
		fmt.Fprintf(r.w, `" mask="url(#%s)`, refMask)
	}
	r.writeClasses(r.w)
	fmt.Fprintf(r.w, `"/>`)
}
