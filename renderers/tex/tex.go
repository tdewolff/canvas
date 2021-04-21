package tex

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	"github.com/tdewolff/canvas"
)

// Writer writes the canvas as a TeX file using PGF (\usepackage{pgf}).
func Writer(w io.Writer, c *canvas.Canvas) error {
	tex := New(w, c.W, c.H)
	c.Render(tex)
	return tex.Close()
}

// TeX is a TeX/PGF renderer. Be aware that TeX/PGF does not support transparency of colors.
type TeX struct {
	w             io.Writer
	width, height float64

	style  canvas.Style
	colors map[color.RGBA]string
}

// New returns a TeX/PGF renderer.
func New(w io.Writer, width, height float64) *TeX {
	fmt.Fprintf(w, "\\begin{pgfpicture}")
	return &TeX{
		w:      w,
		width:  width,
		height: height,
		style:  canvas.DefaultStyle,
		colors: map[color.RGBA]string{},
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

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *TeX) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	if path.Empty() {
		return
	}
	path = path.Transform(m)
	path = path.ReplaceArcs()

	path.Iterate(func(start, end canvas.Point) {
		fmt.Fprintf(r.w, "\n\\pgfpathmoveto{\\pgfpoint{%vmm}{%vmm}}", dec(end.X), dec(end.Y))
	}, func(start, end canvas.Point) {
		fmt.Fprintf(r.w, "\n\\pgfpathlineto{\\pgfpoint{%vmm}{%vmm}}", dec(end.X), dec(end.Y))
	}, func(start, cp, end canvas.Point) {
		fmt.Fprintf(r.w, "\n\\pgfpathquadraticcurveto{\\pgfpoint{%vmm}{%vmm}}{\\pgfpoint{%vmm}{%vmm}}", dec(cp.X), dec(cp.Y), dec(end.X), dec(end.Y))
	}, func(start, cp1, cp2, end canvas.Point) {
		fmt.Fprintf(r.w, "\n\\pgfpathcurveto{\\pgfpoint{%vmm}{%vmm}}{\\pgfpoint{%vmm}{%vmm}}{\\pgfpoint{%vmm}{%vmm}}", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(end.X), dec(end.Y))
	}, func(start canvas.Point, rx, ry, rot float64, large, sweep bool, end canvas.Point) {
		iLarge := 0
		if large {
			iLarge = 1
		}
		iSweep := 0
		if sweep {
			iSweep = 1
		}
		fmt.Fprintf(r.w, "\n\\pgfpatharcto{%vmm}{%vmm}{%v}{%v}{%v}{\\pgfpoint{%vmm}{%vmm}}", dec(rx), dec(ry), dec(rot), iLarge, iSweep, dec(end.X), dec(end.Y))
	}, func(start, end canvas.Point) {
		fmt.Fprintf(r.w, "\n\\pgfpathclose")
	})

	fill := style.FillColor.A != 0
	stroke := style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth

	if fill {
		if style.FillColor.R != r.style.FillColor.R || style.FillColor.G != r.style.FillColor.G || style.FillColor.B != r.style.FillColor.B {
			fmt.Fprintf(r.w, "\n\\pgfsetfillcolor{%v}", r.getColor(style.FillColor))
		}
		if style.FillColor.A != r.style.FillColor.A {
			fmt.Fprintf(r.w, "\n\\pgfsetfillopacity{%v}", dec(float64(style.FillColor.A)/255.0))
		}
	}

	if stroke {
		if style.StrokeCapper != r.style.StrokeCapper {
			if _, ok := style.StrokeCapper.(canvas.RoundCapper); ok {
				fmt.Fprintf(r.w, "\n\\pgfsetroundcap")
			} else if _, ok := style.StrokeCapper.(canvas.SquareCapper); ok {
				fmt.Fprintf(r.w, "\n\\pgfsetrectcap")
			} else if _, ok := style.StrokeCapper.(canvas.ButtCapper); ok {
				fmt.Fprintf(r.w, "\n\\pgfsetbuttcap")
			} else {
				panic("TeX: line cap not support")
			}
		}

		if style.StrokeJoiner != r.style.StrokeJoiner {
			if _, ok := style.StrokeJoiner.(canvas.BevelJoiner); ok {
				fmt.Fprintf(r.w, "\n\\pgfsetbeveljoin")
			} else if _, ok := style.StrokeJoiner.(canvas.RoundJoiner); ok {
				fmt.Fprintf(r.w, "\n\\pgfsetroundjoin")
			} else if miter, ok := style.StrokeJoiner.(canvas.MiterJoiner); ok && !math.IsNaN(miter.Limit) && miter.GapJoiner == canvas.BevelJoin {
				fmt.Fprintf(r.w, "\n\\pgfsetmiterjoin")
				fmt.Fprintf(r.w, "\n\\pgfsetmiterlimit{%vmm}", dec(miter.Limit))
			} else {
				panic("TeX: line join not support")
			}
		}

		if !float64sEqual(style.Dashes, r.style.Dashes) || style.DashOffset != r.style.DashOffset {
			if 0 < len(style.Dashes) {
				dashes := ""
				for _, dash := range style.Dashes {
					dashes += fmt.Sprintf("{%vmm}", dec(dash))
				}
				fmt.Fprintf(r.w, "\n\\pgfsetdash{%v}{%vmm}", dashes, dec(style.DashOffset))
			} else {
				fmt.Fprintf(r.w, "\n\\pgfsetdash{}{0}")
			}
		}

		if style.StrokeWidth != r.style.StrokeWidth {
			fmt.Fprintf(r.w, "\n\\pgfsetlinewidth{%vmm}", dec(style.StrokeWidth))
		}

		if style.StrokeColor.R != r.style.StrokeColor.R || style.StrokeColor.G != r.style.StrokeColor.G || style.StrokeColor.B != r.style.StrokeColor.B {
			fmt.Fprintf(r.w, "\n\\pgfsetstrokecolor{%v}", r.getColor(style.StrokeColor))
		}
		if style.StrokeColor.A != r.style.StrokeColor.A {
			fmt.Fprintf(r.w, "\n\\pgfsetstrokeopacity{%v}", dec(float64(style.StrokeColor.A)/255.0))
		}
	}
	if fill && stroke {
		fmt.Fprintf(r.w, "\n\\pgfusepath{fill,stroke}")
	} else if fill {
		fmt.Fprintf(r.w, "\n\\pgfusepath{fill}")
	} else if stroke {
		fmt.Fprintf(r.w, "\n\\pgfusepath{stroke}")
	}
	r.style = style
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *TeX) RenderText(text *canvas.Text, m canvas.Matrix) {
	// TODO: (TeX) write text natively
	text.RenderAsPath(r, m, canvas.DefaultResolution)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *TeX) RenderImage(img image.Image, m canvas.Matrix) {
	// TODO: (TeX) write image
}
