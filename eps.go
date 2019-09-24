package canvas

import (
	"fmt"
	"image/color"
	"io"
)

var psEllipseDef = `/ellipse {
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
xrad yrad scale
0 0 1 startangle endangle arc
savematrix setmatrix
} def /ellipsen {
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
xrad yrad scale
0 0 1 startangle endangle arcn
savematrix setmatrix
} def`

type epsWriter struct {
	io.Writer
	color color.RGBA
}

func newEPSWriter(writer io.Writer, width, height float64) *epsWriter {
	w := &epsWriter{
		Writer: writer,
		color:  Black,
	}

	fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n%%%%BoundingBox: 0 0 %v %v\n", num(width), num(height))
	fmt.Fprintf(w, psEllipseDef)

	// TODO: generate preview

	return w
}

func (w *epsWriter) SetColor(color color.RGBA) {
	if color != w.color {
		// TODO: transparency
		fmt.Fprintf(w, " %v %v %v setrgbcolor", num(float64(color.R)/255.0), num(float64(color.G)/255.0), num(float64(color.B)/255.0))
		w.color = color
	}
}
