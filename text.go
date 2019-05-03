package canvas

import (
	"image/color"
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
		span := newTextSpan(ff, s, 0.0, Left, 0.0)
		lines = append(lines, []textSpan{span})
	}
	return &Text{
		ff:          ff,
		lines:       lines,
		dy:          0.0,
		lineSpacing: 0.0,
	}
}

func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign, indent float64) *Text {
	// TODO: do inner-word boundaries
	lines := [][]textSpan{}
	var iPrev, iSpace int
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == ' ' {
			iSpace = i
		} else if r == '\r' && i+1 < len(s) && s[i+1] == '\n' {
			size++
		}

		isNewline := r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029'
		if isNewline || width != 0.0 && ff.TextWidth(s[iPrev:i+size])+indent > width {
			iBreak := i
			if !isNewline && iPrev < iSpace {
				iBreak = iSpace // break line at last space
			}

			var span textSpan
			if isNewline {
				span = newTextSpan(ff, s[iPrev:iBreak], 0.0, Left, indent)
			} else {
				span = newTextSpan(ff, s[iPrev:iBreak], width, halign, indent)
			}
			lines = append(lines, []textSpan{span})
			indent = 0.0
			if height != 0.0 && calcTextHeight(ff, len(lines)+1) > height {
				break
			}
			if i == 0 {
				continue
			}
			iPrev = iBreak
			if isNewline || iPrev == iSpace {
				iPrev += size // skip space or newline
			}
		}
		i += size
	}
	if height == 0.0 || calcTextHeight(ff, len(lines)+1) <= height {
		var span textSpan
		if halign == Right || halign == Center {
			span = newTextSpan(ff, s[iPrev:], width, halign, indent)
		} else {
			span = newTextSpan(ff, s[iPrev:], 0.0, Left, indent)
		}
		lines = append(lines, []textSpan{span})
	}

	dy := 0.0
	lineSpacing := 0.0
	if height != 0.0 && (valign == Bottom || valign == Center || valign == Justify) {
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
		for _, span := range line {
			w = math.Max(w, span.dx+span.width)
		}
	}
	h = calcTextHeight(t.ff, len(t.lines)) + t.lineSpacing*float64(len(t.lines)-1)
	return w, h

}

// ToPath makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPath(x, y float64) *Path {
	p := &Path{}
	y -= t.dy
	for _, line := range t.lines {
		for _, span := range line {
			p.Append(span.ToPath(x, y))
		}
		y -= t.ff.Metrics().LineHeight + t.lineSpacing
	}
	return p
}

func (t *Text) splitAtBoundaries(x, y float64, f func(float64, float64, float64, string)) {
	spaceWidth := t.ff.TextWidth(" ")
	for _, line := range t.lines {
		for _, span := range line {
			if span.sentenceSpacing > 0.0 || span.wordSpacing > 0.0 {
				_, _, _, boundaries := calcTextSpanSpacings(span.s)
				boundaries = append(boundaries, TextBoundary{true, len(span.s)})

				prevLoc := 0
				dx := span.dx
				for _, boundary := range boundaries {
					s := span.s[prevLoc:boundary.loc]
					width := 0.0
					if span.glyphSpacing > 0.0 {
						width = t.ff.TextWidth(s) + float64(utf8.RuneCountInString(s)-1)*span.glyphSpacing
					}
					f(x+dx, y, width, s)
					prevLoc = boundary.loc + 1
					dx += t.ff.TextWidth(s) + spaceWidth + float64(utf8.RuneCountInString(s))*span.glyphSpacing
					if boundary.isWord {
						dx += span.wordSpacing
					} else {
						dx += span.sentenceSpacing
					}
				}
			} else {
				width := 0.0
				if span.glyphSpacing > 0.0 {
					width = span.width
				}
				f(x+span.dx, y, width, span.s)
			}
		}
		y += t.ff.Metrics().LineHeight + t.lineSpacing
	}
}

func (t *Text) ToSVG(x, y, rot float64, c color.Color) string {
	y += t.dy
	name, style, size := t.ff.Info()

	sb := strings.Builder{}
	sb.WriteString("<text x=\"")
	writeFloat64(&sb, x)
	sb.WriteString("\" y=\"")
	writeFloat64(&sb, y)
	if rot != 0.0 {
		sb.WriteString("\" transform=\"rotate(")
		writeFloat64(&sb, -rot)
		sb.WriteString(",")
		writeFloat64(&sb, x)
		sb.WriteString(",")
		writeFloat64(&sb, y)
		sb.WriteString(")")
	}
	sb.WriteString("\" font-family=\"")
	sb.WriteString(name)
	sb.WriteString("\" font-size=\"")
	writeFloat64(&sb, size)
	if style&Italic != 0 {
		sb.WriteString("\" font-style=\"italic")
	}
	if style&Bold != 0 {
		sb.WriteString("\" font-weight=\"bold")
	}
	if c != color.Black {
		sb.WriteString("\" fill=\"")
		writeCSSColor(&sb, c)
	}
	sb.WriteString("\">")
	t.splitAtBoundaries(x, y, func(x, y, width float64, s string) {
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
	})
	sb.WriteString("</text>")
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

type TextBoundary struct {
	isWord bool
	loc    int
}

func calcTextSpanSpacings(s string) (int, int, int, []TextBoundary) {
	sentenceSpacings, wordSpacings, glyphSpacings := 0, 0, 0
	locs := []TextBoundary{}
	var rPrev, rPrevPrev rune
	for i, r := range s {
		glyphSpacings++
		if r == ' ' {
			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev)) || rPrev == '!' || rPrev == '?' {
				locs = append(locs, TextBoundary{false, i})
				sentenceSpacings++
			} else if rPrev != ' ' {
				locs = append(locs, TextBoundary{true, i})
				wordSpacings++
			}
		}
		rPrevPrev = rPrev
		rPrev = r
	}
	glyphSpacings -= wordSpacings + sentenceSpacings + 1
	return sentenceSpacings, wordSpacings, glyphSpacings, locs
}

func newTextSpan(ff FontFace, s string, width float64, halign TextAlign, indent float64) textSpan {
	dx := indent
	sentenceSpacing := 0.0
	wordSpacing := 0.0
	glyphSpacing := 0.0
	textWidth := ff.TextWidth(s)
	if halign == Right || halign == Center || halign == Justify {
		if halign == Right {
			dx = width - textWidth - indent
		} else if halign == Center {
			dx = (width - textWidth) / 2.0
		} else if textWidth+indent < width {
			sentenceSpacings, wordSpacings, glyphSpacings, _ := calcTextSpanSpacings(s)
			widthLeft := width - textWidth - indent
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
	if width == 0.0 {
		width = textWidth + indent
	}
	return textSpan{
		ff:              ff,
		s:               s,
		dx:              dx,
		width:           width - dx,
		sentenceSpacing: sentenceSpacing,
		wordSpacing:     wordSpacing,
		glyphSpacing:    glyphSpacing,
	}
}

func (ts textSpan) ToPath(x, y float64) *Path {
	p := &Path{}
	x += ts.dx

	iBoundary := 0
	_, _, _, boundaries := calcTextSpanSpacings(ts.s)

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
		if iBoundary < len(boundaries) && boundaries[iBoundary].loc == i {
			if boundaries[iBoundary].isWord {
				spacing = ts.wordSpacing
			} else {
				spacing = ts.sentenceSpacing
			}
			iBoundary++
		}
		x += spacing
		rPrev = r
	}
	return p
}
