package canvas

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
	ff             FontFace
	s              string
	indent         float64
	hAlign, vAlign TextAlign
}

func NewText(ff FontFace, s string) *Text {
	return &Text{
		ff:     ff,
		s:      s,
		indent: 0.0,
		width:  0.0,
		height: 0.0,
		hAlign: Left,
		vAlign: Top,
	}
}

func (t *Text) SetIndent(indent float64) {
	t.indent = indent
}

func (t *Text) SetAlignment(hAlign, vAlign TextAlign) {
	t.hAlign = hAlign
	t.vAlign = vAlign
}

func (t *Text) ToPath() *Path {
	panic("not implemented")
}

func (t *Text) ToSVG() string {
	// TODO: implement text to SVG using <text> and <tspan> with indent, spacing between characters and lines etc.
	panic("not implemented")
}
