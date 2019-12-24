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

type svgWriter struct {
	io.Writer
	height float64
	fonts  map[*Font]bool
	maskID int
}

func newSVGWriter(writer io.Writer, w, h float64) *svgWriter {
	fmt.Fprintf(writer, `<svg version="1.1" width="%v" height="%v" viewBox="0 0 %v %v" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, dec(w), dec(h), dec(w), dec(h))
	return &svgWriter{writer, h, map[*Font]bool{}, 0}
}

func (w *svgWriter) Close() error {
	fmt.Fprintf(w, "</svg>")
	return nil
}

func (w *svgWriter) EmbedFonts(fonts []*Font) {
	is := []int{}
	for i, font := range fonts {
		if _, ok := w.fonts[font]; !ok {
			is = append(is, i)
			w.fonts[font] = true
		}
	}

	if 0 < len(is) {
		fmt.Fprintf(w, "<style>")
		for _, i := range is {
			mimetype, raw := fonts[i].Raw()
			fmt.Fprintf(w, "\n@font-face{font-family:'%s';src:url('data:%s;base64,", fonts[i].name, mimetype)
			encoder := base64.NewEncoder(base64.StdEncoding, w)
			encoder.Write(raw)
			encoder.Close()
			fmt.Fprintf(w, "');}")
		}
		fmt.Fprintf(w, "\n</style>")
	}
}

func (w *svgWriter) DrawImage(img image.Image, enc ImageEncoding, m Matrix) {
	refMask := ""
	mimetype := "image/png"
	if enc == Lossy {
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
				refMask = fmt.Sprintf("m%v", w.maskID)
				w.maskID++

				fmt.Fprintf(w, `<mask id="%s"><image width="%d" height="%d" xlink:href="data:image/jpg;base64,`, refMask, size.X, size.Y)
				encoder := base64.NewEncoder(base64.StdEncoding, w)
				if err := jpeg.Encode(encoder, mask, nil); err != nil {
					panic(err)
				}
				if err := encoder.Close(); err != nil {
					panic(err)
				}
				fmt.Fprintf(w, `"/></mask>`)
			}
		}
	}

	m = m.Translate(0.0, float64(img.Bounds().Size().Y))
	fmt.Fprintf(w, `<image transform="%s" width="%d" height="%d" xlink:href="data:%s;base64,`,
		m.ToSVG(w.height), img.Bounds().Size().X, img.Bounds().Size().Y, mimetype)

	encoder := base64.NewEncoder(base64.StdEncoding, w)
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
		fmt.Fprintf(w, `" mask="url(#%s)`, refMask)
	}
	fmt.Fprintf(w, `"/>`)
}

func (l pathLayer) WriteSVG(w *svgWriter) {
	fill := l.fillColor.A != 0
	stroke := l.strokeColor.A != 0 && 0.0 < l.strokeWidth

	p := l.path.Transform(Identity.Translate(0.0, w.height).ReflectY())
	fmt.Fprintf(w, `<path d="%s`, p.ToSVG())

	strokeUnsupported := false
	if arcs, ok := l.strokeJoiner.(arcsJoiner); ok && math.IsNaN(arcs.limit) {
		strokeUnsupported = true
	} else if miter, ok := l.strokeJoiner.(miterJoiner); ok {
		if math.IsNaN(miter.limit) {
			strokeUnsupported = true
		} else if _, ok := miter.gapJoiner.(bevelJoiner); !ok {
			strokeUnsupported = true
		}
	}

	if !stroke {
		if fill {
			if l.fillColor != Black {
				fmt.Fprintf(w, `" fill="%v`, cssColor(l.fillColor))
			}
			if l.fillRule == EvenOdd {
				fmt.Fprintf(w, `" fill-rule="evenodd`)
			}
		} else {
			fmt.Fprintf(w, `" fill="none`)
		}
	} else {
		style := &strings.Builder{}
		if fill {
			if l.fillColor != Black {
				fmt.Fprintf(style, ";fill:%v", cssColor(l.fillColor))
			}
			if l.fillRule == EvenOdd {
				fmt.Fprintf(style, ";fill-rule:evenodd")
			}
		} else {
			fmt.Fprintf(style, ";fill:none")
		}
		if stroke && !strokeUnsupported {
			fmt.Fprintf(style, `;stroke:%v`, cssColor(l.strokeColor))
			if l.strokeWidth != 1.0 {
				fmt.Fprintf(style, ";stroke-width:%v", dec(l.strokeWidth))
			}
			if _, ok := l.strokeCapper.(roundCapper); ok {
				fmt.Fprintf(style, ";stroke-linecap:round")
			} else if _, ok := l.strokeCapper.(squareCapper); ok {
				fmt.Fprintf(style, ";stroke-linecap:square")
			} else if _, ok := l.strokeCapper.(buttCapper); !ok {
				panic("SVG: line cap not support")
			}
			if _, ok := l.strokeJoiner.(bevelJoiner); ok {
				fmt.Fprintf(style, ";stroke-linejoin:bevel")
			} else if _, ok := l.strokeJoiner.(roundJoiner); ok {
				fmt.Fprintf(style, ";stroke-linejoin:round")
			} else if arcs, ok := l.strokeJoiner.(arcsJoiner); ok && !math.IsNaN(arcs.limit) {
				fmt.Fprintf(style, ";stroke-linejoin:arcs")
				if !equal(arcs.limit, 4.0) {
					fmt.Fprintf(style, ";stroke-miterlimit:%v", dec(arcs.limit))
				}
			} else if miter, ok := l.strokeJoiner.(miterJoiner); ok && !math.IsNaN(miter.limit) {
				// a miter line join is the default
				if !equal(miter.limit*2.0/l.strokeWidth, 4.0) {
					fmt.Fprintf(style, ";stroke-miterlimit:%v", dec(miter.limit*2.0/l.strokeWidth))
				}
			} else {
				panic("SVG: line join not support")
			}

			if 0 < len(l.dashes) {
				fmt.Fprintf(style, ";stroke-dasharray:%v", dec(l.dashes[0]))
				for _, dash := range l.dashes[1:] {
					fmt.Fprintf(style, " %v", dec(dash))
				}
				if 0.0 != l.dashOffset {
					fmt.Fprintf(style, ";stroke-dashoffset:%v", dec(l.dashOffset))
				}
			}
		}
		if 0 < style.Len() {
			fmt.Fprintf(w, `" style="%s`, style.String()[1:])
		}
	}
	fmt.Fprintf(w, `"/>`)

	if stroke && strokeUnsupported {
		// stroke settings unsupported by PDF, draw stroke explicitly
		if 0 < len(l.dashes) {
			p = p.Dash(l.dashOffset, l.dashes...)
		}
		p = p.Stroke(l.strokeWidth, l.strokeCapper, l.strokeJoiner)
		fmt.Fprintf(w, `<path d="%s`, p.ToSVG())
		if l.strokeColor != Black {
			fmt.Fprintf(w, `" fill="%v`, cssColor(l.strokeColor))
		}
		if l.fillRule == EvenOdd {
			fmt.Fprintf(w, `" fill-rule="evenodd`)
		}
		fmt.Fprintf(w, `"/>`)
	}
}

func svgWriteFontStyle(w *svgWriter, ff, ffMain FontFace) {
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
		fmt.Fprintf(w, `" style="font:`)

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
		buf.WriteTo(w)

		if ff.color != ffMain.color {
			fmt.Fprintf(w, `;fill:%v`, cssColor(ff.color))
		}
	} else if differences == 1 && ff.color != ffMain.color {
		fmt.Fprintf(w, `" fill="%v`, cssColor(ff.color))
	} else if 0 < differences {
		fmt.Fprintf(w, `" style="`)
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
			fmt.Fprintf(buf, `;fill:%v`, cssColor(ff.color))
		}
		buf.ReadByte()
		buf.WriteTo(w)
	}
}

func (l textLayer) WriteSVG(w *svgWriter) {
	w.EmbedFonts(l.text.Fonts())

	if len(l.text.lines) == 0 || len(l.text.lines[0].spans) == 0 {
		return
	}

	ffMain := l.text.mostCommonFontFace()

	x0, y0 := 0.0, 0.0
	if l.m.IsTranslation() {
		x0, y0 = l.m.Pos()
		y0 = w.height - y0
		fmt.Fprintf(w, `<text x="%v" y="%v`, num(x0), num(y0))
	} else {
		fmt.Fprintf(w, `<text transform="%s`, l.m.ToSVG(w.height))
	}
	fmt.Fprintf(w, `" style="font:`)
	if ffMain.style&FontItalic != 0 {
		fmt.Fprintf(w, ` italic`)
	}
	if boldness := ffMain.boldness(); boldness != 400 {
		fmt.Fprintf(w, ` %d`, boldness)
	}
	if ffMain.variant&FontSmallcaps != 0 {
		fmt.Fprintf(w, ` small-caps`)
	}
	fmt.Fprintf(w, ` %vpx %s`, num(ffMain.size*ffMain.scale), ffMain.font.name)
	if ffMain.color != Black {
		fmt.Fprintf(w, `;fill:%v`, cssColor(ffMain.color))
	}
	fmt.Fprintf(w, `">`)

	decorations := []pathLayer{}
	for _, line := range l.text.lines {
		for _, span := range line.spans {
			fmt.Fprintf(w, `<tspan x="%v" y="%v`, num(x0+span.dx), num(y0-line.y-span.ff.voffset))
			if span.wordSpacing > 0.0 {
				fmt.Fprintf(w, `" word-spacing="%v`, num(span.wordSpacing))
			}
			if span.glyphSpacing > 0.0 {
				fmt.Fprintf(w, `" letter-spacing="%v`, num(span.glyphSpacing))
			}
			svgWriteFontStyle(w, span.ff, ffMain)
			s := span.text
			s = strings.ReplaceAll(s, `"`, `&quot;`)
			fmt.Fprintf(w, `">%s</tspan>`, s)
		}
		for _, deco := range line.decos {
			p := deco.ff.Decorate(deco.x1 - deco.x0)
			p = p.Transform(Identity.Mul(l.m).Translate(deco.x0, line.y+deco.ff.voffset))
			decorations = append(decorations, pathLayer{p, style{fillColor: deco.ff.color}, false})
		}
	}
	fmt.Fprintf(w, `</text>`)
	for _, l := range decorations {
		l.WriteSVG(w)
	}
}

func (l imageLayer) WriteSVG(w *svgWriter) {
	w.DrawImage(l.img, l.enc, l.m)
}
