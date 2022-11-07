package pdf

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	"github.com/tdewolff/canvas"
)

type Options struct {
	Compress    bool
	SubsetFonts bool
	canvas.ImageEncoding
}

var DefaultOptions = Options{
	Compress:      true,
	SubsetFonts:   true,
	ImageEncoding: canvas.Lossless,
}

// PDF is a portable document format renderer.
type PDF struct {
	w             *pdfPageWriter
	width, height float64
	opts          *Options
}

// New returns a portable document format (PDF) renderer.
func New(w io.Writer, width, height float64, opts *Options) *PDF {
	if opts == nil {
		defaultOptions := DefaultOptions
		opts = &defaultOptions
	}

	page := newPDFWriter(w).NewPage(width, height)
	page.pdf.SetCompression(opts.Compress)
	page.pdf.SetFontSubsetting(opts.SubsetFonts)
	return &PDF{
		w:      page,
		width:  width,
		height: height,
		opts:   opts,
	}
}

// SetImageEncoding sets the image encoding to Loss or Lossless.
func (r *PDF) SetImageEncoding(enc canvas.ImageEncoding) {
	r.opts.ImageEncoding = enc
}

// SetInfo sets the document's title, subject, keywords, author and creator.
func (r *PDF) SetInfo(title, subject, keywords, author, creator string) {
	r.w.pdf.SetTitle(title)
	r.w.pdf.SetSubject(subject)
	r.w.pdf.SetKeywords(keywords)
	r.w.pdf.SetAuthor(author)
	r.w.pdf.SetCreator(creator)
}

// NewPage starts adds a new page where further rendering will be written to.
func (r *PDF) NewPage(width, height float64) {
	r.w = r.w.pdf.NewPage(width, height)
}

// Close finished and closes the PDF.
func (r *PDF) Close() error {
	return r.w.pdf.Close()
}

// Size returns the size of the canvas in millimeters.
func (r *PDF) Size() (float64, float64) {
	return r.width, r.height
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *PDF) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	differentAlpha := style.HasFill() && style.HasStroke() && style.FillColor.A != style.StrokeColor.A

	// PDFs don't support the arcs joiner, miter joiner (not clipped), or miter joiner (clipped) with non-bevel fallback
	strokeUnsupported := false
	if _, ok := style.StrokeJoiner.(canvas.ArcsJoiner); ok {
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

	// PDFs don't support connecting first and last dashes if path is closed, so we move the start of the path if this is the case
	// TODO: closing dashes
	//if style.DashesClose {
	//	strokeUnsupported = true
	//}

	closed := false
	data := path.Transform(m).ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	if !style.HasStroke() || !strokeUnsupported {
		if style.HasFill() && !style.HasStroke() {
			r.w.SetFillColor(style.FillColor)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			r.w.Write([]byte(" f"))
			if style.FillRule == canvas.EvenOdd {
				r.w.Write([]byte("*"))
			}
		} else if !style.HasFill() && style.HasStroke() {
			r.w.SetStrokeColor(style.StrokeColor)
			r.w.SetLineWidth(style.StrokeWidth)
			r.w.SetLineCap(style.StrokeCapper)
			r.w.SetLineJoin(style.StrokeJoiner)
			r.w.SetDashes(style.DashOffset, style.Dashes)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			if closed {
				r.w.Write([]byte(" s"))
			} else {
				r.w.Write([]byte(" S"))
			}
			if style.FillRule == canvas.EvenOdd {
				r.w.Write([]byte("*"))
			}
		} else if style.HasFill() && style.HasStroke() {
			if !differentAlpha {
				r.w.SetFillColor(style.FillColor)
				r.w.SetStrokeColor(style.StrokeColor)
				r.w.SetLineWidth(style.StrokeWidth)
				r.w.SetLineCap(style.StrokeCapper)
				r.w.SetLineJoin(style.StrokeJoiner)
				r.w.SetDashes(style.DashOffset, style.Dashes)
				r.w.Write([]byte(" "))
				r.w.Write([]byte(data))
				if closed {
					r.w.Write([]byte(" b"))
				} else {
					r.w.Write([]byte(" B"))
				}
				if style.FillRule == canvas.EvenOdd {
					r.w.Write([]byte("*"))
				}
			} else {
				r.w.SetFillColor(style.FillColor)
				r.w.Write([]byte(" "))
				r.w.Write([]byte(data))
				r.w.Write([]byte(" f"))
				if style.FillRule == canvas.EvenOdd {
					r.w.Write([]byte("*"))
				}

				r.w.SetStrokeColor(style.StrokeColor)
				r.w.SetLineWidth(style.StrokeWidth)
				r.w.SetLineCap(style.StrokeCapper)
				r.w.SetLineJoin(style.StrokeJoiner)
				r.w.SetDashes(style.DashOffset, style.Dashes)
				r.w.Write([]byte(" "))
				r.w.Write([]byte(data))
				if closed {
					r.w.Write([]byte(" s"))
				} else {
					r.w.Write([]byte(" S"))
				}
				if style.FillRule == canvas.EvenOdd {
					r.w.Write([]byte("*"))
				}
			}
		}
	} else {
		// style.HasStroke() && strokeUnsupported
		if style.HasFill() {
			r.w.SetFillColor(style.FillColor)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			r.w.Write([]byte(" f"))
			if style.FillRule == canvas.EvenOdd {
				r.w.Write([]byte("*"))
			}
		}

		// stroke settings unsupported by PDF, draw stroke explicitly
		if style.IsDashed() {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)

		r.w.SetFillColor(style.StrokeColor)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(path.Transform(m).ToPDF()))
		r.w.Write([]byte(" f"))
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *PDF) RenderText(text *canvas.Text, m canvas.Matrix) {
	text.WalkDecorations(func(col color.RGBA, p *canvas.Path) {
		style := canvas.DefaultStyle
		style.FillColor = col
		r.RenderPath(p, style, m)
	})

	text.WalkSpans(func(x, y float64, span canvas.TextSpan) {
		if span.IsText() {
			style := canvas.DefaultStyle
			style.FillColor = span.Face.Color

			r.w.StartTextObject()
			r.w.SetFillColor(span.Face.Color)
			r.w.SetFont(span.Face.Font, span.Face.Size, span.Direction)
			r.w.SetTextPosition(m.Translate(x, y).Shear(span.Face.FauxItalic, 0.0))

			if 0.0 < span.Face.FauxBold {
				r.w.SetTextRenderMode(2)
				r.w.SetStrokeColor(span.Face.Color)
				fmt.Fprintf(r.w, " %v w", dec(span.Face.FauxBold*2.0))
			} else {
				r.w.SetTextRenderMode(0)
			}
			r.w.WriteText(text.WritingMode, span.Glyphs)
			r.w.EndTextObject()
		} else {
			for _, obj := range span.Objects {
				rv := canvas.RendererViewer{r, m.Mul(obj.View(x, y, span.Face))}
				obj.Canvas.RenderTo(rv)
			}
		}
	})
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *PDF) RenderImage(img image.Image, m canvas.Matrix) {
	r.w.DrawImage(img, r.opts.ImageEncoding, m)
}
