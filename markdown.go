package canvas

import (
	"io"
	"math"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

type mdRenderer struct {
	c             Renderer
	m             Matrix
	rect          Rect
	roman         *FontFamily
	fontSize      float64
	thematicBreak func() float64

	faceDefault *FontFace
}

type mdState struct {
	rect     Rect
	font     *FontFamily
	fontSize float64
}

func (r *mdRenderer) renderInline(src []byte, n ast.Node, rt *RichText, s mdState) {
	switch v := n.(type) {
	case *ast.Text:
		rt.Write(v.Value(src))
		if v.HardLineBreak() || v.SoftLineBreak() {
			rt.WriteByte('\n')
		}
	case *ast.TextBlock:
		rt.Write(v.Lines().Value(src))
	case *ast.Emphasis:
		switch v.Level {
		case 1:
			rt.SetFace(s.font.Face(s.fontSize, Black, FontItalic))
		case 2:
			rt.SetFace(s.font.Face(s.fontSize, Black, FontBold))
		case 3:
			rt.SetFace(s.font.Face(s.fontSize, Black, FontBold|FontItalic))
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderInline(src, child, rt, s)
		}
		rt.SetFace(s.font.Face(s.fontSize, Black, FontRegular))
	}
}

func (r *mdRenderer) renderBlock(src []byte, n ast.Node, marginTop float64) (float64, error) {
	margin := 1.0
	font := r.roman
	fontSize := r.fontSize
	switch v := n.(type) {
	case *ast.Heading:
		switch v.Level {
		case 1:
			fontSize *= 2.0
			margin = 0.67
		case 2:
			fontSize *= 1.5
			margin = 0.83
		case 3:
			fontSize *= 1.3
		case 4:
			margin = 1.33
		case 5:
			fontSize *= 0.8
			margin = 1.67
		case 6:
			fontSize *= 0.7
			margin = 2.33
		}
	case *ast.List:
		if _, ok := n.Parent().(*ast.ListItem); ok {
			margin = 0.0
		}
	case *ast.ListItem:
		margin = 0.0
	}
	margin *= fontSize * mmPerPt
	if !math.IsInf(marginTop, 0) {
		r.rect.Y1 -= math.Max(marginTop, margin)
	}

	switch v := n.(type) {
	case *ast.List:
		x0 := r.rect.X0
		r.rect.X0 += r.faceDefault.Size * 1.5

		spacing := fontSize * mmPerPt * 0.15
		metrics := r.faceDefault.Metrics()
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child != n.FirstChild() {
				r.rect.Y1 -= spacing
			}

			if item, ok := child.(*ast.ListItem); ok {
				_ = v.Marker
				if v.Start != 0 {
					_ = v.Start + item.Offset
				}

				x := r.rect.X0 - r.faceDefault.Size*1.25
				y := r.rect.Y1 - metrics.Ascent + metrics.CapHeight/2.0
				dash := Line(r.faceDefault.Size/2.0, 0.0)
				style := DefaultStyle
				style.Stroke = Paint{Color: Black}
				style.StrokeWidth = 0.3
				r.c.RenderPath(dash, style, Identity.Translate(x, y))
			}

			if _, err := r.renderBlock(src, child, 0.0); err != nil {
				return margin, err
			}
		}
		r.rect.X0 = x0
		return margin, nil
	case *ast.ListItem:
		spacing := fontSize * mmPerPt * 0.15
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child != n.FirstChild() {
				r.rect.Y1 -= spacing
			}

			if _, err := r.renderBlock(src, child, spacing); err != nil {
				return 0.0, err
			}
		}
	case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
		s := mdState{
			rect:     r.rect,
			font:     font,
			fontSize: fontSize,
		}
		rt := NewRichText(font.Face(fontSize, Black, FontRegular))
		for child := n.FirstChild(); child != nil && child.Type() == ast.TypeInline; child = child.NextSibling() {
			r.renderInline(src, child, rt, s)
		}

	NextPage:
		text := rt.ToText(r.rect.W(), r.rect.H(), Left, Top, &TextOptions{
			LineStretch: 0.15,
		})
		r.c.RenderText(text, Identity.Translate(r.rect.X0, r.rect.Y1))
		r.rect.Y1 -= text.Height

		if text.OverflowsY {
			if 0.0 < text.Height && r.thematicBreak != nil {
				y := r.thematicBreak()
				r.rect.Y1 = r.m.Dot(Point{r.rect.X0, y}).Y
				goto NextPage
			}
			return margin, io.EOF
		}
	case *ast.ThematicBreak:
		if r.thematicBreak != nil {
			y := r.thematicBreak()
			r.rect.Y1 = r.m.Dot(Point{r.rect.X0, y}).Y
			return math.Inf(-1), nil
		} else {
			style := DefaultStyle
			style.Stroke = Paint{Color: Black}
			style.StrokeWidth = 0.3
			r.c.RenderPath(Line(r.rect.X1-r.rect.X0, 0.0), style, Identity.Translate(r.rect.X0, r.rect.Y1))
		}
	}
	if r.rect.Y1 <= r.rect.Y0 {
		return margin, io.EOF
	}
	return margin, nil
}

func (r *mdRenderer) Render(_ io.Writer, src []byte, n ast.Node) error {
	r.faceDefault = r.roman.Face(r.fontSize, Black, FontRegular)

	var err error
	marginBottom := math.Inf(-1)
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if marginBottom, err = r.renderBlock(src, child, marginBottom); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func (r *mdRenderer) AddOptions(options ...renderer.Option) {}

type MarkdownOptions struct {
	Font          *FontFamily
	FontSize      float64
	ThematicBreak func() float64
}

func RenderMarkdown(ctx *Context, rect Rect, text string, opts *MarkdownOptions) (float64, error) {
	if opts == nil {
		opts = &MarkdownOptions{
			FontSize: 12.0,
		}
	}
	if opts.Font == nil {
		opts.Font = NewFontFamily("sans-serif")
		if err := opts.Font.LoadSystemFont("sans-serif", FontRegular); err != nil {
			return 0.0, err
		} else if err := opts.Font.LoadSystemFont("sans-serif", FontItalic); err != nil {
			return 0.0, err
		} else if err := opts.Font.LoadSystemFont("sans-serif", FontBold); err != nil {
			return 0.0, err
		} else if err := opts.Font.LoadSystemFont("sans-serif", FontBold|FontItalic); err != nil {
			return 0.0, err
		}
	}

	rect = rect.Transform(ctx.CoordSystemView())
	renderer := &mdRenderer{
		c:             ctx.Renderer,
		m:             ctx.CoordSystemView(),
		rect:          rect,
		roman:         opts.Font,
		fontSize:      opts.FontSize,
		thematicBreak: opts.ThematicBreak,
	}
	md := goldmark.New(goldmark.WithRenderer(renderer))
	err := md.Convert([]byte(text), nil)
	return rect.Y1 - renderer.rect.Y1, err

}
