package canvas

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TextAlign int

const (
	Left TextAlign = iota
	Right
	Center
	Top
	Bottom
	Justify
)

type Text struct {
	ff          FontFace
	lines       [][]textSpan
	dy          float64
	lineSpacing float64
}

func splitNewlines(s string) []string {
	ss := []string{}
	i := 0
	for j, r := range s {
		if r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029' {
			if r == '\n' && j > 0 && s[j-1] == '\r' {
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

func calcTextHeight(ff FontFace, lines int) float64 {
	return ff.Metrics().Ascent + float64(lines-1)*ff.Metrics().LineHeight + ff.Metrics().Descent
}

func NewText(ff FontFace, s string) *Text {
	ss := splitNewlines(s)
	lines := [][]textSpan{}
	for _, s := range ss {
		span := newTextSpan(ff, s, 0.0, Left)
		lines = append(lines, []textSpan{span})
	}
	return &Text{
		ff:          ff,
		lines:       lines,
		dy:          0.0,
		lineSpacing: 0.0,
	}
}

func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign) *Text {
	// TODO: do inner-word boundaries
	lines := [][]textSpan{}
	var iPrev, iSpace int
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == ' ' {
			iSpace = i
		}
		isNewline := r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029'
		if isNewline || ff.TextWidth(s[iPrev:i+size]) > width {
			iBreak := i
			if i == 0 && !isNewline {
				break // nothing fits
			} else if !isNewline && iPrev < iSpace {
				iBreak = iSpace
			}
			var span textSpan
			if isNewline {
				span = newTextSpan(ff, s[iPrev:iBreak], 0.0, Left)
			} else {
				span = newTextSpan(ff, s[iPrev:iBreak], width, halign)
			}
			lines = append(lines, []textSpan{span})
			if calcTextHeight(ff, len(lines)+1) > height {
				break
			}
			if r == '\r' && i+size < len(s) && s[i+size] == '\n' {
				i++
			}
			iPrev = iBreak
			if iPrev == iSpace {
				iPrev++
			}
		}
		i += size
	}
	if calcTextHeight(ff, len(lines)+1) <= height {
		var span textSpan
		if halign == Right || halign == Center {
			span = newTextSpan(ff, s[iPrev:], width, halign)
		} else {
			span = newTextSpan(ff, s[iPrev:], 0.0, Left)
		}
		lines = append(lines, []textSpan{span})
	}

	dy := 0.0
	lineSpacing := 0.0
	if valign == Bottom || valign == Center || valign == Justify {
		h := calcTextHeight(ff, len(lines))
		if valign == Bottom {
			dy = height - h
		} else if valign == Center {
			dy = (height - h) / 2.0
		} else {
			lineSpacing = (height - h) / float64(len(lines)-1)
		}
	}
	return &Text{
		ff:          ff,
		lines:       lines,
		dy:          dy,
		lineSpacing: lineSpacing,
	}
}

func (t *Text) Bounds() (w, h float64) {
	for _, line := range t.lines {
		for _, ts := range line {
			w = math.Max(w, ts.dx+ts.width)
		}
	}
	h = calcTextHeight(t.ff, len(t.lines))
	return w, h

}

// ToPath makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPath(x, y float64) *Path {
	p := &Path{}
	y -= t.dy
	y -= t.ff.Metrics().Ascent
	for _, line := range t.lines {
		for _, span := range line {
			p.Append(span.ToPath(x, y))
		}
		y -= t.ff.Metrics().LineHeight + t.lineSpacing
	}
	return p
}

func (t *Text) ToSVG(x, y float64) string {
	// TODO: implement with indent, spacing between characters and lines etc.
	y += t.dy + t.ff.Metrics().Ascent
	sb := strings.Builder{}
	writeTSpan := func(x, y, width float64, s string) {
		sb.WriteString("<tspan x=\"")
		writeFloat64(&sb, x)
		sb.WriteString("\" y=\"")
		writeFloat64(&sb, y)
		if width > 0.0 {
			sb.WriteString("\" textLength=\"")
			writeFloat64(&sb, width)
		}
		sb.WriteString("\">")
		sb.WriteString(s)
		sb.WriteString("</tspan>")
	}
	for _, line := range t.lines {
		for _, span := range line {
			//if span.sentenceSpacing > 0.0 || span.wordSpacing > 0.0 {
			//_, _, _, spacingLocs := calcTextSpanSpacings(span.s)
			// TODO: split up into tspans
			//} else {
			width := 0.0
			if span.glyphSpacing > 0.0 {
				width = span.width
			}
			writeTSpan(x+span.dx, y, width, span.s)
			//}
		}
		y += t.ff.Metrics().LineHeight + t.lineSpacing
	}
	return sb.String()
}

const MaxSentenceSpacing = 2.0
const MaxWordSpacing = 1.5
const MaxGlyphSpacing = 1.0

type textSpan struct {
	ff              FontFace
	s               string
	dx, width       float64
	sentenceSpacing float64
	wordSpacing     float64
	glyphSpacing    float64
}

type TextBoundary int

const (
	SentenceBoundary TextBoundary = iota
	WordBoundary
)

func calcTextSpanSpacings(s string) (int, int, int, map[int]TextBoundary) {
	sentenceSpacings, wordSpacings, glyphSpacings := 0, 0, 0
	locs := map[int]TextBoundary{}
	var rPrev, rPrevPrev rune
	for i, r := range s {
		glyphSpacings++
		if r == ' ' {
			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev)) || rPrev == '!' || rPrev == '?' {
				locs[i] = SentenceBoundary
				sentenceSpacings++
			} else if rPrev != ' ' {
				locs[i] = WordBoundary
				wordSpacings++
			}
		}
		rPrevPrev = rPrev
		rPrev = r
	}
	return sentenceSpacings, wordSpacings, glyphSpacings - 1, locs
}

func newTextSpan(ff FontFace, s string, width float64, halign TextAlign) textSpan {
	dx := 0.0
	sentenceSpacing := 0.0
	wordSpacing := 0.0
	glyphSpacing := 0.0
	textWidth := ff.TextWidth(s)
	if width == 0.0 {
		width = textWidth
	} else if halign == Right || halign == Center || halign == Justify {
		if halign == Right {
			dx = width - textWidth
		} else if halign == Center {
			dx = (width - textWidth) / 2.0
		} else if textWidth < width {
			sentenceSpacings, wordSpacings, glyphSpacings, _ := calcTextSpanSpacings(s)
			widthLeft := width - textWidth
			if sentenceSpacings > 0 {
				sentenceSpacing = math.Min(widthLeft/float64(sentenceSpacings), ff.Metrics().XHeight*MaxSentenceSpacing)
				widthLeft -= float64(sentenceSpacings) * sentenceSpacing
			}
			if wordSpacings > 0 {
				wordSpacing = math.Min(widthLeft/float64(wordSpacings), ff.Metrics().XHeight*MaxWordSpacing)
				widthLeft -= float64(wordSpacings) * wordSpacing
			}
			if glyphSpacings > 0 {
				glyphSpacing = math.Min(widthLeft/float64(glyphSpacings), ff.Metrics().XHeight*MaxGlyphSpacing)
			}
		}
	}
	return textSpan{
		ff:              ff,
		s:               s,
		dx:              dx,
		width:           width,
		sentenceSpacing: sentenceSpacing,
		wordSpacing:     wordSpacing,
		glyphSpacing:    glyphSpacing,
	}
}

func (ts textSpan) ToPath(x, y float64) *Path {
	p := &Path{}
	x += ts.dx
	_, _, _, spacingLocs := calcTextSpanSpacings(ts.s)
	var rPrev rune
	for i, r := range ts.s {
		if i > 0 {
			x += ts.ff.Kerning(rPrev, r)
		}

		pr, advance := ts.ff.ToPath(r)
		pr.Translate(x, y)
		p.Append(pr)
		x += advance

		spacing := ts.glyphSpacing
		if boundary, ok := spacingLocs[i]; ok {
			if boundary == SentenceBoundary {
				spacing = ts.sentenceSpacing
			} else {
				spacing = ts.wordSpacing
			}
		}
		x += spacing
		rPrev = r
	}
	return p
}
