package text

import (
	"fmt"
	"math"
	"unicode"

	"github.com/tdewolff/canvas/font"
)

// See: Donald E. Knuth and Michael F. Plass, "Breaking Paragraphs into Lines", 1981

const FairyTales = "In olden times when wish\u200Bing still helped one, there lived a king whose daugh\u200Bters were all beau\u200Bti\u200Bful; and the young\u200Best was so beautiful that the sun it\u200Bself, which has seen so much, was aston\u200Bished when\u200Bever it shone in her face. Close by the king's castle lay a great dark for\u200Best, and un\u200Bder an old lime-tree in the for\u200Best was a well, and when the day was very warm, the king's child went out into the for\u200Best and sat down by the side of the cool foun\u200Btain; and when she was bored she took a golden ball, and threw it up on high and caught it; and this ball was her favor\u200Bite play\u200Bthing."

// FrenchSpacing enforces equal widths for inter-word and inter-sentence spaces
var FrenchSpacing = false

// stretchability and shrinkability of spaces
var SpaceStretch = 1 / 2.0
var SpaceShrink = 1 / 3.0

// stretchability and shrinkability factors for inter-sentence and other types of spaces
// not used if FrenchSpacing is set
var SentenceFactor = 3.0
var ColonFactor = 2.0
var SemicolonFactor = 1.5
var CommaFactor = 1.25

// algorithmic parameters
var Tolerance = 10.0
var DemeritsLine = 10.0
var DemeritsFlagged = 100.0
var DemeritsFitness = 3000.0
var InfPenalty = 1000.0
var HyphenPenalty = 100.0

type Type int

const (
	Box Type = iota
	Glue
	Penalty
)

type Item struct {
	Type
	w, y, z float64
	penalty float64
	flagged bool
	glyphs  []Glyph
}

type Breakpoint struct {
	next, prev *Breakpoint
	parent     *Breakpoint

	position               int
	line                   int
	fitness                int
	width, stretch, shrink float64
	ratio                  float64
	demerits               float64
}

type Breakpoints struct {
	head, tail *Breakpoint
}

func (list *Breakpoints) Has(b *Breakpoint) bool {
	return !(b.prev == nil && b.next == nil && (b != list.head || list.head == nil))
}

func (list *Breakpoints) Push(b *Breakpoint) {
	if list.head == nil {
		list.head = b
		list.tail = b
	} else if !list.Has(b) {
		b.prev = list.tail
		list.tail.next = b
		list.tail = b
	}
}

func (list *Breakpoints) InsertBefore(b *Breakpoint, at *Breakpoint) {
	if list.Has(b) || !list.Has(at) {
		return
	}
	b.next = at
	if at.prev == nil {
		list.head = b
	} else {
		b.prev = at.prev
		at.prev.next = b
	}
	at.prev = b
}

func (list *Breakpoints) Remove(b *Breakpoint) {
	if !list.Has(b) {
		return
	}
	if b.prev == nil {
		list.head = b.next
	} else {
		b.prev.next = b.next
	}
	if b.next == nil {
		list.tail = b.prev
	} else {
		b.next.prev = b.prev
	}
	b.prev = nil
	b.next = nil
}

type Break struct {
	position int
	ratio    float64
}

type Linebreaker struct {
	items       []Item
	activeNodes *Breakpoints
	W, Y, Z     float64
	width       float64
}

func NewLinebreaker(items []Item, width float64) *Linebreaker {
	activeNodes := &Breakpoints{}
	activeNodes.Push(&Breakpoint{fitness: 1})
	return &Linebreaker{
		items:       items,
		activeNodes: activeNodes,
		width:       width,
	}
}

func createItems(sfnt *font.SFNT, ppem, indent float64, glyphs []Glyph) []Item {
	f := ppem / float64(sfnt.Head.UnitsPerEm)

	hyphenWidth := f * float64(sfnt.GlyphAdvance(sfnt.GlyphIndex('-')))

	items := []Item{}
	if indent != 0.0 {
		items = append(items, Item{Type: Box, w: indent})
	} else {
		items = append(items, Item{Type: Box, w: 0.0})
	}
	rs := make([]rune, len(glyphs))
	for i, glyph := range glyphs {
		r := sfnt.Cmap.ToUnicode(glyph.ID)
		if r == ' ' {
			spaceFactor := 1.0
			if !FrenchSpacing {
				j := i - 1
				if 0 <= j && (rs[j] == ')' || rs[j] == ']' || rs[j] == '\'' || rs[j] == '"') {
					j--
				}
				if 0 <= j && (j == 0 || !unicode.IsUpper(rs[j-1])) {
					switch rs[j] {
					case '.', '!', '?':
						spaceFactor = SentenceFactor
					case ':':
						spaceFactor = ColonFactor
					case ';':
						spaceFactor = SemicolonFactor
					case ',':
						spaceFactor = CommaFactor
					}
				}
			}
			w := f * float64(glyph.XAdvance)
			y := f * float64(glyph.XAdvance) * SpaceStretch * spaceFactor
			z := f * float64(glyph.XAdvance) * SpaceShrink / spaceFactor
			if items[len(items)-1].Type == Glue {
				items[len(items)-1].w += w
				items[len(items)-1].y += y
				items[len(items)-1].z += z
			} else {
				items = append(items, Item{Type: Glue, w: w, y: y, z: z})
			}
		} else if r == '\u200B' {
			items = append(items, Item{Type: Penalty, w: hyphenWidth, penalty: HyphenPenalty, flagged: true})
		} else if items[len(items)-1].Type == Box {
			items[len(items)-1].w += f * float64(glyph.XAdvance)
			items[len(items)-1].glyphs = append(items[len(items)-1].glyphs, glyph)
		} else {
			items = append(items, Item{Type: Box, w: f * float64(glyph.XAdvance), glyphs: []Glyph{glyph}})
		}
		if r == '-' {
			items = append(items, Item{Type: Penalty, w: 0.0, penalty: HyphenPenalty, flagged: true})
		}
		rs[i] = r
	}
	items = append(items, Item{Type: Glue, w: 0.0, y: math.Inf(1.0), z: 0.0})
	items = append(items, Item{Type: Penalty, w: 0.0, penalty: -InfPenalty})
	return items
}

func (lb *Linebreaker) computeAdjustmentRatio(b int, active *Breakpoint) float64 {
	// compute the adjustment ratio r from a to b
	L := lb.W - active.width
	if lb.items[b].Type == Penalty {
		L += lb.items[b].w
	}
	//j := active.line + 1
	if L < lb.width {
		Y := lb.Y - active.stretch
		if 0.0 < Y {
			return (lb.width - L) / Y
		}
		return math.Inf(1.0)
	} else if lb.width < L {
		Z := lb.Z - active.shrink
		if 0.0 < Z {
			return (lb.width - L) / Z
		}
		return math.Inf(1.0)
	}
	return 0.0
}

func (lb *Linebreaker) computeSum(b int) (float64, float64, float64) {
	// compute tw=(sum w)after(b), ty=(sum y)after(b), and tz=(sum z)after(b)
	W, Y, Z := lb.W, lb.Y, lb.Z
	for i, item := range lb.items[b:] {
		if item.Type == Glue {
			W += item.w
			Y += item.y
			Z += item.z
		} else if item.Type == Box || (item.Type == Penalty && item.penalty == -InfPenalty && 0 < i) {
			break
		}
	}
	return W, Y, Z
}

func (lb *Linebreaker) mainLoop(b int, tolerance float64, exceed bool) {
	j := 0
	item := lb.items[b]
	active := lb.activeNodes.head
	for active != nil {
		Dmin := math.Inf(1.0)
		D := [4]float64{Dmin, Dmin, Dmin, Dmin}
		A := [4]*Breakpoint{}
		R := [4]float64{}
		for active != nil {
			next := active.next
			j = active.line + 1
			ratio := lb.computeAdjustmentRatio(b, active)
			if ratio < -1.0 || item.Type == Penalty && item.penalty == -InfPenalty {
				lb.activeNodes.Remove(active)
				if lb.activeNodes.head == nil && exceed && math.IsInf(Dmin, 1.0) && ratio < -1.0 {
					ratio = -1.0
				}
			}
			if -1.0 <= ratio && ratio <= tolerance {
				// compute demerits d and fitness class c
				demerits := 0.0
				badness := 100.0 * math.Pow(math.Abs(ratio), 3.0)
				if item.Type == Penalty && 0.0 <= item.penalty {
					// positive penalty
					demerits = math.Pow(DemeritsLine+badness+item.penalty, 2.0)
				} else if item.Type == Penalty && item.penalty != -InfPenalty {
					// negative but not a forced break
					demerits = math.Pow(DemeritsLine+badness, 2.0) - math.Pow(item.penalty, 2.0)
				} else {
					// other cases
					demerits = math.Pow(DemeritsLine+badness, 2.0)
				}
				if item.flagged && item.flagged {
					demerits += DemeritsFlagged
				}

				c := 3
				if ratio < -0.5 {
					c = 0
				} else if ratio <= 0.5 {
					c = 1
				} else if ratio <= 1.0 {
					c = 2
				}
				if 1.0 < math.Abs(float64(c-active.fitness)) {
					demerits += DemeritsFitness
				}
				demerits += active.demerits

				if demerits < D[c] {
					D[c] = demerits
					A[c] = active
					R[c] = ratio
					if demerits < Dmin {
						Dmin = demerits
					}
				}
			}
			active = next

			if active != nil && j <= active.line {
				// we omitted (j < j0) as j0 is difficult to know for complex cases
				break
			}
		}

		if Dmin < math.Inf(1.0) {
			// insert new active nodes for breaks from A[c] to index
			W, Y, Z := lb.computeSum(b)
			for c := 0; c < len(D); c++ {
				if D[c] <= Dmin+DemeritsFitness {
					breakpoint := &Breakpoint{
						parent:   A[c],
						position: b,
						line:     A[c].line + 1,
						fitness:  c,
						width:    W,
						stretch:  Y,
						shrink:   Z,
						ratio:    R[c],
						demerits: D[c],
					}
					if active == nil {
						lb.activeNodes.Push(breakpoint)
					} else {
						lb.activeNodes.InsertBefore(breakpoint, active)
					}
				}
			}
		}
	}
}

func Linebreak(sfnt *font.SFNT, ppem float64, glyphs []Glyph, indent, width float64) [][]Glyph {
	const q = 0 // looseness

	items := createItems(sfnt, ppem, indent, glyphs)
	tolerance := Tolerance
	exceed := false

START:
	// create an active node representing the beginning of the paragraph
	lb := NewLinebreaker(items, width)
	// if index is a legal breakpoint then main loop
	for index, item := range items {
		if item.Type == Box {
			lb.W += item.w
		} else if item.Type == Glue {
			if items[index-1].Type == Box {
				lb.mainLoop(index, tolerance, exceed)
			}
			lb.W += item.w
			lb.Y += item.y
			lb.Z += item.z
		} else if item.Type == Penalty && item.penalty != InfPenalty {
			lb.mainLoop(index, tolerance, exceed)
		}
	}

	// do something drastic since there is no feasible solution
	if !exceed && lb.activeNodes.head == nil {
		tolerance = math.Inf(1.0)
		exceed = true
		goto START
	}

	// choose the active node with fewest total demerits
	b := &Breakpoint{demerits: math.Inf(1.0)}
	for a := lb.activeNodes.head; a != nil; a = a.next {
		if a.demerits < b.demerits {
			b = a
		}
	}

	// choose the appropriate active node
	if q != 0 {
		s := 0
		k := b.line
		for a := lb.activeNodes.head; a != nil; a = a.next {
			delta := a.line - k
			if q <= delta && delta < s || s < delta && delta <= q {
				s = delta
				b = a
			} else if delta == s && a.demerits < b.demerits {
				b = a
			}
		}
	}

	// use the chosen node to determine the optimum breakpoint sequence
	breaks := []Break{}
	for b != nil {
		breaks = append(breaks, Break{
			position: b.position,
			ratio:    b.ratio,
		})
		b = b.parent
	}
	// reverse order of breaks
	for i, j := 0, len(breaks)-1; i < j; i, j = i+1, j-1 {
		breaks[i], breaks[j] = breaks[j], breaks[i]
	}
	breaks = breaks[1:]
	if len(breaks) == 0 {
		breaks = append(breaks, Break{position: len(items)})
	}
	for j, br := range breaks {
		fmt.Println(j, br.ratio)
	}

	spaceID := sfnt.GlyphIndex(' ')
	hyphenID := sfnt.GlyphIndex('-')
	fInv := float64(sfnt.Head.UnitsPerEm) / ppem

	j := 0
	glyphLines := [][]Glyph{[]Glyph{}}
	for position, item := range items {
		if position == breaks[j].position {
			if item.Type == Penalty && item.flagged && item.w != 0.0 {
				glyphLines[j] = append(glyphLines[j], Glyph{ID: hyphenID, XAdvance: int32(item.w * fInv)})
			}
			glyphLines = append(glyphLines, []Glyph{})
			if j+1 < len(breaks) {
				j++
			}
		} else if item.Type == Box {
			glyphLines[j] = append(glyphLines[j], item.glyphs...)
		} else if item.Type == Glue {
			width := item.w
			if 0.0 <= breaks[j].ratio {
				width += breaks[j].ratio * item.y
			} else {
				width += breaks[j].ratio * item.z
			}
			glyphLines[j] = append(glyphLines[j], Glyph{ID: spaceID, XAdvance: int32(width * fInv)})
		}
	}
	return glyphLines
}
