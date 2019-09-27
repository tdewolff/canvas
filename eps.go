package canvas

import (
	"fmt"
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

type epsWriter struct {
	io.Writer
	color color.RGBA
}

func newEPSWriter(writer io.Writer, width, height float64) *epsWriter {
	w := &epsWriter{
		Writer: writer,
		color:  Black,
	}

	fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n%%%%BoundingBox: 0 0 %v %v\n", dec(width), dec(height))
	fmt.Fprintf(w, psEllipseDef)

	// TODO: (EPS) generate and add preview

	return w
}

func (w *epsWriter) SetColor(color color.RGBA) {
	if color != w.color {
		fmt.Fprintf(w, " %v %v %v setrgbcolor", dec(float64(color.R)/255.0), dec(float64(color.G)/255.0), dec(float64(color.B)/255.0))
		w.color = color
	}
}
