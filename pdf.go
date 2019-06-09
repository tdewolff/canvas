package canvas

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"image/color"
	"io"
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
	dict    PDFDict
	filters []PDFFilter
	b       []byte
}

const (
	PDFFilterASCII85 PDFFilter = "ASCII85Decode"
	PDFFilterFlate   PDFFilter = "FlateDecode"
)

func (w *PDFWriter) writeVal(i interface{}) {
	switch v := i.(type) {
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
	case PDFName:
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
		for key, val := range v {
			w.writeVal(key)
			w.write(" ")
			w.writeVal(val)
			w.write(" ")
		}
		w.write(">>")
	case PDFStream:
		filters := PDFArray{}
		for j := len(v.filters) - 1; j >= 0; j-- {
			filters = append(filters, PDFName(v.filters[j]))
		}

		b := v.b
		for _, filter := range v.filters {
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

		dict := v.dict
		if dict == nil {
			dict = PDFDict{}
		}
		if len(filters) == 1 {
			dict["Filter"] = filters[0]
		} else if len(filters) > 1 {
			dict["Filter"] = filters
		}
		dict["Length"] = len(b)
		w.writeVal(dict)
		w.write("\nstream\n")
		w.writeBytes(b)
		w.write("\nendstream")
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

	mimetype, _ := font.Raw()
	if mimetype != "font/ttf" {
		panic("only TTF format support for embedding fonts in PDFs")
	}

	// TODO: implement
	baseFont := strings.ReplaceAll(font.name, " ", "_")
	ref := w.writeObject(PDFStream{
		dict: PDFDict{
			"Type":     PDFName("Font"),
			"Subtype":  PDFName("TrueType"),
			"BaseFont": PDFName(baseFont),
		},
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

	color          color.RGBA
	font           *Font
	graphicsStates map[float64]PDFName
}

func (w *PDFWriter) NewPage(width, height float64) *PDFPageWriter {
	w.pages = append(w.pages, &PDFPageWriter{
		Buffer:         &bytes.Buffer{},
		pdf:            w,
		width:          width,
		height:         height,
		resources:      PDFDict{},
		color:          Black,
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
	return w.pdf.writeObject(PDFDict{
		"Type":      PDFName("Page"),
		"Parent":    parent,
		"MediaBox":  PDFArray{0.0, 0.0, w.width, w.height},
		"Resources": w.resources,
		"Contents": PDFStream{
			filters: []PDFFilter{}, // TODO: use filter
			b:       b,
		},
	})
}

func (w *PDFPageWriter) SetColor(color color.RGBA) {
	if color != w.color {
		if w.color.A != 255 && color.A != w.color.A {
			fmt.Fprintf(w, " Q")
		}
		fmt.Fprintf(w, " %f %f %f rg", float64(color.R)/255.0, float64(color.G)/255.0, float64(color.B)/255.0)
		if color.A != w.color.A {
			gs := w.getOpacityGS(float64(color.A) / 255.0)
			fmt.Fprintf(w, " q /%v gs", gs)
		}
		w.color = color
	}
}

func (w *PDFPageWriter) SetFont(font *Font, size float64) {
	if font != w.font {
		ref := w.pdf.getFont(font)
		if _, ok := w.resources["Font"]; !ok {
			w.resources["Font"] = PDFDict{}
		}
		name := PDFName(fmt.Sprintf("F%d", len(w.resources["Font"].(PDFDict))))
		w.resources["Font"].(PDFDict)[name] = ref
		fmt.Fprintf(w, " /%v %f Tf", name, size)
	}
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
		"ca": a,
	}
	return name
}
