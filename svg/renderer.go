package svg

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	"github.com/tdewolff/canvas"
)

type SVG struct {
	w             io.Writer
	width, height float64
	embedFonts    bool
	fonts         map[*canvas.Font]bool
	maskID        int
	imgEnc        canvas.ImageEncoding

	classes []string
}

// New creates a scalable vector graphics (SVG) renderer.
func New(w io.Writer, width, height float64) *SVG {
	fmt.Fprintf(w, `<svg version="1.1" width="%vmm" height="%vmm" viewBox="0 0 %v %v" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, dec(width), dec(height), dec(width), dec(height))
	return &SVG{
		w:          w,
		width:      width,
		height:     height,
		embedFonts: true,
		fonts:      map[*canvas.Font]bool{},
		maskID:     0,
		imgEnc:     canvas.Lossless,
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

func (r *SVG) SetImageEncoding(enc canvas.ImageEncoding) {
	r.imgEnc = enc
}

func (r *SVG) writeFonts(fonts []*canvas.Font) {
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
			mediatype, raw := fonts[i].Raw()
			fmt.Fprintf(r.w, "\n@font-face{font-family:'%s';src:url('data:%s;base64,", fonts[i].Name(), mediatype)
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

func (r *SVG) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	fill := style.FillColor.A != 0
	stroke := style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth

	path = path.Transform(canvas.Identity.ReflectYAbout(r.height / 2.0).Mul(m))
	fmt.Fprintf(r.w, `<path d="%s`, path.ToSVG())

	strokeUnsupported := false
	if arcs, ok := style.StrokeJoiner.(canvas.ArcsJoiner); ok && math.IsNaN(arcs.Limit) {
		strokeUnsupported = true
	} else if miter, ok := style.StrokeJoiner.(canvas.MiterJoiner); ok {
		if math.IsNaN(miter.Limit) {
			strokeUnsupported = true
		} else if _, ok := miter.GapJoiner.(canvas.BevelJoiner); !ok {
			strokeUnsupported = true
		}
	}

	if !stroke {
		if fill {
			if style.FillColor != canvas.Black {
				fmt.Fprintf(r.w, `" fill="%v`, canvas.CSSColor(style.FillColor))
			}
			if style.FillRule == canvas.EvenOdd {
				fmt.Fprintf(r.w, `" fill-rule="evenodd`)
			}
		} else {
			fmt.Fprintf(r.w, `" fill="none`)
		}
	} else {
		b := &strings.Builder{}
		if fill {
			if style.FillColor != canvas.Black {
				fmt.Fprintf(b, ";fill:%v", canvas.CSSColor(style.FillColor))
			}
			if style.FillRule == canvas.EvenOdd {
				fmt.Fprintf(b, ";fill-rule:evenodd")
			}
		} else {
			fmt.Fprintf(b, ";fill:none")
		}
		if stroke && !strokeUnsupported {
			fmt.Fprintf(b, `;stroke:%v`, canvas.CSSColor(style.StrokeColor))
			if style.StrokeWidth != 1.0 {
				fmt.Fprintf(b, ";stroke-width:%v", dec(style.StrokeWidth))
			}
			if _, ok := style.StrokeCapper.(canvas.RoundCapper); ok {
				fmt.Fprintf(b, ";stroke-linecap:round")
			} else if _, ok := style.StrokeCapper.(canvas.SquareCapper); ok {
				fmt.Fprintf(b, ";stroke-linecap:square")
			} else if _, ok := style.StrokeCapper.(canvas.ButtCapper); !ok {
				panic("SVG: line cap not support")
			}
			if _, ok := style.StrokeJoiner.(canvas.BevelJoiner); ok {
				fmt.Fprintf(b, ";stroke-linejoin:bevel")
			} else if _, ok := style.StrokeJoiner.(canvas.RoundJoiner); ok {
				fmt.Fprintf(b, ";stroke-linejoin:round")
			} else if arcs, ok := style.StrokeJoiner.(canvas.ArcsJoiner); ok && !math.IsNaN(arcs.Limit) {
				fmt.Fprintf(b, ";stroke-linejoin:arcs")
				if !canvas.Equal(arcs.Limit, 4.0) {
					fmt.Fprintf(b, ";stroke-miterlimit:%v", dec(arcs.Limit))
				}
			} else if miter, ok := style.StrokeJoiner.(canvas.MiterJoiner); ok && !math.IsNaN(miter.Limit) {
				// a miter line join is the default
				if !canvas.Equal(miter.Limit*2.0/style.StrokeWidth, 4.0) {
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
		if style.StrokeColor != canvas.Black {
			fmt.Fprintf(r.w, `" fill="%v`, canvas.CSSColor(style.StrokeColor))
		}
		if style.FillRule == canvas.EvenOdd {
			fmt.Fprintf(r.w, `" fill-rule="evenodd`)
		}
		r.writeClasses(r.w)
		fmt.Fprintf(r.w, `"/>`)
	}
}

func (r *SVG) writeFontStyle(ff, ffMain canvas.FontFace) {
	boldness := ff.Boldness()
	differences := 0

	if ff.Style&canvas.FontItalic != ffMain.Style&canvas.FontItalic {
		differences++
	}
	if boldness != ffMain.Boldness() {
		differences++
	}
	if ff.Variant&canvas.FontSmallcaps != ffMain.Variant&canvas.FontSmallcaps {
		differences++
	}
	if ff.Color != ffMain.Color {
		differences++
	}
	if ff.Name() != ffMain.Name() || ff.Size*ff.Scale != ffMain.Size || differences == 3 {
		fmt.Fprintf(r.w, `" style="font:`)

		buf := &bytes.Buffer{}
		if ff.Style&canvas.FontItalic != ffMain.Style&canvas.FontItalic {
			fmt.Fprintf(buf, ` italic`)
		}

		if boldness != ffMain.Boldness() {
			fmt.Fprintf(buf, ` %d`, boldness)
		}

		if ff.Variant&canvas.FontSmallcaps != ffMain.Variant&canvas.FontSmallcaps {
			fmt.Fprintf(buf, ` small-caps`)
		}

		fmt.Fprintf(buf, ` %vpx %s`, num(ff.Size*ff.Scale), ff.Name())
		buf.ReadByte()
		buf.WriteTo(r.w)

		if ff.Color != ffMain.Color {
			fmt.Fprintf(r.w, `;fill:%v`, canvas.CSSColor(ff.Color))
		}
	} else if differences == 1 && ff.Color != ffMain.Color {
		fmt.Fprintf(r.w, `" fill="%v`, canvas.CSSColor(ff.Color))
	} else if 0 < differences {
		fmt.Fprintf(r.w, `" style="`)
		buf := &bytes.Buffer{}
		if ff.Style&canvas.FontItalic != ffMain.Style&canvas.FontItalic {
			fmt.Fprintf(buf, `;font-style:italic`)
		}
		if boldness != ffMain.Boldness() {
			fmt.Fprintf(buf, `;font-weight:%d`, boldness)
		}
		if ff.Variant&canvas.FontSmallcaps != ffMain.Variant&canvas.FontSmallcaps {
			fmt.Fprintf(buf, `;font-variant:small-caps`)
		}
		if ff.Color != ffMain.Color {
			fmt.Fprintf(buf, `;fill:%v`, canvas.CSSColor(ff.Color))
		}
		buf.ReadByte()
		buf.WriteTo(r.w)
	}
}

func (r *SVG) RenderText(text *canvas.Text, m canvas.Matrix) {
	if r.embedFonts {
		r.writeFonts(text.Fonts())
	}

	if text.Empty() {
		return
	}

	ffMain := text.MostCommonFontFace()

	x0, y0 := 0.0, 0.0
	if m.IsTranslation() {
		x0, y0 = m.Pos()
		y0 = r.height - y0
		fmt.Fprintf(r.w, `<text x="%v" y="%v`, num(x0), num(y0))
	} else {
		fmt.Fprintf(r.w, `<text transform="%s`, m.ToSVG(r.height))
	}
	fmt.Fprintf(r.w, `" style="font:`)
	if ffMain.Style&canvas.FontItalic != 0 {
		fmt.Fprintf(r.w, ` italic`)
	}
	if boldness := ffMain.Boldness(); boldness != 400 {
		fmt.Fprintf(r.w, ` %d`, boldness)
	}
	if ffMain.Variant&canvas.FontSmallcaps != 0 {
		fmt.Fprintf(r.w, ` small-caps`)
	}
	fmt.Fprintf(r.w, ` %vpx %s`, num(ffMain.Size*ffMain.Scale), ffMain.Name())
	if ffMain.Color != canvas.Black {
		fmt.Fprintf(r.w, `;fill:%v`, canvas.CSSColor(ffMain.Color))
	}
	r.writeClasses(r.w)
	fmt.Fprintf(r.w, `">`)

	text.WalkSpans(func(y, dx float64, span canvas.TextSpan) {
		fmt.Fprintf(r.w, `<tspan x="%v" y="%v`, num(x0+dx), num(y0-y-span.Face.Voffset))
		if span.WordSpacing > 0.0 {
			fmt.Fprintf(r.w, `" word-spacing="%v`, num(span.WordSpacing))
		}
		if span.GlyphSpacing > 0.0 {
			fmt.Fprintf(r.w, `" letter-spacing="%v`, num(span.GlyphSpacing))
		}
		r.writeFontStyle(span.Face, ffMain)
		r.writeClasses(r.w)
		fmt.Fprintf(r.w, `">`)
		xml.EscapeText(r.w, []byte(span.Text))
		fmt.Fprintf(r.w, `</tspan>`)
	})
	fmt.Fprintf(r.w, `</text>`)
	text.RenderDecoration(r, m)
}

func (r *SVG) RenderImage(img image.Image, m canvas.Matrix) {
	size := img.Bounds().Size()
	writeTo, refMask, mimetype := r.prepareImage(img)

	m = m.Translate(0.0, float64(size.Y))
	fmt.Fprintf(r.w, `<image transform="%s" width="%d" height="%d" xlink:href="data:%s;base64,`,
		m.ToSVG(r.height), size.X, size.Y, mimetype.String())

	encoder := base64.NewEncoder(base64.StdEncoding, r.w)
	err := writeTo(encoder)
	if err != nil {
		panic(err)
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

// return a WriterTo, a refMask and a mimetype
func (r *SVG) prepareImage(img image.Image) (func(io.Writer) error, string, canvas.ImageMimetype) {
	if cimg, ok := img.(canvas.Image); ok && len(cimg.Bytes) > 0 {
		if cimg.Mimetype == canvas.ImageJPEG || cimg.Mimetype == canvas.ImagePNG {
			return func(w io.Writer) error {
				_, err := w.Write(cimg.Bytes)
				return err
			}, "", cimg.Mimetype
		}
	}

	// lossy: jpeg
	if r.imgEnc == canvas.Lossy {
		var refMask string
		if opaqueImg, ok := img.(interface{ Opaque() bool }); !ok || !opaqueImg.Opaque() {
			img, refMask = r.renderOpacityMask(img)
		}
		return func(w io.Writer) error {
			return jpeg.Encode(w, img, nil)
		}, refMask, canvas.ImageJPEG
	}

	// lossless: png
	return func(w io.Writer) error {
		return png.Encode(w, img)
	}, "", canvas.ImagePNG
}

func (r *SVG) renderOpacityMask(img image.Image) (image.Image, string) {
	opaque, mask := getOpacityMask(img)
	if mask == nil {
		return opaque, ""
	}

	refMask := fmt.Sprintf("m%v", r.maskID)
	r.maskID++

	size := img.Bounds().Size()
	fmt.Fprintf(r.w, `<mask id="%s"><image width="%d" height="%d" xlink:href="data:image/jpeg;base64,`, refMask, size.X, size.Y)

	encoder := base64.NewEncoder(base64.StdEncoding, r.w)
	if err := jpeg.Encode(encoder, mask, nil); err != nil {
		panic(err)
	}
	if err := encoder.Close(); err != nil {
		panic(err)
	}
	fmt.Fprintf(r.w, `"/></mask>`)
	return opaque, refMask
}

func getOpacityMask(img image.Image) (image.Image, image.Image) {
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
	if !hasMask {
		return img, nil
	}

	return opaque, mask
}
