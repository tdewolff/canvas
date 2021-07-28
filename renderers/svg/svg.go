package svg

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"strings"

	"github.com/tdewolff/canvas"
	canvasText "github.com/tdewolff/canvas/text"
)

// Writer writes the canvas as an SVG file.
// DEPRECATED
func Writer(w io.Writer, c *canvas.Canvas) error {
	log.Println("WARNING: github.com/tdewolff/canvas/renderers/svg.Writer is deprecated, please use github.com/tdewolff/canvas/renderers.SVG")
	svg := New(w, c.W, c.H, nil)
	c.Render(svg)
	return svg.Close()
}

type Options struct {
	Compression int
	EmbedFonts  bool
	SubsetFonts bool
	canvas.ImageEncoding
}

var DefaultOptions = Options{
	EmbedFonts:    true,
	SubsetFonts:   false, // TODO: enable when properly handling GPOS and GSUB tables
	ImageEncoding: canvas.Lossless,
}

// SVG is a scalable vector graphics renderer.
type SVG struct {
	w             io.Writer
	width, height float64
	fonts         map[*canvas.Font]bool
	maskID        int
	classes       []string
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

	fmt.Fprintf(w, `<svg version="1.1" width="%vmm" height="%vmm" viewBox="0 0 %v %v" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`, dec(width), dec(height), dec(width), dec(height))
	return &SVG{
		w:       w,
		width:   width,
		height:  height,
		fonts:   map[*canvas.Font]bool{},
		maskID:  0,
		classes: []string{},
		opts:    opts,
	}
}

// Close finished and closes the SVG.
func (r *SVG) Close() error {
	if r.opts.EmbedFonts {
		r.writeFonts()
	}
	_, err := fmt.Fprintf(r.w, "</svg>")
	if r.opts.Compression != 0 {
		r.w.(*gzip.Writer).Close() // does not close underlying writer
	}
	return err
}

func (r *SVG) writeFonts() {
	if 0 < len(r.fonts) {
		fmt.Fprintf(r.w, "<style>")
		for font := range r.fonts {
			b := font.SFNT.Data
			if r.opts.SubsetFonts {
				b, _ = font.SFNT.Subset(font.SubsetIDs())
			}
			fmt.Fprintf(r.w, "\n@font-face{font-family:'%s'", font.Name())
			if font.Style().Weight() != canvas.FontRegular {
				fmt.Fprintf(r.w, ";font-weight:%d", font.Style().CSS())
			}
			if font.Style().Italic() {
				fmt.Fprintf(r.w, ";font-style:italic")
			}
			fmt.Fprintf(r.w, ";src:url('data:type/opentype;base64,")
			encoder := base64.NewEncoder(base64.StdEncoding, r.w)
			encoder.Write(b)
			encoder.Close()
			fmt.Fprintf(r.w, "');}")
		}
		fmt.Fprintf(r.w, "\n</style>")
	}
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

// Size returns the size of the canvas in millimeters.
func (r *SVG) Size() (float64, float64) {
	return r.width, r.height
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *SVG) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	stroke := path
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
	if !strokeUnsupported {
		if m.IsSimilarity() {
			scale := math.Sqrt(m.Det())
			style.StrokeWidth *= scale
			style.DashOffset *= scale
			dashes := make([]float64, len(style.Dashes))
			for i := range style.Dashes {
				dashes[i] = style.Dashes[i] * scale
			}
			style.Dashes = dashes
		} else {
			strokeUnsupported = true
		}
	}

	if !style.HasStroke() {
		if style.HasFill() {
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
		if style.HasFill() {
			if style.FillColor != canvas.Black {
				fmt.Fprintf(b, ";fill:%v", canvas.CSSColor(style.FillColor))
			}
			if style.FillRule == canvas.EvenOdd {
				fmt.Fprintf(b, ";fill-rule:evenodd")
			}
		} else {
			fmt.Fprintf(b, ";fill:none")
		}
		if style.HasStroke() && !strokeUnsupported {
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
		stroke = stroke.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)
		stroke = stroke.Transform(canvas.Identity.ReflectYAbout(r.height / 2.0).Mul(m))
		fmt.Fprintf(r.w, `<path d="%s`, stroke.ToSVG())
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

func (r *SVG) writeFontStyle(face, faceMain *canvas.FontFace) {
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
	if face.Color != faceMain.Color {
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

		if face.Color != faceMain.Color {
			fmt.Fprintf(r.w, `;fill:%v`, canvas.CSSColor(face.Color))
		}
	} else if differences == 1 && face.Color != faceMain.Color {
		fmt.Fprintf(r.w, `" fill="%v`, canvas.CSSColor(face.Color))
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
		if face.Color != faceMain.Color {
			fmt.Fprintf(buf, `;fill:%v`, canvas.CSSColor(face.Color))
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

	text.WalkDecorations(func(col color.RGBA, p *canvas.Path) {
		style := canvas.DefaultStyle
		style.FillColor = col
		r.RenderPath(p, style, m)
	})

	faceMain := text.Face
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
	if faceMain.Color != canvas.Black {
		fmt.Fprintf(r.w, `;fill:%v`, canvas.CSSColor(faceMain.Color))
	}
	if faceMain.Direction == canvasText.TopToBottom || faceMain.Direction == canvasText.BottomToTop {
		fmt.Fprintf(r.w, `;writing-mode:vertical-lr`)
	}
	r.writeClasses(r.w)
	fmt.Fprintf(r.w, `">`)

	text.WalkSpans(func(x, y float64, span canvas.TextSpan) {
		r.fonts[span.Face.Font] = true
		for _, r := range span.Text {
			glyphID := span.Face.Font.SFNT.GlyphIndex(r)
			_ = span.Face.Font.SubsetID(glyphID) // register usage of glyph for subsetting
		}

		x += x0
		y = y0 - y
		fmt.Fprintf(r.w, `<tspan x="%v" y="%v`, num(x), num(y))
		r.writeFontStyle(span.Face, faceMain)
		r.writeClasses(r.w)
		fmt.Fprintf(r.w, `">`)
		xml.EscapeText(r.w, []byte(span.Text))
		fmt.Fprintf(r.w, `</tspan>`)
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
