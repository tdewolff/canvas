package tex

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	"github.com/tdewolff/canvas"
)

// TeX is a TeX/PGF renderer. Be aware that TeX/PGF does not support transparency of colors.
type TeX struct {
	w             io.Writer
	width, height float64

	style      canvas.Style
	miterLimit float64
	colors     map[color.RGBA]string
}

// New returns a TeX/PGF renderer.
func New(w io.Writer, width, height float64) *TeX {
	fmt.Fprintf(w, "\\begin{pgfpicture}")
	style := canvas.DefaultStyle
	style.StrokeWidth = 0.0
	return &TeX{
		w:          w,
		width:      width,
		height:     height,
		style:      style,
		miterLimit: 10.0,
		colors:     map[color.RGBA]string{},
	}
}

// Close finished and closes the TeX file.
func (r *TeX) Close() error {
	_, err := fmt.Fprintf(r.w, "\n\\end{pgfpicture}")
	return err
}

// Size returns the size of the canvas in millimeters.
func (r *TeX) Size() (float64, float64) {
	return r.width, r.height
}

func (r *TeX) getColor(col color.RGBA) string {
	if name, ok := r.colors[col]; ok {
		return name
	}

	name := fmt.Sprintf("canvasColor%v", len(r.colors))
	A := float64(col.A) / 255.0
	R := float64(col.R) / A
	G := float64(col.G) / A
	B := float64(col.B) / A
	fmt.Fprintf(r.w, "\n\\definecolor{%v}{RGB}{%v,%v,%v}", name, dec(R), dec(G), dec(B))
	r.colors[col] = name
	return name
}

func (r *TeX) writePath(path *canvas.Path) {
	path = path.ReplaceArcs() // sometimes arcs generate errors of the form: Dimension too large
	for scanner := path.Scanner(); scanner.Scan(); {
		end := scanner.End()
		switch scanner.Cmd() {
		case canvas.MoveToCmd:
			fmt.Fprintf(r.w, "\n\\pgfpathmoveto{\\pgfpoint{%vmm}{%vmm}}", dec(end.X), dec(end.Y))
		case canvas.LineToCmd:
			fmt.Fprintf(r.w, "\n\\pgfpathlineto{\\pgfpoint{%vmm}{%vmm}}", dec(end.X), dec(end.Y))
		case canvas.QuadToCmd:
			cp := scanner.CP1()
			fmt.Fprintf(r.w, "\n\\pgfpathquadraticcurveto{\\pgfpoint{%vmm}{%vmm}}{\\pgfpoint{%vmm}{%vmm}}", dec(cp.X), dec(cp.Y), dec(end.X), dec(end.Y))
		case canvas.CubeToCmd:
			cp1, cp2 := scanner.CP1(), scanner.CP2()
			fmt.Fprintf(r.w, "\n\\pgfpathcurveto{\\pgfpoint{%vmm}{%vmm}}{\\pgfpoint{%vmm}{%vmm}}{\\pgfpoint{%vmm}{%vmm}}", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(end.X), dec(end.Y))
		case canvas.ArcToCmd:
			rx, ry, rot, large, sweep := scanner.Arc()
			iLarge := 0
			if large {
				iLarge = 1
			}
			iSweep := 0
			if sweep {
				iSweep = 1
			}
			fmt.Fprintf(r.w, "\n\\pgfpatharcto{%v}{%v}{%v}{%v}{%v}{\\pgfpoint{%v}{%v}}", dec(rx), dec(ry), dec(rot), iLarge, iSweep, dec(end.X), dec(end.Y))
		case canvas.CloseCmd:
			fmt.Fprintf(r.w, "\n\\pgfpathclose")
		}
	}
}

func (r *TeX) setFill(paint canvas.Paint) {
	color := paint.Color
	if color.R != r.style.Fill.Color.R || color.G != r.style.Fill.Color.G || color.B != r.style.Fill.Color.B {
		fmt.Fprintf(r.w, "\n\\pgfsetfillcolor{%v}", r.getColor(color))
		r.style.Fill.Color.R = color.R
		r.style.Fill.Color.G = color.G
		r.style.Fill.Color.B = color.B
	}
	if color.A != r.style.Fill.Color.A {
		fmt.Fprintf(r.w, "\n\\pgfsetfillopacity{%v}", dec(float64(color.A)/255.0))
		r.style.Fill.Color.A = color.A
	}
}

func (r *TeX) setStroke(paint canvas.Paint) {
	color := paint.Color
	if color.R != r.style.Stroke.Color.R || color.G != r.style.Stroke.Color.G || color.B != r.style.Stroke.Color.B {
		fmt.Fprintf(r.w, "\n\\pgfsetstrokecolor{%v}", r.getColor(color))
		r.style.Stroke.Color.R = color.R
		r.style.Stroke.Color.G = color.G
		r.style.Stroke.Color.B = color.B
	}
	if color.A != r.style.Stroke.Color.A {
		fmt.Fprintf(r.w, "\n\\pgfsetstrokeopacity{%v}", dec(float64(color.A)/255.0))
		r.style.Stroke.Color.A = color.A
	}
}

func (r *TeX) setStrokeWidth(width float64) {
	if width != r.style.StrokeWidth {
		fmt.Fprintf(r.w, "\n\\pgfsetlinewidth{%vmm}", dec(width))
		r.style.StrokeWidth = width
	}
}

func (r *TeX) setMiterLimit(limit float64) {
	if limit != r.miterLimit {
		fmt.Fprintf(r.w, "\n\\pgfsetmiterlimit{%v}", dec(limit))
		r.miterLimit = limit
	}
}

func (r *TeX) setStrokeCap(capper canvas.Capper) {
	if capper != r.style.StrokeCapper {
		if _, ok := capper.(canvas.RoundCapper); ok {
			fmt.Fprintf(r.w, "\n\\pgfsetroundcap")
		} else if _, ok := capper.(canvas.SquareCapper); ok {
			fmt.Fprintf(r.w, "\n\\pgfsetrectcap")
		} else if _, ok := capper.(canvas.ButtCapper); ok {
			fmt.Fprintf(r.w, "\n\\pgfsetbuttcap")
		} else {
			panic("TeX: line cap not support")
		}
		r.style.StrokeCapper = capper
	}
}

func (r *TeX) setStrokeJoin(joiner canvas.Joiner) {
	if joiner != r.style.StrokeJoiner {
		if _, ok := joiner.(canvas.BevelJoiner); ok {
			fmt.Fprintf(r.w, "\n\\pgfsetbeveljoin")
		} else if _, ok := joiner.(canvas.RoundJoiner); ok {
			fmt.Fprintf(r.w, "\n\\pgfsetroundjoin")
		} else if miter, ok := joiner.(canvas.MiterJoiner); ok && !math.IsNaN(miter.Limit) && miter.GapJoiner == canvas.BevelJoin {
			fmt.Fprintf(r.w, "\n\\pgfsetmiterjoin")
			r.setMiterLimit(miter.Limit)
		} else {
			panic("TeX: line join not support")
		}
		r.style.StrokeJoiner = joiner
	}
}

func (r *TeX) setDashes(offset float64, dashes []float64) {
	if !float64sEqual(dashes, r.style.Dashes) || offset != r.style.DashOffset {
		if 0 < len(dashes) {
			pgfDashes := ""
			for _, dash := range dashes {
				pgfDashes += fmt.Sprintf("{%vmm}", dec(dash))
			}
			fmt.Fprintf(r.w, "\n\\pgfsetdash{%v}{%vmm}", pgfDashes, dec(offset))
		} else {
			fmt.Fprintf(r.w, "\n\\pgfsetdash{}{0}")
		}
		r.style.DashOffset = offset
		r.style.Dashes = dashes
	}
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *TeX) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	if path.Empty() {
		return
	}

	strokeUnsupported := false
	if m.IsSimilarity() {
		scale := math.Sqrt(math.Abs(m.Det()))
		style.StrokeWidth *= scale
		style.DashOffset, style.Dashes = canvas.ScaleDash(style.StrokeWidth, style.DashOffset, style.Dashes)
	} else {
		strokeUnsupported = true
	}

	if style.HasFill() || style.HasStroke() && !strokeUnsupported {
		r.writePath(path.Copy().Transform(m))
	}

	if style.HasFill() {
		r.setFill(style.Fill)
	}

	if style.HasStroke() && !strokeUnsupported {
		r.setStroke(style.Stroke)
		r.setStrokeWidth(style.StrokeWidth)
		r.setStrokeCap(style.StrokeCapper)
		r.setStrokeJoin(style.StrokeJoiner)
		r.setDashes(style.DashOffset, style.Dashes)
	}
	if style.HasFill() && style.HasStroke() && !strokeUnsupported {
		fmt.Fprintf(r.w, "\n\\pgfusepath{fill,stroke}")
	} else if style.HasFill() {
		fmt.Fprintf(r.w, "\n\\pgfusepath{fill}")
	} else if style.HasStroke() && !strokeUnsupported {
		fmt.Fprintf(r.w, "\n\\pgfusepath{stroke}")
	}

	if style.HasStroke() && strokeUnsupported {
		// stroke settings unsupported by TeX, draw stroke explicitly
		if style.IsDashed() {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner, canvas.Tolerance)
		r.writePath(path.Transform(m))
		r.setFill(style.Stroke)
		fmt.Fprintf(r.w, "\n\\pgfusepath{fill}")
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *TeX) RenderText(text *canvas.Text, m canvas.Matrix) {
	// TODO: (TeX) write text natively
	text.RenderAsPath(r, m, 0.0)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *TeX) RenderImage(img image.Image, m canvas.Matrix) {
	// TODO: (TeX) write image
}
