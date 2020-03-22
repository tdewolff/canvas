package canvas

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	canvasFont "github.com/tdewolff/canvas/font"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
)

type PDF struct {
	w             *pdfPageWriter
	width, height float64
	imgEnc        ImageEncoding
}

// NewPDF creates a portable document format renderer.
func NewPDF(w io.Writer, width, height float64) *PDF {
	return &PDF{
		w:      newPDFWriter(w).NewPage(width, height),
		width:  width,
		height: height,
		imgEnc: Lossless,
	}
}

func (r *PDF) SetImageEncoding(enc ImageEncoding) {
	r.imgEnc = enc
}

func (r *PDF) SetCompression(compress bool) {
	r.w.pdf.SetCompression(compress)
}

func (r *PDF) SetInfo(title, subject, keywords, author string) {
	r.w.pdf.SetTitle(title)
	r.w.pdf.SetSubject(subject)
	r.w.pdf.SetKeywords(keywords)
	r.w.pdf.SetAuthor(author)
}

// NewPage starts adds a new page where further rendering will be written to
func (r *PDF) NewPage(width, height float64) {
	r.w = r.w.pdf.NewPage(width, height)
}

func (r *PDF) Close() error {
	return r.w.pdf.Close()
}

func (r *PDF) Size() (float64, float64) {
	return r.width, r.height
}

func (r *PDF) RenderPath(path *Path, style Style, m Matrix) {
	fill := style.FillColor.A != 0
	stroke := style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth
	differentAlpha := fill && stroke && style.FillColor.A != style.StrokeColor.A

	// PDFs don't support the arcs joiner, miter joiner (not clipped), or miter joiner (clipped) with non-bevel fallback
	strokeUnsupported := false
	if _, ok := style.StrokeJoiner.(ArcsJoiner); ok {
		strokeUnsupported = true
	} else if miter, ok := style.StrokeJoiner.(MiterJoiner); ok {
		if math.IsNaN(miter.Limit) {
			strokeUnsupported = true
		} else if _, ok := miter.GapJoiner.(BevelJoiner); !ok {
			strokeUnsupported = true
		}
	}

	// PDFs don't support connecting first and last dashes if path is closed, so we move the start of the path if this is the case
	// TODO
	//if style.DashesClose {
	//	strokeUnsupported = true
	//}

	closed := false
	data := path.Transform(m).ToPDF()
	if 1 < len(data) && data[len(data)-1] == 'h' {
		data = data[:len(data)-2]
		closed = true
	}

	if !stroke || !strokeUnsupported {
		if fill && !stroke {
			r.w.SetFillColor(style.FillColor)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			r.w.Write([]byte(" f"))
			if style.FillRule == EvenOdd {
				r.w.Write([]byte("*"))
			}
		} else if !fill && stroke {
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
			if style.FillRule == EvenOdd {
				r.w.Write([]byte("*"))
			}
		} else if fill && stroke {
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
				if style.FillRule == EvenOdd {
					r.w.Write([]byte("*"))
				}
			} else {
				r.w.SetFillColor(style.FillColor)
				r.w.Write([]byte(" "))
				r.w.Write([]byte(data))
				r.w.Write([]byte(" f"))
				if style.FillRule == EvenOdd {
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
				if style.FillRule == EvenOdd {
					r.w.Write([]byte("*"))
				}
			}
		}
	} else {
		// stroke && strokeUnsupported
		if fill {
			r.w.SetFillColor(style.FillColor)
			r.w.Write([]byte(" "))
			r.w.Write([]byte(data))
			r.w.Write([]byte(" f"))
			if style.FillRule == EvenOdd {
				r.w.Write([]byte("*"))
			}
		}

		// stroke settings unsupported by PDF, draw stroke explicitly
		if 0 < len(style.Dashes) {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)

		r.w.SetFillColor(style.StrokeColor)
		r.w.Write([]byte(" "))
		r.w.Write([]byte(path.ToPDF()))
		r.w.Write([]byte(" f"))
		if style.FillRule == EvenOdd {
			r.w.Write([]byte("*"))
		}
	}
}

func (r *PDF) RenderText(text *Text, m Matrix) {
	r.w.StartTextObject()
	decoPaths := []*Path{}
	decoColors := []color.RGBA{}
	for _, line := range text.lines {
		for _, span := range line.spans {
			r.w.SetFillColor(span.ff.color)
			r.w.SetFont(span.ff.font, span.ff.size*span.ff.scale)
			r.w.SetTextPosition(m.Translate(span.dx, line.y).Shear(span.ff.fauxItalic, 0.0))
			r.w.SetTextCharSpace(span.glyphSpacing)

			if 0.0 < span.ff.fauxBold {
				r.w.SetTextRenderMode(2)
				fmt.Fprintf(r.w, " %v w", dec(span.ff.fauxBold*2.0))
			} else {
				r.w.SetTextRenderMode(0)
			}

			i := 0
			TJ := []interface{}{}
			for _, boundary := range span.boundaries {
				if boundary.kind == wordBoundary || boundary.kind == eofBoundary {
					j := boundary.pos + boundary.size
					TJ = append(TJ, span.text[i:j])
					if boundary.kind == wordBoundary {
						TJ = append(TJ, span.wordSpacing)
					}
					i = j
				}
			}
			r.w.WriteText(TJ...)
		}
		for _, deco := range line.decos {
			p := deco.ff.Decorate(deco.x1 - deco.x0)
			p = p.Transform(Identity.Mul(m).Translate(deco.x0, line.y+deco.ff.voffset))
			decoPaths = append(decoPaths, p)
			decoColors = append(decoColors, deco.ff.color)
		}
	}
	r.w.EndTextObject()
	style := DefaultStyle
	for i := range decoPaths {
		style.FillColor = decoColors[i]
		r.RenderPath(decoPaths[i], style, Identity)
	}
}

func (r *PDF) RenderImage(img image.Image, m Matrix) {
	r.w.DrawImage(img, r.imgEnc, m)
}

type pdfWriter struct {
	w   io.Writer
	err error

	pos        int
	objOffsets []int

	fonts    map[*Font]pdfRef
	pages    []*pdfPageWriter
	compress bool
	title    string
	subject  string
	keywords string
	author   string
}

func newPDFWriter(writer io.Writer) *pdfWriter {
	w := &pdfWriter{
		w:     writer,
		fonts: map[*Font]pdfRef{},
	}

	w.write("%%PDF-1.7\n")
	return w
}

func (w *pdfWriter) SetCompression(compress bool) {
	w.compress = compress
}

func (w *pdfWriter) SetTitle(title string) {
	w.title = title
}

func (w *pdfWriter) SetSubject(subject string) {
	w.subject = subject
}

func (w *pdfWriter) SetKeywords(keywords string) {
	w.keywords = keywords
}

func (w *pdfWriter) SetAuthor(author string) {
	w.author = author
}

func (w *pdfWriter) writeBytes(b []byte) {
	if w.err != nil {
		return
	}
	n, err := w.w.Write(b)
	w.pos += n
	w.err = err
}

func (w *pdfWriter) write(s string, v ...interface{}) {
	if w.err != nil {
		return
	}
	n, err := fmt.Fprintf(w.w, s, v...)
	w.pos += n
	w.err = err
}

type pdfRef int
type pdfName string
type pdfArray []interface{}
type pdfDict map[pdfName]interface{}
type pdfFilter string
type pdfStream struct {
	dict   pdfDict
	stream []byte
}

const (
	pdfFilterASCII85 pdfFilter = "ASCII85Decode"
	pdfFilterFlate   pdfFilter = "FlateDecode"
)

func (w *pdfWriter) writeVal(i interface{}) {
	switch v := i.(type) {
	case bool:
		if v {
			w.write("true")
		} else {
			w.write("false")
		}
	case int:
		w.write("%d", v)
	case float64:
		w.write("%v", dec(v))
	case string:
		v = strings.Replace(v, `\`, `\\`, -1)
		v = strings.Replace(v, `(`, `\(`, -1)
		v = strings.Replace(v, `)`, `\)`, -1)
		w.write("(%v)", v)
	case pdfRef:
		w.write("%v 0 R", v)
	case pdfName, pdfFilter:
		w.write("/%v", v)
	case pdfArray:
		w.write("[")
		for j, val := range v {
			if j != 0 {
				w.write(" ")
			}
			w.writeVal(val)
		}
		w.write("]")
	case pdfDict:
		w.write("<< ")
		if val, ok := v["Type"]; ok {
			w.write("/Type ")
			w.writeVal(val)
			w.write(" ")
		}
		if val, ok := v["Subtype"]; ok {
			w.write("/Subtype ")
			w.writeVal(val)
			w.write(" ")
		}
		keys := []string{}
		for key := range v {
			if key != "Type" && key != "Subtype" {
				keys = append(keys, string(key))
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			w.writeVal(pdfName(key))
			w.write(" ")
			w.writeVal(v[pdfName(key)])
			w.write(" ")
		}
		w.write(">>")
	case pdfStream:
		if v.dict == nil {
			v.dict = pdfDict{}
		}

		filters := []pdfFilter{}
		if filter, ok := v.dict["Filter"].(pdfFilter); ok {
			filters = append(filters, filter)
		} else if filterArray, ok := v.dict["Filter"].(pdfArray); ok {
			for i := len(filterArray) - 1; i >= 0; i-- {
				if filter, ok := filterArray[i].(pdfFilter); ok {
					filters = append(filters, filter)
				}
			}
		}

		b := v.stream
		for _, filter := range filters {
			var b2 bytes.Buffer
			switch filter {
			case pdfFilterASCII85:
				w := ascii85.NewEncoder(&b2)
				w.Write(b)
				w.Close()
			case pdfFilterFlate:
				w := zlib.NewWriter(&b2)
				w.Write(b)
				w.Close()
			}
			b = b2.Bytes()
		}

		v.dict["Length"] = len(b)
		w.writeVal(v.dict)
		w.write(" stream\n")
		w.writeBytes(b)
		w.write("\nendstream")
	default:
		panic(fmt.Sprintf("unknown PDF type %T", i))
	}
}

func (w *pdfWriter) writeObject(val interface{}) pdfRef {
	w.objOffsets = append(w.objOffsets, w.pos)
	w.write("%v 0 obj\n", len(w.objOffsets))
	w.writeVal(val)
	w.write("\nendobj\n")
	return pdfRef(len(w.objOffsets))
}

func (w *pdfWriter) getFont(font *Font) pdfRef {
	if ref, ok := w.fonts[font]; ok {
		return ref
	}

	mimetype, b := font.Raw()
	if mimetype != "font/truetype" && mimetype != "font/opentype" {
		var err error
		b, mimetype, err = canvasFont.ToSFNT(b)
		if err != nil {
			panic(err)
		}
		if mimetype != "font/truetype" && mimetype != "font/opentype" {
			panic("only TTF and OTF formats supported for embedding fonts in PDFs")
		}
	}

	ffSubtype := ""
	cidSubtype := ""
	if mimetype == "font/truetype" {
		ffSubtype = "TrueType"
		cidSubtype = "CIDFontType2"
	} else if mimetype == "font/opentype" {
		ffSubtype = "OpenType"
		cidSubtype = "CIDFontType0"
	}

	bounds, italicAngle, ascent, descent, capHeight, widths := font.pdfInfo()

	// shorten glyph widths array
	DW := widths[0]
	W := pdfArray{}
	i, j := 1, 1
	for k, width := range widths {
		if k != 0 && width != widths[j] {
			if 4 < k-j { // at about 5 equal widths, it would be shorter using the other notation format
				if i < j {
					arr := pdfArray{}
					for _, w := range widths[i:j] {
						arr = append(arr, w)
					}
					W = append(W, i, arr)
				}
				if widths[j] != DW {
					W = append(W, j, k-1, widths[j])
				}
				i = k
			}
			j = k
		}
	}
	if i < len(widths) {
		arr := pdfArray{}
		for _, w := range widths[i:] {
			arr = append(arr, w)
		}
		W = append(W, i, arr)
	}

	baseFont := strings.ReplaceAll(font.name, " ", "_")
	fontfileRef := w.writeObject(pdfStream{
		dict: pdfDict{
			"Subtype": pdfName(ffSubtype),
			"Filter":  pdfFilterFlate,
		},
		stream: b,
	})
	ref := w.writeObject(pdfDict{
		"Type":     pdfName("Font"),
		"Subtype":  pdfName("Type0"),
		"BaseFont": pdfName(baseFont),
		"Encoding": pdfName("Identity-H"),
		"DescendantFonts": pdfArray{pdfDict{
			"Type":        pdfName("Font"),
			"Subtype":     pdfName(cidSubtype),
			"BaseFont":    pdfName(baseFont),
			"CIDToGIDMap": pdfName("Identity"),
			"DW":          DW,
			"W":           W,
			"CIDSystemInfo": pdfDict{
				"Registry":   "Adobe",
				"Ordering":   "Identity",
				"Supplement": 0,
			},
			"FontDescriptor": pdfDict{
				"Type":        pdfName("FontDescriptor"),
				"FontName":    pdfName(baseFont),
				"Flags":       4,
				"FontBBox":    pdfArray{int(bounds.X), -int(bounds.Y + bounds.H), int(bounds.X + bounds.W), -int(bounds.Y)},
				"ItalicAngle": italicAngle,
				"Ascent":      int(ascent),
				"Descent":     -int(descent),
				"CapHeight":   -int(capHeight),
				"StemV":       80, // taken from Inkscape, should be calculated somehow
				"StemH":       80,
				"FontFile3":   fontfileRef,
			},
		}},
	})
	w.fonts[font] = ref
	return ref
}

func (w *pdfWriter) Close() error {
	parent := pdfRef(len(w.objOffsets) + 2 + len(w.pages))
	kids := pdfArray{}
	for _, p := range w.pages {
		kids = append(kids, p.writePage(parent))
	}

	refPages := w.writeObject(pdfDict{
		"Type":  pdfName("Pages"),
		"Kids":  pdfArray(kids),
		"Count": len(kids),
	})

	info := pdfDict{
		"Producer":     "tdewolff/canvas",
		"CreationDate": time.Now().Format("D:20060102150405Z0700"),
	}
	if w.title != "" {
		info["title"] = w.title
	}
	if w.subject != "" {
		info["subject"] = w.subject
	}
	if w.keywords != "" {
		info["keywords"] = w.keywords
	}
	if w.author != "" {
		info["author"] = w.author
	}
	refInfo := w.writeObject(info)

	refCatalog := w.writeObject(pdfDict{
		"Type":  pdfName("Catalog"),
		"Pages": refPages,
	})

	xrefOffset := w.pos
	w.write("xref\n0 %d\n0000000000 65535 f\n", len(w.objOffsets)+1)
	for _, objOffset := range w.objOffsets {
		w.write("%010d 00000 n\n", objOffset)
	}
	w.write("trailer\n")
	w.writeVal(pdfDict{
		"Root": refCatalog,
		"Size": len(w.objOffsets) + 1,
		"Info": refInfo,
	})
	w.write("\nstartxref\n%v\n%%%%EOF", xrefOffset)
	return w.err
}

type pdfPageWriter struct {
	*bytes.Buffer
	pdf           *pdfWriter
	width, height float64
	resources     pdfDict

	graphicsStates map[float64]pdfName
	alpha          float64
	fillColor      color.RGBA
	strokeColor    color.RGBA
	lineWidth      float64
	lineCap        int
	lineJoin       int
	miterLimit     float64
	dashes         []float64
	font           *Font
	fontSize       float64
	inTextObject   bool
	textPosition   Matrix
	textCharSpace  float64
	textRenderMode int
}

func (w *pdfWriter) NewPage(width, height float64) *pdfPageWriter {
	// for defaults see https://help.adobe.com/pdfl_sdk/15/PDFL_SDK_HTMLHelp/PDFL_SDK_HTMLHelp/API_References/PDFL_API_Reference/PDFEdit_Layer/General.html#_t_PDEGraphicState
	page := &pdfPageWriter{
		Buffer:         &bytes.Buffer{},
		pdf:            w,
		width:          width,
		height:         height,
		resources:      pdfDict{},
		graphicsStates: map[float64]pdfName{},
		alpha:          1.0,
		fillColor:      Black,
		strokeColor:    Black,
		lineWidth:      1.0,
		lineCap:        0,
		lineJoin:       0,
		miterLimit:     10.0,
		dashes:         []float64{0.0}, // dashArray and dashPhase
		font:           nil,
		fontSize:       0.0,
		inTextObject:   false,
		textPosition:   Identity,
		textCharSpace:  0.0,
		textRenderMode: 0,
	}
	w.pages = append(w.pages, page)

	m := Identity.Scale(ptPerMm, ptPerMm)
	fmt.Fprintf(page, " %v %v %v %v %v %v cm", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]))
	return page
}

func (w *pdfPageWriter) writePage(parent pdfRef) pdfRef {
	b := w.Bytes()
	if 0 < len(b) && b[0] == ' ' {
		b = b[1:]
	}
	stream := pdfStream{
		dict:   pdfDict{},
		stream: b,
	}
	if w.pdf.compress {
		stream.dict["Filter"] = pdfFilterFlate
	}
	contents := w.pdf.writeObject(stream)
	return w.pdf.writeObject(pdfDict{
		"Type":      pdfName("Page"),
		"Parent":    parent,
		"MediaBox":  pdfArray{0.0, 0.0, w.width * ptPerMm, w.height * ptPerMm},
		"Resources": w.resources,
		"Group": pdfDict{
			"Type": pdfName("Group"),
			"S":    pdfName("Transparency"),
			"I":    true,
			"CS":   pdfName("DeviceRGB"),
		},
		"Contents": contents,
	})
}

func (w *pdfPageWriter) SetAlpha(alpha float64) {
	if alpha != w.alpha {
		gs := w.getOpacityGS(alpha)
		fmt.Fprintf(w, " /%v gs", gs)
		w.alpha = alpha
	}
}

func (w *pdfPageWriter) SetFillColor(fillColor color.RGBA) {
	a := float64(fillColor.A) / 255.0
	if fillColor != w.fillColor {
		if fillColor.R == fillColor.G && fillColor.R == fillColor.B {
			fmt.Fprintf(w, " %v g", dec(float64(fillColor.R)/255.0/a))
		} else {
			fmt.Fprintf(w, " %v %v %v rg", dec(float64(fillColor.R)/255.0/a), dec(float64(fillColor.G)/255.0/a), dec(float64(fillColor.B)/255.0/a))
		}
		w.fillColor = fillColor
	}
	w.SetAlpha(a)
}

func (w *pdfPageWriter) SetStrokeColor(strokeColor color.RGBA) {
	a := float64(strokeColor.A) / 255.0
	if strokeColor != w.strokeColor {
		if strokeColor.R == strokeColor.G && strokeColor.R == strokeColor.B {
			fmt.Fprintf(w, " %v G", dec(float64(strokeColor.R)/255.0/a))
		} else {
			fmt.Fprintf(w, " %v %v %v RG", dec(float64(strokeColor.R)/255.0/a), dec(float64(strokeColor.G)/255.0/a), dec(float64(strokeColor.B)/255.0/a))
		}
		w.strokeColor = strokeColor
	}
	w.SetAlpha(a)
}

func (w *pdfPageWriter) SetLineWidth(lineWidth float64) {
	if lineWidth != w.lineWidth {
		fmt.Fprintf(w, " %v w", dec(lineWidth))
		w.lineWidth = lineWidth
	}
}

func (w *pdfPageWriter) SetLineCap(capper Capper) {
	var lineCap int
	if _, ok := capper.(ButtCapper); ok {
		lineCap = 0
	} else if _, ok := capper.(RoundCapper); ok {
		lineCap = 1
	} else if _, ok := capper.(SquareCapper); ok {
		lineCap = 2
	} else {
		panic("PDF: line cap not support")
	}
	if lineCap != w.lineCap {
		fmt.Fprintf(w, " %d J", lineCap)
		w.lineCap = lineCap
	}
}

func (w *pdfPageWriter) SetLineJoin(joiner Joiner) {
	var lineJoin int
	var miterLimit float64
	if _, ok := joiner.(BevelJoiner); ok {
		lineJoin = 2
	} else if _, ok := joiner.(RoundJoiner); ok {
		lineJoin = 1
	} else if miter, ok := joiner.(MiterJoiner); ok {
		lineJoin = 0
		if math.IsNaN(miter.Limit) {
			panic("PDF: line join not support")
		} else {
			miterLimit = miter.Limit
		}
	} else {
		panic("PDF: line join not support")
	}
	if lineJoin != w.lineJoin {
		fmt.Fprintf(w, " %d j", lineJoin)
		w.lineJoin = lineJoin
	}
	if lineJoin == 0 && miterLimit != w.miterLimit {
		fmt.Fprintf(w, " %v M", dec(miterLimit))
		w.miterLimit = miterLimit
	}
}

func (w *pdfPageWriter) SetDashes(dashPhase float64, dashArray []float64) {
	if len(dashArray)%2 == 1 {
		dashArray = append(dashArray, dashArray...)
	}

	// PDF can't handle negative dash phases
	if dashPhase < 0.0 {
		totalLength := 0.0
		for _, dash := range dashArray {
			totalLength += dash
		}
		for dashPhase < 0.0 {
			dashPhase += totalLength
		}
	}

	dashes := append(dashArray, dashPhase)
	if !float64sEqual(dashes, w.dashes) {
		if len(dashes) == 1 {
			fmt.Fprintf(w, " [] 0 d")
			dashes[0] = 0.0
		} else {
			fmt.Fprintf(w, " [%v", dec(dashes[0]))
			for _, dash := range dashes[1 : len(dashes)-1] {
				fmt.Fprintf(w, " %v", dec(dash))
			}
			fmt.Fprintf(w, "] %v d", dec(dashes[len(dashes)-1]))
		}
		w.dashes = dashes
	}
}

func (w *pdfPageWriter) SetFont(font *Font, size float64) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if font != w.font || w.fontSize != size {
		w.font = font
		w.fontSize = size

		ref := w.pdf.getFont(font)
		if _, ok := w.resources["Font"]; !ok {
			w.resources["Font"] = pdfDict{}
		} else {
			for name, fontRef := range w.resources["Font"].(pdfDict) {
				if ref == fontRef {
					fmt.Fprintf(w, " /%v %v Tf", name, dec(size))
					return
				}
			}
		}

		name := pdfName(fmt.Sprintf("F%d", len(w.resources["Font"].(pdfDict))))
		w.resources["Font"].(pdfDict)[name] = ref
		fmt.Fprintf(w, " /%v %v Tf", name, dec(size))
	}
}

func (w *pdfPageWriter) SetTextPosition(m Matrix) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if m.Equals(w.textPosition) {
		return
	}

	if equal(m[0][0], w.textPosition[0][0]) && equal(m[0][1], w.textPosition[0][1]) && equal(m[1][0], w.textPosition[1][0]) && equal(m[1][1], w.textPosition[1][1]) {
		d := w.textPosition.Inv().Dot(Point{m[0][2], m[1][2]})
		fmt.Fprintf(w, " %v %v Td", dec(d.X), dec(d.Y))
	} else {
		fmt.Fprintf(w, " %v %v %v %v %v %v Tm", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]))
	}
	w.textPosition = m
}

func (w *pdfPageWriter) SetTextRenderMode(mode int) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if w.textRenderMode != mode {
		fmt.Fprintf(w, " %d Tr", mode)
		w.textRenderMode = mode
	}
}

func (w *pdfPageWriter) SetTextCharSpace(space float64) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if !equal(w.textCharSpace, space) {
		fmt.Fprintf(w, " %v Tc", dec(space))
		w.textCharSpace = space
	}
}

func (w *pdfPageWriter) StartTextObject() {
	if w.inTextObject {
		panic("already in text object")
	}
	fmt.Fprintf(w, " BT")
	w.textPosition = Identity
	w.inTextObject = true
}

func (w *pdfPageWriter) EndTextObject() {
	if !w.inTextObject {
		panic("must be in text object")
	}
	fmt.Fprintf(w, " ET")
	w.inTextObject = false
}

func (w *pdfPageWriter) WriteText(TJ ...interface{}) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if len(TJ) == 0 || w.font == nil {
		return
	}

	units := float64(w.font.sfnt.UnitsPerEm())

	first := true
	write := func(s string) {
		if first {
			fmt.Fprintf(w, "(")
			first = false
		} else {
			fmt.Fprintf(w, " (")
		}

		buf := &bytes.Buffer{}
		indices := w.font.toIndices(s)
		binary.Write(buf, binary.BigEndian, indices)

		s = buf.String()
		s = strings.Replace(s, "\\", "\\\\", -1)
		s = strings.Replace(s, "(", "\\(", -1)
		s = strings.Replace(s, ")", "\\)", -1)
		fmt.Fprintf(w, "%s)", s)
	}

	var sfntBuffer sfnt.Buffer
	fmt.Fprintf(w, "[")
	for _, tj := range TJ {
		switch val := tj.(type) {
		case string:
			i := 0
			var rPrev rune
			for j, r := range val {
				if i < j {
					i0, err0 := w.font.sfnt.GlyphIndex(&sfntBuffer, rPrev)
					i1, err1 := w.font.sfnt.GlyphIndex(&sfntBuffer, r)
					if err0 == nil && err1 == nil {
						kern, err := w.font.sfnt.Kern(&sfntBuffer, i0, i1, toI26_6(units), font.HintingNone)
						if err == nil && kern != 0.0 {
							write(val[i:j])
							fmt.Fprintf(w, " %d", -int(fromI26_6(kern)*1000.0/units+0.5))
							i = j
						}
					}
				}
				rPrev = r
			}
			write(val[i:])
		case float64:
			fmt.Fprintf(w, " %d", -int(val*1000.0/w.fontSize+0.5))
		case int:
			fmt.Fprintf(w, " %d", -int(float64(val)*1000.0/w.fontSize+0.5))
		}
	}
	fmt.Fprintf(w, "]TJ")
}

func (w *pdfPageWriter) DrawImage(img image.Image, enc ImageEncoding, m Matrix) {
	size := img.Bounds().Size()

	// add clipping path around image for smooth edges when rotating
	outerRect := Rect{0.0, 0.0, float64(size.X), float64(size.Y)}.Transform(m)
	bl := m.Dot(Point{0, 0})
	br := m.Dot(Point{float64(size.X), 0})
	tl := m.Dot(Point{0, float64(size.Y)})
	tr := m.Dot(Point{float64(size.X), float64(size.Y)})
	fmt.Fprintf(w, " q %v %v %v %v re W n", dec(outerRect.X), dec(outerRect.Y), dec(outerRect.W), dec(outerRect.H))
	fmt.Fprintf(w, " %v %v m %v %v l %v %v l %v %v l h W n", dec(bl.X), dec(bl.Y), dec(tl.X), dec(tl.Y), dec(tr.X), dec(tr.Y), dec(br.X), dec(br.Y))

	name := w.embedImage(img, enc)
	m = m.Scale(float64(size.X), float64(size.Y))
	w.SetAlpha(1.0)
	fmt.Fprintf(w, " %v %v %v %v %v %v cm /%v Do Q", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]), name)
}

func (w *pdfPageWriter) embedImage(img image.Image, enc ImageEncoding) pdfName {
	size := img.Bounds().Size()
	b := make([]byte, size.X*size.Y*3)
	bMask := make([]byte, size.X*size.Y)
	hasMask := false
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 3
			R, G, B, A := img.At(x, y).RGBA()
			if A != 0 {
				b[i+0] = byte((R * 65535 / A) >> 8)
				b[i+1] = byte((G * 65535 / A) >> 8)
				b[i+2] = byte((B * 65535 / A) >> 8)
				bMask[y*size.X+x] = byte(A >> 8)
			}
			if A>>8 != 255 {
				hasMask = true
			}
		}
	}

	dict := pdfDict{
		"Type":             pdfName("XObject"),
		"Subtype":          pdfName("Image"),
		"Width":            size.X,
		"Height":           size.Y,
		"ColorSpace":       pdfName("DeviceRGB"),
		"BitsPerComponent": 8,
		"Interpolate":      true,
		"Filter":           pdfFilterFlate,
	}

	if hasMask {
		dict["SMask"] = w.pdf.writeObject(pdfStream{
			dict: pdfDict{
				"Type":             pdfName("XObject"),
				"Subtype":          pdfName("Image"),
				"Width":            size.X,
				"Height":           size.Y,
				"ColorSpace":       pdfName("DeviceGray"),
				"BitsPerComponent": 8,
				"Interpolate":      true,
				"Filter":           pdfFilterFlate,
			},
			stream: bMask,
		})
	}

	// TODO: (PDF) implement JPXFilter for lossy image compression
	ref := w.pdf.writeObject(pdfStream{
		dict:   dict,
		stream: b,
	})

	if _, ok := w.resources["XObject"]; !ok {
		w.resources["XObject"] = pdfDict{}
	}
	name := pdfName(fmt.Sprintf("Im%d", len(w.resources["XObject"].(pdfDict))))
	w.resources["XObject"].(pdfDict)[name] = ref
	return name
}

func (w *pdfPageWriter) getOpacityGS(a float64) pdfName {
	if name, ok := w.graphicsStates[a]; ok {
		return name
	}
	name := pdfName(fmt.Sprintf("A%d", len(w.graphicsStates)))
	w.graphicsStates[a] = name

	if _, ok := w.resources["ExtGState"]; !ok {
		w.resources["ExtGState"] = pdfDict{}
	}
	w.resources["ExtGState"].(pdfDict)[name] = pdfDict{
		"CA": a,
		"ca": a,
	}
	return name
}
