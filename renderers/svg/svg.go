package svg

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"

	"github.com/tdewolff/canvas"
	canvasText "github.com/tdewolff/canvas/text"
	"github.com/tdewolff/font"
)

type Options struct {
	Compression int
	EmbedFonts  bool
	SubsetFonts bool
	SizeUnits   string
	canvas.ImageEncoding
}

var DefaultOptions = Options{
	EmbedFonts:    true,
	SubsetFonts:   false, // TODO: enable when properly handling GPOS and GSUB tables
	SizeUnits:     "mm",
	ImageEncoding: canvas.Lossless,
}

// SVG is a scalable vector graphics renderer.
type SVG struct {
	w             io.Writer
	width, height float64
	fonts         map[*canvas.Font]bool
	fontSubset    map[*canvas.Font]*canvas.FontSubsetter
	maskID        int
	defs          map[any][2]string
	classes       []string
	customStyle   string
	opts          *Options
}

// New returns a scalable vector graphics (SVG) renderer.
func New(w io.Writer, width, height float64, opts *Options) *SVG {
	if opts == nil {
		defaultOptions := DefaultOptions
		opts = &defaultOptions
	}

	if opts.Compression != 0 {
		if opts.Compression < gzip.HuffmanOnly || gzip.BestCompression < opts.Compression {
			opts.Compression = -1
		}
		w, _ = gzip.NewWriterLevel(w, opts.Compression)
	}

	fmt.Fprintf(w, `<svg version="1.1" width="%v%s" height="%v%s" viewBox="0 0 %v %v" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, dec(width), opts.SizeUnits, dec(height), opts.SizeUnits, dec(width), dec(height))
	return &SVG{
		w:          w,
		width:      width,
		height:     height,
		fonts:      map[*canvas.Font]bool{},
		fontSubset: map[*canvas.Font]*canvas.FontSubsetter{},
		defs:       map[any][2]string{},
		opts:       opts,
	}
}

// Close finished and closes the SVG.
func (r *SVG) Close() error {
	if 0 < len(r.defs) {
		for _, v := range r.defs {
			fmt.Fprintf(r.w, "<defs>%s</defs>", v[1])
		}
	}
	if r.customStyle != "" || r.opts.EmbedFonts && 0 < len(r.fonts) {
		fmt.Fprintf(r.w, "<style>")
		if r.customStyle != "" {
			fmt.Fprintf(r.w, "%s\n", html.EscapeString(r.customStyle))
		}
		if r.opts.EmbedFonts && 0 < len(r.fonts) {
			for f := range r.fonts {
				sfnt := f.SFNT
				if r.opts.SubsetFonts {
					glyphIDs := r.fontSubset[f].List()
					sfntSubset, err := sfnt.Subset(glyphIDs, font.SubsetOptions{Tables: font.KeepMinTables})
					if err == nil {
						//	// TODO: report error?
						sfnt = sfntSubset
					}
				}
				fontProgram := sfnt.Write()

				fmt.Fprintf(r.w, "\n@font-face{font-family:'%s'", f.Name())
				if f.Style().Weight() != canvas.FontRegular {
					fmt.Fprintf(r.w, ";font-weight:%d", f.Style().CSS())
				}
				if f.Style().Italic() {
					fmt.Fprintf(r.w, ";font-style:italic")
				}
				fmt.Fprintf(r.w, ";src:url('data:type/opentype;base64,")
				encoder := base64.NewEncoder(base64.StdEncoding, r.w)
				encoder.Write(fontProgram)
				encoder.Close()
				fmt.Fprintf(r.w, "');}")
			}
		}
		fmt.Fprintf(r.w, "\n</style>")
	}
	_, err := fmt.Fprintf(r.w, "</svg>")
	if r.opts.Compression != 0 {
		r.w.(*gzip.Writer).Close() // does not close underlying writer
	}
	return err
}

func (r *SVG) writeClasses(w io.Writer) {
	if len(r.classes) != 0 {
		fmt.Fprintf(w, `" class="%s`, strings.Join(r.classes, " "))
	}
}

// SetClass sets the classes to be assigned to drawn objects.
func (r *SVG) SetClass(classes ...string) {
	r.classes = classes
}

// AddClass adds a class to the class list.
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

// RemoveClass removes a class from the class list.
func (r *SVG) RemoveClass(class string) {
	for i, c := range r.classes {
		if c == class {
			r.classes = append(r.classes[:i], r.classes[i+1:]...)
			return
		}
	}
}

// SetImageEncoding sets the image encoding to Loss or Lossless.
func (r *SVG) SetImageEncoding(enc canvas.ImageEncoding) {
	r.opts.ImageEncoding = enc
}

// SetCustomStyle defines a custom CSS code to add in the SVG
func (r *SVG) SetCustomStyle(style string) {
	r.customStyle = style
}

// Size returns the size of the canvas in millimeters.
func (r *SVG) Size() (float64, float64) {
	return r.width, r.height
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *SVG) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	fillPaint := r.writePaint(style.Fill, m)
	strokePaint := r.writePaint(style.Stroke, m)

	stroke := path
	path = path.Copy().Transform(canvas.Identity.ReflectYAbout(r.height / 2.0).Mul(m))
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
	if !strokeUnsupported {
		if m.IsSimilarity() {
			scale := math.Sqrt(math.Abs(m.Det()))
			style.StrokeWidth *= scale
			style.DashOffset, style.Dashes = canvas.ScaleDash(style.StrokeWidth, style.DashOffset, style.Dashes)
		} else {
			strokeUnsupported = true
		}
	}

	if !style.HasStroke() {
		if style.HasFill() {
			if !style.Fill.IsColor() || style.Fill.Color != canvas.Black {
				fmt.Fprintf(r.w, `" fill="%v`, fillPaint)
			}
			if style.FillRule == canvas.EvenOdd {
				fmt.Fprintf(r.w, `" fill-rule="evenodd`)
			}
		} else {
			fmt.Fprintf(r.w, `" fill="none`)
		}
	} else {
		b := &strings.Builder{}
		if style.HasFill() {
			if !style.Fill.IsColor() || style.Fill.Color != canvas.Black {
				fmt.Fprintf(b, ";fill:%v", fillPaint)
			}
			if style.FillRule == canvas.EvenOdd {
				fmt.Fprintf(b, ";fill-rule:evenodd")
			}
		} else {
			fmt.Fprintf(b, ";fill:none")
		}
		if style.HasStroke() && !strokeUnsupported {
			fmt.Fprintf(b, `;stroke:%v`, strokePaint)
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
				if !canvas.Equal(miter.Limit, 4.0) {
					fmt.Fprintf(b, ";stroke-miterlimit:%v", dec(miter.Limit))
				}
			} else {
				panic("SVG: line join not support")
			}

			if style.IsDashed() {
				fmt.Fprintf(b, ";stroke-dasharray:%v", dec(style.Dashes[0]))
				for _, dash := range style.Dashes[1:] {
					fmt.Fprintf(b, " %v", dec(dash))
				}
				if style.DashOffset != 0.0 {
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

	if style.HasStroke() && strokeUnsupported {
		// stroke settings unsupported by SVG, draw stroke explicitly
		if style.IsDashed() {
			stroke = stroke.Dash(style.DashOffset, style.Dashes...)
		}
		stroke = stroke.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner, canvas.Tolerance)
		stroke = stroke.Transform(canvas.Identity.ReflectYAbout(r.height / 2.0).Mul(m))
		fmt.Fprintf(r.w, `<path d="%s`, stroke.ToSVG())
		if !style.Stroke.IsColor() || style.Stroke.Color != canvas.Black {
			fmt.Fprintf(r.w, `" fill="%v`, strokePaint)
		}
		if style.FillRule == canvas.EvenOdd {
			fmt.Fprintf(r.w, `" fill-rule="evenodd`)
		}
		r.writeClasses(r.w)
		fmt.Fprintf(r.w, `"/>`)
	}
}

func (r *SVG) writeFontStyle(face, faceMain *canvas.FontFace, rtl bool, fill string) {
	differences := 0
	boldness := face.Style.CSS()
	if face.Style&canvas.FontItalic != faceMain.Style&canvas.FontItalic {
		differences++
	}
	if boldness != faceMain.Style.CSS() {
		differences++
	}
	if (face.Variant == canvas.FontSmallcaps) != (faceMain.Variant == canvas.FontSmallcaps) {
		differences++
	}
	if !face.Fill.Equal(faceMain.Fill) {
		differences++
	}
	if rtl {
		differences++
	}
	if face.Name() != faceMain.Name() || face.Size != faceMain.Size || differences == 3 {
		fmt.Fprintf(r.w, `" style="font:`)

		buf := &bytes.Buffer{}
		if face.Style&canvas.FontItalic != faceMain.Style&canvas.FontItalic {
			fmt.Fprintf(buf, ` italic`)
		}

		if boldness != faceMain.Style.CSS() {
			fmt.Fprintf(buf, ` %d`, boldness)
		}

		if face.Variant == canvas.FontSmallcaps && faceMain.Variant != canvas.FontSmallcaps {
			fmt.Fprintf(buf, ` small-caps`)
		} else if face.Variant != canvas.FontSmallcaps && faceMain.Variant == canvas.FontSmallcaps {
			fmt.Fprintf(buf, ` normal`)
		}

		fmt.Fprintf(buf, ` %vpx %s`, num(face.Size), face.Name())
		buf.ReadByte()
		buf.WriteTo(r.w)

		if !face.Fill.Equal(faceMain.Fill) {
			fmt.Fprintf(r.w, `;fill:%v`, fill)
		}
		if rtl {
			fmt.Fprintf(r.w, `;direction:rtl`)
		}
	} else if differences == 1 && !face.Fill.Equal(faceMain.Fill) {
		fmt.Fprintf(r.w, `" fill="%v`, fill)
	} else if 0 < differences {
		fmt.Fprintf(r.w, `" style="`)
		buf := &bytes.Buffer{}
		if face.Style&canvas.FontItalic != faceMain.Style&canvas.FontItalic {
			fmt.Fprintf(buf, `;font-style:italic`)
		}
		if boldness != faceMain.Style.CSS() {
			fmt.Fprintf(buf, `;font-weight:%d`, boldness)
		}
		if face.Variant == canvas.FontSmallcaps && faceMain.Variant != canvas.FontSmallcaps {
			fmt.Fprintf(buf, `;font-variant:small-caps`)
		} else if face.Variant != canvas.FontSmallcaps && faceMain.Variant == canvas.FontSmallcaps {
			fmt.Fprintf(buf, `;font-variant:normal`)
		}
		if !face.Fill.Equal(faceMain.Fill) {
			fmt.Fprintf(buf, `;fill:%v`, fill)
		}
		if rtl {
			fmt.Fprintf(r.w, `;direction:rtl`)
		}
		buf.ReadByte()
		buf.WriteTo(r.w)
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *SVG) RenderText(text *canvas.Text, m canvas.Matrix) {
	if text.Empty() {
		return
	}

	text.WalkDecorations(func(paint canvas.Paint, p *canvas.Path) {
		style := canvas.DefaultStyle
		style.Fill = paint
		r.RenderPath(p, style, m)
	})

	text.WalkSpans(func(x, y float64, span canvas.TextSpan) {
		if !span.IsText() {
			for _, obj := range span.Objects {
				obj.Canvas.RenderViewTo(r, m.Mul(obj.View(x, y, span.Face)))
			}
		}
	})

	faceMain := text.MostCommonFontFace()
	x0, y0 := 0.0, 0.0
	if m.IsTranslation() {
		x0, y0 = m.Pos()
		y0 = r.height - y0
		fmt.Fprintf(r.w, `<text x="%v" y="%v`, num(x0), num(y0))
	} else {
		fmt.Fprintf(r.w, `<text transform="%s`, m.ToSVG(r.height))
	}
	fmt.Fprintf(r.w, `" style="font:`)
	if faceMain.Style&canvas.FontItalic != 0 {
		fmt.Fprintf(r.w, ` italic`)
	}
	if boldness := faceMain.Style.CSS(); boldness != 400 {
		fmt.Fprintf(r.w, ` %d`, boldness)
	}
	if faceMain.Variant == canvas.FontSmallcaps {
		fmt.Fprintf(r.w, ` small-caps`)
	}
	fmt.Fprintf(r.w, ` %vpx %s`, num(faceMain.Size), faceMain.Name())
	if !faceMain.Fill.IsColor() || faceMain.Fill.Color != canvas.Black {
		fmt.Fprintf(r.w, `;fill:%v`, r.writePaint(faceMain.Fill, m))
	}
	if text.WritingMode != canvas.HorizontalTB {
		if text.WritingMode == canvas.VerticalLR {
			fmt.Fprintf(r.w, `;writing-mode:vertical-lr`)
		} else if text.WritingMode == canvas.VerticalRL {
			fmt.Fprintf(r.w, `;writing-mode:vertical-rl`)
		}
		if text.TextOrientation == canvas.Upright {
			fmt.Fprintf(r.w, `;text-orientation:upright`)
		}
	}
	r.writeClasses(r.w)
	fmt.Fprintf(r.w, `">`)

	text.WalkSpans(func(x, y float64, span canvas.TextSpan) {
		if span.IsText() {
			if ok, _ := r.fonts[span.Face.Font]; !ok {
				r.fonts[span.Face.Font] = true
				r.fontSubset[span.Face.Font] = canvas.NewFontSubsetter()
			}

			subset := r.fontSubset[span.Face.Font]
			for _, r := range span.Text {
				glyphID := span.Face.Font.SFNT.GlyphIndex(r)
				_ = subset.Get(glyphID) // register usage of glyph for subsetting
			}

			x += x0
			y = y0 - y
			if span.Direction == canvasText.RightToLeft {
				x += span.Width
			}
			fmt.Fprintf(r.w, `<tspan x="%v" y="%v`, num(x), num(y))
			r.writeFontStyle(span.Face, faceMain, span.Direction == canvasText.RightToLeft, r.writePaint(span.Face.Fill, m))
			r.writeClasses(r.w)
			fmt.Fprintf(r.w, `">`)
			xml.EscapeText(r.w, []byte(span.Text))
			fmt.Fprintf(r.w, `</tspan>`)
		}
	})
	fmt.Fprintf(r.w, `</text>`)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *SVG) RenderImage(img image.Image, m canvas.Matrix) {
	size := img.Bounds().Size()
	writeTo, refMask, mimetype := r.encodableImage(img)

	m = m.Translate(0.0, float64(size.Y))
	fmt.Fprintf(r.w, `<image transform="%s" width="%d" height="%d" xlink:href="data:%s;base64,`,
		m.ToSVG(r.height), size.X, size.Y, mimetype)

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
func (r *SVG) encodableImage(img image.Image) (func(io.Writer) error, string, string) {
	if cimg, ok := img.(canvas.Image); ok && 0 < len(cimg.Bytes) {
		if cimg.Mimetype == "image/jpeg" || cimg.Mimetype == "image/png" {
			return func(w io.Writer) error {
				_, err := w.Write(cimg.Bytes)
				return err
			}, "", cimg.Mimetype
		}
	}

	// lossy: jpeg
	if r.opts.ImageEncoding == canvas.Lossy {
		var refMask string
		if opaqueImg, ok := img.(interface{ Opaque() bool }); !ok || !opaqueImg.Opaque() {
			img, refMask = r.renderOpacityMask(img)
		}
		return func(w io.Writer) error {
			return jpeg.Encode(w, img, nil)
		}, refMask, "image/jpeg"
	}

	// lossless: png
	return func(w io.Writer) error {
		return png.Encode(w, img)
	}, "", "image/png"
}

func (r *SVG) renderOpacityMask(img image.Image) (image.Image, string) {
	opaque, mask := splitImageAlphaChannel(img)
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

func splitImageAlphaChannel(img image.Image) (image.Image, image.Image) {
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

func (r *SVG) writePaint(paint canvas.Paint, m canvas.Matrix) string {
	if paint.IsPattern() {
		// TODO
		return ""
	} else if paint.IsGradient() {
		var def string
		ref := fmt.Sprintf("d%v", len(r.defs))
		if v, ok := r.defs[[2]any{paint.Gradient, m}]; ok {
			return fmt.Sprintf("url(#%v)", v[0])
		} else if linearGradient, ok := paint.Gradient.(*canvas.LinearGradient); ok {
			sb := strings.Builder{}
			if m.IsSimilarity() {
				start := m.Dot(linearGradient.Start)
				end := m.Dot(linearGradient.End)
				fmt.Fprintf(&sb, `<linearGradient id="%v" gradientUnits="userSpaceOnUse" x1="%v" y1="%v" x2="%v" y2="%v">`, ref, dec(start.X), dec(r.height-start.Y), dec(end.X), dec(r.height-end.Y))
			} else {
				// negate the Y coordinates because ToSVG(r.height) applies Y-axis reflection in its translation component,
				// so negating the gradient coordinates cancels out the double Y-axis reflection to achieve correct positioning.
				fmt.Fprintf(&sb, `<linearGradient id="%v" gradientUnits="userSpaceOnUse" gradientTransform="%v" x1="%v" y1="%v" x2="%v" y2="%v">`, ref, m.ToSVG(r.height), dec(linearGradient.Start.X), -dec(linearGradient.Start.Y), dec(linearGradient.End.X), -dec(linearGradient.End.Y))
			}
			for _, stop := range linearGradient.Grad {
				fmt.Fprintf(&sb, `<stop offset="%v" stop-color="%v"/>`, dec(stop.Offset), canvas.CSSColor(stop.Color))
			}
			fmt.Fprintf(&sb, `</linearGradient>`)
			def = sb.String()
		} else if radialGradient, ok := paint.Gradient.(*canvas.RadialGradient); ok {
			sb := strings.Builder{}
			if m.IsSimilarity() {
				c0 := m.Dot(radialGradient.C0)
				c1 := m.Dot(radialGradient.C1)
				scale := math.Sqrt(math.Abs(m.Det()))
				r0 := scale * radialGradient.R0
				r1 := scale * radialGradient.R1
				fmt.Fprintf(&sb, `<radialGradient id="%v" gradientUnits="userSpaceOnUse" fx="%v" fy="%v" fr="%v" cx="%v" cy="%v" r="%v">`, ref, dec(c0.X), dec(r.height-c0.Y), dec(r0), dec(c1.X), dec(r.height-c1.Y), dec(r1))
			} else {
				// negate the Y coordinates because ToSVG(r.height) applies Y-axis reflection in its translation component,
				// so negating the gradient coordinates cancels out the double Y-axis reflection to achieve correct positioning.
				fmt.Fprintf(&sb, `<radialGradient id="%v" gradientUnits="userSpaceOnUse" gradientTransform="%v" fx="%v" fy="%v" fr="%v" cx="%v" cy="%v" r="%v">`, ref, m.ToSVG(r.height), dec(radialGradient.C0.X), -dec(radialGradient.C0.Y), dec(radialGradient.R0), dec(radialGradient.C1.X), -dec(radialGradient.C1.Y), dec(radialGradient.R1))
			}
			for _, stop := range radialGradient.Grad {
				fmt.Fprintf(&sb, `<stop offset="%v" stop-color="%v"/>`, dec(stop.Offset), canvas.CSSColor(stop.Color))
			}
			fmt.Fprintf(&sb, `</radialGradient>`)
			def = sb.String()
		}
		r.defs[[2]any{paint.Gradient, m}] = [2]string{ref, def}
		return fmt.Sprintf("url(#%v)", ref)
	} else {
		return fmt.Sprintf("%v", canvas.CSSColor(paint.Color))
	}
}
