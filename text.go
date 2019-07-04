package canvas

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math"
	"unicode"
	"unicode/utf8"
)

const MaxSentenceSpacing = 3.0 // times width of space
const MaxWordSpacing = 2.5     // times width of space
const MaxGlyphSpacing = 0.5    // times x-height

// TextAlign specifies how the text should align or whether it should be justified.
type TextAlign int

const (
	Left TextAlign = iota
	Right
	Center
	Top
	Bottom
	Justify
)

type line struct {
	spans []textSpan
	decos []decoSpan
	y     float64
}

func (l line) Heights() (float64, float64, float64, float64) {
	top, ascent, descent, bottom := 0.0, 0.0, 0.0, 0.0
	for _, s := range l.spans {
		spanAscent, spanDescent, lineSpacing := s.Heights()
		top = math.Max(top, spanAscent+lineSpacing)
		ascent = math.Max(ascent, spanAscent)
		descent = math.Max(descent, spanDescent)
		bottom = math.Max(bottom, spanDescent+lineSpacing)
	}
	return top, ascent, descent, bottom
}

////////////////////////////////////////////////////////////////

// Text holds the representation of text using lines and text spans.
type Text struct {
	lines []line
	fonts map[*Font]bool
}

func NewTextLine(ff FontFace, s string, halign TextAlign) *Text {
	// TODO: use halign
	s, _, _ = ff.font.substituteTypography(s, false, false)
	ss := splitNewlines(s)
	y := 0.0
	lines := []line{}
	for _, s := range ss {
		spans := []textSpan{newTextSpan(ff, s)}
		decos := []decoSpan{}
		if len(ff.deco) != 0 {
			decos = append(decos, decoSpan{ff, 0.0, ff.TextWidth(s)})
		}
		lines = append(lines, line{spans, decos, y})

		ascent, descent, spacing := spans[0].Heights()
		y -= spacing + ascent + descent + spacing
	}
	return &Text{lines, map[*Font]bool{ff.font: true}}
}

func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	return NewRichText().Add(ff, s).ToText(width, height, halign, valign, indent, lineStretch)
}

// RichText allows to build up a rich text with text spans of different font faces and by fitting that into a box.
type RichText struct {
	spans                        []textSpan
	fonts                        map[*Font]bool
	inSingleQuote, inDoubleQuote bool
}

func NewRichText() *RichText {
	// TODO: allow for default font and use font face modifiers (color, style, faux style, decorations)
	return &RichText{
		fonts: map[*Font]bool{},
	}
}

// Add adds a new text span element.
func (rt *RichText) Add(ff FontFace, s string) *RichText {
	s, rt.inSingleQuote, rt.inDoubleQuote = ff.font.substituteTypography(s, rt.inSingleQuote, rt.inDoubleQuote)

	i := 0
	boundaries := calcTextBoundaries(s, 0, len(s))
	for _, boundary := range boundaries {
		if boundary.kind == lineBoundary || boundary.kind == sentenceBoundary {
			j := boundary.pos + boundary.size
			rt.spans = append(rt.spans, newTextSpan(ff, s[i:j]))
			i = j
		}
	}
	if i < len(s) {
		rt.spans = append(rt.spans, newTextSpan(ff, s[i:]))
	}
	rt.fonts[ff.font] = true
	return rt
}

// ToText takes the added text spans and fits them within a given box of certain width and height.
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	if len(rt.spans) == 0 {
		return &Text{[]line{}, rt.fonts}
	}
	spans := []textSpan{rt.spans[0]}

	k := 0 // index into rt.spans and rt.positions
	lines := []line{}
	kSpans := [][]int{} // indices (k) per line, per span
	//iBoundary := 0
	y, prevLineSpacing := 0.0, 0.0
	for k < len(rt.spans) {
		dx := indent
		indent = 0.0

		// trim left spaces
		spans[0] = spans[0].TrimLeft()
		for spans[0].text == "" {
			if k+1 == len(rt.spans) {
				break
			}
			k++
			spans = []textSpan{rt.spans[k]}
			spans[0] = spans[0].TrimLeft()
		}

		// accumulate line spans for a full line, ie. either split span1 to fit or if it fits retrieve the next span1 and repeat
		ss := []textSpan{}
		kSpans = append(kSpans, []int{})
		for {
			// space or inter-word splitting
			var ok bool
			if width == 0.0 {
				spans, ok = spans[0].Split(0.0)
			} else {
				spans, ok = spans[0].Split(width - dx)
			}
			if !ok && len(ss) != 0 {
				// span couln't fit, but we have no choice as it's the only span on the line
				break
			}

			spans[0].dx = dx
			spans[0].w = spans[0].textWidth
			ss = append(ss, spans[0])
			kSpans[len(lines)] = append(kSpans[len(lines)], k)
			dx += spans[0].textWidth

			spans = spans[1:]
			if len(spans) == 0 {
				k++
				if k == len(rt.spans) {
					break
				}
				spans = []textSpan{rt.spans[k]}
			} else {
				break // span couldn't fully fit, we have a full line
			}
		}

		// trim right spaces
		for 0 < len(ss) {
			ss[len(ss)-1] = ss[len(ss)-1].TrimRight()
			if ss[len(ss)-1].text == "" {
				ss = ss[:len(ss)-1]
			} else {
				break
			}
		}

		l := line{ss, []decoSpan{}, 0.0}
		top, ascent, descent, bottom := l.Heights()
		lineSpacing := math.Max(top-ascent, prevLineSpacing)
		if len(lines) != 0 {
			y -= lineSpacing * (1.0 + lineStretch)
			y -= ascent * lineStretch
		}
		y -= ascent
		l.y = y
		y -= descent * (1.0 + lineStretch)
		prevLineSpacing = bottom - descent

		if height != 0.0 && y < -height {
			break
		}
		lines = append(lines, l)
	}

	if len(lines) == 0 {
		return &Text{lines, rt.fonts}
	}

	// apply horizontal alignment
	if halign == Right || halign == Center {
		for _, l := range lines {
			firstSpan := l.spans[0]
			lastSpan := l.spans[len(l.spans)-1]
			dx := width - lastSpan.dx - lastSpan.w - firstSpan.dx
			if halign == Center {
				dx /= 2.0
			}
			for i := range l.spans {
				l.spans[i].dx += dx
			}
		}
	} else if 0.0 < width && halign == Justify {
		for _, l := range lines[:len(lines)-1] {
			// get the width range of our spans (eg. for text width can increase with extra character spacing)
			textWidth, maxSentenceSpacing, maxWordSpacing, maxGlyphSpacing := 0.0, 0.0, 0.0, 0.0
			origDiff := 0.0 // difference between textWidth with ligatures and without
			for i, span := range l.spans {
				sentences, words := 0, 0
				for _, boundary := range span.boundaries {
					if boundary.kind == sentenceBoundary {
						sentences++
					} else if boundary.kind == wordBoundary {
						words++
					}
				}
				glyphs := utf8.RuneCountInString(span.origText)

				textWidth += span.textWidth
				if i == 0 {
					textWidth += span.dx
				}
				origDiff += span.origTextWidth - span.textWidth

				xHeight := span.ff.Metrics().XHeight
				maxSentenceSpacing += float64(sentences) * MaxSentenceSpacing * xHeight
				maxWordSpacing += float64(words) * MaxWordSpacing * xHeight
				maxGlyphSpacing += float64(glyphs) * MaxGlyphSpacing * xHeight
			}

			// only expand if we can reach the line width
			if textWidth < width && width < textWidth+maxSentenceSpacing+maxWordSpacing+maxGlyphSpacing {
				widthLeft := width - textWidth
				sentenceFactor, wordFactor, glyphFactor := 0.0, 0.0, 0.0
				if Epsilon < widthLeft && maxSentenceSpacing > 0 {
					sentenceFactor = math.Min(widthLeft/maxSentenceSpacing, 1.0)
					widthLeft -= sentenceFactor * maxSentenceSpacing
				}
				if Epsilon < widthLeft && maxWordSpacing > 0 {
					wordFactor = math.Min(widthLeft/maxWordSpacing, 1.0)
					widthLeft -= wordFactor * maxWordSpacing
				}
				if Epsilon < widthLeft && maxGlyphSpacing > 0 {
					glyphFactor = math.Min((widthLeft-origDiff)/maxGlyphSpacing, 1.0)
				}

				dx := 0.0
				for i, span := range l.spans {
					sentences, words := 0, 0
					for _, boundary := range span.boundaries {
						if boundary.kind == sentenceBoundary {
							sentences++
						} else if boundary.kind == wordBoundary {
							words++
						}
					}
					glyphs := utf8.RuneCountInString(span.origText)

					xHeight := span.ff.Metrics().XHeight
					sentenceSpacing := MaxSentenceSpacing * xHeight * sentenceFactor
					wordSpacing := MaxWordSpacing * xHeight * wordFactor
					glyphSpacing := MaxGlyphSpacing * xHeight * glyphFactor

					if 0.0 < glyphSpacing {
						l.spans[i].text = l.spans[i].origText
						l.spans[i].textWidth = l.spans[i].origTextWidth
					}

					w := span.textWidth + float64(sentences)*sentenceSpacing + float64(words)*wordSpacing + float64(glyphs)*glyphSpacing
					l.spans[i].dx += dx
					dx += w - span.w
					l.spans[i].w = w
					l.spans[i].sentenceSpacing = sentenceSpacing
					l.spans[i].wordSpacing = wordSpacing
					l.spans[i].glyphSpacing = glyphSpacing
				}
			}
		}
	}

	// apply vertical alignment
	dy := 0.0
	h := -y
	extraLineSpacing := 0.0
	if height != 0.0 && (valign == Bottom || valign == Center || valign == Justify) {
		if valign == Bottom {
			dy = height - h
		} else if valign == Center {
			dy = (height - h) / 2.0
		} else if len(lines) > 1 {
			extraLineSpacing = (height - h) / float64(len(lines)-1)
		}
	}
	for j := range lines {
		lines[j].y -= dy + float64(j)*extraLineSpacing
	}

	// set decorations
	for j, line := range lines {
		ff := FontFace{}
		x0, x1 := 0.0, 0.0
		for _, span := range line.spans {
			if 0.0 < x1-x0 && !span.ff.Equals(ff) {
				if ff.deco != nil {
					lines[j].decos = append(lines[j].decos, decoSpan{ff, x0, x1})
				}
				x0 = x1
			}
			ff = span.ff
			if x0 == x1 {
				x0 = span.dx // skip space when starting new decoSpan
			}
			x1 = span.dx + span.w
		}
		if 0.0 < x1-x0 && ff.deco != nil {
			lines[j].decos = append(lines[j].decos, decoSpan{ff, x0, x1})
		}
	}
	return &Text{lines, rt.fonts}
}

// Bounds returns the rectangle that contains the entire text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return Rect{}
	}
	r := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			spanBounds := span.Bounds(span.w)
			spanBounds = spanBounds.Move(Point{span.dx, line.y})
			r = r.Add(spanBounds)
		}
	}
	return r
}

func (t *Text) mostCommonFontFace() FontFace {
	families := map[*FontFamily]int{}
	sizes := map[float64]int{}
	styles := map[FontStyle]int{}
	variants := map[FontVariant]int{}
	colors := map[color.RGBA]int{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			families[span.ff.family]++
			sizes[span.ff.size]++
			styles[span.ff.style]++
			variants[span.ff.variant]++
			colors[span.ff.color]++
		}
	}
	if len(families) == 0 {
		return FontFace{}
	}

	family, size, style, variant, col := (*FontFamily)(nil), 0.0, FontRegular, FontNormal, Black
	for key, val := range families {
		if families[family] < val {
			family = key
		}
	}
	for key, val := range sizes {
		if sizes[size] < val {
			size = key
		}
	}
	for key, val := range styles {
		if styles[style] < val {
			style = key
		}
	}
	for key, val := range variants {
		if variants[variant] < val {
			variant = key
		}
	}
	for key, val := range colors {
		if colors[col] < val {
			col = key
		}
	}
	return family.Face(size*PtPerMm, col, style, variant)
}

// ToPath makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPaths() ([]*Path, []color.RGBA) {
	paths := []*Path{}
	colors := []color.RGBA{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			p, _, col := span.ToPath(span.w)
			p = p.Translate(span.dx, line.y)
			paths = append(paths, p)
			colors = append(colors, col)
		}
		for _, deco := range line.decos {
			p := deco.ff.Decorate(deco.x1 - deco.x0)
			p = p.Translate(deco.x0, line.y)
			paths = append(paths, p)
			colors = append(colors, deco.ff.color)
		}
	}
	return paths, colors
}

// WriteSVG will write out the text in the SVG file format.
func (t *Text) WriteSVG(w io.Writer, x, y, rot float64) {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return
	}

	writeStyle := func(ff, ffMain FontFace) {
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
			if ff.style&FontItalic != ffMain.style&FontItalic {
				fmt.Fprintf(w, ` italic`)
			}

			if boldness != ffMain.boldness() {
				fmt.Fprintf(w, ` %d`, boldness)
			}

			if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
				fmt.Fprintf(w, ` small-caps`)
			}

			fmt.Fprintf(w, ` %gpx %s`, ff.size*ff.scale, ff.font.name)
			if ff.color != ffMain.color {
				fmt.Fprintf(w, `;fill:%s`, toCSSColor(ff.color))
			}
		} else if differences == 1 && ff.color != ffMain.color {
			fmt.Fprintf(w, `" fill="%s`, toCSSColor(ff.color))
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
				fmt.Fprintf(buf, `;fill:%s`, toCSSColor(ff.color))
			}
			buf.ReadByte()
			buf.WriteTo(w)
		}
	}

	ffMain := t.mostCommonFontFace()

	fmt.Fprintf(w, `<text x="%g" y="%g`, x, y)
	if rot != 0.0 {
		fmt.Fprintf(w, `" transform="rotate(%g,%g,%g)`, -rot, x, y)
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
	fmt.Fprintf(w, ` %gpx %s`, ffMain.size*ffMain.scale, ffMain.font.name)
	if ffMain.color != Black {
		fmt.Fprintf(w, `;fill:%s`, toCSSColor(ffMain.color))
	}
	fmt.Fprintf(w, `">`)

	decorations := []pathLayer{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			// TODO: sentence spacing
			fmt.Fprintf(w, `<tspan x="%g" y="%g`, x+span.dx, y-line.y-span.ff.voffset)
			if span.wordSpacing > 0.0 {
				fmt.Fprintf(w, `" word-spacing="%g`, span.wordSpacing)
			}
			if span.glyphSpacing > 0.0 {
				fmt.Fprintf(w, `" letter-spacing="%g`, span.glyphSpacing)
			}
			writeStyle(span.ff, ffMain)
			fmt.Fprintf(w, `">%s</tspan>`, span.text)
		}
		for _, deco := range line.decos {
			p := deco.ff.Decorate(deco.x1 - deco.x0)
			p = p.Transform(Identity.Translate(x+deco.x0, -y+line.y+deco.ff.voffset).RotateAt(rot, x, -y))
			decorations = append(decorations, pathLayer{p, drawState{fillColor: deco.ff.color}})
		}
	}
	fmt.Fprintf(w, `</text>`)
	for _, l := range decorations {
		l.WriteSVG(w, 0.0)
	}
}

// WritePDF will write out the text in the PDF file format.
func (t *Text) WritePDF(w *PDFPageWriter) {
	// TODO: Text.WritePDF
}

// TODO: Text.WriteEPS
// TODO: Text.WriteImage

////////////////////////////////////////////////////////////////

type decoSpan struct {
	ff     FontFace
	x0, x1 float64
}

type textSpan struct {
	dx, w           float64
	path            *Path // either path is set, or the text related attributes below
	text            string
	origText        string
	ff              FontFace
	textWidth       float64
	origTextWidth   float64
	boundaries      []textBoundary
	sentenceSpacing float64
	wordSpacing     float64
	glyphSpacing    float64
}

// TODO: add newPathSpan

// TODO: proper transformation of typographic elements, ie. including surrounding text
func newTextSpan(ff FontFace, origText string) textSpan {
	text := ff.font.substituteLigatures(origText)
	boundaries := calcTextBoundaries(text, 0, len(text))
	return textSpan{
		dx:              0.0,
		w:               0.0,
		path:            nil,
		text:            text,
		origText:        origText,
		ff:              ff,
		textWidth:       ff.TextWidth(text),
		origTextWidth:   ff.TextWidth(origText),
		boundaries:      boundaries,
		sentenceSpacing: 0.0,
		wordSpacing:     0.0,
		glyphSpacing:    0.0,
	}
}

func (span textSpan) TrimLeft() textSpan {
	if 0 < len(span.boundaries) {
		firstBoundary := span.boundaries[0]
		if firstBoundary.pos == 0 {
			wDiff := span.ff.TextWidth(span.text[:firstBoundary.size])
			span.w -= wDiff
			span.textWidth -= wDiff
			span.origTextWidth -= wDiff
			span.text = span.text[firstBoundary.size:]
			span.origText = span.origText[firstBoundary.size:]
			span.boundaries = span.boundaries[1:]
			for i := range span.boundaries {
				span.boundaries[i].pos -= firstBoundary.size
			}
		}
	}
	return span
}

func (span textSpan) TrimRight() textSpan {
	if 0 < len(span.boundaries) {
		lastBoundary := span.boundaries[len(span.boundaries)-1]
		if lastBoundary.pos+lastBoundary.size == len(span.text) {
			wDiff := span.ff.TextWidth(span.text[len(span.text)-lastBoundary.size:])
			span.w -= wDiff
			span.textWidth -= wDiff
			span.origTextWidth -= wDiff
			span.text = span.text[:len(span.text)-lastBoundary.size]
			span.origText = span.origText[:len(span.origText)-lastBoundary.size]
			span.boundaries = span.boundaries[:len(span.boundaries)-1]
		}
	}
	return span
}

func (span textSpan) Bounds(width float64) Rect {
	p, deco, _ := span.ToPath(width)
	return p.Bounds().Add(deco.Bounds()) // TODO: make more efficient?
}

func (span textSpan) Heights() (float64, float64, float64) {
	return span.ff.Metrics().Ascent, span.ff.Metrics().Descent, span.ff.Metrics().LineHeight - span.ff.Metrics().Ascent - span.ff.Metrics().Descent
}

func (span textSpan) Split(width float64) ([]textSpan, bool) {
	if width == 0.0 || span.textWidth <= width {
		return []textSpan{span}, true
	}
	for i := len(span.boundaries) - 1; i >= 0; i-- {
		boundary := span.boundaries[i]
		s0 := span.text[:boundary.pos]
		if boundary.kind == breakBoundary {
			s0 += "-"
		}
		if span.ff.TextWidth(s0) <= width {
			s1 := span.text[boundary.pos+boundary.size:]
			if boundary.pos == 0 {
				return []textSpan{span}, false
			}
			return []textSpan{newTextSpan(span.ff, s0), newTextSpan(span.ff, s1)}, true
		}
	}
	return []textSpan{span}, false
}

// TODO: transform to Draw to canvas and cache the glyph rasterizations?
func (span textSpan) ToPath(width float64) (*Path, *Path, color.RGBA) {
	iBoundary := 0

	x := 0.0
	p := &Path{}
	var rPrev rune
	for i, r := range span.text {
		if i > 0 {
			x += span.ff.Kerning(rPrev, r)
		}

		pr, advance := span.ff.ToPath(string(r))
		pr = pr.Translate(x, 0.0)
		p = p.Append(pr)

		x += advance + span.glyphSpacing
		if iBoundary < len(span.boundaries) && span.boundaries[iBoundary].pos == i {
			boundary := span.boundaries[iBoundary]
			if boundary.kind == sentenceBoundary {
				x += span.sentenceSpacing
			} else if boundary.kind == wordBoundary {
				x += span.wordSpacing
			}
			iBoundary++
		}
		rPrev = r
	}
	return p, span.ff.Decorate(width), span.ff.color
}

////////////////////////////////////////////////////////////////

type textBoundaryKind int

const (
	lineBoundary textBoundaryKind = iota
	sentenceBoundary
	wordBoundary
	breakBoundary // zero-width space indicates word boundary
)

type textBoundary struct {
	kind textBoundaryKind
	pos  int
	size int
}

func mergeBoundaries(a, b []textBoundary) []textBoundary {
	if 0 < len(a) && 0 < len(b) && a[len(a)-1].pos+a[len(a)-1].size == b[0].pos {
		if b[0].kind < a[len(a)-1].kind {
			a[len(a)-1].kind = b[0].kind
		}
		a[len(a)-1].size += b[0].size
		b = b[1:]
	}
	return append(a, b...)
}

func calcTextBoundaries(s string, a, b int) []textBoundary {
	boundaries := []textBoundary{}
	var rPrev, rPrevPrev rune
	if 0 < a {
		var size int
		rPrev, size = utf8.DecodeLastRuneInString(s[:a])
		if size < a {
			rPrevPrev, _ = utf8.DecodeLastRuneInString(s[:a-size])
		}
	}
	i := 0
	for _, r := range s[a:b] {
		size := utf8.RuneLen(r)
		if isNewline(r) {
			if r == '\n' && 0 < i && s[i-1] == '\r' {
				boundaries[len(boundaries)-1].size++
			} else {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{lineBoundary, i, size}})
			}
		} else if isWhitespace(r) {
			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev) && !isWhitespace(rPrevPrev)) || rPrev == '!' || rPrev == '?' {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{sentenceBoundary, i, size}})
			} else {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{wordBoundary, i, size}})
			}
		} else if r == '\u200b' {
			boundaries = mergeBoundaries(boundaries, []textBoundary{{breakBoundary, i, size}})
		}
		rPrevPrev = rPrev
		rPrev = r
		i += size
	}
	return boundaries
}

func splitNewlines(s string) []string {
	ss := []string{}
	i := 0
	for j, r := range s {
		if r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029' {
			if r == '\n' && 0 < j && s[j-1] == '\r' {
				i++
				continue
			}
			ss = append(ss, s[i:j])
			i = j + utf8.RuneLen(r)
		}
	}
	ss = append(ss, s[i:])
	return ss
}

func isNewline(r rune) bool {
	return r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029'
}

func isWhitespace(r rune) bool {
	// TODO: add breaking spaces such as en quad, em space, hair space, ...
	// see https://unicode.org/reports/tr14/#Properties
	return r == ' ' || r == '\t' || isNewline(r)
}
