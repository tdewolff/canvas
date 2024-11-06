package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/tdewolff/canvas"
	canvasText "github.com/tdewolff/canvas/text"
	canvasFont "github.com/tdewolff/font"
	"golang.org/x/text/encoding/charmap"
)

// TODO: Invalid graphics transparency, Group has a transparency S entry or the S entry is null
// TODO: Invalid Color space, The operator "g" can't be used without Color Profile

type pdfWriter struct {
	w   io.Writer
	err error

	pos        int
	objOffsets []int
	pages      []pdfRef

	page       *pdfPageWriter
	fontSubset map[*canvas.Font]*canvas.FontSubsetter
	fontsH     map[*canvas.Font]pdfRef
	fontsV     map[*canvas.Font]pdfRef
	fontsStd   map[*canvas.Font]pdfRef
	images     map[image.Image]pdfRef
	compress   bool
	subset     bool
	title      string
	subject    string
	keywords   string
	author     string
	creator    string
}

func newPDFWriter(writer io.Writer) *pdfWriter {
	w := &pdfWriter{
		w:          writer,
		objOffsets: []int{0, 0, 0}, // catalog, metadata, page tree
		fontSubset: map[*canvas.Font]*canvas.FontSubsetter{},
		fontsH:     map[*canvas.Font]pdfRef{},
		fontsV:     map[*canvas.Font]pdfRef{},
		fontsStd:   map[*canvas.Font]pdfRef{},
		images:     map[image.Image]pdfRef{},
		compress:   true,
		subset:     true,
	}

	w.write("%%PDF-1.7\n%%Ŧǟċơ\n")
	return w
}

// SetCompression enable the compression of the streams.
func (w *pdfWriter) SetCompression(compress bool) {
	w.compress = compress
}

// SeFontSubsetting enables the subsetting of embedded fonts.
func (w *pdfWriter) SetFontSubsetting(subset bool) {
	w.subset = subset
}

// SetTitle sets the document's title.
func (w *pdfWriter) SetTitle(title string) {
	w.title = title
}

// SetSubject sets the document's subject.
func (w *pdfWriter) SetSubject(subject string) {
	w.subject = subject
}

// SetKeywords sets the document's keywords.
func (w *pdfWriter) SetKeywords(keywords string) {
	w.keywords = keywords
}

// SetAuthor sets the document's author.
func (w *pdfWriter) SetAuthor(author string) {
	w.author = author
}

// SetCreator sets the document's creator.
func (w *pdfWriter) SetCreator(creator string) {
	w.creator = creator
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
	pdfFilterDCT     pdfFilter = "DCTDecode"
)

func pdfValContinuesName(val any) bool {
	switch val.(type) {
	case string, pdfName, pdfFilter, pdfArray, pdfDict, pdfStream:
		return false
	}
	return true
}

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
		w.write("<<")
		if val, ok := v["Type"]; ok {
			w.write("/Type")
			if pdfValContinuesName(val) {
				w.write(" ")
			}
			w.writeVal(val)
		}
		if val, ok := v["Subtype"]; ok {
			w.write("/Subtype")
			if pdfValContinuesName(val) {
				w.write(" ")
			}
			w.writeVal(val)
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
			if pdfValContinuesName(v[pdfName(key)]) {
				w.write(" ")
			}
			w.writeVal(v[pdfName(key)])
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
				fmt.Fprintf(&b2, "~>")
				b = b2.Bytes()
			case pdfFilterFlate:
				w := zlib.NewWriter(&b2)
				w.Write(b)
				w.Close()
				b = b2.Bytes()
			default:
				// assume already in the right format
			}
		}

		v.dict["Length"] = len(b)
		w.writeVal(v.dict)
		w.write("stream\n")
		w.writeBytes(b)
		w.write("\nendstream")
	default:
		panic(fmt.Sprintf("unknown PDF type %T", i))
	}
}

func (w *pdfWriter) writeObject(val interface{}) pdfRef {
	w.objOffsets = append(w.objOffsets, w.pos)
	w.write("%v 0 obj", len(w.objOffsets))
	w.writeVal(val)
	w.write("endobj\n")
	return pdfRef(len(w.objOffsets))
}

func standardFontName(font *canvas.Font) string {
	switch strings.ToLower(font.Name()) {
	case "courier":
		if font.Style() == canvas.FontRegular {
			return "Courier"
		} else if font.Style() == canvas.FontBold {
			return "Courier-Bold"
		} else if font.Style() == canvas.FontItalic {
			return "Courier-Oblique"
		} else if font.Style() == canvas.FontBold|canvas.FontItalic {
			return "Courier-BoldOblique"
		}
	case "dingbats":
		if font.Style() == canvas.FontRegular {
			return "ZapfDingbats"
		}
	case "helvetica":
		if font.Style() == canvas.FontRegular {
			return "Helvetica"
		} else if font.Style() == canvas.FontBold {
			return "Helvetica-Bold"
		} else if font.Style() == canvas.FontItalic {
			return "Helvetica-Oblique"
		} else if font.Style() == canvas.FontBold|canvas.FontItalic {
			return "Helvetica-BoldOblique"
		}
	case "symbol":
		if font.Style() == canvas.FontRegular {
			return "Symbol"
		}
	case "times":
		if font.Style() == canvas.FontRegular {
			return "Times-Roman"
		} else if font.Style() == canvas.FontBold {
			return "Times-Bold"
		} else if font.Style() == canvas.FontItalic {
			return "Times-Italic"
		} else if font.Style() == canvas.FontBold|canvas.FontItalic {
			return "Times-BoldItalic"
		}
	}
	return ""
}

func (w *pdfWriter) getFont(font *canvas.Font, vertical bool) pdfRef {
	if standardFont := standardFontName(font); standardFont != "" {
		// handle 14 embedded standard fonts in PDF if name and style matches
		if ref, ok := w.fontsStd[font]; ok {
			return ref
		}

		dict := pdfDict{
			"Type":     pdfName("Font"),
			"Subtype":  pdfName("Type1"),
			"BaseFont": pdfName(standardFont),
			"Encoding": pdfName("WinAnsiEncoding"),
		}
		ref := w.writeObject(dict)
		w.fontsStd[font] = ref
		// don't set fontSubset
		return ref
	}

	fonts := w.fontsH
	if vertical {
		fonts = w.fontsV
	}
	if ref, ok := fonts[font]; ok {
		return ref
	}
	w.objOffsets = append(w.objOffsets, 0)
	ref := pdfRef(len(w.objOffsets))
	fonts[font] = ref
	w.fontSubset[font] = canvas.NewFontSubsetter()
	return ref
}

func (w *pdfWriter) writeFont(ref pdfRef, font *canvas.Font, vertical bool) {
	// subset the font, we only write the used characters to the PDF CMap object to reduce its
	// length. At the end of the function we add a CID to GID mapping to correctly select the
	// right glyphID.
	sfnt := font.SFNT
	glyphIDs := w.fontSubset[font].List() // also when not subsetting, to minimize cmap table
	if w.subset {
		if sfnt.IsCFF && sfnt.CFF != nil {
			sfnt.CFF.SetGlyphNames(nil)
		}
		sfntSubset, err := sfnt.Subset(glyphIDs, canvasFont.SubsetOptions{Tables: canvasFont.KeepPDFTables})
		if err == nil {
			sfnt = sfntSubset
		} else {
			fmt.Println("WARNING: font subsetting failed:", err)
		}
	}
	fontProgram := sfnt.Write()

	// calculate the character widths for the W array and shorten it
	f := 1000.0 / float64(font.SFNT.Head.UnitsPerEm)
	widths := make([]int, len(glyphIDs)+1)
	for subsetGlyphID, glyphID := range glyphIDs {
		widths[subsetGlyphID] = int(f*float64(font.SFNT.GlyphAdvance(glyphID)) + 0.5)
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
	for subsetGlyphID, glyphID := range glyphIDs[1:] {
		unicode := uint32(font.SFNT.Cmap.ToUnicode(glyphID))
		if 0x010000 <= unicode && unicode <= 0x10FFFF {
			// UTF-16 surrogates
			unicode -= 0x10000
			unicode = (0xD800+(unicode>>10)&0x3FF)<<16 + 0xDC00 + unicode&0x3FF
		}
		if uint16(subsetGlyphID+1) == startGlyphID+length && unicode == startUnicode+uint32(length) {
			length++
		} else {
			if 1 < length {
				fmt.Fprintf(&bfRange, "<%04X> <%04X> <%04X>\n", startGlyphID, startGlyphID+length-1, startUnicode)
			} else {
				fmt.Fprintf(&bfChar, "<%04X> <%04X>\n", startGlyphID, startUnicode)
			}
			startGlyphID = uint16(subsetGlyphID + 1)
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
		baseFont = "SUBSET+" + baseFont // TODO: give unique subset name
	}

	encoding := "Identity-H"
	if vertical {
		encoding = "Identity-V"
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
		"Encoding":  pdfName(encoding), // map character codes in the stream to CID with identity encoding, we additionally map CID to GID in the descendant font when subsetting, otherwise that is also identity
		"ToUnicode": toUnicodeRef,
		"DescendantFonts": pdfArray{pdfDict{
			"Type":        pdfName("Font"),
			"Subtype":     pdfName(cidSubtype),
			"BaseFont":    pdfName(baseFont),
			"DW":          DW,
			"W":           W,
			"CIDToGIDMap": pdfName("Identity"),
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
				"FontFile3":   fontfileRef,
			},
		}},
	}

	if !w.subset {
		cidToGIDMap := make([]byte, 2*len(glyphIDs))
		for subsetGlyphID, glyphID := range glyphIDs {
			j := int(subsetGlyphID) * 2
			cidToGIDMap[j+0] = byte((glyphID & 0xFF00) >> 8)
			cidToGIDMap[j+1] = byte(glyphID & 0x00FF)
		}
		cidToGIDMapStream := pdfStream{
			dict:   pdfDict{},
			stream: cidToGIDMap,
		}
		if w.compress {
			cidToGIDMapStream.dict["Filter"] = pdfFilterFlate
		}
		cidToGIDMapRef := w.writeObject(cidToGIDMapStream)
		dict["DescendantFonts"].(pdfArray)[0].(pdfDict)["CIDToGIDMap"] = cidToGIDMapRef
	}

	w.objOffsets[ref-1] = w.pos
	w.write("%v 0 obj", ref)
	w.writeVal(dict)
	w.write("endobj\n")
}

func (w *pdfWriter) writeFonts(fontMap map[*canvas.Font]pdfRef, vertical bool) {
	// sort fonts by ref to make PDF deterministic
	refs := make([]pdfRef, 0, len(fontMap))
	refMap := make(map[pdfRef]*canvas.Font, len(fontMap))
	for font, ref := range fontMap {
		refs = append(refs, ref)
		refMap[ref] = font
	}
	sort.Slice(refs, func(i, j int) bool {
		return refs[i] < refs[j]
	})
	for _, ref := range refs {
		w.writeFont(ref, refMap[ref], vertical)
	}
}

// Close finished the document.
func (w *pdfWriter) Close() error {
	// TODO: support cross reference table streams and compressed objects for all dicts
	if w.page != nil {
		w.pages = append(w.pages, w.page.writePage(pdfRef(3)))
	}

	kids := pdfArray{}
	for _, page := range w.pages {
		kids = append(kids, page)
	}

	// write fonts
	w.writeFonts(w.fontsH, false)
	w.writeFonts(w.fontsV, false)

	// document catalog
	w.objOffsets[0] = w.pos
	w.write("%v 0 obj", 1)
	w.writeVal(pdfDict{
		"Type":  pdfName("Catalog"),
		"Pages": pdfRef(3),
		// TODO: add metadata?
	})
	w.write("endobj\n")

	// metadata
	info := pdfDict{
		"Producer":     "tdewolff/canvas",
		"CreationDate": time.Now().Format("D:20060102150405Z0700"),
	}

	encode := func(s string) string {
		// TODO: make clean
		ascii := true
		for _, r := range s {
			if 0x80 <= r {
				ascii = false
				break
			}
		}
		if ascii {
			return s
		}

		rs := utf16.Encode([]rune(s))
		b := make([]byte, 2+2*len(rs))
		b[0] = 254
		b[1] = 255
		for i, r := range rs {
			b[2+2*i+0] = byte(r >> 8)
			b[2+2*i+1] = byte(r & 0x00FF)
		}
		return string(b)
	}
	if w.title != "" {
		info["Title"] = encode(w.title)
	}
	if w.subject != "" {
		info["Subject"] = encode(w.subject)
	}
	if w.keywords != "" {
		info["Keywords"] = encode(w.keywords)
	}
	if w.author != "" {
		info["Author"] = encode(w.author)
	}
	if w.creator != "" {
		info["Creator"] = encode(w.creator)
	}

	w.objOffsets[1] = w.pos
	w.write("%v 0 obj", 2)
	w.writeVal(info)
	w.write("endobj\n")

	// page tree
	w.objOffsets[2] = w.pos
	w.write("%v 0 obj", 3)
	w.writeVal(pdfDict{
		"Type":  pdfName("Pages"),
		"Kids":  pdfArray(kids),
		"Count": len(kids),
	})
	w.write("endobj\n")

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
	w.write("\nstartxref\n%v\n%%%%EOF\n", xrefOffset)
	return w.err
}

type pdfPageWriter struct {
	*bytes.Buffer
	pdf           *pdfWriter
	width, height float64
	resources     pdfDict
	annots        pdfArray

	graphicsStates map[float64]pdfName
	alpha          float64
	fill           canvas.Paint
	stroke         canvas.Paint
	lineWidth      float64
	lineCap        int
	lineJoin       int
	miterLimit     float64
	dashes         []float64
	font           *canvas.Font
	fontSize       float64
	fontDirection  canvasText.Direction
	inTextObject   bool
	textPosition   canvas.Matrix
	textCharSpace  float64
	textRenderMode int
}

// NewPage starts a new page.
func (w *pdfWriter) NewPage(width, height float64) *pdfPageWriter {
	if w.page != nil {
		w.pages = append(w.pages, w.page.writePage(pdfRef(3)))
	}

	// for defaults see https://help.adobe.com/pdfl_sdk/15/PDFL_SDK_HTMLHelp/PDFL_SDK_HTMLHelp/API_References/PDFL_API_Reference/PDFEdit_Layer/General.html#_t_PDEGraphicState
	w.page = &pdfPageWriter{
		Buffer:         &bytes.Buffer{},
		pdf:            w,
		width:          width,
		height:         height,
		resources:      pdfDict{},
		graphicsStates: map[float64]pdfName{},
		alpha:          1.0,
		fill:           canvas.Paint{Color: canvas.Black},
		stroke:         canvas.Paint{Color: canvas.Black},
		lineWidth:      1.0,
		lineCap:        0,
		lineJoin:       0,
		miterLimit:     10.0,
		dashes:         []float64{0.0}, // dashArray and dashPhase
		font:           nil,
		fontSize:       0.0,
		fontDirection:  canvasText.LeftToRight,
		inTextObject:   false,
		textPosition:   canvas.Identity,
		textCharSpace:  0.0,
		textRenderMode: 0,
	}

	m := canvas.Identity.Scale(ptPerMm, ptPerMm)
	fmt.Fprintf(w.page, " %v %v %v %v %v %v cm", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]))
	return w.page
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
	page := pdfDict{
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
	}
	if 0 < len(w.annots) {
		page["Annots"] = w.annots
	}
	return w.pdf.writeObject(page)
}

// AddAnnotation adds an annotation.
func (w *pdfPageWriter) AddURIAction(uri string, rect canvas.Rect) {
	annot := pdfDict{
		"Type":     pdfName("Annot"),
		"Subtype":  pdfName("Link"),
		"Border":   pdfArray{0, 0, 0},
		"Rect":     pdfArray{rect.X0 * ptPerMm, rect.Y0 * ptPerMm, rect.X1 * ptPerMm, rect.Y1 * ptPerMm},
		"Contents": uri,
		"A": pdfDict{
			"S":   pdfName("URI"),
			"URI": uri,
		},
	}
	w.annots = append(w.annots, annot)
}

// SetAlpha sets the transparency value.
func (w *pdfPageWriter) SetAlpha(alpha float64) {
	if alpha != w.alpha {
		gs := w.getOpacityGS(alpha)
		fmt.Fprintf(w, " /%v gs", gs)
		w.alpha = alpha
	}
}

// SetFill sets the filling paint.
func (w *pdfPageWriter) SetFill(fill canvas.Paint) {
	if fill.Equal(w.fill) {
		return
	}
	if fill.IsPattern() {
		// TODO
	} else if fill.IsGradient() {
		// TODO: should we unset cs?
		fmt.Fprintf(w, " /Pattern cs /%v scn", w.getPattern(fill.Gradient))
	} else {
		a := float64(fill.Color.A) / 255.0
		if fill.Color.R == fill.Color.G && fill.Color.R == fill.Color.B {
			fmt.Fprintf(w, " %v g", dec(float64(fill.Color.R)/255.0/a))
		} else {
			fmt.Fprintf(w, " %v %v %v rg", dec(float64(fill.Color.R)/255.0/a), dec(float64(fill.Color.G)/255.0/a), dec(float64(fill.Color.B)/255.0/a))
		}
		w.SetAlpha(a)
	}
	w.fill = fill
}

// SetStroke sets the stroking paint.
func (w *pdfPageWriter) SetStroke(stroke canvas.Paint) {
	if stroke.Equal(w.stroke) {
		return
	}
	if stroke.IsPattern() {
		// TODO
	} else if stroke.IsGradient() {
		// TODO: should we unset CS?
		fmt.Fprintf(w, " /Pattern CS /%v SCN", w.getPattern(stroke.Gradient))
	} else {
		a := float64(stroke.Color.A) / 255.0
		if stroke.Color.R == stroke.Color.G && stroke.Color.R == stroke.Color.B {
			fmt.Fprintf(w, " %v G", dec(float64(stroke.Color.R)/255.0/a))
		} else {
			fmt.Fprintf(w, " %v %v %v RG", dec(float64(stroke.Color.R)/255.0/a), dec(float64(stroke.Color.G)/255.0/a), dec(float64(stroke.Color.B)/255.0/a))
		}
		w.SetAlpha(a)
	}
	w.stroke = stroke
}

// SetLineWidth sets the stroke width.
func (w *pdfPageWriter) SetLineWidth(lineWidth float64) {
	if lineWidth != w.lineWidth {
		fmt.Fprintf(w, " %v w", dec(lineWidth))
		w.lineWidth = lineWidth
	}
}

// SetLineCap sets the stroke cap type.
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

// SetLineJoin sets the stroke join type.
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

// SetDashes sets the dash phase and array.
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

// SetFont sets the font.
func (w *pdfPageWriter) SetFont(font *canvas.Font, size float64, direction canvasText.Direction) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if font != w.font || w.fontSize != size || w.fontDirection != direction {
		w.font = font
		w.fontSize = size
		w.fontDirection = direction

		vertical := direction == canvasText.TopToBottom || direction == canvasText.BottomToTop
		ref := w.pdf.getFont(font, vertical)
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

// SetTextPosition sets the text position.
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

// SetTextRenderMode sets the text rendering mode.
func (w *pdfPageWriter) SetTextRenderMode(mode int) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if w.textRenderMode != mode {
		fmt.Fprintf(w, " %d Tr", mode)
		w.textRenderMode = mode
	}
}

// SetTextCharSpace sets the text character spacing.
func (w *pdfPageWriter) SetTextCharSpace(space float64) {
	if !w.inTextObject {
		panic("must be in text object")
	}
	if !canvas.Equal(w.textCharSpace, space) {
		fmt.Fprintf(w, " %v Tc", dec(space))
		w.textCharSpace = space
	}
}

// StartTextObject starts a text object.
func (w *pdfPageWriter) StartTextObject() {
	if w.inTextObject {
		panic("already in text object")
	}
	fmt.Fprintf(w, " BT")
	w.textPosition = canvas.Identity
	w.inTextObject = true
}

// EndTextObject ends a text object.
func (w *pdfPageWriter) EndTextObject() {
	if !w.inTextObject {
		panic("must be in text object")
	}
	fmt.Fprintf(w, " ET")
	w.inTextObject = false
}

// WriteText writes text using a writing mode and a list of strings and inter-character distance modifiers (ints or float64s).
func (w *pdfPageWriter) WriteText(mode canvas.WritingMode, TJ ...interface{}) {
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
		subset := w.pdf.fontSubset[w.font]
		if subset == nil {
			for _, glyph := range glyphs {
				c, ok := charmap.Windows1252.EncodeRune(glyph.Text)
				if !ok {
					if '\u2000' <= glyph.Text && glyph.Text <= '\u200A' {
						c = ' '
					}
				}
				if c == '\n' {
					w.WriteByte('\\')
					w.WriteByte('n')
				} else if c == '\r' {
					w.WriteByte('\\')
					w.WriteByte('r')
				} else if c == '\t' {
					w.WriteByte('\\')
					w.WriteByte('t')
				} else if c == '\b' {
					w.WriteByte('\\')
					w.WriteByte('b')
				} else if c == '\f' {
					w.WriteByte('\\')
					w.WriteByte('f')
				} else if c == '\\' || c == '(' || c == ')' {
					w.WriteByte('\\')
					w.WriteByte(c)
				} else {
					w.WriteByte(c)
				}
			}
		} else {
			for _, glyph := range glyphs {
				glyphID := subset.Get(glyph.ID)
				for _, c := range []uint8{uint8((glyphID & 0xff00) >> 8), uint8(glyphID & 0x00ff)} {
					if c == '\n' {
						w.WriteByte('\\')
						w.WriteByte('n')
					} else if c == '\r' {
						w.WriteByte('\\')
						w.WriteByte('r')
					} else if c == '\t' {
						w.WriteByte('\\')
						w.WriteByte('t')
					} else if c == '\b' {
						w.WriteByte('\\')
						w.WriteByte('b')
					} else if c == '\f' {
						w.WriteByte('\\')
						w.WriteByte('f')
					} else if c == '\\' || c == '(' || c == ')' {
						w.WriteByte('\\')
						w.WriteByte(c)
					} else {
						w.WriteByte(c)
					}
				}
			}
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

	position := w.textPosition
	if glyphs, ok := TJ[0].([]canvasText.Glyph); ok && 0 < len(glyphs) && mode != canvas.HorizontalTB && !glyphs[0].Vertical {
		glyphRotation, glyphOffset := glyphs[0].Rotation(), glyphs[0].YOffset-int32(glyphs[0].SFNT.Head.UnitsPerEm/2)
		if glyphRotation != canvasText.NoRotation || glyphOffset != 0 {
			w.SetTextPosition(position.Rotate(float64(glyphRotation)).Translate(0.0, glyphs[0].Size/float64(glyphs[0].SFNT.Head.UnitsPerEm)*mmPerPt*float64(glyphOffset)))
		}
	}

	f := 1000.0 / float64(w.font.SFNT.Head.UnitsPerEm)
	fmt.Fprintf(w, "[")
	for _, tj := range TJ {
		switch val := tj.(type) {
		case []canvasText.Glyph:
			i := 0
			for j, glyph := range val {
				if mode == canvas.HorizontalTB || !glyph.Vertical {
					origXAdvance := int32(w.font.SFNT.GlyphAdvance(glyph.ID))
					if glyph.XAdvance != origXAdvance {
						write(val[i : j+1])
						fmt.Fprintf(w, " %d", -int(f*float64(glyph.XAdvance-origXAdvance)+0.5))
						i = j + 1
					}
				} else {
					origYAdvance := -int32(w.font.SFNT.GlyphVerticalAdvance(glyph.ID))
					if glyph.YAdvance != origYAdvance {
						write(val[i : j+1])
						fmt.Fprintf(w, " %d", -int(f*float64(glyph.YAdvance-origYAdvance)+0.5))
						i = j + 1
					}
				}
			}
			write(val[i:])
		case string:
			i := 0
			if mode == canvas.HorizontalTB {
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

// DrawImage embeds and draws an image.
func (w *pdfPageWriter) DrawImage(img image.Image, enc canvas.ImageEncoding, m canvas.Matrix) {
	size := img.Bounds().Size()

	// add clipping path around image for smooth edges when rotating
	outerRect := canvas.Rect{0.0, 0.0, float64(size.X), float64(size.Y)}.Transform(m)
	bl := m.Dot(canvas.Point{0, 0})
	br := m.Dot(canvas.Point{float64(size.X), 0})
	tl := m.Dot(canvas.Point{0, float64(size.Y)})
	tr := m.Dot(canvas.Point{float64(size.X), float64(size.Y)})
	fmt.Fprintf(w, " q %v %v %v %v re W n", dec(outerRect.X0), dec(outerRect.Y0), dec(outerRect.W()), dec(outerRect.H()))
	fmt.Fprintf(w, " %v %v m %v %v l %v %v l %v %v l h W n", dec(bl.X), dec(bl.Y), dec(tl.X), dec(tl.Y), dec(tr.X), dec(tr.Y), dec(br.X), dec(br.Y))

	ref := w.embedImage(img, enc)
	if _, ok := w.resources["XObject"]; !ok {
		w.resources["XObject"] = pdfDict{}
	}
	name := pdfName(fmt.Sprintf("Im%d", len(w.resources["XObject"].(pdfDict))))
	w.resources["XObject"].(pdfDict)[name] = ref

	m = m.Scale(float64(size.X), float64(size.Y))
	w.SetAlpha(1.0)
	fmt.Fprintf(w, " %v %v %v %v %v %v cm /%v Do Q", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]), name)
}

func (w *pdfPageWriter) embedImage(img image.Image, enc canvas.ImageEncoding) pdfRef {
	if ref, ok := w.pdf.images[img]; ok {
		return ref
	}

	var filter pdfFilter
	var stream []byte
	var streamMask []byte
	var hasMask bool

	size := img.Bounds().Size()
	if enc == canvas.Lossy {
		filter = pdfFilterDCT
		sp := img.Bounds().Min // starting point
		streamMask = make([]byte, size.X*size.Y)
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				_, _, _, A := img.At(sp.X+x, sp.Y+y).RGBA()
				if A != 0 {
					streamMask[y*size.X+x] = byte(A >> 8)
				}
				if A>>8 != 255 {
					hasMask = true
				}
			}
		}

		var buf bytes.Buffer
		_ = jpeg.Encode(&buf, img, nil)
		stream = buf.Bytes()
	} else {
		filter = pdfFilterFlate
		sp := img.Bounds().Min // starting point
		stream = make([]byte, size.X*size.Y*3)
		streamMask = make([]byte, size.X*size.Y)
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				i := (y*size.X + x) * 3
				R, G, B, A := img.At(sp.X+x, sp.Y+y).RGBA()
				if A != 0 {
					stream[i+0] = byte((R * 65535 / A) >> 8)
					stream[i+1] = byte((G * 65535 / A) >> 8)
					stream[i+2] = byte((B * 65535 / A) >> 8)
					streamMask[y*size.X+x] = byte(A >> 8)
				}
				if A>>8 != 255 {
					hasMask = true
				}
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
		"Filter":           filter,
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
			stream: streamMask,
		})
	}

	ref := w.pdf.writeObject(pdfStream{
		dict:   dict,
		stream: stream,
	})
	w.pdf.images[img] = ref
	return ref
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

func (w *pdfPageWriter) getPattern(gradient canvas.Gradient) pdfName {
	// TODO: support patterns/gradients with alpha channel
	shading := pdfDict{
		"ColorSpace": pdfName("DeviceRGB"),
	}
	if g, ok := gradient.(*canvas.LinearGradient); ok {
		shading["ShadingType"] = 2
		shading["Coords"] = pdfArray{g.Start.X * ptPerMm, g.Start.Y * ptPerMm, g.End.X * ptPerMm, g.End.Y * ptPerMm}
		shading["Function"] = patternStopsFunction(g.Stops)
		shading["Extend"] = pdfArray{true, true}
	} else if g, ok := gradient.(*canvas.RadialGradient); ok {
		shading["ShadingType"] = 3
		shading["Coords"] = pdfArray{g.C0.X * ptPerMm, g.C0.Y * ptPerMm, g.R0 * ptPerMm, g.C1.X * ptPerMm, g.C1.Y * ptPerMm, g.R1 * ptPerMm}
		shading["Function"] = patternStopsFunction(g.Stops)
		shading["Extend"] = pdfArray{true, true}
	}
	pattern := pdfDict{
		"Type":        pdfName("Pattern"),
		"PatternType": 2,
		"Shading":     shading,
	}

	if _, ok := w.resources["Pattern"]; !ok {
		w.resources["Pattern"] = pdfDict{}
	}
	for name, pat := range w.resources["Pattern"].(pdfDict) {
		if reflect.DeepEqual(pat, pattern) {
			return name
		}
	}
	name := pdfName(fmt.Sprintf("P%d", len(w.resources["Pattern"].(pdfDict))))
	w.resources["Pattern"].(pdfDict)[name] = pattern
	return name
}

func patternStopsFunction(stops canvas.Stops) pdfDict {
	if len(stops) < 2 {
		return pdfDict{}
	}

	fs := []pdfDict{}
	encode := pdfArray{}
	bounds := pdfArray{}
	if !canvas.Equal(stops[0].Offset, 0.0) {
		fs = append(fs, patternStopFunction(stops[0], stops[0]))
		encode = append(encode, 0, 1)
		bounds = append(bounds, stops[0].Offset)
	}
	for i := 0; i < len(stops)-1; i++ {
		fs = append(fs, patternStopFunction(stops[i], stops[i+1]))
		encode = append(encode, 0, 1)
		if i != 0 {
			bounds = append(bounds, stops[1].Offset)
		}
	}
	if !canvas.Equal(stops[len(stops)-1].Offset, 1.0) {
		fs = append(fs, patternStopFunction(stops[len(stops)-1], stops[len(stops)-1]))
		encode = append(encode, 0, 1)
	}
	if len(fs) == 1 {
		return fs[0]
	}
	return pdfDict{
		"FunctionType": 3,
		"Domain":       pdfArray{0, 1},
		"Encode":       encode,
		"Bounds":       bounds,
		"Functions":    fs,
	}
}

func patternStopFunction(s0, s1 canvas.Stop) pdfDict {
	a0 := float64(s0.Color.A) / 255.0
	a1 := float64(s1.Color.A) / 255.0
	return pdfDict{
		"FunctionType": 2,
		"Domain":       pdfArray{0, 1},
		"N":            1,
		"C0":           pdfArray{float64(s0.Color.R) / 255.0 / a0, float64(s0.Color.G) / 255.0 / a0, float64(s0.Color.B) / 255.0 / a0},
		"C1":           pdfArray{float64(s1.Color.R) / 255.0 / a1, float64(s1.Color.G) / 255.0 / a1, float64(s1.Color.B) / 255.0 / a1},
	}
}
