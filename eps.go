package canvas

import (
	"fmt"
	"image"
	"image/color"
	"io"
)

var psEllipseDef = `/ellipse {
/rot exch def
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
rot rotate
xrad yrad scale
0 0 1 startangle endangle arc
savematrix setmatrix
} def /ellipsen {
/rot exch def
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
rot rotate
xrad yrad scale
0 0 1 startangle endangle arcn
savematrix setmatrix
} def`

type eps struct {
	*Context
	w     io.Writer
	color color.RGBA
}

func EPS(w io.Writer, width, height float64) *eps {
	fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n%%%%BoundingBox: 0 0 %v %v\n", dec(width), dec(height))
	fmt.Fprintf(w, psEllipseDef)
	// TODO: (EPS) generate and add preview

	r := &eps{
		Context: nil,
		w:       w,
		color:   Black,
	}
	r.Context = newContext(r, width, height)
	return r
}

func (r *eps) SetColor(color color.RGBA) {
	if color != r.color {
		fmt.Fprintf(r.w, " %v %v %v setrgbcolor", dec(float64(color.R)/255.0), dec(float64(color.G)/255.0), dec(float64(color.B)/255.0))
		r.color = color
	}
}

func (r *eps) renderPath(path *Path, style Style, m Matrix) {
	// TODO: (EPS) test ellipse, rotations etc
	// TODO: (EPS) add drawState support
	r.SetColor(style.FillColor)
	r.w.Write([]byte(" "))
	r.w.Write([]byte(path.Transform(m).ToPS()))
	r.w.Write([]byte(" fill"))
}

func (r *eps) renderText(text *Text, m Matrix) {
	// TODO: (EPS) write text natively
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := DefaultStyle
		style.FillColor = colors[i]
		r.renderPath(path, style, m)
	}
}

func (r *eps) renderImage(img image.Image, m Matrix) {
	// TODO: (EPS) write image
}
