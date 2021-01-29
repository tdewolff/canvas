package pdf

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

	"github.com/tdewolff/canvas"
	canvasFont "github.com/tdewolff/canvas/font"
	canvasText "github.com/tdewolff/canvas/text"
)

type pdfWriter struct {
	w   io.Writer
	err error

	pos        int
	objOffsets []int

	fonts    map[*canvas.Font]pdfRef
	pages    []*pdfPageWriter
	compress bool
	subset   bool
	title    string
	subject  string
	keywords string
	author   string
}

func newPDFWriter(writer io.Writer) *pdfWriter {
	w := &pdfWriter{
		w:          writer,
		objOffsets: []int{0, 0, 0}, // catalog, metadata, page tree
		fonts:      map[*canvas.Font]pdfRef{},
		compress:   true,
		subset:     true,
	}

	w.write("%%PDF-1.7\n%%Ŧǟċơ\n")
	return w
}

func (w *pdfWriter) SetCompression(compress bool) {
	w.compress = compress
}

func (w *pdfWriter) SetFontSubsetting(subset bool) {
	w.subset = subset
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

func (w *pdfWriter) getFont(font *canvas.Font) pdfRef {
	if ref, ok := w.fonts[font]; ok {
		return ref
	}
	w.objOffsets = append(w.objOffsets, 0)
	ref := pdfRef(len(w.objOffsets))
	w.fonts[font] = ref
	return ref
}

func (w *pdfWriter) writeFonts() {
	for font, ref := range w.fonts {
		// subset the font
		var fontProgram []byte
		var glyphIDs []uint16 // actual to original glyph ID, identical if we're not subsetting
		if w.subset {
			fontProgram, glyphIDs = font.SFNT.Subset(font.SubsetIDs())
		} else {
			fontProgram = font.SFNT.Data
			glyphIDs = make([]uint16, font.SFNT.Maxp.NumGlyphs)
			for glyphID, _ := range glyphIDs {
				glyphIDs[glyphID] = uint16(glyphID)
			}
		}

		// calculate the character widths for the W array and shorten it
		f := 1000.0 / float64(font.SFNT.Head.UnitsPerEm)
		widths := make([]int, glyphIDs[len(glyphIDs)-1]+1)
		for _, glyphID := range glyphIDs {
			widths[glyphID] = int(f*float64(font.SFNT.GlyphAdvance(glyphID)) + 0.5)
		}
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

		// create ToUnicode CMap
		var bfRange, bfChar strings.Builder
		var bfRangeCount, bfCharCount int
		startGlyphID := uint16(0)
		startUnicode := uint32('\uFFFD')
		length := uint16(1)
		for _, glyphID := range glyphIDs[1:] {
			unicode := uint32(font.SFNT.Cmap.ToUnicode(glyphID))
			if 0x010000 <= unicode && unicode <= 0x10FFFF {
				// UTF-16 surrogates
				unicode -= 0x10000
				unicode = (0xD800+(unicode>>10)&0x3FF)<<16 + 0xDC00 + unicode&0x3FF
			}
			if glyphID == startGlyphID+length && unicode == startUnicode+uint32(length) {
				length++
			} else {
				if 1 < length {
					fmt.Fprintf(&bfRange, "<%04X> <%04X> <%04X>\n", startGlyphID, startGlyphID+length-1, startUnicode)
				} else {
					fmt.Fprintf(&bfChar, "<%04X> <%04X>\n", startGlyphID, startUnicode)
				}
				startGlyphID = glyphID
				startUnicode = unicode
				length = 1
			}
		}
		if 1 < length {
			fmt.Fprintf(&bfRange, "<%04X> <%04X> <%04X>\n", startGlyphID, startGlyphID+length-1, startUnicode)
		} else {
			fmt.Fprintf(&bfChar, "<%04X> <%04X>\n", startGlyphID, startUnicode)
		}

		toUnicode := fmt.Sprintf(`/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo
<< /Registry (Adobe)
   /Ordering (UCS)
   /Supplement 0
>> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
%d beginbfrange
%sendbfrange
%d beginbfchar
%sendbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end`, bfRangeCount, bfRange.String(), bfCharCount, bfChar.String())
		toUnicodeStream := pdfStream{
			dict:   pdfDict{},
			stream: []byte(toUnicode),
		}
		if w.compress {
			toUnicodeStream.dict["Filter"] = pdfFilterFlate
		}
		toUnicodeRef := w.writeObject(toUnicodeStream)

		// write font program
		fontfileRef := w.writeObject(pdfStream{
			dict: pdfDict{
				"Subtype": pdfName("OpenType"),
				"Filter":  pdfFilterFlate,
			},
			stream: fontProgram,
		})

		// get name and CID subtype
		name := font.Name()
		if records := font.SFNT.Name.Get(canvasFont.NamePostScript); 0 < len(records) {
			name = records[0].String()
		}
		baseFont := strings.ReplaceAll(name, " ", "")
		if w.subset {
			baseFont = "SUBSET+" + baseFont
		}

		cidSubtype := ""
		if font.SFNT.IsTrueType {
			cidSubtype = "CIDFontType2"
		} else if font.SFNT.IsCFF {
			cidSubtype = "CIDFontType0"
		}

		// in order to support more than 256 characters, we need to use a CIDFont dictionary which must be inside a Type0 font. Character codes in the stream are glyph IDs, however for subsetted fonts they are the _old_ glyph IDs, which is why we need the CIDToGIDMap
		dict := pdfDict{
			"Type":      pdfName("Font"),
			"Subtype":   pdfName("Type0"),
			"BaseFont":  pdfName(baseFont),
			"Encoding":  pdfName("Identity-H"), // map character codes in the stream to CID with identity encoding, we additionally map CID to GID in the descendant font when subsetting, otherwise that is also identity
			"ToUnicode": toUnicodeRef,
			"DescendantFonts": pdfArray{pdfDict{
				"Type":     pdfName("Font"),
				"Subtype":  pdfName(cidSubtype),
				"BaseFont": pdfName(baseFont),
				"DW":       DW,
				"W":        W,
				"CIDSystemInfo": pdfDict{
					"Registry":   "Adobe",
					"Ordering":   "Identity",
					"Supplement": 0,
				},
				"FontDescriptor": pdfDict{
					"Type":     pdfName("FontDescriptor"),
					"FontName": pdfName(baseFont),
					"Flags":    4, // Symbolic
					"FontBBox": pdfArray{
						int(f * float64(font.SFNT.Head.XMin)),
						int(f * float64(font.SFNT.Head.YMin)),
						int(f * float64(font.SFNT.Head.XMax)),
						int(f * float64(font.SFNT.Head.YMax)),
					},
					"ItalicAngle": float64(font.SFNT.Post.ItalicAngle),
					"Ascent":      int(f * float64(font.SFNT.Hhea.Ascender)),
					"Descent":     -int(f * float64(font.SFNT.Hhea.Descender)),
					"CapHeight":   int(f * float64(font.SFNT.OS2.SCapHeight)),
					"StemV":       80, // taken from Inkscape, should be calculated somehow, maybe use: 10+220*(usWeightClass-50)/900
					"StemH":       80, // idem
					"FontFile3":   fontfileRef,
				},
			}},
		}

		// add CIDToGIDMap as the characters use the original glyphIDs of the non-subsetted font
		if w.subset {
			cidToGIDMap := make([]byte, 2*glyphIDs[len(glyphIDs)-1]+2)
			for i, glyphID := range glyphIDs {
				j := int(glyphID) * 2
				cidToGIDMap[j+0] = byte((i & 0xFF00) >> 8)
				cidToGIDMap[j+1] = byte(i & 0x00FF)
			}
			cidToGIDMapStream := pdfStream{
				dict:   pdfDict{},
				stream: cidToGIDMap,
			}
			if w.compress {
				cidToGIDMapStream.dict["Filter"] = pdfFilterFlate
			}
			dict["DescendantFonts"].(pdfArray)[0].(pdfDict)["CIDToGIDMap"] = cidToGIDMapStream

			cidSetLength := (glyphIDs[len(glyphIDs)-1] + 1 + 7) / 8 // ceil of number of bytes
			cidSet := make([]byte, cidSetLength)
			for _, glyphID := range glyphIDs {
				i := glyphID / 8
				j := glyphID % 8
				cidSet[i] |= 0x80 >> j
			}
			cidSetStream := pdfStream{
				stream: cidSet,
			}
			dict["DescendantFonts"].(pdfArray)[0].(pdfDict)["CIDSet"] = cidSetStream
		}

		w.objOffsets[ref-1] = w.pos
		w.write("%v 0 obj\n", ref)
		w.writeVal(dict)
		w.write("\nendobj\n")
	}
}

func (w *pdfWriter) Close() error {
	// TODO: write pages directly to stream instead of using bytes.Buffer
	kids := pdfArray{}
	for _, p := range w.pages {
		kids = append(kids, p.writePage(pdfRef(3)))
	}

	w.writeFonts()

	// document catalog
	w.objOffsets[0] = w.pos
	w.write("%v 0 obj\n", 1)
	w.writeVal(pdfDict{
		"Type":  pdfName("Catalog"),
		"Pages": pdfRef(3),
	})
	w.write("\nendobj\n")

	// metadata
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

	w.objOffsets[1] = w.pos
	w.write("%v 0 obj\n", 2)
	w.writeVal(info)
	w.write("\nendobj\n")

	// page tree
	w.objOffsets[2] = w.pos
	w.write("%v 0 obj\n", 3)
	w.writeVal(pdfDict{
		"Type":  pdfName("Pages"),
		"Kids":  pdfArray(kids),
		"Count": len(kids),
	})
	w.write("\nendobj\n")

	xrefOffset := w.pos
	w.write("xref\n0 %d\n0000000000 65535 f \n", len(w.objOffsets)+1)
	for _, objOffset := range w.objOffsets {
		w.write("%010d 00000 n \n", objOffset)
	}
	w.write("trailer\n")
	w.writeVal(pdfDict{
		"Root": pdfRef(1),
		"Size": len(w.objOffsets) + 1,
		"Info": pdfRef(2),
		// TODO: write document ID
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
	font           *canvas.Font
	fontSize       float64
	inTextObject   bool
	textPosition   canvas.Matrix
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
		fillColor:      canvas.Black,
		strokeColor:    canvas.Black,
		lineWidth:      1.0,
		lineCap:        0,
		lineJoin:       0,
		miterLimit:     10.0,
		dashes:         []float64{0.0}, // dashArray and dashPhase
		font:           nil,
		fontSize:       0.0,
		inTextObject:   false,
		textPosition:   canvas.Identity,
		textCharSpace:  0.0,
		textRenderMode: 0,
	}
	w.pages = append(w.pages, page)

	m := canvas.Identity.Scale(ptPerMm, ptPerMm)
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

func (w *pdfPageWriter) SetLineCap(capper canvas.Capper) {
	var lineCap int
	if _, ok := capper.(canvas.ButtCapper); ok {
		lineCap = 0
	} else if _, ok := capper.(canvas.RoundCapper); ok {
		lineCap = 1
	} else if _, ok := capper.(canvas.SquareCapper); ok {
		lineCap = 2
	} else {
		panic("PDF: line cap not support")
	}
	if lineCap != w.lineCap {
		fmt.Fprintf(w, " %d J", lineCap)
		w.lineCap = lineCap
	}
}

func (w *pdfPageWriter) SetLineJoin(joiner canvas.Joiner) {
	var lineJoin int
	var miterLimit float64
	if _, ok := joiner.(canvas.BevelJoiner); ok {
		lineJoin = 2
	} else if _, ok := joiner.(canvas.RoundJoiner); ok {
		lineJoin = 1
	} else if miter, ok := joiner.(canvas.MiterJoiner); ok {
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

func (w *pdfPageWriter) SetFont(font *canvas.Font, size float64) {
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

func (w *pdfPageWriter) SetTextPosition(m canvas.Matrix) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if m.Equals(w.textPosition) {
		return
	}

	if canvas.Equal(m[0][0], w.textPosition[0][0]) && canvas.Equal(m[0][1], w.textPosition[0][1]) && canvas.Equal(m[1][0], w.textPosition[1][0]) && canvas.Equal(m[1][1], w.textPosition[1][1]) {
		d := w.textPosition.Inv().Dot(canvas.Point{m[0][2], m[1][2]})
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
	if !canvas.Equal(w.textCharSpace, space) {
		fmt.Fprintf(w, " %v Tc", dec(space))
		w.textCharSpace = space
	}
}

func (w *pdfPageWriter) StartTextObject() {
	if w.inTextObject {
		panic("already in text object")
	}
	fmt.Fprintf(w, " BT")
	w.textPosition = canvas.Identity
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

	first := true
	write := func(glyphs []canvasText.Glyph) {
		if first {
			fmt.Fprintf(w, "(")
			first = false
		} else {
			fmt.Fprintf(w, " (")
		}
		for _, glyph := range glyphs {
			w.font.Use(glyph.ID)
			if r := rune(glyph.ID); r == '\\' || r == '(' || r == ')' {
				binary.Write(w, binary.BigEndian, '\\')
			}
			binary.Write(w, binary.BigEndian, glyph.ID)
		}
		fmt.Fprintf(w, ")")
	}
	writeString := func(s string) {
		rs := []rune(s)
		glyphs := make([]canvasText.Glyph, len(rs))
		for i, r := range rs {
			glyphs[i].ID = w.font.SFNT.GlyphIndex(r)
		}
		write(glyphs)
	}

	f := 1000.0 / float64(w.font.SFNT.Head.UnitsPerEm)
	fmt.Fprintf(w, "[")
	for _, tj := range TJ {
		switch val := tj.(type) {
		case []canvasText.Glyph:
			i := 0
			for j, glyph := range val {
				origXAdvance := int32(w.font.SFNT.GlyphAdvance(glyph.ID))
				if glyph.XAdvance != origXAdvance {
					write(val[i : j+1])
					fmt.Fprintf(w, " %d", -int(f*float64(glyph.XAdvance-origXAdvance)+0.5))
					i = j + 1
				}
			}
			write(val[i:])
		case string:
			i := 0
			var rPrev rune
			for j, r := range val {
				if i < j {
					kern := w.font.SFNT.Kerning(w.font.SFNT.GlyphIndex(rPrev), w.font.SFNT.GlyphIndex(r))
					if kern != 0 {
						writeString(val[i:j])
						fmt.Fprintf(w, " %d", -int(f*float64(kern)+0.5))
						i = j
					}
				}
				rPrev = r
			}
			writeString(val[i:])
		case float64:
			fmt.Fprintf(w, " %d", -int(val*1000.0/w.fontSize+0.5))
		case int:
			fmt.Fprintf(w, " %d", -int(float64(val)*1000.0/w.fontSize+0.5))
		}
	}
	fmt.Fprintf(w, "]TJ")
}

func (w *pdfPageWriter) DrawImage(img image.Image, enc canvas.ImageEncoding, m canvas.Matrix) {
	size := img.Bounds().Size()

	// add clipping path around image for smooth edges when rotating
	outerRect := canvas.Rect{0.0, 0.0, float64(size.X), float64(size.Y)}.Transform(m)
	bl := m.Dot(canvas.Point{0, 0})
	br := m.Dot(canvas.Point{float64(size.X), 0})
	tl := m.Dot(canvas.Point{0, float64(size.Y)})
	tr := m.Dot(canvas.Point{float64(size.X), float64(size.Y)})
	fmt.Fprintf(w, " q %v %v %v %v re W n", dec(outerRect.X), dec(outerRect.Y), dec(outerRect.W), dec(outerRect.H))
	fmt.Fprintf(w, " %v %v m %v %v l %v %v l %v %v l h W n", dec(bl.X), dec(bl.Y), dec(tl.X), dec(tl.Y), dec(tr.X), dec(tr.Y), dec(br.X), dec(br.Y))

	name := w.embedImage(img, enc)
	m = m.Scale(float64(size.X), float64(size.Y))
	w.SetAlpha(1.0)
	fmt.Fprintf(w, " %v %v %v %v %v %v cm /%v Do Q", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]), name)
}

func (w *pdfPageWriter) embedImage(img image.Image, enc canvas.ImageEncoding) pdfName {
	size := img.Bounds().Size()
	sp := img.Bounds().Min // starting point
	b := make([]byte, size.X*size.Y*3)
	bMask := make([]byte, size.X*size.Y)
	hasMask := false
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 3
			R, G, B, A := img.At(sp.X+x, sp.Y+y).RGBA()
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
