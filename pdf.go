package canvas

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"sort"
	"strings"
)

type PDFWriter struct {
	w   io.Writer
	err error

	pos        int
	objOffsets []int

	fonts map[*Font]PDFRef
	pages []*PDFPageWriter
}

func NewPDFWriter(writer io.Writer) *PDFWriter {
	w := &PDFWriter{
		w:     writer,
		fonts: map[*Font]PDFRef{},
	}

	w.write("%%PDF-1.7\n")
	return w
}

func (w *PDFWriter) writeBytes(b []byte) {
	if w.err != nil {
		return
	}
	n, err := w.w.Write(b)
	w.pos += n
	w.err = err
}

func (w *PDFWriter) write(s string, v ...interface{}) {
	if w.err != nil {
		return
	}
	n, err := fmt.Fprintf(w.w, s, v...)
	w.pos += n
	w.err = err
}

type PDFRef int
type PDFName string
type PDFArray []interface{}
type PDFDict map[PDFName]interface{}
type PDFFilter string
type PDFStream struct {
	dict   PDFDict
	stream []byte
}

const (
	PDFFilterASCII85 PDFFilter = "ASCII85Decode"
	PDFFilterFlate   PDFFilter = "FlateDecode"
)

func (w *PDFWriter) writeVal(i interface{}) {
	switch v := i.(type) {
	case bool:
		if v {
			w.write("1")
		} else {
			w.write("0")
		}
	case int:
		w.write("%d", v)
	case float64:
		w.write("%f", v)
	case string:
		v = strings.Replace(v, `\`, `\\`, -1)
		v = strings.Replace(v, `(`, `\(`, -1)
		v = strings.Replace(v, `)`, `\)`, -1)
		w.write("(%v)", v)
	case PDFRef:
		w.write("%v 0 R", v)
	case PDFName, PDFFilter:
		w.write("/%v", v)
	case PDFArray:
		w.write("[")
		for j, val := range v {
			if j != 0 {
				w.write(" ")
			}
			w.writeVal(val)
		}
		w.write("]")
	case PDFDict:
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
			w.writeVal(PDFName(key))
			w.write(" ")
			w.writeVal(v[PDFName(key)])
			w.write(" ")
		}
		w.write(">>")
	case PDFStream:
		if v.dict == nil {
			v.dict = PDFDict{}
		}

		filters := []PDFFilter{}
		if filter, ok := v.dict["Filter"].(PDFFilter); ok {
			filters = append(filters, filter)
		} else if filterArray, ok := v.dict["Filter"].(PDFArray); ok {
			for i := len(filterArray) - 1; i >= 0; i-- {
				if filter, ok := filterArray[i].(PDFFilter); ok {
					filters = append(filters, filter)
				}
			}
		}

		b := v.stream
		for _, filter := range filters {
			var b2 bytes.Buffer
			switch filter {
			case PDFFilterASCII85:
				w := ascii85.NewEncoder(&b2)
				w.Write(b)
				w.Close()
			case PDFFilterFlate:
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

func (w *PDFWriter) writeObject(val interface{}) PDFRef {
	w.objOffsets = append(w.objOffsets, w.pos)
	w.write("%v 0 obj\n", len(w.objOffsets))
	w.writeVal(val)
	w.write("\nendobj\n")
	return PDFRef(len(w.objOffsets))
}

func (w *PDFWriter) getFont(font *Font) PDFRef {
	if ref, ok := w.fonts[font]; ok {
		return ref
	}

	mimetype, b := font.Raw()
	if mimetype != "font/truetype" && mimetype != "font/opentype" {
		panic("only TTF and OTF formats supported for embedding fonts in PDFs")
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

	widths := font.Widths()

	baseFont := strings.ReplaceAll(font.name, " ", "_")
	fontfileRef := w.writeObject(PDFStream{
		dict: PDFDict{
			"Subtype": PDFName(ffSubtype),
			"Filter":  PDFFilterFlate,
		},
		stream: b,
	})
	ref := w.writeObject(PDFDict{
		"Type":     PDFName("Font"),
		"Subtype":  PDFName("Type0"),
		"BaseFont": PDFName(baseFont),
		"Encoding": PDFName("Identity-H"),
		"DescendantFonts": PDFArray{PDFDict{
			"Type":        PDFName("Font"),
			"Subtype":     PDFName(cidSubtype),
			"BaseFont":    PDFName(baseFont),
			"CIDToGIDMap": PDFName("Identity"),
			"DW":          widths[0],
			"W":           PDFArray{0, PDFArray(widths)},
			"CIDSystemInfo": PDFDict{
				"Registry":   "Adobe",
				"Ordering":   "Identity",
				"Supplement": 0,
			},
			"FontDescriptor": PDFDict{
				"Type":        PDFName("FontDescriptor"),
				"FontName":    PDFName(baseFont),
				"Flags":       4,
				"FontBBox":    PDFArray{0.0, 0.0, 0.0, 0.0},
				"ItalicAngle": 0,
				"Ascent":      0, // not used since it's embedded in a Type 1 font?
				"Descent":     0,
				"CapHeight":   0,
				"StemV":       0,
				"FontFile3":   fontfileRef,
			},
		}},
	})
	w.fonts[font] = ref
	return ref
}

func (w *PDFWriter) Close() error {
	parent := PDFRef(len(w.objOffsets) + 1 + len(w.pages))
	kids := PDFArray{}
	for _, p := range w.pages {
		kids = append(kids, p.writePage(parent))
	}

	refPages := w.writeObject(PDFDict{
		"Type":  PDFName("Pages"),
		"Kids":  PDFArray(kids),
		"Count": len(kids),
	})

	refCatalog := w.writeObject(PDFDict{
		"Type":  PDFName("Catalog"),
		"Pages": refPages,
	})

	xrefOffset := w.pos
	w.write("xref\n0 %d\n0000000000 65535 f\n", len(w.objOffsets)+1)
	for _, objOffset := range w.objOffsets {
		w.write("%010d 00000 n\n", objOffset)
	}
	w.write("trailer\n")
	w.writeVal(PDFDict{
		"Root": refCatalog,
		"Size": len(w.objOffsets),
	})
	w.write("\nstarxref\n%v\n%%%%EOF", xrefOffset)
	return w.err
}

type PDFPageWriter struct {
	*bytes.Buffer
	pdf           *PDFWriter
	width, height float64
	resources     PDFDict

	graphicsStates map[float64]PDFName
	alpha          float64
	fillColor      color.RGBA
	strokeColor    color.RGBA
	textColor      color.RGBA
	lineWidth      float64
	lineCap        int
	lineJoin       int
	miterLimit     float64
	dashes         []float64
	font           *Font
}

func (w *PDFWriter) NewPage(width, height float64) *PDFPageWriter {
	// for defaults see https://help.adobe.com/pdfl_sdk/15/PDFL_SDK_HTMLHelp/PDFL_SDK_HTMLHelp/API_References/PDFL_API_Reference/PDFEdit_Layer/General.html#_t_PDEGraphicState
	w.pages = append(w.pages, &PDFPageWriter{
		Buffer:         &bytes.Buffer{},
		pdf:            w,
		width:          width,
		height:         height,
		resources:      PDFDict{},
		alpha:          1.0,
		fillColor:      Black,
		strokeColor:    Black,
		textColor:      Black,
		lineWidth:      1.0,
		lineCap:        0,
		lineJoin:       0,
		miterLimit:     10.0,
		dashes:         []float64{0.0}, // dashArray and dashPhase
		font:           nil,
		graphicsStates: map[float64]PDFName{},
	})
	return w.pages[len(w.pages)-1]
}

func (w *PDFPageWriter) writePage(parent PDFRef) PDFRef {
	b := w.Bytes()
	if 0 < len(b) && b[0] == ' ' {
		b = b[1:]
	}
	contents := w.pdf.writeObject(PDFStream{
		dict: PDFDict{
			"Filter": PDFFilterFlate,
		},
		stream: b,
	})
	return w.pdf.writeObject(PDFDict{
		"Type":      PDFName("Page"),
		"Parent":    parent,
		"MediaBox":  PDFArray{0.0, 0.0, w.width, w.height},
		"Resources": w.resources,
		"Group": PDFDict{
			"Type": PDFName("Group"),
			"S":    PDFName("Transparency"),
			"I":    true,
			"CS":   PDFName("DeviceRGB"),
		},
		"Contents": contents,
	})
}

func (w *PDFPageWriter) SetAlpha(alpha float64) {
	if alpha != w.alpha {
		gs := w.getOpacityGS(alpha)
		fmt.Fprintf(w, " /%v gs", gs)
		w.alpha = alpha
	}
}

func (w *PDFPageWriter) SetFillColor(fillColor color.RGBA) {
	if fillColor != w.fillColor {
		if fillColor.R == fillColor.G && fillColor.R == fillColor.B {
			fmt.Fprintf(w, " %f g", float64(fillColor.R)/255.0)
		} else {
			fmt.Fprintf(w, " %f %f %f rg", float64(fillColor.R)/255.0, float64(fillColor.G)/255.0, float64(fillColor.B)/255.0)
		}
		w.fillColor = fillColor
	}
	w.SetAlpha(float64(fillColor.A) / 255.0)
}

func (w *PDFPageWriter) SetStrokeColor(strokeColor color.RGBA) {
	if strokeColor != w.strokeColor {
		if strokeColor.R == strokeColor.G && strokeColor.R == strokeColor.B {
			fmt.Fprintf(w, " %f G", float64(strokeColor.R)/255.0)
		} else {
			fmt.Fprintf(w, " %f %f %f RG", float64(strokeColor.R)/255.0, float64(strokeColor.G)/255.0, float64(strokeColor.B)/255.0)
		}
		w.strokeColor = strokeColor
	}
	w.SetAlpha(float64(strokeColor.A) / 255.0)
}

func (w *PDFPageWriter) SetTextColor(textColor color.RGBA) {
	if textColor != w.textColor {
		if textColor.R == textColor.G && textColor.R == textColor.B {
			fmt.Fprintf(w, " %f g", float64(textColor.R)/255.0)
		} else {
			fmt.Fprintf(w, " %f %f %f rg", float64(textColor.R)/255.0, float64(textColor.G)/255.0, float64(textColor.B)/255.0)
		}
		w.textColor = textColor
	}
	w.SetAlpha(float64(textColor.A) / 255.0)
}

func (w *PDFPageWriter) SetLineWidth(lineWidth float64) {
	if lineWidth != w.lineWidth {
		fmt.Fprintf(w, " %f w", lineWidth)
		w.lineWidth = lineWidth
	}
}

func (w *PDFPageWriter) SetLineCap(capper Capper) {
	var lineCap int
	if _, ok := capper.(buttCapper); ok {
		lineCap = 0
	} else if _, ok := capper.(roundCapper); ok {
		lineCap = 1
	} else if _, ok := capper.(squareCapper); ok {
		lineCap = 2
	} else {
		panic("PDF: line cap not support")
	}
	if lineCap != w.lineCap {
		fmt.Fprintf(w, " %d J", lineCap)
		w.lineCap = lineCap
	}
}

func (w *PDFPageWriter) SetLineJoin(joiner Joiner) {
	var lineJoin int
	var miterLimit float64
	if _, ok := joiner.(bevelJoiner); ok {
		lineJoin = 2
	} else if _, ok := joiner.(roundJoiner); ok {
		lineJoin = 1
	} else if miter, ok := joiner.(miterJoiner); ok {
		lineJoin = 0
		if math.IsNaN(miter.limit) {
			panic("PDF: line join not support")
		} else {
			miterLimit = miter.limit
		}
	} else {
		panic("PDF: line join not support")
	}
	if lineJoin != w.lineJoin {
		fmt.Fprintf(w, " %d j", lineJoin)
		w.lineJoin = lineJoin
	}
	if lineJoin == 0 && miterLimit != w.miterLimit {
		fmt.Fprintf(w, " %g M", miterLimit)
		w.miterLimit = miterLimit
	}
}

func (w *PDFPageWriter) SetDashes(dashPhase float64, dashArray []float64) {
	// TODO: connect the first and last dash if they coincide
	// TODO: mind that dash pattern is restarted for each path segment, in contrary to Dash()
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
	equal := false
	if len(dashes) == len(w.dashes) {
		equal = true
		for i, dash := range dashes {
			if dash != w.dashes[i] {
				equal = false
				break
			}
		}
	}

	if !equal {
		if len(dashes) == 1 {
			fmt.Fprintf(w, " [] 0 d")
			dashes[0] = 0.0
		} else {
			fmt.Fprintf(w, " [%f", dashes[0])
			for _, dash := range dashes[1 : len(dashes)-1] {
				fmt.Fprintf(w, " %f", dash)
			}
			fmt.Fprintf(w, "] %f d", dashes[len(dashes)-1])
		}
		w.dashes = dashes
	}
}

func (w *PDFPageWriter) SetFont(font *Font, size float64) {
	if font != w.font {
		ref := w.pdf.getFont(font)
		if _, ok := w.resources["Font"]; !ok {
			w.resources["Font"] = PDFDict{}
		} else {
			for name, fontRef := range w.resources["Font"].(PDFDict) {
				if ref == fontRef {
					fmt.Fprintf(w, " /%v %f Tf", name, size)
					return
				}
			}
		}

		name := PDFName(fmt.Sprintf("F%d", len(w.resources["Font"].(PDFDict))))
		w.resources["Font"].(PDFDict)[name] = ref
		fmt.Fprintf(w, " /%v %f Tf", name, size)
	}
}

func (w *PDFPageWriter) DrawImage(img image.Image, enc ImageEncoding, m Matrix) {
	name := w.embedImage(img, enc)
	size := img.Bounds().Size()
	m = m.Scale(float64(size.X), float64(size.Y))
	w.SetAlpha(1.0)
	fmt.Fprintf(w, " q %f %f %f %f %f %f cm /%v Do Q", m[0][0], -m[0][1], -m[1][0], m[1][1], m[0][2], m[1][2], name)
}

func (w *PDFPageWriter) embedImage(img image.Image, enc ImageEncoding) PDFName {
	size := img.Bounds().Size()
	b := make([]byte, size.X*size.Y*3)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 3
			R, G, B, _ := img.At(x, y).RGBA()
			b[i+0] = byte(R >> 8)
			b[i+1] = byte(G >> 8)
			b[i+2] = byte(B >> 8)
			// TODO: handle alpha channel
		}
	}

	// TODO: implement JPXFilter for Lossy image compression
	ref := w.pdf.writeObject(PDFStream{
		dict: PDFDict{
			"Type":             PDFName("XObject"),
			"Subtype":          PDFName("Image"),
			"Width":            size.X,
			"Height":           size.Y,
			"ColorSpace":       PDFName("DeviceRGB"),
			"BitsPerComponent": 8,
			"Interpolation":    true,
			"Filter":           PDFFilterFlate,
		},
		stream: b,
	})

	if _, ok := w.resources["XObject"]; !ok {
		w.resources["XObject"] = PDFDict{}
	}
	name := PDFName(fmt.Sprintf("Im%d", len(w.resources["XObject"].(PDFDict))))
	w.resources["XObject"].(PDFDict)[name] = ref
	return name
}

func (w *PDFPageWriter) getOpacityGS(a float64) PDFName {
	if name, ok := w.graphicsStates[a]; ok {
		return name
	}
	name := PDFName(fmt.Sprintf("GS%d", len(w.graphicsStates)))
	w.graphicsStates[a] = name

	if _, ok := w.resources["ExtGState"]; !ok {
		w.resources["ExtGState"] = PDFDict{}
	}
	w.resources["ExtGState"].(PDFDict)[name] = PDFDict{
		"CA": a,
		"ca": a,
	}
	return name
}
