package canvas

import (
	"fmt"
	"image"
	"io"
	"math"
)

type TeX struct {
	w             io.Writer
	width, height float64
}

// NewTeX creates a TeX/pgf renderer.
func NewTeX(w io.Writer, width, height float64) *TeX {
	fmt.Fprintf(w, `\begin{pgfpicture}`)
	return &TeX{
		w:      w,
		width:  width,
		height: height,
	}
}

func (r *TeX) Close() error {
	_, err := fmt.Fprintf(r.w, `\end{pgfpicture}`)
	return err
}

func (r *TeX) Size() (float64, float64) {
	return r.width, r.height
}

func (r *TeX) RenderPath(path *Path, style Style, m Matrix) {
	if path.Empty() {
		return
	}
	path = path.Transform(m)
	Precision = 4

	fmt.Fprintf(r.w, "\n")
	path.Iterate(func(start, end Point) {
		fmt.Fprintf(r.w, `\pgfpathmoveto{\pgfpoint{%vmm}{%vmm}}`, dec(end.X), dec(end.Y))
	}, func(start, end Point) {
		fmt.Fprintf(r.w, `\pgfpathlineto{\pgfpoint{%vmm}{%vmm}}`, dec(end.X), dec(end.Y))
	}, func(start, cp, end Point) {
		fmt.Fprintf(r.w, `\pgfpathquadraticcurveto{\pgfpoint{%vmm}{%vmm}}{\pgfpoint{%vmm}{%vmm}}`, dec(cp.X), dec(cp.Y), dec(end.X), dec(end.Y))
	}, func(start, cp1, cp2, end Point) {
		fmt.Fprintf(r.w, `\pgfpathcurveto{\pgfpoint{%vmm}{%vmm}}{\pgfpoint{%vmm}{%vmm}}{\pgfpoint{%vmm}{%vmm}}`, dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(end.X), dec(end.Y))
	}, func(start Point, rx, ry, rot float64, large, sweep bool, end Point) {
		iLarge := 0
		if large {
			iLarge = 1
		}
		iSweep := 0
		if sweep {
			iSweep = 1
		}
		fmt.Fprintf(r.w, `\pgfpatharcto{%vmm}{%vmm}{%v}{%v}{%v}{\pgfpoint{%vmm}{%vmm}}`, dec(rx), dec(ry), dec(rot), iLarge, iSweep, dec(end.X), dec(end.Y))
	}, func(start, end Point) {
		fmt.Fprintf(r.w, `\pgfpathclose`)
	})

	if style.FillColor.A != 0 {
		//a := float64(style.FillColor.A) / 255.0
		//fmt.Fprintf(r.w, `\pgfsetcolor{\color[rgb]{%v,%v,%v}}`, dec(float64(style.FillColor.R)/255.0/a), dec(float64(style.FillColor.G)/255.0/a), dec(float64(style.FillColor.B)/255.0/a))
		fmt.Fprintf(r.w, `\pgfusepath{fill}`)
	}
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		if _, ok := style.StrokeCapper.(RoundCapper); ok {
			fmt.Fprintf(r.w, `\pgfsetroundcap`)
		} else if _, ok := style.StrokeCapper.(SquareCapper); ok {
			fmt.Fprintf(r.w, `\pgfsetrectcap`)
		} else if _, ok := style.StrokeCapper.(ButtCapper); ok {
			fmt.Fprintf(r.w, `\pgfsetbuttcap`)
		} else {
			panic("TeX: line cap not support")
		}

		if _, ok := style.StrokeJoiner.(BevelJoiner); ok {
			fmt.Fprintf(r.w, `\pgfsetbeveljoin`)
		} else if _, ok := style.StrokeJoiner.(RoundJoiner); ok {
			fmt.Fprintf(r.w, `\pgfsetroundjoin`)
		} else if miter, ok := style.StrokeJoiner.(MiterJoiner); ok && !math.IsNaN(miter.Limit) && miter.GapJoiner == BevelJoin {
			fmt.Fprintf(r.w, `\pgfsetmiterjoin`)
			fmt.Fprintf(r.w, `\pgfsetmiterlimit{%v}`, dec(miter.Limit))
		} else {
			panic("TeX: line join not support")
		}

		if 0 < len(style.Dashes) {
			dashes := ""
			for _, dash := range style.Dashes {
				dashes += fmt.Sprintf("{%v}", dec(dash))
			}
			fmt.Fprintf(r.w, `\pgfsetdash{%v}{%v}`, dashes, dec(style.DashOffset))
		}

		fmt.Fprintf(r.w, `\pgfsetlinewidth{%v}`, dec(style.StrokeWidth))
		//a := float64(style.StrokeColor.A) / 255.0
		//fmt.Fprintf(r.w, `\pgfsetcolor{\color[rgb]{%v,%v,%v}}`, dec(float64(style.StrokeColor.R)/255.0/a), dec(float64(style.StrokeColor.G)/255.0/a), dec(float64(style.StrokeColor.B)/255.0/a))
		fmt.Fprintf(r.w, `\pgfusepath{stroke}`)
	}
}

func (r *TeX) RenderText(text *Text, m Matrix) {
	// TODO: (TeX) write text natively
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := DefaultStyle
		style.FillColor = colors[i]
		r.RenderPath(path, style, m)
	}
}

func (r *TeX) RenderImage(img image.Image, m Matrix) {
	// TODO: (TeX) write image
}
