package canvas

import (
	"image/color"
	"math"
	"sort"
	"unicode"
	"unicode/utf8"
)

// MaxSentenceSpacing is the maximum amount times the x-height of the font that sentence spaces can expand.
const MaxSentenceSpacing = 3.5

// MaxWordSpacing is the maximum amount times the x-height of the font that word spaces can expand.
const MaxWordSpacing = 2.5

// MaxGlyphSpacing is the maximum amount times the x-height of the font that glyphs can be spaced.
const MaxGlyphSpacing = 0.5

// TextAlign specifies how the text should align or whether it should be justified.
type TextAlign int

// see TextAlign
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
	for _, span := range l.spans {
		spanAscent, spanDescent, lineSpacing := span.ff.Metrics().Ascent, span.ff.Metrics().Descent, span.ff.Metrics().LineHeight-span.ff.Metrics().Ascent-span.ff.Metrics().Descent
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

// NewTextLine is a simple text line using a font face, a string (supporting new lines) and horizontal alignment (Left, Center, Right).
func NewTextLine(ff FontFace, s string, halign TextAlign) *Text {
	ascent, descent, spacing := ff.Metrics().Ascent, ff.Metrics().Descent, ff.Metrics().LineHeight-ff.Metrics().Ascent-ff.Metrics().Descent

	i := 0
	y := 0.0
	lines := []line{}
	for _, boundary := range calcTextBoundaries(s, 0, len(s)) {
		if boundary.kind == lineBoundary || boundary.kind == eofBoundary {
			j := boundary.pos + boundary.size
			if i < j {
				l := line{y: y}
				span := newTextSpan(ff, s[:j], i)
				if halign == Center {
					span.dx = -span.width / 2.0
				} else if halign == Right {
					span.dx = -span.width
				}

				l.spans = append(l.spans, span)
				if len(ff.deco) != 0 {
					l.decos = append(l.decos, decoSpan{ff, span.dx, span.dx + span.width})
				}
				lines = append(lines, l)
			}
			y -= spacing + ascent + descent + spacing
			i = j
		}
	}
	return &Text{lines, map[*Font]bool{ff.font: true}}
}

// NewTextBox is an advanced text formatter that will calculate text placement based on the setteings. It takes a font face, a string, the width or height of the box (can be zero for no limit), horizontal and vertical alignment (Left, Center, Right, Top, Bottom or Justify), text indentation for the first line and line stretch (percentage to stretch the line based on the line height).
func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	return NewRichText().Add(ff, s).ToText(width, height, halign, valign, indent, lineStretch)
}

// RichText allows to build up a rich text with text spans of different font faces and by fitting that into a box.
type RichText struct {
	spans []textSpan
	fonts map[*Font]bool
	text  string
}

// NewRichText returns a new RichText.
func NewRichText() *RichText {
	return &RichText{
		fonts: map[*Font]bool{},
	}
}

// Add adds a new text span element.
func (rt *RichText) Add(ff FontFace, s string) *RichText {
	if 0 < len(s) {
		rPrev := ' '
		rNext, size := utf8.DecodeRuneInString(s)
		if 0 < len(rt.text) {
			rPrev, _ = utf8.DecodeLastRuneInString(rt.text)
		}
		if isWhitespace(rPrev) && isWhitespace(rNext) {
			s = s[size:]
		}
	}

	start := len(rt.text)
	rt.text += s

	// TODO: can we simplify this? Just merge adjacent spans, don't split at newlines or sentences?
	i := 0
	for _, boundary := range calcTextBoundaries(s, 0, len(s)) {
		if boundary.kind == lineBoundary || boundary.kind == sentenceBoundary || boundary.kind == eofBoundary {
			j := boundary.pos + boundary.size
			if i < j {
				extendPrev := false
				if i == 0 && boundary.kind != lineBoundary && 0 < len(rt.spans) && rt.spans[len(rt.spans)-1].ff.Equals(ff) {
					prevSpan := rt.spans[len(rt.spans)-1]
					if 1 < len(prevSpan.boundaries) {
						prevBoundaryKind := prevSpan.boundaries[len(prevSpan.boundaries)-2].kind
						if prevBoundaryKind != lineBoundary && prevBoundaryKind != sentenceBoundary {
							extendPrev = true
						}
					} else {
						extendPrev = true
					}
				}

				if extendPrev {
					diff := len(rt.spans[len(rt.spans)-1].text)
					rt.spans[len(rt.spans)-1] = newTextSpan(ff, rt.text[:start+j], start+i-diff)
				} else {
					rt.spans = append(rt.spans, newTextSpan(ff, rt.text[:start+j], start+i))
				}
			}
			i = j
		}
	}
	rt.fonts[ff.font] = true
	return rt
}

func (rt *RichText) halign(lines []line, yoverflow bool, width float64, halign TextAlign) {
	if halign == Right || halign == Center {
		for _, l := range lines {
			firstSpan := l.spans[0]
			lastSpan := l.spans[len(l.spans)-1]
			dx := width - lastSpan.dx - lastSpan.width - firstSpan.dx
			if halign == Center {
				dx /= 2.0
			}
			for i := range l.spans {
				l.spans[i].dx += dx
			}
		}
	} else if 0.0 < width && halign == Justify {
		n := len(lines) - 1
		if yoverflow {
			n++
		}
		for _, l := range lines[:n] {
			// get the width range of our spans (eg. for text width can increase with extra character spacing)
			textWidth, maxSentenceSpacing, maxWordSpacing, maxGlyphSpacing := 0.0, 0.0, 0.0, 0.0
			for i, span := range l.spans {
				sentences, words := 0, 0
				for _, boundary := range span.boundaries {
					if boundary.kind == sentenceBoundary {
						sentences++
					} else if boundary.kind == wordBoundary {
						words++
					}
				}
				glyphs := span.CountGlyphs()
				if i+1 == len(l.spans) {
					glyphs--
				}

				textWidth += span.width
				if i == 0 {
					textWidth += span.dx
				}

				xHeight := span.ff.Metrics().XHeight
				maxSentenceSpacing += float64(sentences) * MaxSentenceSpacing * xHeight
				maxWordSpacing += float64(words) * MaxWordSpacing * xHeight
				maxGlyphSpacing += float64(glyphs) * MaxGlyphSpacing * xHeight
			}

			// use non-ligature versions so we can stretch glyph spacings
			if textWidth+maxSentenceSpacing+maxWordSpacing < width && width <= textWidth+maxSentenceSpacing+maxWordSpacing+maxGlyphSpacing {
				textWidth = 0.0
				for i, span := range l.spans {
					l.spans[i] = span.ReplaceLigatures()
					textWidth += span.width
					if i == 0 {
						textWidth += span.dx
					}
				}
			}

			// only expand if we can reach the line width
			if textWidth < width && width <= textWidth+maxSentenceSpacing+maxWordSpacing+maxGlyphSpacing {
				widthLeft := width - textWidth
				sentenceFactor, wordFactor, glyphFactor := 0.0, 0.0, 0.0
				if Epsilon < widthLeft && (0 < maxWordSpacing || 0 < maxSentenceSpacing) {
					sentenceFactor = math.Min(widthLeft/(maxWordSpacing+maxSentenceSpacing), 1.0)
					wordFactor = sentenceFactor
					widthLeft -= sentenceFactor * maxSentenceSpacing
					widthLeft -= wordFactor * maxWordSpacing
				}
				if Epsilon < widthLeft && 0 < maxGlyphSpacing {
					glyphFactor = math.Min(widthLeft/maxGlyphSpacing, 1.0)
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
					glyphs := utf8.RuneCountInString(span.text)
					if i+1 == len(l.spans) {
						glyphs--
					}

					xHeight := span.ff.Metrics().XHeight
					sentenceSpacing := MaxSentenceSpacing * xHeight * sentenceFactor
					wordSpacing := MaxWordSpacing * xHeight * wordFactor
					glyphSpacing := MaxGlyphSpacing * xHeight * glyphFactor

					w := span.width + float64(sentences)*sentenceSpacing + float64(words)*wordSpacing + float64(glyphs)*glyphSpacing
					l.spans[i].dx += dx
					l.spans[i].width = w
					l.spans[i].sentenceSpacing = sentenceSpacing
					l.spans[i].wordSpacing = wordSpacing
					l.spans[i].glyphSpacing = glyphSpacing
					dx += w - span.width
				}
			}
		}
	}
}

func (rt *RichText) valign(lines []line, h, height float64, valign TextAlign) {
	dy := 0.0
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
}

func (rt *RichText) decorate(lines []line) {
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
			x1 = span.dx + span.width
		}
		if 0.0 < x1-x0 && ff.deco != nil {
			lines[j].decos = append(lines[j].decos, decoSpan{ff, x0, x1})
		}
	}
}

// ToText takes the added text spans and fits them within a given box of certain width and height.
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	if len(rt.spans) == 0 {
		return &Text{[]line{}, rt.fonts}
	}
	spans := []textSpan{rt.spans[0]}

	k := 0 // index into rt.spans and rt.positions
	lines := []line{}
	yoverflow := false
	y, prevLineSpacing := 0.0, 0.0
	for k < len(rt.spans) {
		dx := indent
		indent = 0.0

		// trim left spaces
		spans[0] = spans[0].TrimLeft()
		for spans[0].text == "" {
			// TODO: reachable?
			if k+1 == len(rt.spans) {
				break
			}
			k++
			spans = []textSpan{rt.spans[k]}
			spans[0] = spans[0].TrimLeft()
		}

		// accumulate line spans for a full line, ie. either split span1 to fit or if it fits retrieve the next span1 and repeat
		ss := []textSpan{}
		for {
			// space or inter-word splitting
			if width != 0.0 && len(spans) == 1 {
				// there is a width limit and we have only one (unsplit) span to process
				var ok bool
				spans, ok = spans[0].Split(width - dx)
				if !ok && len(ss) != 0 {
					// span couln't fit but this line already has a span, try next line
					break
				}
			}

			// if this span ends with a newline, split off that newline boundary
			newline := 1 < len(spans[0].boundaries) && spans[0].boundaries[len(spans[0].boundaries)-2].kind == lineBoundary
			if newline {
				spans[0], _ = spans[0].split(len(spans[0].boundaries) - 2)
			}

			spans[0].dx = dx
			ss = append(ss, spans[0])
			dx += spans[0].width

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
			if newline {
				break
			}
		}

		// trim right spaces
		for 0 < len(ss) {
			ss[len(ss)-1] = ss[len(ss)-1].TrimRight()
			if 1 < len(ss) && ss[len(ss)-1].text == "" {
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
			yoverflow = true
			break
		}
		lines = append(lines, l)
	}

	if len(lines) == 0 {
		return &Text{lines, rt.fonts}
	}

	// apply horizontal alignment
	rt.halign(lines, yoverflow, width, halign)

	// apply vertical alignment
	rt.valign(lines, -y, height, valign)

	// set decorations
	rt.decorate(lines)

	return &Text{lines, rt.fonts}
}

// Empty is true if there are no text lines or no text spans.
func (t *Text) Empty() bool {
	for _, line := range t.lines {
		if len(line.spans) != 0 {
			return false
		}
	}
	return true
}

// Height returns the height of the text using the font metrics, this is usually more than the bounds of the glyph outlines.
func (t *Text) Height() float64 {
	if len(t.lines) == 0 {
		return 0.0
	}
	lastLine := t.lines[len(t.lines)-1]
	_, _, descent, _ := lastLine.Heights()
	return -lastLine.y + descent
}

// Bounds returns the bounding rectangle that defines the text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return Rect{}
	}
	r := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			r = r.Add(Rect{span.dx, line.y - span.ff.Metrics().Descent, span.ff.TextWidth(span.text), span.ff.Metrics().Ascent + span.ff.Metrics().Descent})
		}
	}
	return r
}

// OutlineBounds returns the rectangle that contains the entire text box, ie. the glyph outlines (slow).
func (t *Text) OutlineBounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return Rect{}
	}
	r := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			spanBounds := span.Bounds(span.width)
			spanBounds = spanBounds.Move(Point{span.dx, line.y})
			r = r.Add(spanBounds)
		}
	}
	return r
}

// Fonts returns list of fonts used.
func (t *Text) Fonts() []*Font {
	fonts := []*Font{}
	fontNames := []string{}
	fontMap := map[string]*Font{}
	for font := range t.fonts {
		name := font.Name()
		fontNames = append(fontNames, name)
		fontMap[name] = font
	}
	sort.Strings(fontNames)
	for _, name := range fontNames {
		fonts = append(fonts, fontMap[name])
	}
	return fonts
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
	return family.Face(size*ptPerMm, col, style, variant)
}

// ToPaths makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPaths() ([]*Path, []color.RGBA) {
	paths := []*Path{}
	colors := []color.RGBA{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			p, _, col := span.ToPath(span.width)
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

////////////////////////////////////////////////////////////////

type decoSpan struct {
	ff     FontFace
	x0, x1 float64
}

type textSpan struct {
	ff         FontFace
	text       string
	width      float64
	boundaries []textBoundary

	dx              float64
	sentenceSpacing float64
	wordSpacing     float64
	glyphSpacing    float64
}

func newTextSpan(ff FontFace, text string, i int) textSpan {
	return textSpan{
		ff:              ff,
		text:            text[i:],
		width:           ff.TextWidth(text[i:]),
		boundaries:      calcTextBoundaries(text, i, len(text)),
		dx:              0.0,
		sentenceSpacing: 0.0,
		wordSpacing:     0.0,
		glyphSpacing:    0.0,
	}
}

func (span textSpan) TrimLeft() textSpan {
	if 0 < len(span.boundaries) && span.boundaries[0].pos == 0 && span.boundaries[0].kind != lineBoundary {
		_, span1 := span.split(0)
		return span1
	}
	return span
}

func (span textSpan) TrimRight() textSpan {
	i := len(span.boundaries) - 2 // the last one is EOF
	if 1 < len(span.boundaries) && span.boundaries[i].pos+span.boundaries[i].size == len(span.text) && span.boundaries[i].kind != lineBoundary {
		span0, _ := span.split(i)
		return span0
	}
	return span
}

func (span textSpan) Bounds(width float64) Rect {
	p, deco, _ := span.ToPath(width)
	return p.Bounds().Add(deco.Bounds()) // TODO: make more efficient?
}

func (span textSpan) split(i int) (textSpan, textSpan) {
	dash := ""
	if span.boundaries[i].kind == breakBoundary {
		dash = "-"
	}

	span0 := textSpan{}
	span0.ff = span.ff
	span0.text = span.text[:span.boundaries[i].pos] + dash
	span0.width = span.ff.TextWidth(span0.text)
	span0.boundaries = append(span.boundaries[:i:i], textBoundary{eofBoundary, len(span0.text), 0})
	span0.dx = span.dx

	span1 := textSpan{}
	span1.ff = span.ff
	span1.text = span.text[span.boundaries[i].pos+span.boundaries[i].size:]
	span1.width = span.ff.TextWidth(span1.text)
	span1.boundaries = make([]textBoundary, len(span.boundaries)-i-1)
	copy(span1.boundaries, span.boundaries[i+1:])
	span1.dx = span.dx
	for j := range span1.boundaries {
		span1.boundaries[j].pos -= span.boundaries[i].pos + span.boundaries[i].size
	}
	return span0, span1
}

func (span textSpan) Split(width float64) ([]textSpan, bool) {
	if width == 0.0 || span.width <= width {
		return []textSpan{span}, true // span fits
	}
	for i := len(span.boundaries) - 2; i >= 0; i-- {
		if span.boundaries[i].pos == 0 {
			return []textSpan{span}, false // boundary is at the beginning, do not split
		}

		span0, span1 := span.split(i)
		if span0.width <= width {
			// span fits up to this boundary
			if span1.width == 0.0 {
				return []textSpan{span0}, true // there is no text between the last two boundaries (e.g. space followed by end)
			}
			return []textSpan{span0, span1}, true
		}
	}
	return []textSpan{span}, false // does not fit, but there are no boundaries to split
}

// CountGlyphs counts all the glyphs, where ligatures are separated into their constituent parts
func (span textSpan) CountGlyphs() int {
	n := 0
	for _, r := range span.text {
		if s, ok := ligatures[r]; ok {
			n += len(s)
		} else {
			n++
		}
	}
	return n
}

// ReplaceLigatures replaces all ligatures by their constituent parts
func (span textSpan) ReplaceLigatures() textSpan {
	shift := 0
	iBoundary := 0
	for i, r := range span.text {
		if span.boundaries[iBoundary].pos == i {
			span.boundaries[iBoundary].pos += shift
			iBoundary++
		} else if s, ok := ligatures[r]; ok {
			span.text = span.text[:i] + s + span.text[i+utf8.RuneLen(r):]
			shift += len(s) - 1
		}
	}
	span.boundaries[len(span.boundaries)-1].pos = len(span.text)
	span.width = span.ff.TextWidth(span.text)
	return span
}

// TODO: transform to Draw to canvas and cache the glyph rasterizations?
// TODO: remove width argument and use span.width?
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
	eofBoundary textBoundaryKind = iota
	lineBoundary
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
		if a[len(a)-1].kind != lineBoundary || b[0].kind != lineBoundary {
			if b[0].kind < a[len(a)-1].kind {
				a[len(a)-1].kind = b[0].kind
			} else if a[len(a)-1].kind < b[0].kind {
				b[0].kind = a[len(a)-1].kind
			}
			a[len(a)-1].size += b[0].size
			b = b[1:]
		}
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
	for i, r := range s[a:b] {
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
	}
	boundaries = append(boundaries, textBoundary{eofBoundary, b - a, 0})
	return boundaries
}

func isNewline(r rune) bool {
	return r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029'
}

func isWhitespace(r rune) bool {
	// see https://unicode.org/reports/tr14/#Properties
	return unicode.IsSpace(r) || r == '\t' || r == '\u2028' || r == '\u2029'
}
