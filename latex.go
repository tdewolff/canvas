// +build !latex

package canvas

import (
	"fmt"

	"github.com/go-latex/latex/drawtex"
	"github.com/go-latex/latex/font/ttf"
	"github.com/go-latex/latex/mtex"
	"github.com/go-latex/latex/tex"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

func ParseLaTeX(s string) (*Path, error) {
	// TODO: use original LaTeX font?
	//regular, err := sfnt.Parse(stix2textregular.TTF)
	//if err != nil {
	//	return nil, fmt.Errorf("could not load font: %w", err)
	//}
	//italic, err := sfnt.Parse(stix2textitalic.TTF)
	//if err != nil {
	//	return nil, fmt.Errorf("could not load font: %w", err)
	//}
	//bold, err := sfnt.Parse(stix2textbold.TTF)
	//if err != nil {
	//	return nil, fmt.Errorf("could not load font: %w", err)
	//}
	//bolditalic, err := sfnt.Parse(stix2textbolditalic.TTF)
	//if err != nil {
	//	return nil, fmt.Errorf("could not load font: %w", err)
	//}

	c := drawtex.New()
	fontsize := 12.0
	//fonts := &ttf.Fonts{
	//	Default: regular,
	//	Rm:      regular,
	//	It:      italic,
	//	Bf:      bold,
	//	BfIt:    bolditalic,
	//}
	node, err := mtex.Parse(s, fontsize, 72.0, ttf.New(c))
	if err != nil {
		return nil, fmt.Errorf("could not parse expression: %w", err)
	}

	var sh tex.Ship
	sh.Call(0, 0, node.(tex.Tree))

	p := &Path{}
	height := 1.5 * node.Height()
	for _, op := range c.Ops() {
		switch op := op.(type) {
		case drawtex.GlyphOp:
			glyph := op.Glyph
			buf := &sfnt.Buffer{}
			ppem := fixed.Int26_6(0.5 + glyph.Size*64.0)

			segs, err := glyph.Font.LoadGlyph(buf, glyph.Num, ppem, nil)
			if err != nil {
				return nil, fmt.Errorf("unknown glyph: %v", glyph.Num)
			}
			for _, seg := range segs {
				switch seg.Op {
				case sfnt.SegmentOpMoveTo:
					p.MoveTo(op.X+fromI26_6(seg.Args[0].X), height-op.Y-fromI26_6(seg.Args[0].Y))
				case sfnt.SegmentOpLineTo:
					p.LineTo(op.X+fromI26_6(seg.Args[0].X), height-op.Y-fromI26_6(seg.Args[0].Y))
				case sfnt.SegmentOpQuadTo:
					p.QuadTo(op.X+fromI26_6(seg.Args[0].X), height-op.Y-fromI26_6(seg.Args[0].Y), op.X+fromI26_6(seg.Args[1].X), height-op.Y-fromI26_6(seg.Args[1].Y))
				case sfnt.SegmentOpCubeTo:
					p.CubeTo(op.X+fromI26_6(seg.Args[0].X), height-op.Y-fromI26_6(seg.Args[0].Y), op.X+fromI26_6(seg.Args[1].X), height-op.Y-fromI26_6(seg.Args[1].Y), op.X+fromI26_6(seg.Args[2].X), height-op.Y-fromI26_6(seg.Args[2].Y))
				}
			}
			if 0 < len(segs) {
				p.Close()
			}
		case drawtex.RectOp:
			p.MoveTo(op.X1, height-op.Y1)
			p.LineTo(op.X2, height-op.Y1)
			p.LineTo(op.X2, height-op.Y2)
			p.LineTo(op.X1, height-op.Y2)
			p.Close()
		default:
			return nil, fmt.Errorf("unknown operation: %v", op)
		}
	}
	return p, nil
}
